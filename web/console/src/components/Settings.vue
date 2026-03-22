<template>
  <section class="console-module-layout">
    <aside class="console-module-sidebar">
      <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
    </aside>
    <div class="console-module-main">
      <div id="setting-domain" class="info-card">
        <div class="section-title">域名设置</div>
        <div class="record-meta mb-3">绑定登录域名并记录当前验证状态。</div>
        <div v-for="(item, index) in organizationDomainRows" :key="item.id" class="row g-2 align-items-center mb-2">
          <div class="col-md-6">
            <BFormInput v-model="item.host" placeholder="login.example.com" />
          </div>
          <div class="col-md-2">
            <span class="badge" :class="item.verified ? 'text-bg-success' : 'text-bg-secondary'">{{ item.verified ? '已验证' : '未验证' }}</span>
          </div>
          <div class="col-md-4 d-flex gap-2">
            <BButton type="button" size="sm" variant="outline-primary" @click="verifyOrganizationDomain(index)">验证域名</BButton>
            <BButton type="button" size="sm" variant="outline-danger" @click="removeOrganizationDomainRow(index)">删除</BButton>
          </div>
        </div>
        <div class="d-flex justify-content-between align-items-center mt-3">
          <BButton type="button" variant="outline-secondary" @click="addOrganizationDomainRow">添加域名</BButton>
          <BButton type="button" variant="primary" @click="saveOrganizationDomainSettings">保存域名设置</BButton>
        </div>
      </div>

      <div id="setting-login-policy" class="info-card">
        <div class="section-title">登录策略设置</div>
        <BForm @submit.prevent="saveOrganizationLoginPolicy">
          <div class="row g-3">
            <div class="col-md-6">
              <div class="form-check">
                <input id="setting-password-login-enabled" v-model="organizationLoginPolicyForm.passwordLoginEnabled" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-password-login-enabled">启用密码登录</label>
              </div>
            </div>
            <div class="col-md-6">
              <div class="form-check">
                <input id="setting-webauthn-login-enabled" v-model="organizationLoginPolicyForm.webauthnLoginEnabled" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-webauthn-login-enabled">启用通行密钥登录</label>
              </div>
            </div>
            <div class="col-md-4">
              <div class="form-check mb-2">
                <input id="setting-allow-username" v-model="organizationLoginPolicyForm.allowUsername" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-allow-username">支持用户名</label>
              </div>
              <BFormSelect v-model="organizationLoginPolicyForm.usernameMode" :options="fieldVisibilityOptions" />
            </div>
            <div class="col-md-4">
              <div class="form-check mb-2">
                <input id="setting-allow-email" v-model="organizationLoginPolicyForm.allowEmail" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-allow-email">支持邮箱</label>
              </div>
              <BFormSelect v-model="organizationLoginPolicyForm.emailMode" :options="fieldVisibilityOptions" />
            </div>
            <div class="col-md-4">
              <div class="form-check mb-2">
                <input id="setting-allow-phone" v-model="organizationLoginPolicyForm.allowPhone" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-allow-phone">支持手机</label>
              </div>
              <BFormSelect v-model="organizationLoginPolicyForm.phoneMode" :options="fieldVisibilityOptions" />
            </div>
          </div>
          <div class="d-flex justify-content-end mt-3">
            <BButton type="submit" variant="primary">保存登录策略</BButton>
          </div>
        </BForm>
      </div>

      <div id="setting-password-policy" class="info-card">
        <div class="section-title">密码策略设置</div>
        <BForm @submit.prevent="saveOrganizationPasswordPolicy">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label">最少位数</label>
              <BFormInput v-model="organizationPasswordPolicyForm.minLength" type="number" min="6" />
            </div>
            <div class="col-md-4">
              <div class="form-check mt-4">
                <input id="setting-password-uppercase" v-model="organizationPasswordPolicyForm.requireUppercase" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-password-uppercase">必须包含大写字母</label>
              </div>
            </div>
            <div class="col-md-4">
              <div class="form-check mt-4">
                <input id="setting-password-lowercase" v-model="organizationPasswordPolicyForm.requireLowercase" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-password-lowercase">必须包含小写字母</label>
              </div>
            </div>
            <div class="col-md-4">
              <div class="form-check">
                <input id="setting-password-number" v-model="organizationPasswordPolicyForm.requireNumber" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-password-number">必须包含数字</label>
              </div>
            </div>
            <div class="col-md-4">
              <div class="form-check">
                <input id="setting-password-symbol" v-model="organizationPasswordPolicyForm.requireSymbol" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-password-symbol">必须包含特殊符号</label>
              </div>
            </div>
            <div class="col-md-4">
              <div class="form-check">
                <input id="setting-password-expires" v-model="organizationPasswordPolicyForm.passwordExpires" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-password-expires">密码过期</label>
              </div>
            </div>
            <div class="col-md-4">
              <label class="form-label">过期时间（天）</label>
              <BFormInput v-model="organizationPasswordPolicyForm.expiryDays" type="number" min="1" :disabled="!organizationPasswordPolicyForm.passwordExpires" />
            </div>
          </div>
          <div class="d-flex justify-content-end mt-3">
            <BButton type="submit" variant="primary">保存密码策略</BButton>
          </div>
        </BForm>
      </div>

      <div id="setting-mfa-policy" class="info-card">
        <div class="section-title">两步验证策略</div>
        <BForm @submit.prevent="saveOrganizationMFAPolicy">
          <div class="row g-3">
            <div class="col-12">
              <div class="form-check">
                <input id="setting-mfa-required" v-model="organizationMFAPolicyForm.requireForAllUsers" class="form-check-input" type="checkbox" />
                <label class="form-check-label" for="setting-mfa-required">强制所有用户启用两步验证</label>
              </div>
            </div>
            <div class="col-md-4"><div class="form-check"><input id="setting-mfa-webauthn" v-model="organizationMFAPolicyForm.allowWebauthn" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-webauthn">通行密钥</label></div></div>
            <div class="col-md-4"><div class="form-check"><input id="setting-mfa-totp" v-model="organizationMFAPolicyForm.allowTotp" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-totp">身份验证器（TOTP）</label></div></div>
            <div class="col-md-4"><div class="form-check"><input id="setting-mfa-email" v-model="organizationMFAPolicyForm.allowEmailCode" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-email">邮箱验证码</label></div></div>
            <div class="col-md-4"><div class="form-check"><input id="setting-mfa-sms" v-model="organizationMFAPolicyForm.allowSmsCode" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-sms">手机验证码</label></div></div>
            <div class="col-md-4"><div class="form-check"><input id="setting-mfa-u2f" v-model="organizationMFAPolicyForm.allowU2f" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-u2f">安全密钥</label></div></div>
            <div class="col-md-4"><div class="form-check"><input id="setting-mfa-recovery" v-model="organizationMFAPolicyForm.allowRecoveryCode" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-recovery">备用验证码</label></div></div>
            <div class="col-12">
              <div class="detail-card h-100">
                <div class="d-flex align-items-center justify-content-between mb-3">
                  <div>
                    <div class="section-subtitle mb-1">邮箱验证码通道</div>
                    <div class="record-meta">用于组织级邮箱验证码发送配置。</div>
                  </div>
                  <div class="form-check m-0">
                    <input id="setting-mfa-email-channel-enabled" v-model="organizationMFAPolicyForm.emailChannelEnabled" class="form-check-input" type="checkbox" />
                    <label class="form-check-label ms-2" for="setting-mfa-email-channel-enabled">启用 SMTP</label>
                  </div>
                </div>
                <div class="row g-3">
                  <div class="col-md-6">
                    <label class="form-label">发件人邮箱</label>
                    <BFormInput v-model="organizationMFAPolicyForm.emailChannelFrom" type="email" />
                  </div>
                  <div class="col-md-6">
                    <label class="form-label">SMTP 主机</label>
                    <BFormInput v-model="organizationMFAPolicyForm.emailChannelHost" />
                  </div>
                  <div class="col-md-4">
                    <label class="form-label">SMTP 端口</label>
                    <BFormInput v-model="organizationMFAPolicyForm.emailChannelPort" type="number" min="1" />
                  </div>
                  <div class="col-md-4">
                    <label class="form-label">SMTP 用户名</label>
                    <BFormInput v-model="organizationMFAPolicyForm.emailChannelUsername" />
                  </div>
                  <div class="col-md-4">
                    <label class="form-label">SMTP 密码</label>
                    <BFormInput v-model="organizationMFAPolicyForm.emailChannelPassword" type="password" />
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="d-flex justify-content-end mt-3">
            <BButton type="submit" variant="primary">保存两步验证策略</BButton>
          </div>
        </BForm>
      </div>

      <div id="setting-captcha" class="info-card">
        <div class="section-title">验证码设置</div>
        <BForm @submit.prevent="saveOrganizationCaptchaSettings">
          <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label">验证码类型</label>
              <BFormSelect v-model="organizationCaptchaForm.provider" :options="captchaProviderOptions" />
            </div>
            <template v-if="organizationCaptchaForm.provider === 'google' || organizationCaptchaForm.provider === 'cloudflare'">
              <div class="col-md-4">
                <label class="form-label">client_key</label>
                <BFormInput v-model="organizationCaptchaForm.client_key" />
              </div>
              <div class="col-md-4">
                <label class="form-label">client_secret</label>
                <BFormInput v-model="organizationCaptchaForm.client_secret" type="password" />
              </div>
            </template>
          </div>
          <div class="record-meta mt-3">选择 Google 或 Cloudflare 时，必须填写 client_key 和 client_secret。默认验证码不需要这两个值。</div>
          <div class="d-flex justify-content-end mt-3">
            <BButton type="submit" variant="primary">保存验证码设置</BButton>
          </div>
        </BForm>
      </div>

      <div id="setting-external-idp" class="info-card">
        <div class="section-title">外部 IdP 设置</div>
        <div class="record-meta mb-3">采用预置 Provider 模板，点击启用后在弹窗中配置应用参数。</div>
        <div class="record-list">
          <div v-for="item in externalIdpProviderRows" :key="item.id" class="record-row">
            <div>
              <strong>{{ item.label }}</strong>
              <div class="record-meta">{{ item.summary }}</div>
            </div>
            <BButton size="sm" variant="outline-primary" @click="openExternalIdpEditor(item.id)">{{ item.enabled ? '配置' : (item.id.startsWith('custom_') ? '添加' : '启用') }}</BButton>
          </div>
        </div>
        <div v-if="customExternalIdps.length" class="detail-card mt-3">
          <div class="record-meta mb-2">已添加的自定义 Provider</div>
          <div v-for="item in customExternalIdps" :key="item.id" class="record-row">
            <div>
              <strong>{{ item.name }}</strong>
              <div class="record-meta">{{ String(item.protocol || '').toUpperCase() }} · {{ item.issuer || '-' }}</div>
            </div>
            <BButton size="sm" variant="outline-primary" @click="openExistingExternalIdp(item)">配置</BButton>
          </div>
        </div>
      </div>
    </div>
  </section>

  <ExternalIdpConfigModal
    :visible="externalIDPConfigModalVisible"
    :kind="currentExternalIDPKind"
    :form="externalIDPForm"
    @update:visible="externalIDPConfigModalVisible = $event"
    @submit="submitExternalIDPConfig"
  />
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch, watchEffect } from 'vue'
import { useToast } from '@shared/composables/toast'
import ExternalIdpConfigModal from '../modal/ExternalIdpConfigModal.vue'
import {
  createExternalIdp as apiCreateExternalIdp,
  queryExternalIdps as apiQueryExternalIdps,
  updateExternalIdp as apiUpdateExternalIdp
} from '../api/manage/external_idp'
import { useConsoleStore } from '../stores/console'
import { useOrganizationStore } from '../stores/organization'
import { BButton, BForm, BFormInput, BFormSelect } from 'bootstrap-vue-next'

