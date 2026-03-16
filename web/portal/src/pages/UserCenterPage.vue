<template>
  <div class="portal-center">
    <header class="portal-topbar border-bottom bg-white">
      <div class="container-fluid">
        <div class="portal-topbar-main">
          <div>
            <div class="portal-title">用户中心</div>
            <div class="portal-subtitle">维护个人资料、安全设置、身份绑定与设备。</div>
          </div>
          <BButton variant="outline-secondary" size="sm" @click="logout">退出登录</BButton>
        </div>
      </div>
    </header>

    <main class="container-fluid py-4">
      <ToastHost />

      <div class="row g-4">
        <aside class="col-lg-3">
          <div class="portal-nav-card">
            <button v-for="section in sections" :key="section.id" type="button" class="portal-nav-link" @click="scrollTo(section.id)">
              {{ section.label }}
            </button>
          </div>
        </aside>

        <section class="col-lg-9">
          <div class="d-grid gap-4 portal-sections">
            <div id="profile-basic" class="info-card">
              <div class="section-title">基本信息</div>
              <BForm @submit.prevent="saveProfile">
                <div class="row g-3">
                  <div class="col-md-6">
                    <label class="form-label">姓名</label>
                    <BFormInput v-model="profile.name" />
                  </div>
                  <div class="col-md-6">
                    <label class="form-label">用户名</label>
                    <BFormInput v-model="profile.username" />
                  </div>
                  <div class="col-md-6">
                    <label class="form-label">邮箱</label>
                    <BFormInput v-model="profile.email" type="email" />
                  </div>
                  <div class="col-md-6">
                    <label class="form-label">手机</label>
                    <BFormInput v-model="profile.phoneNumber" />
                  </div>
                </div>
                <div class="d-flex justify-content-end mt-3">
                  <BButton type="submit" variant="primary" size="sm">保存基本信息</BButton>
                </div>
              </BForm>
            </div>

            <div id="profile-login" class="info-card">
              <div class="section-title">登录设置</div>
              <div class="row g-3">
                <div class="col-lg-6">
                  <div class="detail-card h-100">
                    <div class="record-meta mb-3">密码登录：{{ detail?.passwordCredential ? '已启用' : '未配置' }}</div>
                    <BForm @submit.prevent="savePassword">
                      <BFormInput v-model="passwordForm.currentPassword" type="password" placeholder="当前密码" class="mb-2" />
                      <BFormInput v-model="passwordForm.newPassword" type="password" placeholder="新密码" class="mb-3" />
                      <BButton type="submit" variant="outline-primary" size="sm">更新密码</BButton>
                    </BForm>
                  </div>
                </div>
                <div class="col-lg-6">
                  <div class="detail-card h-100">
                    <div class="d-flex justify-content-between align-items-center mb-2">
                      <div class="record-meta mb-0">通行密钥：{{ loginPasskeys.length }} · {{ passkeyLoginEnabled ? '已启用登录' : '已关闭登录' }}</div>
                      <div class="d-flex gap-2">
                        <BButton size="sm" :variant="passkeyLoginEnabled ? 'outline-danger' : 'outline-secondary'" :disabled="!loginPasskeys.length" @click="togglePasskeyLogin(!passkeyLoginEnabled)">
                          {{ passkeyLoginEnabled ? '关闭登录' : '启用登录' }}
                        </BButton>
                        <BButton size="sm" variant="outline-primary" @click="registerPasskey('passkey')">注册通行密钥</BButton>
                      </div>
                    </div>
                    <div v-if="!loginPasskeys.length" class="record-meta">当前没有通行密钥，注册后才可启用通行密钥登录。</div>
                    <div v-for="passkey in loginPasskeys" :key="passkey.id" class="record-row">
                      <div>
                        <strong>{{ passkey.identifier || '通行密钥' }}</strong>
                        <div class="record-meta small-break">{{ passkey.publicKeyId }}</div>
                      </div>
                      <BButton size="sm" variant="outline-danger" @click="deletePasskey(passkey.id)">删除</BButton>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div id="profile-binding" class="info-card">
              <div class="section-title">账号绑定</div>
              <div class="record-meta mb-3">绑定外部 OAuth / OIDC 身份的 Subject，用于后续识别已有账号。</div>
              <div class="row g-3">
                <div class="col-lg-7">
                  <div v-if="!(detail?.bindings?.length ?? 0)" class="detail-card">
                    <div class="record-meta">当前没有第三方身份绑定。</div>
                  </div>
                  <div v-for="binding in detail?.bindings || []" :key="binding.id" class="record-card mb-2">
                    <div class="record-head">
                      <strong>{{ binding.providerName || binding.externalIdpId }}</strong>
                      <code>{{ binding.subject }}</code>
                    </div>
                    <div class="record-meta">Issuer：{{ binding.issuer }}</div>
                    <div class="record-meta">绑定时间：{{ formatDateTime(binding.createdAt) }}</div>
                    <div class="record-actions">
                      <BButton size="sm" variant="outline-danger" @click="deleteBinding(binding.id)">解绑</BButton>
                    </div>
                  </div>
                </div>
                <div class="col-lg-5">
                  <BForm @submit.prevent="createBinding">
                    <label class="form-label">外部 IdP</label>
                    <BFormSelect v-model="bindingForm.externalIdpId" class="mb-2" @update:model-value="syncBindingIssuer">
                      <option value="">请选择</option>
                      <option v-for="provider in detail?.externalIdps || []" :key="provider.id" :value="provider.id">{{ provider.name }}</option>
                    </BFormSelect>
                    <label class="form-label">Issuer</label>
                    <BFormInput v-model="bindingForm.issuer" class="mb-2" />
                    <label class="form-label">Subject</label>
                    <BFormInput v-model="bindingForm.subject" class="mb-3" />
                    <BButton type="submit" variant="primary" size="sm">新增绑定</BButton>
                  </BForm>
                </div>
              </div>
            </div>

            <div id="profile-mfa" class="info-card">
              <div class="section-title">两步验证</div>
              <div class="record-list">
                <div v-for="item in mfaRows" :key="item.id" class="record-row">
                  <div>
                    <strong>{{ item.label }}</strong>
                    <div class="record-meta">{{ item.summary }}</div>
                  </div>
                  <div class="d-flex gap-2">
                    <BButton
                      v-if="item.id === 'email_code' || item.id === 'sms_code'"
                      size="sm"
                      :variant="item.enabled ? 'outline-danger' : 'outline-primary'"
                      @click="toggleSimpleMFA(item.id, !item.enabled)"
                    >
                      {{ item.enabled ? '关闭' : '开启' }}
                    </BButton>
                    <BButton v-else size="sm" variant="outline-primary" @click="openMFAModal(item.id)">
                      {{ item.enabled ? '配置' : '开启' }}
                    </BButton>
                  </div>
                </div>
              </div>
            </div>

            <div id="profile-device" class="info-card">
              <div class="section-title">会话管理</div>
              <div v-if="!(detail?.devices?.length ?? 0)" class="detail-card">
                <div class="record-meta">当前没有设备记录。</div>
              </div>
              <div v-for="device in detail?.devices || []" :key="device.id" class="record-card mb-2">
                <div class="record-head">
                  <strong>{{ device.label }}</strong>
                  <div class="d-flex align-items-center gap-2">
                    <span v-if="device.trusted" class="badge text-bg-primary">可信</span>
                    <span class="badge" :class="device.online ? 'text-bg-success' : 'text-bg-secondary'">{{ device.online ? '在线' : '离线' }}</span>
                  </div>
                </div>
                <div class="record-meta">上次登录 IP：{{ formatIPLine(device.ipAddress, device.ipLocation) }}</div>
                <div class="record-meta">上次登录时间：{{ formatDateTime(device.lastLoginAt) }}</div>
                <div class="record-meta">初次登录日期：{{ formatDateTime(device.firstLoginAt) }}</div>
                <div v-if="device.fingerprint" class="record-meta small-break">设备指纹：{{ device.fingerprint }}</div>
                <div v-if="device.trusted" class="record-actions">
                  <BButton size="sm" variant="outline-danger" @click="untrustDevice(device.id)">取消可信</BButton>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </main>

    <BModal v-model="showMFAModal" :title="mfaModalTitle" centered>
      <template v-if="mfaModalType === 'totp'">
        <div v-if="activeTotpEnrollment" class="record-meta mb-3">当前已配置身份验证器，可重新生成或关闭。</div>
        <div v-else class="record-meta mb-3">当前没有已激活的身份验证器。</div>
        <div v-if="totpQRCode" class="text-center mb-3">
          <img :src="totpQRCode" alt="身份验证器二维码" class="img-fluid border rounded bg-white p-2" />
        </div>
        <div v-if="totpEnrollment.manualEntryKey" class="record-meta mb-3">手动输入密钥：{{ totpEnrollment.manualEntryKey }}</div>
        <BForm @submit.prevent="verifyTotp">
          <BFormInput v-model="totpCode" placeholder="输入 6 位验证码" class="mb-3" />
          <div class="d-flex gap-2">
            <BButton type="button" variant="outline-secondary" @click="generateTotp">生成身份验证器配置</BButton>
            <BButton type="submit" variant="primary">确认并启用</BButton>
            <BButton v-if="activeTotpEnrollment" type="button" variant="outline-danger" @click="disableTotp">关闭</BButton>
          </div>
        </BForm>
      </template>

      <template v-else-if="mfaModalType === 'u2f'">
        <div class="record-meta mb-3">安全密钥与通行密钥共用同一套 WebAuthn 凭据。</div>
        <div class="record-meta mb-3">当前已注册 {{ u2fPasskeys.length }} 把安全密钥。</div>
        <div v-for="passkey in u2fPasskeys" :key="passkey.id" class="record-row mb-2">
          <div>
            <strong>{{ passkey.identifier || '安全密钥' }}</strong>
            <div class="record-meta small-break">{{ passkey.publicKeyId }}</div>
          </div>
          <div class="d-flex align-items-center gap-2">
            <code>{{ formatDateTime(passkey.createdAt) }}</code>
            <BButton size="sm" variant="outline-danger" @click="deletePasskey(passkey.id)">删除</BButton>
          </div>
        </div>
        <div v-if="!u2fPasskeys.length" class="record-meta mb-3">当前没有已注册的安全密钥。</div>
        <BButton variant="outline-primary" @click="registerPasskey('u2f')">新增安全密钥</BButton>
      </template>

      <template v-else-if="mfaModalType === 'recovery_code'">
        <div class="record-meta mb-2">剩余有效码：{{ detail?.recoverySummary?.available ?? 0 }}</div>
        <div class="record-meta mb-3">上次生成时间：{{ formatDateTime(detail?.recoverySummary?.lastGeneratedAt) }}</div>
        <div v-if="recoveryCodes.length" class="portal-code-grid mb-3">
          <code v-for="code in recoveryCodes" :key="code">{{ code }}</code>
        </div>
        <BButton variant="primary" @click="generateRecoveryCodes">重新生成备用验证码</BButton>
      </template>

      <template #footer>
        <BButton variant="outline-secondary" @click="showMFAModal = false">关闭</BButton>
      </template>
    </BModal>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import {
  BButton,
  BForm,
  BFormInput,
  BFormSelect,
  BModal
} from 'bootstrap-vue-next'
import QRCode from 'qrcode'
import { normalizeCreationOptions, serializeCredential } from '@shared/api/webauthn'
import ToastHost from '@shared/components/ToastHost.vue'
import { useToast } from '@shared/composables/toast'
import { apiPost } from '../api/client'
import { startPortalLogout } from '../auth'

