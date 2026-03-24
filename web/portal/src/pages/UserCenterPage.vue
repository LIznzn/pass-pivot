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
      <div class="row g-4">
        <aside class="col-lg-3">
          <div class="console-module-sidebar">
            <button v-for="section in sections" :key="section.id" type="button" class="console-module-sidebar-link" @click="scrollTo(section.id)">
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
                    <div class="login-card-title">登录方式</div>
                    <div class="login-toggle-list">
                      <div class="login-setting-row">
                        <div>
                          <div class="login-setting-name">密码登录</div>
                          <div class="record-meta">{{ detail?.passwordCredential ? '已启用' : '未配置' }}</div>
                        </div>
                        <span class="record-meta">{{ detail?.passwordCredential ? '当前账号已设置密码' : '当前账号尚未设置密码' }}</span>
                      </div>
                      <div class="login-setting-row">
                        <div>
                          <div class="login-setting-name">通行密钥登录</div>
                          <div class="record-meta">{{ webauthnLoginEnabled ? '已启用' : '未启用' }}</div>
                        </div>
                        <BButton size="sm" :variant="webauthnLoginEnabled ? 'outline-danger' : 'outline-primary'" @click="toggleWebAuthnLogin(!webauthnLoginEnabled)">
                          {{ webauthnLoginEnabled ? '关闭' : '开启' }}
                        </BButton>
                      </div>
                    </div>
                  </div>
                </div>
                <div class="col-lg-6">
                  <div class="detail-card h-100">
                    <div class="login-card-title">密码修改</div>
                    <BForm @submit.prevent="savePassword">
                      <BFormInput v-model="passwordForm.currentPassword" type="password" placeholder="当前密码" class="mb-2" />
                      <BFormInput v-model="passwordForm.newPassword" type="password" placeholder="新密码" class="mb-3" />
                      <BButton type="submit" variant="outline-primary" size="sm">更新密码</BButton>
                    </BForm>
                  </div>
                </div>
                <div class="col-12">
                  <div class="detail-card">
                    <div class="d-flex justify-content-between align-items-center gap-3 flex-wrap">
                      <div>
                        <div class="login-card-title mb-1">密钥管理</div>
                        <div class="record-meta">{{ allSecureKeys.length ? `当前账号已绑定 ${allSecureKeys.length} 把密钥，可查看每把密钥支持的能力。` : '当前账号还没有绑定密钥。' }}</div>
                      </div>
                      <BButton size="sm" variant="outline-primary" @click="openKeyModal">管理密钥</BButton>
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
              <div class="section-title">多因素验证</div>
              <div class="record-card mb-3">
                <div class="mfa-summary-row">
                  <div>
                    <strong>启用多因素验证</strong>
                    <div class="record-meta">{{ mfaSummaryText }}</div>
                  </div>
                  <div class="d-flex gap-2">
                    <BButton v-if="mfaEnabled" size="sm" variant="outline-primary" @click="openMFAModal('recovery_code')">查看备用验证码</BButton>
                    <BButton size="sm" :variant="mfaEnabled ? 'outline-danger' : 'outline-primary'" @click="toggleMFAEnabled(!mfaEnabled)">
                      {{ mfaEnabled ? '关闭' : '开启' }}
                    </BButton>
                  </div>
                </div>
              </div>
              <div v-if="mfaEnabled" class="record-list">
                <div v-for="item in mfaRows" :key="item.id" class="record-card">
                  <div class="mfa-summary-row">
                    <div>
                      <strong>{{ item.label }}</strong>
                      <div class="record-meta">{{ item.summary }}</div>
                    </div>
                    <div class="d-flex gap-2">
                      <BButton
                        v-if="item.id === 'email_code' || item.id === 'sms_code' || item.id === 'u2f'"
                        size="sm"
                        :variant="item.enabled ? 'outline-danger' : 'outline-primary'"
                        @click="toggleSimpleMFA(item.id, !item.enabled)"
                      >
                        {{ item.enabled ? '关闭' : '开启' }}
                      </BButton>
                      <BButton v-else size="sm" variant="outline-primary" @click="openMFAModal(item.id)">
                        配置
                      </BButton>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div id="profile-device" class="info-card">
              <div class="section-title">会话管理</div>
              <div v-if="!deviceRows.length" class="detail-card">
                <div class="record-meta">当前没有设备记录。</div>
              </div>
              <div v-for="device in deviceRows" :key="device.id" class="record-card mb-2">
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
        <div class="record-meta mb-3">安全密钥的注册与删除已迁移到“登录设置 > 密钥管理”。</div>
      </template>

      <template v-else-if="mfaModalType === 'recovery_code'">
        <div class="record-meta mb-2">剩余有效码：{{ detail?.recoverySummary?.available ?? 0 }}</div>
        <div class="record-meta mb-3">上次生成时间：{{ formatDateTime(detail?.recoverySummary?.lastGeneratedAt) }}</div>
        <div class="record-meta mb-3">当前有效备用验证码：{{ recoveryCodes.length ? `共 ${recoveryCodes.length} 个` : '暂无' }}</div>
        <div v-if="recoveryCodes.length" class="portal-code-grid mb-3">
          <code v-for="code in recoveryCodes" :key="code">{{ code }}</code>
        </div>
        <BButton variant="primary" @click="generateRecoveryCodes">重新生成备用验证码</BButton>
      </template>

      <template #footer>
        <BButton variant="outline-secondary" @click="showMFAModal = false">关闭</BButton>
      </template>
    </BModal>

    <BModal v-model="showKeyModal" title="密钥管理" size="lg" centered>
      <div class="d-flex gap-2 flex-wrap mb-3">
        <BButton size="sm" variant="outline-primary" @click="registerSecureKey('webauthn')">注册为通行密钥（WebAuthn）</BButton>
        <BButton size="sm" variant="outline-secondary" @click="registerSecureKey('u2f')">注册为安全密钥（U2F）</BButton>
      </div>
      <div v-if="!allSecureKeys.length" class="record-meta">当前没有已注册的密钥。</div>
      <div v-for="secureKey in allSecureKeys" :key="secureKey.id" class="record-card mb-2">
        <div class="record-head">
          <div>
            <BFormInput
              v-model="keyNameDrafts[secureKey.id]"
              placeholder="密钥名称"
              class="mb-2"
            />
            <div class="record-meta">{{ keyCapabilityLabel(secureKey) }}</div>
          </div>
          <div class="d-flex align-items-center gap-2">
            <BButton size="sm" variant="outline-primary" @click="updateSecureKey(secureKey.id, keyNameDrafts[secureKey.id] || '')">保存名称</BButton>
            <BButton size="sm" variant="outline-danger" @click="deleteSecureKey(secureKey.id)">删除密钥</BButton>
          </div>
        </div>
        <div class="record-meta small-break">{{ secureKey.publicKeyId }}</div>
        <div class="record-meta mt-2">注册时间：{{ formatDateTime(secureKey.createdAt) }}</div>
      </div>
      <template #footer>
        <BButton variant="outline-secondary" @click="showKeyModal = false">关闭</BButton>
      </template>
    </BModal>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import {
  BButton,
  BForm,
  BFormInput,
  BFormSelect,
  BModal,
  useToast
} from 'bootstrap-vue-next'
import QRCode from 'qrcode'
import { normalizeCreationOptions, serializeCredential } from '@shared/utils/webauthn'
import { formatDateTime as formatSharedDateTime } from '@shared/utils/datetime'
import { notifyToast } from '@shared/utils/notify'
import { apiPost } from '@/api/client'
import { startPortalLogout } from '@/auth'