const toast = useToast()
const console = useConsoleStore()
const organizationStore = useOrganizationStore()

watchEffect(() => {
  console.setPageHeader('设置', '配置外部 OAuth/OIDC 联邦与身份绑定。')
})

const fieldVisibilityOptions = [
  { value: 'hidden', text: '隐藏' },
  { value: 'optional', text: '选填' },
  { value: 'required', text: '必填' }
]
const captchaProviderOptions = [
  { value: 'disabled', text: '不开启' },
  { value: 'default', text: '默认' },
  { value: 'google', text: 'Google' },
  { value: 'cloudflare', text: 'Cloudflare' }
]
const externalIdps = ref<any[]>([])
const externalIDPConfigModalVisible = ref(false)
const currentExternalIDPKind = ref<'google' | 'github' | 'apple' | 'qq' | 'weibo' | 'custom_oauth' | 'custom_oidc'>('google')
const organizationDomainRows = ref<OrganizationDomainRow[]>([])

const externalIDPForm = reactive({
  id: '',
  organizationId: '',
  protocol: 'oidc',
  name: '',
  issuer: '',
  clientId: '',
  clientSecret: '',
  scopes: '',
  authorizationUrl: '',
  tokenUrl: '',
  userInfoUrl: '',
  jwksUrl: ''
})
const organizationLoginPolicyForm = reactive({
  passwordLoginEnabled: true,
  webauthnLoginEnabled: true,
  allowUsername: true,
  allowEmail: true,
  allowPhone: true,
  usernameMode: 'optional' as 'hidden' | 'optional' | 'required',
  emailMode: 'required' as 'hidden' | 'optional' | 'required',
  phoneMode: 'optional' as 'hidden' | 'optional' | 'required'
})
const organizationPasswordPolicyForm = reactive({
  minLength: 12,
  requireUppercase: true,
  requireLowercase: true,
  requireNumber: true,
  requireSymbol: false,
  passwordExpires: false,
  expiryDays: 90
})
const organizationMFAPolicyForm = reactive({
  requireForAllUsers: false,
  allowWebauthn: true,
  allowTotp: true,
  allowEmailCode: true,
  allowSmsCode: false,
  allowU2f: true,
  allowRecoveryCode: true,
  emailChannelEnabled: false,
  emailChannelFrom: '',
  emailChannelHost: '',
  emailChannelPort: 587,
  emailChannelUsername: '',
  emailChannelPassword: ''
})
const organizationCaptchaForm = reactive({
  provider: 'disabled' as 'disabled' | 'default' | 'google' | 'cloudflare',
  client_key: '',
  client_secret: ''
})