type DetailData = {
  user: {
    id: string
    username: string
    name: string
    email: string
    phoneNumber: string
  }
  passwordCredential: boolean
  passkeys: Array<{ id: string; identifier: string; publicKeyId: string; isPasskey: boolean; isU2f: boolean; createdAt: string }>
  bindings: Array<{ id: string; providerName: string; externalIdpId: string; issuer: string; subject: string; createdAt: string }>
  externalIdps: Array<{ id: string; name: string; issuer: string }>
  mfaEnrollments: Array<{ id: string; method: string; label: string; target: string; status: string; lastUsedAt?: string }>
  devices: Array<{ id: string; label: string; online: boolean; trusted: boolean; ipAddress: string; ipLocation?: string; firstLoginAt?: string; lastLoginAt?: string; fingerprint?: string }>
  recoverySummary: { available: number; lastGeneratedAt?: string }
}

type SettingData = {
  session?: { applicationId?: string }
}

type ProfileData = {
  username: string
  name: string
  email: string
  phoneNumber: string
}

const toast = useToast()
const detail = ref<DetailData | null>(null)
const setting = ref<SettingData | null>(null)
const profile = reactive({
  username: '',
  name: '',
  email: '',
  phoneNumber: ''
})
const passwordForm = reactive({
  currentPassword: '',
  newPassword: ''
})
const bindingForm = reactive({
  externalIdpId: '',
  issuer: '',
  subject: ''
})
const showMFAModal = ref(false)
const mfaModalType = ref<'totp' | 'u2f' | 'recovery_code' | ''>('')
const totpEnrollment = reactive({
  enrollmentId: '',
  provisioningUri: '',
  manualEntryKey: ''
})
const totpCode = ref('')
const totpQRCode = ref('')
const recoveryCodes = ref<string[]>([])