type DetailData = {
  user: {
    id: string
    username: string
    name: string
    email: string
    phoneNumber: string
  }
  passwordCredential: boolean
  secureKeys: Array<{ id: string; identifier: string; publicKeyId: string; webauthnEnable: boolean; u2fEnable: boolean; createdAt: string }>
  bindings: Array<{ id: string; providerName: string; externalIdpId: string; issuer: string; subject: string; createdAt: string }>
  externalIdps: Array<{ id: string; name: string; issuer: string }>
  mfaEnrollments: Array<{ id: string; method: string; label: string; target: string; status: string; lastUsedAt?: string }>
  devices: Array<{ id: string; userAgent?: string; online: boolean; trusted: boolean; lastLoginIp?: string; ipLocation?: string; firstLoginAt?: string; lastLoginAt?: string; deviceFingerprint?: string }>
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

function showToast(
  message: string,
  variant: 'success' | 'danger',
  options: {
    source: string
    trigger?: string
    error?: unknown
    metadata?: Record<string, unknown>
  } = {
    source: 'portal/UserCenterPage'
  }
) {
  notifyToast({
    toast,
    message,
    variant,
    source: options.source,
    trigger: options.trigger,
    error: options.error,
    metadata: options.metadata
  })
}

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
const showKeyModal = ref(false)
const mfaModalType = ref<'totp' | 'u2f' | 'recovery_code' | ''>('')
const totpEnrollment = reactive({
  enrollmentId: '',
  provisioningUri: '',
  manualEntryKey: ''
})
const totpCode = ref('')
const totpQRCode = ref('')
const recoveryCodes = ref<string[]>([])
const keyNameDrafts = reactive<Record<string, string>>({})

const sections = [
  { id: 'profile-basic', label: '基本信息' },
  { id: 'profile-login', label: '登录设置' },
  { id: 'profile-binding', label: '账号绑定' },
  { id: 'profile-mfa', label: '多因素验证' },
  { id: 'profile-device', label: '会话管理' }
]

const activeTotpEnrollment = computed(() => detail.value?.mfaEnrollments.find((item) => item.method === 'totp' && item.status === 'active') ?? null)
const mfaEnrollment = computed(() => detail.value?.mfaEnrollments.find((item) => item.method === 'mfa' && item.status === 'active') ?? null)
const webauthnEnrollment = computed(() => detail.value?.mfaEnrollments.find((item) => item.method === 'webauthn') ?? null)
const loginSecureKeys = computed(() => (detail.value?.secureKeys ?? []).filter((item) => item.webauthnEnable))
const u2fSecureKeys = computed(() => (detail.value?.secureKeys ?? []).filter((item) => item.u2fEnable))
const allSecureKeys = computed(() => detail.value?.secureKeys ?? [])
const deviceRows = computed(() => (detail.value?.devices ?? []).map((device) => ({
  id: device.id,
  label: inferDeviceName(device.userAgent),
  online: Boolean(device.online),
  trusted: Boolean(device.trusted),
  ipAddress: device.lastLoginIp || '',
  ipLocation: device.ipLocation || '',
  firstLoginAt: device.firstLoginAt,
  lastLoginAt: device.lastLoginAt,
  fingerprint: device.deviceFingerprint || ''
})))
const webauthnLoginEnabled = computed(() => webauthnEnrollment.value?.status === 'active' && loginSecureKeys.value.length > 0)
const mfaEnabled = computed(() => Boolean(mfaEnrollment.value))
const mfaRows = computed(() => {
  const enrollments = detail.value?.mfaEnrollments ?? []
  const u2fEnabled = enrollments.some((item) => item.method === 'u2f' && item.status === 'active') && u2fSecureKeys.value.length > 0
  const emailEnabled = enrollments.some((item) => item.method === 'email_code' && item.status === 'active')
  const smsEnabled = enrollments.some((item) => item.method === 'sms_code' && item.status === 'active')
  const totpEnabled = enrollments.some((item) => item.method === 'totp' && item.status === 'active')
  return [
    { id: 'email_code', label: '邮箱验证码', enabled: emailEnabled, summary: emailEnabled ? `目标：${profile.email || '未配置邮箱'}` : '使用邮箱接收验证码' },
    { id: 'sms_code', label: '手机验证码', enabled: smsEnabled, summary: smsEnabled ? `目标：${profile.phoneNumber || '未配置手机'}` : '使用手机接收验证码' },
    { id: 'totp', label: '身份验证器（TOTP）', enabled: totpEnabled, summary: totpEnabled ? '已配置身份验证器' : '使用身份验证器 App 生成动态验证码' },
    { id: 'u2f', label: '安全密钥', enabled: u2fEnabled, summary: u2fSecureKeys.value.length ? `已登记 ${u2fSecureKeys.value.length} 把安全密钥` : '当前没有可用于安全密钥验证的密钥' }
  ]
})
const configuredPrimaryMfaCount = computed(() => mfaRows.value.filter((item) => item.enabled).length)
const recoveryCodeCount = computed(() => detail.value?.recoverySummary?.available ?? 0)
const mfaSummaryText = computed(() => {
  if (!mfaEnabled.value) {
    return '开启后可配置主验证方式，并自动准备备用验证码。'
  }
  if (configuredPrimaryMfaCount.value === 0) {
    return `已生成 ${recoveryCodeCount.value} 个备用验证码，但尚未配置其他验证方式；当前登录不会触发多因素验证。`
  }
  return `已配置 ${configuredPrimaryMfaCount.value} 种主验证方式，备用验证码剩余 ${recoveryCodeCount.value} 个。`
})
const mfaModalTitle = computed(() => {
  if (mfaModalType.value === 'totp') return '身份验证器（TOTP）'
  if (mfaModalType.value === 'u2f') return '安全密钥'
  if (mfaModalType.value === 'recovery_code') return '备用验证码'
  return '多因素验证'
})

watch(
  allSecureKeys,
  (items) => {
    Object.keys(keyNameDrafts).forEach((key) => delete keyNameDrafts[key])
    for (const item of items) {
      keyNameDrafts[item.id] = item.identifier || ''
    }
  },
  { immediate: true }
)

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
    showToast(String(error), 'danger', {
      source: 'portal/UserCenterPage.loadPortalData',
      trigger: 'loadPortalData',
      error
    })
  }
}