type ProviderKind = typeof providerKinds[number]
type OrganizationDomainRow = {
  id: string
  host: string
  verified: boolean
}

type OrganizationConsoleSettings = {
  domains: Array<{
    host: string
    verified: boolean
  }>
  loginPolicy: {
    passwordLoginEnabled: boolean
    webauthnLoginEnabled: boolean
    allowUsername: boolean
    allowEmail: boolean
    allowPhone: boolean
    usernameMode: 'hidden' | 'optional' | 'required'
    emailMode: 'hidden' | 'optional' | 'required'
    phoneMode: 'hidden' | 'optional' | 'required'
  }
  passwordPolicy: {
    minLength: number
    requireUppercase: boolean
    requireLowercase: boolean
    requireNumber: boolean
    requireSymbol: boolean
    passwordExpires: boolean
    expiryDays: number
  }
  mfaPolicy: {
    requireForAllUsers: boolean
    allowWebauthn: boolean
    allowTotp: boolean
    allowEmailCode: boolean
    allowSmsCode: boolean
    allowU2f: boolean
    allowRecoveryCode: boolean
    emailChannel: {
      enabled: boolean
      from: string
      host: string
      port: number
      username: string
      password: string
    }
  }
  captcha: {
    provider: 'disabled' | 'default' | 'google' | 'cloudflare'
    client_key: string
    client_secret: string
  }
}