const sections = [
  { id: 'profile-basic', label: '基本信息' },
  { id: 'profile-login', label: '登录设置' },
  { id: 'profile-binding', label: '账号绑定' },
  { id: 'profile-mfa', label: '两步验证' },
  { id: 'profile-device', label: '会话管理' }
]

const activeTotpEnrollment = computed(() => detail.value?.mfaEnrollments.find((item) => item.method === 'totp' && item.status === 'active') ?? null)
const passkeyEnrollment = computed(() => detail.value?.mfaEnrollments.find((item) => item.method === 'passkey') ?? null)
const loginPasskeys = computed(() => (detail.value?.passkeys ?? []).filter((item) => item.isPasskey))
const u2fPasskeys = computed(() => (detail.value?.passkeys ?? []).filter((item) => item.isU2f))
const passkeyLoginEnabled = computed(() => passkeyEnrollment.value?.status === 'active' && loginPasskeys.value.length > 0)
const mfaRows = computed(() => {
  const enrollments = detail.value?.mfaEnrollments ?? []
  const u2fEnabled = enrollments.some((item) => item.method === 'u2f' && item.status === 'active') && u2fPasskeys.value.length > 0
  const emailEnabled = enrollments.some((item) => item.method === 'email_code' && item.status === 'active')
  const smsEnabled = enrollments.some((item) => item.method === 'sms_code' && item.status === 'active')
  const totpEnabled = enrollments.some((item) => item.method === 'totp' && item.status === 'active')
  return [
    { id: 'email_code', label: '邮箱验证码', enabled: emailEnabled, summary: emailEnabled ? `目标：${profile.email || '未配置邮箱'}` : '使用邮箱接收验证码' },
    { id: 'sms_code', label: '手机验证码', enabled: smsEnabled, summary: smsEnabled ? `目标：${profile.phoneNumber || '未配置手机'}` : '使用手机接收验证码' },
    { id: 'totp', label: '身份验证器（TOTP）', enabled: totpEnabled, summary: totpEnabled ? '已配置身份验证器' : '使用身份验证器 App 生成动态验证码' },
    { id: 'u2f', label: '安全密钥', enabled: u2fEnabled, summary: u2fEnabled ? `已登记 ${u2fPasskeys.value.length} 把安全密钥` : '使用 WebAuthn 安全密钥进行验证' },
    { id: 'recovery_code', label: '备用验证码', enabled: (detail.value?.recoverySummary?.available ?? 0) > 0, summary: `剩余可用 ${detail.value?.recoverySummary?.available ?? 0} 个` }
  ]
})
const mfaModalTitle = computed(() => {
  if (mfaModalType.value === 'totp') return '身份验证器（TOTP）'
  if (mfaModalType.value === 'u2f') return '安全密钥'
  if (mfaModalType.value === 'recovery_code') return '备用验证码'
  return '两步验证'
})