async function saveProfile() {
  await apiPost('/api/user/v1/profile/update', { ...profile })
  showToast('基本信息已保存', 'success')
  await loadPortalData()
}

async function savePassword() {
  await apiPost('/api/user/v1/setting/update', { ...passwordForm })
  passwordForm.currentPassword = ''
  passwordForm.newPassword = ''
  showToast('密码已更新', 'success')
}

async function registerSecureKey(purpose: 'webauthn' | 'u2f') {
  const begin = await apiPost<{ challengeId: string; options: any }>('/api/user/v1/securekey/register/begin', { purpose })
  const credential = await navigator.credentials.create({
    publicKey: normalizeCreationOptions(begin.options)
  })
  if (!credential) {
    return
  }
  await apiPost('/api/user/v1/securekey/register/finish', {
    challengeId: begin.challengeId,
    response: serializeCredential(credential as PublicKeyCredential)
  })
  showToast(purpose === 'webauthn' ? '通行密钥已注册' : '安全密钥已注册', 'success')
  await loadPortalData()
}

async function deleteSecureKey(credentialId: string) {
  await apiPost('/api/user/v1/securekey/delete', { credentialId })
  showToast('密钥已删除', 'success')
  await loadPortalData()
}

async function updateSecureKey(credentialId: string, identifier: string) {
  await apiPost('/api/user/v1/securekey/update', { credentialId, identifier })
  showToast('密钥名称已更新', 'success')
  await loadPortalData()
}