type ExternalIdpFormPayload = {
  kind: ProviderKind
  form: {
    id: string
    organizationId: string
    protocol: string
    name: string
    issuer: string
    clientId: string
    clientSecret: string
    scopes: string
    authorizationUrl: string
    tokenUrl: string
    userInfoUrl: string
    jwksUrl: string
  }
}

const currentModulePanels = [
  { id: 'setting-domain', label: '域名设置' },
  { id: 'setting-login-policy', label: '登录策略设置' },
  { id: 'setting-password-policy', label: '密码策略设置' },
  { id: 'setting-mfa-policy', label: '两步验证策略' },
  { id: 'setting-captcha', label: '验证码设置' },
  { id: 'setting-external-idp', label: '外部 IdP 设置' }
]

const providerKinds = ['google', 'github', 'apple', 'qq', 'weibo', 'custom_oauth', 'custom_oidc'] as const

const externalIdpProviderRows = computed(() => providerKinds.map((kind) => {
  const provider = findExistingExternalIdp(kind)
  return {
    id: kind,
    label: providerLabel(kind),
    summary: provider ? `已配置 · ${provider.clientId || provider.issuer || provider.name}` : '未启用',
    enabled: Boolean(provider)
  }
}))

const customExternalIdps = computed(() => externalIdps.value.filter((item: any) => {
  const kind = normalizeProviderKind(item)
  return kind === 'custom_oauth' || kind === 'custom_oidc'
}))

