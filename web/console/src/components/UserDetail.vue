<template>
  <section class="console-module-shell">
    <div class="console-module-summary-card">
      <div class="console-module-hero">
        <div class="console-module-hero-copy">
          <button type="button" class="console-back-button" @click="emit('back')" aria-label="返回用户列表">
            <i class="bi bi-arrow-left console-back-button-icon" aria-hidden="true"></i>
          </button>
          <div>
            <div class="console-module-eyebrow">用户</div>
            <h2 class="console-module-title">{{ currentUserRecord?.name || currentUserRecord?.username || currentUserRecord?.email || '用户' }}</h2>
            <p class="console-module-subtitle">{{ currentUserRecord?.name || currentUserRecord?.email ? '从用户列表选择条目后，在详情区维护基本信息、登录设置、账号绑定、两步验证、会话与角色分配。' : '管理用户、通行密钥、身份验证器、备用验证码与管理员动作。' }}</p>
          </div>
        </div>
        <BButton variant="primary" @click="emit('run-module-action')">刷新用户</BButton>
      </div>
      <div class="console-module-metrics">
        <div v-for="item in currentModuleMetrics" :key="item.label" class="console-module-metric">
          <span class="console-module-metric-label">{{ item.label }}</span>
          <div class="console-module-metric-value-row">
            <strong class="console-module-metric-value">{{ item.value }}</strong>
            <button
              v-if="item.copyable"
              type="button"
              class="console-module-metric-copy"
              :aria-label="`复制${item.label}`"
              @click="consoleStore.copyMetricValue(item.copyValue || item.value)"
            >
              <i class="bi bi-copy" aria-hidden="true"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="console-module-workspace">
      <aside class="console-module-sidebar">
        <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="consoleStore.scrollToPanel(item.id)">{{ item.label }}</button>
      </aside>
      <div class="console-module-main">
        <div id="user-basic" class="info-card">
          <div class="section-title">基本信息</div>
          <BForm @submit.prevent="emit('update-user')">
            <div class="row g-3">
              <div class="col-md-6">
                <label class="form-label">姓名</label>
                <BFormInput v-model="userUpdateForm.name" />
              </div>
              <div class="col-md-6">
                <label class="form-label">用户名</label>
                <BFormInput v-model="userUpdateForm.username" />
              </div>
              <div class="col-md-6">
                <label class="form-label">邮箱</label>
                <BFormInput v-model="userUpdateForm.email" type="email" />
              </div>
              <div class="col-md-6">
                <label class="form-label">手机</label>
                <div class="phone-input-group">
                  <BFormSelect v-model="userUpdatePhoneInput.countryCode" :options="props.phoneCountryOptions" class="phone-country-select" />
                  <BFormInput v-model="userUpdatePhoneInput.localNumber" class="phone-local-input" />
                </div>
              </div>
            </div>
            <div class="d-flex justify-content-between align-items-center mt-3">
              <div class="record-meta mb-0">创建时间：{{ formatDateTime(currentUserRecord?.createdAt) }} | 更新时间：{{ formatDateTime(currentUserRecord?.updatedAt) }}</div>
              <BButton type="submit" variant="primary">保存基本信息</BButton>
            </div>
          </BForm>
        </div>

        <div id="user-login-setting" class="info-card">
          <div class="section-title">登录设置</div>
          <div class="row g-3">
            <div class="col-lg-6">
              <div class="detail-card h-100">
                <div class="record-meta mb-3">密码登录：{{ userDetail?.passwordCredential ? '已启用' : '未配置' }}</div>
                <BForm @submit.prevent="emit('reset-user-password')">
                  <BFormInput v-model="userAdminForm.password" type="password" placeholder="新密码" class="mb-2" />
                  <BButton type="submit" variant="outline-primary" size="sm">重置密码</BButton>
                </BForm>
              </div>
            </div>
            <div class="col-lg-6">
              <div class="detail-card h-100">
                <div class="d-flex justify-content-between align-items-center mb-2">
                  <div class="record-meta mb-0">通行密钥数：{{ loginSecureKeys.length }} · {{ webauthnLoginEnabled ? '已启用登录' : '已关闭登录' }}</div>
                  <div class="d-flex gap-2">
                    <BButton size="sm" :variant="webauthnLoginEnabled ? 'outline-danger' : 'outline-secondary'" :disabled="!loginSecureKeys.length" @click="emit('toggle-webauthn-login', !webauthnLoginEnabled)">
                      {{ webauthnLoginEnabled ? '关闭登录' : '启用登录' }}
                    </BButton>
                    <BButton size="sm" variant="outline-primary" @click="emit('register-secure-key', 'webauthn')">注册通行密钥</BButton>
                  </div>
                </div>
                <div v-if="!loginSecureKeys.length" class="record-meta">当前没有通行密钥，注册后才可启用通行密钥登录。</div>
                <div v-for="secureKey in loginSecureKeys" :key="secureKey.id" class="record-row">
                  <div>
                    <strong>{{ secureKey.identifier || '通行密钥' }}</strong>
                    <div class="record-meta">{{ secureKey.publicKeyId }}</div>
                  </div>
                  <code>{{ formatDateTime(secureKey.createdAt) }}</code>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div id="user-binding" class="info-card">
          <div class="section-title">账号绑定</div>
          <div class="record-meta mb-3">绑定外部 OAuth/OIDC 身份的 UUID/Subject，用于登录识别，不会自动创建用户。</div>
          <div class="row g-3">
            <div class="col-lg-7">
              <div v-if="!(userDetail?.bindings?.length ?? 0)" class="detail-card">
                <div class="record-meta">当前没有第三方身份绑定。</div>
              </div>
              <div v-for="binding in userDetail?.bindings || []" :key="binding.id" class="record-card mb-2">
                <div class="record-head">
                  <strong>{{ binding.providerName || binding.externalIdpId }}</strong>
                  <code>{{ binding.subject }}</code>
                </div>
                <div class="record-meta">Issuer：{{ binding.issuer }}</div>
                <div class="record-meta">绑定时间：{{ formatDateTime(binding.createdAt) }}</div>
                <div class="record-actions">
                  <BButton size="sm" variant="outline-danger" @click="emit('delete-external-binding', binding.id)">解绑</BButton>
                </div>
              </div>
            </div>
            <div class="col-lg-5">
              <BForm @submit.prevent="emit('create-external-binding')">
                <label class="form-label">外部 IdP</label>
                <BFormSelect v-model="externalBindingForm.externalIdpId" class="mb-2" @update:model-value="syncExternalBindingIssuer">
                  <option v-for="item in userDetail?.externalIdps || []" :key="item.id" :value="item.id">{{ item.name }}</option>
                </BFormSelect>
                <label class="form-label">Issuer</label>
                <BFormInput v-model="externalBindingForm.issuer" class="mb-2" />
                <label class="form-label">Subject / UUID</label>
                <BFormInput v-model="externalBindingForm.subject" class="mb-3" />
                <BButton type="submit" variant="primary" size="sm">新增绑定</BButton>
              </BForm>
            </div>
          </div>
        </div>

        <div id="user-mfa" class="info-card">
          <div class="section-title">两步验证</div>
          <div class="record-list">
            <div v-for="item in userMfaMethodRows" :key="item.id" class="record-row">
              <div>
                <strong>{{ item.label }}</strong>
                <div class="record-meta">{{ item.summary }}</div>
              </div>
              <BButton
                v-if="item.id === 'email_code' || item.id === 'sms_code'"
                size="sm"
                :variant="item.enabled ? 'outline-danger' : 'outline-primary'"
                @click="emit('handle-inline-mfa-method-action', item)"
              >
                {{ item.enabled ? '关闭' : '开启' }}
              </BButton>
              <BButton v-else size="sm" variant="outline-primary" @click="emit('open-mfa-modal', item.id)">{{ item.enabled ? '配置' : '开启' }}</BButton>
            </div>
          </div>
        </div>

        <div id="user-session" class="info-card">
          <div class="section-title">会话管理</div>
          <div class="d-flex justify-content-end mb-3">
            <BButton variant="outline-danger" size="sm" @click="emit('revoke-all-user-sessions')">吊销全部 Session</BButton>
          </div>
          <div v-if="!userDeviceList.length" class="detail-card">
            <div class="record-meta">当前没有设备记录。</div>
          </div>
          <div v-for="device in userDeviceList" :key="device.id" class="record-card mb-2">
            <div class="record-head">
              <strong>{{ device.label }}</strong>
              <div class="d-flex align-items-center gap-2">
                <span class="badge text-bg-success" v-if="device.online">在线</span>
                <span class="badge text-bg-secondary" v-else>离线</span>
                <span class="badge text-bg-primary" v-if="device.trusted">可信</span>
                <span class="badge text-bg-light text-dark border" v-else>非可信</span>
              </div>
            </div>
            <div class="record-meta">上次登录 IP：{{ formatIpLine(device.ipAddress, device.ipLocation) }}</div>
            <div class="record-meta">上次登录时间：{{ formatDateTime(device.lastLoginAt) }}</div>
            <div class="record-meta">初次登录日期：{{ formatDateTime(device.firstLoginAt) }}</div>
            <div v-if="device.fingerprint" class="record-meta">设备指纹：{{ device.fingerprint }}</div>
            <div v-if="device.trusted" class="record-actions">
              <BButton size="sm" variant="outline-danger" @click="emit('untrust-managed-device', device.id)">取消可信</BButton>
            </div>
          </div>
        </div>

        <div id="user-role-assignment" class="info-card">
          <div class="section-title">角色分配</div>
          <div class="record-meta mb-3">用户表中的角色以标签数组保存，若角色标签未来被删除，则自动忽略。</div>
          <div class="row g-2 mb-3">
            <div v-for="role in userAssignableRoles" :key="role.id" class="col-md-6 col-xl-4">
              <label class="detail-card d-flex align-items-center gap-2 mb-0">
                <input class="form-check-input mt-0" type="checkbox" :checked="userRoleAssignments.includes(role.name)" @change="emit('toggle-user-role', role.name, ($event.target as HTMLInputElement).checked)" />
                <span>
                  <strong>{{ role.name }}</strong>
                  <span class="record-meta d-block">{{ role.description || '无描述' }}</span>
                </span>
              </label>
            </div>
          </div>
          <BButton variant="primary" size="sm" @click="emit('update-user')">保存角色分配</BButton>
        </div>

        <div id="user-danger-zone" class="info-card">
          <div class="section-title text-danger">危险区</div>
          <div class="record-meta mb-3">以下操作会直接影响该用户的凭据、会话与访问状态，请谨慎执行。</div>
          <div class="d-flex gap-2 flex-wrap mb-3">
            <BButton v-if="currentUserRecord?.status !== 'disabled'" variant="outline-warning" size="sm" @click="emit('disable-user')">停用用户</BButton>
            <BButton v-else variant="outline-success" size="sm" @click="emit('enable-user')">启用用户</BButton>
          </div>
          <div class="d-flex gap-2 flex-wrap">
            <BButton variant="outline-warning" size="sm" @click="emit('reset-user-ukid')">轮换用户主密钥</BButton>
            <BButton variant="outline-warning" size="sm" @click="emit('rotate-user-token')">轮换用户主 Token</BButton>
            <BButton variant="outline-danger" size="sm" @click="emit('revoke-all-user-sessions')">吊销全部 Session</BButton>
            <BButton variant="outline-danger" size="sm" @click="emit('delete-single-user', selectedUserId)">删除用户</BButton>
          </div>
          <div class="detail-card mt-3">
            <div class="record-meta">最近管理员动作结果：{{ formatAdminResult(userAdminResult) }}</div>
          </div>
        </div>
      </div>
      <RightSide :items="moduleRecentChanges" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { BButton, BForm, BFormInput, BFormSelect } from 'bootstrap-vue-next'