async function toggleWebAuthnLogin(enabled: boolean) {
  if (enabled && loginSecureKeys.value.length === 0) {
    openKeyModal()
    return
  }
  await apiPost('/api/user/v1/mfa_method/update', { method: 'webauthn', enabled })
  showToast(enabled ? '已启用通行密钥登录' : '已关闭通行密钥登录', 'success')
  await loadPortalData()
}

function syncBindingIssuer() {
  const current = detail.value?.externalIdps.find((item) => item.id === bindingForm.externalIdpId)
  bindingForm.issuer = current?.issuer ?? ''
}

async function createBinding() {
  await apiPost('/api/user/v1/external_identity_binding/create', { ...bindingForm })
  bindingForm.subject = ''
  showToast('账号绑定已新增', 'success')
  await loadPortalData()
}

async function deleteBinding(bindingId: string) {
  await apiPost('/api/user/v1/external_identity_binding/delete', { bindingId })
  showToast('账号绑定已删除', 'success')
  await loadPortalData()
}

async function toggleSimpleMFA(method: string, enabled: boolean) {
  if (method === 'u2f' && enabled && u2fSecureKeys.value.length === 0) {
    openKeyModal()
    return
  }
  await apiPost('/api/user/v1/mfa_method/update', { method, enabled })
  showToast(enabled ? '已开启' : '已关闭', 'success')
  await loadPortalData()
}