async function loadPortalData() {
  try {
    const [profileResponse, detailResponse, settingResponse] = await Promise.all([
      apiPost<ProfileData>('/api/user/v1/profile/query', {}),
      apiPost<DetailData>('/api/user/v1/detail/query', {}),
      apiPost<SettingData>('/api/user/v1/setting/query', {})
    ])
    profile.username = profileResponse.username || ''
    profile.name = profileResponse.name || ''
    profile.email = profileResponse.email || ''
    profile.phoneNumber = profileResponse.phoneNumber || ''
    detail.value = detailResponse
    setting.value = settingResponse
    if (!bindingForm.externalIdpId && detailResponse.externalIdps.length) {
      bindingForm.externalIdpId = detailResponse.externalIdps[0].id
      bindingForm.issuer = detailResponse.externalIdps[0].issuer
    }
  } catch (error) {
    toast.error(String(error))
  }
}

async function saveProfile() {
  await apiPost('/api/user/v1/profile/update', { ...profile })
  toast.success('基本信息已保存')
  await loadPortalData()
}

async function savePassword() {
  await apiPost('/api/user/v1/setting/update', { ...passwordForm })
  passwordForm.currentPassword = ''
  passwordForm.newPassword = ''
  toast.success('密码已更新')
}

async function registerPasskey(purpose: 'passkey' | 'u2f' = 'passkey') {
  const begin = await apiPost<{ challengeId: string; options: any }>('/api/user/v1/passkey/register/begin', { purpose })
  const credential = await navigator.credentials.create({
    publicKey: normalizeCreationOptions(begin.options)
  })
  if (!credential) {
    return
  }
  await apiPost('/api/user/v1/passkey/register/finish', {
    challengeId: begin.challengeId,
    response: serializeCredential(credential as PublicKeyCredential)
  })
  toast.success(purpose === 'u2f' ? '安全密钥已注册' : '通行密钥已注册')
  await loadPortalData()
}