watch(
  () => organizationStore.currentOrganization,
  (organization) => {
    const settings = parseOrganizationConsoleSettings(organization)
    externalIDPForm.organizationId = organization?.id || externalIDPForm.organizationId
    organizationLoginPolicyForm.passwordLoginEnabled = settings.loginPolicy.passwordLoginEnabled
    organizationLoginPolicyForm.webauthnLoginEnabled = settings.loginPolicy.webauthnLoginEnabled
    organizationLoginPolicyForm.allowUsername = settings.loginPolicy.allowUsername
    organizationLoginPolicyForm.allowEmail = settings.loginPolicy.allowEmail
    organizationLoginPolicyForm.allowPhone = settings.loginPolicy.allowPhone
    organizationLoginPolicyForm.usernameMode = settings.loginPolicy.usernameMode
    organizationLoginPolicyForm.emailMode = settings.loginPolicy.emailMode
    organizationLoginPolicyForm.phoneMode = settings.loginPolicy.phoneMode
    organizationPasswordPolicyForm.minLength = settings.passwordPolicy.minLength
    organizationPasswordPolicyForm.requireUppercase = settings.passwordPolicy.requireUppercase
    organizationPasswordPolicyForm.requireLowercase = settings.passwordPolicy.requireLowercase
    organizationPasswordPolicyForm.requireNumber = settings.passwordPolicy.requireNumber
    organizationPasswordPolicyForm.requireSymbol = settings.passwordPolicy.requireSymbol
    organizationPasswordPolicyForm.passwordExpires = settings.passwordPolicy.passwordExpires
    organizationPasswordPolicyForm.expiryDays = settings.passwordPolicy.expiryDays
    organizationMFAPolicyForm.requireForAllUsers = settings.mfaPolicy.requireForAllUsers
    organizationMFAPolicyForm.allowWebauthn = settings.mfaPolicy.allowWebauthn
    organizationMFAPolicyForm.allowTotp = settings.mfaPolicy.allowTotp
    organizationMFAPolicyForm.allowEmailCode = settings.mfaPolicy.allowEmailCode
    organizationMFAPolicyForm.allowSmsCode = settings.mfaPolicy.allowSmsCode
    organizationMFAPolicyForm.allowU2f = settings.mfaPolicy.allowU2f
    organizationMFAPolicyForm.allowRecoveryCode = settings.mfaPolicy.allowRecoveryCode
    organizationMFAPolicyForm.emailChannelEnabled = settings.mfaPolicy.emailChannel.enabled
    organizationMFAPolicyForm.emailChannelFrom = settings.mfaPolicy.emailChannel.from
    organizationMFAPolicyForm.emailChannelHost = settings.mfaPolicy.emailChannel.host
    organizationMFAPolicyForm.emailChannelPort = settings.mfaPolicy.emailChannel.port
    organizationMFAPolicyForm.emailChannelUsername = settings.mfaPolicy.emailChannel.username
    organizationMFAPolicyForm.emailChannelPassword = settings.mfaPolicy.emailChannel.password
    organizationCaptchaForm.provider = settings.captcha.provider
    organizationCaptchaForm.client_key = settings.captcha.client_key
    organizationCaptchaForm.client_secret = settings.captcha.client_secret
    organizationDomainRows.value = settings.domains.map((item) => ({
      id: createLocalRowId(),
      host: item.host,
      verified: item.verified
    }))
  },
  { immediate: true }
)

watch(
  () => console.currentOrganizationId,
  async (organizationId) => {
    externalIDPForm.organizationId = organizationId || ''
    if (!organizationId) {
      externalIdps.value = []
      return
    }
    await loadExternalIdps(organizationId)
  },
  { immediate: true }
)

function createLocalRowId() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function providerLabel(kind: typeof providerKinds[number]) {
  if (kind === 'google') return 'Google'
  if (kind === 'github') return 'GitHub'
  if (kind === 'apple') return 'Apple'
  if (kind === 'qq') return 'QQ'
  if (kind === 'weibo') return '新浪微博'
  if (kind === 'custom_oauth') return '自定义 OAuth'
  return '自定义 OIDC'
}

function normalizeProviderName(value?: string) {
  return String(value || '').trim().toLowerCase()
}

function normalizeProviderKind(item: any) {
  const metadataKind = normalizeProviderName(item?.metadata?.providerKind)
  if (metadataKind) {
    return metadataKind
  }
  const normalizedName = normalizeProviderName(item?.name)
  if (normalizedName === 'google') return 'google'
  if (normalizedName === 'github') return 'github'
  if (normalizedName === 'apple') return 'apple'
  if (normalizedName === 'qq') return 'qq'
  if (normalizedName === 'weibo' || normalizedName === '新浪微博') return 'weibo'
  if (item?.protocol === 'oidc') return 'custom_oidc'
  return 'custom_oauth'
}

function findExistingExternalIdp(kind: typeof providerKinds[number]) {
  if (kind === 'custom_oauth' || kind === 'custom_oidc') {
    return null
  }
  return externalIdps.value.find((item: any) => {
    const normalizedKind = normalizeProviderKind(item)
    const normalizedName = normalizeProviderName(item?.name)
    return normalizedKind === kind || normalizedName === kind
  }) || null
}