import RightSide from '../layout/RightSide.vue'
import { useAuditStore } from '../stores/audit'
import { useConsoleStore } from '../stores/console'

type MFAMethod = 'totp' | 'email_code' | 'sms_code' | 'u2f' | 'recovery_code'

const props = defineProps<{
  userUpdateForm: { name: string; username: string; email: string }
  userUpdatePhoneInput: { countryCode: string; localNumber: string }
  phoneCountryOptions: Array<{ value: string; text: string }>
  currentUserRecord: any
  userDetail: any
  userAdminForm: { password: string }
  externalBindingForm: { externalIdpId: string; issuer: string; subject: string }
  userAssignableRoles: any[]
  userRoleAssignments: string[]
  userAdminResult: unknown
  selectedUserId: string
}>()

const auditStore = useAuditStore()
const consoleStore = useConsoleStore()
const moduleRecentChanges = computed(() => auditStore.moduleRecentChanges)
const formatDateTime = consoleStore.formatDateTime

const currentModulePanels = [
  { id: 'user-basic', label: '基本信息' },
  { id: 'user-login-setting', label: '登录设置' },
  { id: 'user-binding', label: '账号绑定' },
  { id: 'user-mfa', label: '两步验证' },
  { id: 'user-session', label: '会话管理' },
  { id: 'user-role-assignment', label: '角色分配' },
  { id: 'user-danger-zone', label: '危险区' }
]