async function deletePasskey(credentialId: string) {
  await apiPost('/api/user/v1/passkey/delete', { credentialId })
  toast.success('通行密钥已删除')
  await loadPortalData()
}

async function togglePasskeyLogin(enabled: boolean) {
  await apiPost('/api/user/v1/mfa_method/update', { method: 'passkey', enabled })
  toast.success(enabled ? '已启用通行密钥登录' : '已关闭通行密钥登录')
  await loadPortalData()
}

function syncBindingIssuer() {
  const current = detail.value?.externalIdps.find((item) => item.id === bindingForm.externalIdpId)
  bindingForm.issuer = current?.issuer ?? ''
}

async function createBinding() {
  await apiPost('/api/user/v1/external_identity_binding/create', { ...bindingForm })
  bindingForm.subject = ''
  toast.success('账号绑定已新增')
  await loadPortalData()
}

async function deleteBinding(bindingId: string) {
  await apiPost('/api/user/v1/external_identity_binding/delete', { bindingId })
  toast.success('账号绑定已删除')
  await loadPortalData()
}

async function toggleSimpleMFA(method: string, enabled: boolean) {
  await apiPost('/api/user/v1/mfa_method/update', { method, enabled })
  toast.success(enabled ? '已开启' : '已关闭')
  await loadPortalData()
}

function openMFAModal(type: string) {
  if (type !== 'totp' && type !== 'u2f' && type !== 'recovery_code') {
    return
  }
  mfaModalType.value = type
  showMFAModal.value = true
  if (type === 'totp') {
    totpCode.value = ''
    totpEnrollment.enrollmentId = ''
    totpEnrollment.provisioningUri = ''
    totpEnrollment.manualEntryKey = ''
    totpQRCode.value = ''
  }
  if (type === 'recovery_code') {
    recoveryCodes.value = []
  }
}