async function toggleMFAEnabled(enabled: boolean) {
  await apiPost('/api/user/v1/mfa_method/update', { method: 'mfa', enabled })
  if (enabled) {
    const result = await apiPost<{ codes: string[] }>('/api/user/v1/recovery_code/query', {})
    recoveryCodes.value = result.codes
    mfaModalType.value = 'recovery_code'
    showMFAModal.value = true
  } else {
    recoveryCodes.value = []
  }
  showToast(enabled ? '已更新多因素验证主开关，并已准备备用验证码' : '已关闭多因素验证', 'success')
  await loadPortalData()
}

function openKeyModal() {
  showKeyModal.value = true
}

function keyCapabilityLabel(secureKey: DetailData['secureKeys'][number]) {
  if (secureKey.webauthnEnable && secureKey.u2fEnable) {
    return '支持通行密钥登录与安全密钥验证'
  }
  if (secureKey.webauthnEnable) {
    return '仅支持通行密钥登录'
  }
  if (secureKey.u2fEnable) {
    return '仅支持安全密钥验证'
  }
  return '能力未识别'
}

function inferDeviceName(userAgent?: string) {
  const source = String(userAgent || '').trim()
  if (!source) return '未知设备'
  const browser = source.includes('Edg/') ? 'Edge'
    : source.includes('Chrome/') && !source.includes('Edg/') ? 'Chrome'
    : source.includes('Firefox/') ? 'Firefox'
    : source.includes('Safari/') && !source.includes('Chrome/') ? 'Safari'
    : source.includes('MSIE') || source.includes('Trident/') ? 'Internet Explorer'
    : ''
  const os = source.includes('Windows NT') ? 'Windows'
    : source.includes('Mac OS X') || source.includes('Macintosh') ? 'macOS'
    : source.includes('Android') ? 'Android'
    : source.includes('iPhone') || source.includes('iPad') || source.includes('iOS') ? 'iOS'
    : source.includes('Linux') ? 'Linux'
    : ''
  if (!browser && !os) return source
  return browser && os ? `${browser} (${os})` : (browser || os)
}

async function openMFAModal(type: string) {
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
    const result = await apiPost<{ codes: string[] }>('/api/user/v1/recovery_code/query', {})
    recoveryCodes.value = result.codes
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
  showToast('身份验证器已启用', 'success')
  showMFAModal.value = false
  await loadPortalData()
}

async function disableTotp() {
  await apiPost('/api/user/v1/mfa_enrollment/delete', { method: 'totp' })
  showToast('身份验证器已关闭', 'success')
  showMFAModal.value = false
  await loadPortalData()
}

async function generateRecoveryCodes() {
  const result = await apiPost<{ codes: string[] }>('/api/user/v1/recovery_code/generate', {})
  recoveryCodes.value = result.codes
  showToast('已重新生成备用验证码', 'success')
  await loadPortalData()
}

async function untrustDevice(deviceId: string) {
  await apiPost('/api/user/v1/device/untrust', { deviceId })
  showToast('设备已取消可信', 'success')
  await loadPortalData()
}

function formatDateTime(value?: string) {
  return formatSharedDateTime(value)
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

.portal-empty-card {
  background: #fff;
  border: 1px solid #dee2e6;
  border-radius: 0.75rem;
  padding: 1rem;
  box-shadow: 0 0.125rem 0.25rem rgba(0, 0, 0, 0.04);
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

.login-card-title {
  font-size: 0.98rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
}

.login-toggle-list {
  display: grid;
  gap: 0.25rem;
}

.login-setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 0;
  border-top: 1px solid #edf0f2;
}

.login-setting-name {
  font-size: 1rem;
  font-weight: 500;
}

.login-setting-row:first-child {
  border-top: 0;
  padding-top: 0;
}

.login-setting-row:last-child {
  padding-bottom: 0;
}

.mfa-summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
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
  padding: 0.9rem 1rem;
  border-top: 1px solid #edf0f2;
}

.record-row:first-child {
  border-top: 0;
  padding-top: 0;
}

.record-row-plain {
  border-top: 0;
  padding: 0 1rem 0.9rem;
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
  .console-module-sidebar {
    position: static;
  }
}
</style>