const currentModuleMetrics = computed(() => [
  { label: '用户 ID', value: props.currentUserRecord?.id || '-', copyable: Boolean(props.currentUserRecord?.id), copyValue: props.currentUserRecord?.id || '' },
  { label: '状态', value: props.currentUserRecord?.status || '-' },
  { label: '通行密钥', value: String(props.userDetail?.secureKeys?.length ?? 0) },
  { label: '绑定数', value: String(props.userDetail?.bindings?.length ?? 0) },
  { label: '会话数', value: String(props.userDetail?.recentSessions?.length ?? 0) },
  { label: '最近变更', value: formatDateTime(props.currentUserRecord?.updatedAt) }
])

const activeTotpEnrollments = computed(() => (props.userDetail?.mfaEnrollments || []).filter((item: any) => item.method === 'totp'))
const emailCodeEnrollment = computed(() => (props.userDetail?.mfaEnrollments || []).find((item: any) => item.method === 'email_code'))
const smsCodeEnrollment = computed(() => (props.userDetail?.mfaEnrollments || []).find((item: any) => item.method === 'sms_code'))
const webauthnEnrollment = computed(() => (props.userDetail?.mfaEnrollments || []).find((item: any) => item.method === 'webauthn'))
const loginSecureKeys = computed(() => (props.userDetail?.secureKeys || []).filter((item: any) => item.webauthnEnable))
const u2fSecureKeys = computed(() => (props.userDetail?.secureKeys || []).filter((item: any) => item.u2fEnable))
const u2fEnrollment = computed(() => (props.userDetail?.mfaEnrollments || []).find((item: any) => item.method === 'u2f'))
const webauthnLoginEnabled = computed(() => webauthnEnrollment.value?.status === 'active' && loginSecureKeys.value.length > 0)