function providerPreset(kind: ProviderKind) {
  if (kind === 'google') {
    return {
      protocol: 'oidc',
      name: 'Google',
      issuer: 'https://accounts.google.com',
      scopes: 'openid profile email',
      authorizationUrl: 'https://accounts.google.com/o/oauth2/v2/auth',
      tokenUrl: 'https://oauth2.googleapis.com/token',
      userInfoUrl: 'https://openidconnect.googleapis.com/v1/userinfo',
      jwksUrl: 'https://www.googleapis.com/oauth2/v3/certs'
    }
  }
  if (kind === 'github') {
    return {
      protocol: 'oauth',
      name: 'GitHub',
      issuer: 'https://github.com',
      scopes: 'read:user user:email',
      authorizationUrl: 'https://github.com/login/oauth/authorize',
      tokenUrl: 'https://github.com/login/oauth/access_token',
      userInfoUrl: 'https://api.github.com/user',
      jwksUrl: ''
    }
  }
  if (kind === 'apple') {
    return {
      protocol: 'oidc',
      name: 'Apple',
      issuer: 'https://appleid.apple.com',
      scopes: 'name email',
      authorizationUrl: 'https://appleid.apple.com/auth/authorize',
      tokenUrl: 'https://appleid.apple.com/auth/token',
      userInfoUrl: '',
      jwksUrl: 'https://appleid.apple.com/auth/keys'
    }
  }
  if (kind === 'qq') {
    return {
      protocol: 'oauth',
      name: 'QQ',
      issuer: 'https://graph.qq.com',
      scopes: 'get_user_info',
      authorizationUrl: 'https://graph.qq.com/oauth2.0/authorize',
      tokenUrl: 'https://graph.qq.com/oauth2.0/token',
      userInfoUrl: 'https://graph.qq.com/user/get_user_info',
      jwksUrl: ''
    }
  }
  if (kind === 'weibo') {
    return {
      protocol: 'oauth',
      name: 'Weibo',
      issuer: 'https://api.weibo.com',
      scopes: 'email',
      authorizationUrl: 'https://api.weibo.com/oauth2/authorize',
      tokenUrl: 'https://api.weibo.com/oauth2/access_token',
      userInfoUrl: 'https://api.weibo.com/2/users/show.json',
      jwksUrl: ''
    }
  }
  if (kind === 'custom_oauth') {
    return {
      protocol: 'oauth',
      name: 'Custom OAuth',
      issuer: '',
      scopes: '',
      authorizationUrl: '',
      tokenUrl: '',
      userInfoUrl: '',
      jwksUrl: ''
    }
  }
  return {
    protocol: 'oidc',
    name: 'Custom OIDC',
    issuer: '',
    scopes: 'openid profile email',
    authorizationUrl: '',
    tokenUrl: '',
    userInfoUrl: '',
    jwksUrl: ''
  }
}

function buildExternalIdpForm(kind: ProviderKind, item?: any): ExternalIdpFormPayload {
  const preset = providerPreset(kind)
  return {
    kind,
    form: {
      id: item?.id || '',
      organizationId: console.currentOrganizationId || '',
      protocol: item?.protocol || preset.protocol,
      name: item?.name || preset.name,
      issuer: item?.issuer || preset.issuer,
      clientId: item?.clientId || '',
      clientSecret: '',
      scopes: item?.scopes || preset.scopes,
      authorizationUrl: item?.authorizationUrl || preset.authorizationUrl,
      tokenUrl: item?.tokenUrl || preset.tokenUrl,
      userInfoUrl: item?.userInfoUrl || preset.userInfoUrl,
      jwksUrl: item?.jwksUrl || preset.jwksUrl
    }
  }
}

function scrollToPanel(id: string) {
  const target = document.getElementById(id)
  if (!target) {
    return
  }
  const topbar = document.querySelector('.admin-topbar') as HTMLElement | null
  const offset = (topbar?.offsetHeight ?? 0) + 32
  const targetTop = target.getBoundingClientRect().top + window.scrollY - offset
  window.scrollTo({ top: Math.max(targetTop, 0), behavior: 'smooth' })
}

async function withFeedback(fn: () => Promise<void>, successMessage = '操作成功') {
  try {
    await fn()
    toast.success(successMessage)
  } catch (error) {
    toast.error(String(error))
  }
}

async function loadExternalIdps(organizationId = console.currentOrganizationId) {
  if (!organizationId) {
    externalIdps.value = []
    return
  }
  const response = await apiQueryExternalIdps(organizationId)
  externalIdps.value = response.items
}

async function saveOrganizationConsoleSettings(options: { name?: string; description?: string } = {}) {
  await organizationStore.saveOrganizationConsoleSettings(buildOrganizationConsoleSettings(), options)
}