async function generateTotp() {
  const applicationId = setting.value?.session?.applicationId ?? ''
  const result = await apiPost<{ enrollmentId: string; provisioningUri: string; manualEntryKey: string }>('/api/user/v1/totp/enroll', {
    applicationId
  })
  totpEnrollment.enrollmentId = result.enrollmentId
  totpEnrollment.provisioningUri = result.provisioningUri
  totpEnrollment.manualEntryKey = result.manualEntryKey
  totpQRCode.value = await QRCode.toDataURL(result.provisioningUri, { width: 180, margin: 1 })
}

async function verifyTotp() {
  if (!totpEnrollment.enrollmentId) {
    await generateTotp()
  }
  await apiPost('/api/user/v1/totp/verify', {
    enrollmentId: totpEnrollment.enrollmentId,
    code: totpCode.value
  })
  toast.success('身份验证器已启用')
  showMFAModal.value = false
  await loadPortalData()
}

async function disableTotp() {
  await apiPost('/api/user/v1/mfa_enrollment/delete', { method: 'totp' })
  toast.success('身份验证器已关闭')
  showMFAModal.value = false
  await loadPortalData()
}

async function generateRecoveryCodes() {
  const result = await apiPost<{ codes: string[] }>('/api/user/v1/recovery_code/generate', {})
  recoveryCodes.value = result.codes
  toast.success('已重新生成备用验证码')
  await loadPortalData()
}

async function untrustDevice(deviceId: string) {
  await apiPost('/api/user/v1/device/untrust', { deviceId })
  toast.success('设备已取消可信')
  await loadPortalData()
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function formatIPLine(ipAddress?: string, ipLocation?: string) {
  const ip = String(ipAddress || '').trim() || '-'
  const location = String(ipLocation || '').trim()
  return location ? `${ip} (${location})` : ip
}

async function scrollTo(id: string) {
  await nextTick()
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function logout() {
  startPortalLogout()
}

onMounted(loadPortalData)
</script>

<style scoped>
.portal-center {
  min-height: 100vh;
  background: #f8f9fa;
}

.portal-topbar {
  position: sticky;
  top: 0;
  z-index: 1000;
}

.portal-topbar-main {
  min-height: 4.25rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.portal-title {
  font-size: 1.15rem;
  font-weight: 700;
}

.portal-subtitle {
  color: #6c757d;
  font-size: 0.9rem;
}

.portal-empty-card,
.portal-nav-card {
  background: #fff;
  border: 1px solid #dee2e6;
  border-radius: 0.75rem;
  padding: 1rem;
  box-shadow: 0 0.125rem 0.25rem rgba(0, 0, 0, 0.04);
}

.portal-nav-card {
  display: grid;
  gap: 0.35rem;
  position: sticky;
  top: 5.5rem;
}

.portal-nav-link {
  width: 100%;
  text-align: left;
  border: 0;
  border-radius: 0.65rem;
  background: transparent;
  padding: 0.7rem 0.8rem;
  font-size: 0.95rem;
  color: #212529;
}

.portal-nav-link:hover {
  background: #eef2f7;
}

.portal-sections {
  padding-bottom: 8rem;
}

.detail-card,
.record-card {
  background: #fff;
  border: 1px solid #dee2e6;
  border-radius: 0.65rem;
  padding: 0.9rem 1rem;
}

.record-list {
  display: grid;
  gap: 0.75rem;
}

.record-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 0;
  border-top: 1px solid #edf0f2;
}

.record-row:first-child {
  border-top: 0;
  padding-top: 0;
}

.record-row:last-child {
  padding-bottom: 0;
}

.record-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.35rem;
}

.record-meta {
  color: #6c757d;
  font-size: 0.88rem;
}

.record-actions {
  margin-top: 0.65rem;
}

.small-break {
  overflow-wrap: anywhere;
}

.portal-code-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 0.5rem;
}

.portal-code-grid code {
  background: #f8f9fa;
  border: 1px solid #dee2e6;
  border-radius: 0.5rem;
  padding: 0.55rem 0.65rem;
}

@media (max-width: 991.98px) {
  .portal-nav-card {
    position: static;
  }
}
</style>