const userMfaMethodRows = computed<Array<{ id: MFAMethod; label: string; summary: string; enabled: boolean; disabled?: boolean }>>(() => [
  {
    id: 'totp',
    label: '身份验证器（TOTP）',
    summary: activeTotpEnrollments.value.length > 0 ? '已配置' : '未开启',
    enabled: activeTotpEnrollments.value.length > 0
  },
  {
    id: 'email_code',
    label: '邮箱验证码',
    summary: props.currentUserRecord?.email ? `${emailCodeEnrollment.value?.status === 'active' ? '已开启' : '已关闭'}，目标邮箱：${props.currentUserRecord.email}` : '未配置邮箱',
    enabled: emailCodeEnrollment.value?.status === 'active',
    disabled: !props.currentUserRecord?.email
  },
  {
    id: 'sms_code',
    label: '手机验证码',
    summary: props.currentUserRecord?.phoneNumber ? `${smsCodeEnrollment.value?.status === 'active' ? '已开启' : '已关闭'}，目标手机号：${props.currentUserRecord.phoneNumber}` : '未配置手机号',
    enabled: smsCodeEnrollment.value?.status === 'active',
    disabled: !props.currentUserRecord?.phoneNumber
  },
  {
    id: 'u2f',
    label: '安全密钥',
    summary: u2fEnrollment.value?.status === 'active' && u2fSecureKeys.value.length > 0 ? `已注册 ${u2fSecureKeys.value.length} 个密钥` : '未开启',
    enabled: u2fEnrollment.value?.status === 'active' && u2fSecureKeys.value.length > 0
  },
  {
    id: 'recovery_code',
    label: '备用验证码',
    summary: (props.userDetail?.recoverySummary?.total ?? 0) > 0
      ? `剩余有效码 ${props.userDetail?.recoverySummary?.available ?? 0} 个，最近生成于 ${formatDateTime(props.userDetail?.recoverySummary?.lastGeneratedAt)}`
      : '未开启',
    enabled: (props.userDetail?.recoverySummary?.total ?? 0) > 0
  }
])

const userDeviceList = computed(() => (props.userDetail?.devices || []).map((device: any) => ({
  id: device.id,
  label: inferDeviceName(device.userAgent),
  online: Boolean(device.online),
  trusted: Boolean(device.trusted),
  ipAddress: device.lastLoginIp || '-',
  ipLocation: device.ipLocation || '',
  lastLoginAt: device.lastLoginAt || '',
  firstLoginAt: device.firstLoginAt || '',
  fingerprint: device.deviceFingerprint || ''
})))

function syncExternalBindingIssuer() {
  const provider = props.userDetail?.externalIdps?.find((item: any) => item.id === props.externalBindingForm.externalIdpId)
  if (provider?.issuer) {
    props.externalBindingForm.issuer = provider.issuer
  }
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

function formatIpLine(ipAddress?: string, ipLocation?: string) {
  const ip = String(ipAddress || '').trim()
  const location = String(ipLocation || '').trim()
  if (ip && location) return `${ip} (${location})`
  return ip || location || '-'
}

function formatAdminResult(value: unknown) {
  if (value == null || value === '') return '暂无'
  if (typeof value === 'string') return value
  return JSON.stringify(value)
}

const emit = defineEmits<{
  back: []
  'run-module-action': []
  'update-user': []
  'reset-user-password': []
  'toggle-webauthn-login': [enabled: boolean]
  'register-secure-key': [purpose: 'webauthn' | 'u2f']
  'delete-external-binding': [bindingId: string]
  'create-external-binding': []
  'handle-inline-mfa-method-action': [item: { id: MFAMethod; enabled: boolean; disabled?: boolean }]
  'open-mfa-modal': [method: MFAMethod]
  'revoke-all-user-sessions': []
  'untrust-managed-device': [deviceId: string]
  'toggle-user-role': [roleName: string, checked: boolean]
  'disable-user': []
  'enable-user': []
  'reset-user-ukid': []
  'rotate-user-token': []
  'delete-single-user': [userId: string]
}>()
</script>