function parseOrganizationConsoleSettings(organization?: any): OrganizationConsoleSettings {
  const defaults: OrganizationConsoleSettings = {
    domains: [],
    loginPolicy: {
      passwordLoginEnabled: true,
      webauthnLoginEnabled: true,
      allowUsername: true,
      allowEmail: true,
      allowPhone: true,
      usernameMode: 'optional',
      emailMode: 'required',
      phoneMode: 'optional'
    },
    passwordPolicy: {
      minLength: 12,
      requireUppercase: true,
      requireLowercase: true,
      requireNumber: true,
      requireSymbol: false,
      passwordExpires: false,
      expiryDays: 90
    },
    mfaPolicy: {
      requireForAllUsers: false,
      allowWebauthn: true,
      allowTotp: true,
      allowEmailCode: true,
      allowSmsCode: false,
      allowU2f: true,
      allowRecoveryCode: true,
      emailChannel: {
        enabled: false,
        from: '',
        host: '',
        port: 587,
        username: '',
        password: ''
      }
    },
    captcha: {
      provider: 'disabled',
      client_key: '',
      client_secret: ''
    }
  }
  const parsed = organization?.consoleSettings
  if (!parsed || typeof parsed !== 'object') {
    return defaults
  }
  return {
    ...defaults,
    ...parsed,
    loginPolicy: { ...defaults.loginPolicy, ...(parsed.loginPolicy || {}) },
    passwordPolicy: { ...defaults.passwordPolicy, ...(parsed.passwordPolicy || {}) },
    mfaPolicy: {
      ...defaults.mfaPolicy,
      ...(parsed.mfaPolicy || {}),
      emailChannel: {
        ...defaults.mfaPolicy.emailChannel,
        ...((parsed.mfaPolicy && parsed.mfaPolicy.emailChannel) || {})
      }
    },
    captcha: {
      ...defaults.captcha,
      ...(parsed.captcha || {}),
      provider: ['disabled', 'default', 'google', 'cloudflare'].includes(String(parsed?.captcha?.provider || '').toLowerCase())
        ? String(parsed.captcha.provider).toLowerCase() as OrganizationConsoleSettings['captcha']['provider']
        : defaults.captcha.provider,
      client_key: String(parsed?.captcha?.client_key || parsed?.captcha?.clientKey || ''),
      client_secret: String(parsed?.captcha?.client_secret || parsed?.captcha?.clientSecret || '')
    },
    domains: Array.isArray(parsed.domains)
      ? parsed.domains.map((item: any) => ({
          host: String(item.host || ''),
          verified: Boolean(item.verified)
        })).filter((item: OrganizationConsoleSettings['domains'][number]) => item.host)
      : []
  }
}

function buildOrganizationConsoleSettings(): OrganizationConsoleSettings {
  return {
    domains: organizationDomainRows.value
      .map((item) => ({
        host: item.host.trim(),
        verified: item.verified
      }))
      .filter((item) => item.host),
    loginPolicy: {
      passwordLoginEnabled: organizationLoginPolicyForm.passwordLoginEnabled,
      webauthnLoginEnabled: organizationLoginPolicyForm.webauthnLoginEnabled,
      allowUsername: organizationLoginPolicyForm.allowUsername,
      allowEmail: organizationLoginPolicyForm.allowEmail,
      allowPhone: organizationLoginPolicyForm.allowPhone,
      usernameMode: organizationLoginPolicyForm.usernameMode,
      emailMode: organizationLoginPolicyForm.emailMode,
      phoneMode: organizationLoginPolicyForm.phoneMode
    },
    passwordPolicy: {
      minLength: Number(organizationPasswordPolicyForm.minLength),
      requireUppercase: organizationPasswordPolicyForm.requireUppercase,
      requireLowercase: organizationPasswordPolicyForm.requireLowercase,
      requireNumber: organizationPasswordPolicyForm.requireNumber,
      requireSymbol: organizationPasswordPolicyForm.requireSymbol,
      passwordExpires: organizationPasswordPolicyForm.passwordExpires,
      expiryDays: Number(organizationPasswordPolicyForm.expiryDays)
    },
    mfaPolicy: {
      requireForAllUsers: organizationMFAPolicyForm.requireForAllUsers,
      allowWebauthn: organizationMFAPolicyForm.allowWebauthn,
      allowTotp: organizationMFAPolicyForm.allowTotp,
      allowEmailCode: organizationMFAPolicyForm.allowEmailCode,
      allowSmsCode: organizationMFAPolicyForm.allowSmsCode,
      allowU2f: organizationMFAPolicyForm.allowU2f,
      allowRecoveryCode: organizationMFAPolicyForm.allowRecoveryCode,
      emailChannel: {
        enabled: organizationMFAPolicyForm.emailChannelEnabled,
        from: organizationMFAPolicyForm.emailChannelFrom.trim(),
        host: organizationMFAPolicyForm.emailChannelHost.trim(),
        port: Number(organizationMFAPolicyForm.emailChannelPort),
        username: organizationMFAPolicyForm.emailChannelUsername.trim(),
        password: organizationMFAPolicyForm.emailChannelPassword
      }
    },
    captcha: {
      provider: organizationCaptchaForm.provider,
      client_key: organizationCaptchaForm.client_key.trim(),
      client_secret: organizationCaptchaForm.client_secret.trim()
    }
  }
}

async function saveOrganizationDomainSettings() {
  await withFeedback(async () => {
    await saveOrganizationConsoleSettings()
  })
}

async function saveOrganizationLoginPolicy() {
  await withFeedback(async () => {
    await saveOrganizationConsoleSettings()
  })
}

async function saveOrganizationPasswordPolicy() {
  await withFeedback(async () => {
    await saveOrganizationConsoleSettings()
  })
}

async function saveOrganizationMFAPolicy() {
  await withFeedback(async () => {
    await saveOrganizationConsoleSettings()
  })
}

async function saveOrganizationCaptchaSettings() {
  await withFeedback(async () => {
    await saveOrganizationConsoleSettings()
  })
}

function addOrganizationDomainRow() {
  organizationDomainRows.value.push({
    id: createLocalRowId(),
    host: '',
    verified: false
  })
}

function removeOrganizationDomainRow(index: number) {
  organizationDomainRows.value.splice(index, 1)
}

function verifyOrganizationDomain(index: number) {
  const item = organizationDomainRows.value[index]
  if (!item || !item.host.trim()) {
    toast.error('请先填写域名')
    return
  }
  item.verified = true
  toast.success('域名已标记为已验证')
}

async function createExternalIDP(form: {
  id?: string
  name: string
  protocol: string
  issuer: string
  clientId: string
  clientSecret: string
  scopes: string
  authorizationUrl: string
  tokenUrl: string
  userInfoUrl: string
  jwksUrl: string
}) {
  await withFeedback(async () => {
    await apiCreateExternalIdp({
      organizationId: console.currentOrganizationId || externalIDPForm.organizationId,
      protocol: form.protocol,
      name: form.name,
      issuer: form.issuer,
      clientId: form.clientId,
      clientSecret: form.clientSecret,
      scopes: form.scopes,
      authorizationUrl: form.authorizationUrl,
      tokenUrl: form.tokenUrl,
      userInfoUrl: form.userInfoUrl,
      jwksUrl: form.jwksUrl,
      metadata: {
        providerKind: currentExternalIDPKind.value
      }
    })
    await loadExternalIdps()
  })
}

async function updateExternalIDP(form: {
  id?: string
  name: string
  protocol: string
  issuer: string
  clientId: string
  clientSecret: string
  scopes: string
  authorizationUrl: string
  tokenUrl: string
  userInfoUrl: string
  jwksUrl: string
}) {
  await withFeedback(async () => {
    await apiUpdateExternalIdp({
      id: form.id || '',
      organizationId: console.currentOrganizationId || externalIDPForm.organizationId,
      protocol: form.protocol,
      name: form.name,
      issuer: form.issuer,
      clientId: form.clientId,
      clientSecret: form.clientSecret,
      scopes: form.scopes,
      authorizationUrl: form.authorizationUrl,
      tokenUrl: form.tokenUrl,
      userInfoUrl: form.userInfoUrl,
      jwksUrl: form.jwksUrl,
      metadata: {
        providerKind: currentExternalIDPKind.value
      }
    })
    await loadExternalIdps()
  })
}

async function submitExternalIDPConfig(form: {
  id?: string
  name: string
  protocol: string
  issuer: string
  clientId: string
  clientSecret: string
  scopes: string
  authorizationUrl: string
  tokenUrl: string
  userInfoUrl: string
  jwksUrl: string
}) {
  if (form.id) {
    await updateExternalIDP(form)
  } else {
    await createExternalIDP(form)
  }
  externalIDPConfigModalVisible.value = false
}

function openExternalIDPEditor(payload: {
  kind: 'google' | 'github' | 'apple' | 'qq' | 'weibo' | 'custom_oauth' | 'custom_oidc'
  form: {
    id: string
    organizationId: string
    protocol: string
    name: string
    issuer: string
    clientId: string
    clientSecret: string
    scopes: string
    authorizationUrl: string
    tokenUrl: string
    userInfoUrl: string
    jwksUrl: string
  }
}) {
  currentExternalIDPKind.value = payload.kind
  externalIDPForm.id = payload.form.id
  externalIDPForm.organizationId = payload.form.organizationId
  externalIDPForm.protocol = payload.form.protocol
  externalIDPForm.name = payload.form.name
  externalIDPForm.issuer = payload.form.issuer
  externalIDPForm.clientId = payload.form.clientId
  externalIDPForm.clientSecret = payload.form.clientSecret
  externalIDPForm.scopes = payload.form.scopes
  externalIDPForm.authorizationUrl = payload.form.authorizationUrl
  externalIDPForm.tokenUrl = payload.form.tokenUrl
  externalIDPForm.userInfoUrl = payload.form.userInfoUrl
  externalIDPForm.jwksUrl = payload.form.jwksUrl
  externalIDPConfigModalVisible.value = true
}

function openExternalIdpEditor(kind: ProviderKind) {
  const provider = findExistingExternalIdp(kind)
  openExternalIDPEditor(buildExternalIdpForm(kind, provider))
}

function openExistingExternalIdp(item: any) {
  const kind = normalizeProviderKind(item) as ProviderKind
  openExternalIDPEditor(buildExternalIdpForm(kind, item))
}
</script>
