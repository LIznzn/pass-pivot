<template>
  <section class="console-module-layout">
    <aside class="console-module-sidebar">
      <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
    </aside>
    <div class="console-module-main">
      <div id="setting-domain" class="info-card">
        <div class="section-title">域名设置</div>
        <div class="record-meta mb-3">点击“添加域名”后在弹窗中输入域名并保存。新增域名默认为未验证，可在列表中继续验证所有权或删除。</div>
        <div v-for="(item, index) in organizationDomainRows" :key="item.id" class="detail-card mb-3">
          <div class="row g-2 align-items-center">
            <div class="col-md-7">
              <div class="form-control bg-body-tertiary">{{ item.host }}</div>
            </div>
            <div class="col-md-2">
              <span class="badge" :class="item.verified ? 'text-bg-success' : 'text-bg-secondary'">{{ item.verified ? '已验证' : '未验证' }}</span>
            </div>
            <div class="col-md-3 d-flex gap-2 justify-content-md-end">
              <BButton type="button" size="sm" variant="outline-primary" :disabled="item.verified" @click="openDomainVerificationModal(index)">验证所有权</BButton>
              <BButton type="button" size="sm" variant="outline-danger" @click="deleteOrganizationDomain(index)">删除</BButton>
            </div>
          </div>
          <div v-if="item.verifiedAt" class="record-meta mt-2">最近验证时间：{{ formatDateTime(item.verifiedAt) }}</div>
        </div>
        <div class="d-flex justify-content-start align-items-center mt-3">
          <BButton type="button" variant="outline-secondary" @click="openCreateDomainModal">添加域名</BButton>
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
            <BButton size="sm" variant="outline-primary" @click="openExternalIdpEditor(item.id)">{{ item.enabled ? '配置' : '启用' }}</BButton>
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
  <DomainCreateModal
    :visible="domainCreateModalVisible"
    :form="domainCreateForm"
    @update:visible="domainCreateModalVisible = $event"
    @hidden="resetCreateDomainForm"
    @submit="submitCreateDomain"
  />
  <DomainVerificationModal
    :visible="domainVerificationModalVisible"
    :host="currentDomainVerificationRow?.host || ''"
    :method="currentDomainVerificationRow?.verificationMethod || 'http_file'"
    :verified="Boolean(currentDomainVerificationRow?.verified)"
    :challenge="currentDomainVerificationChallenge"
    @update:visible="domainVerificationModalVisible = $event"
    @prepare="prepareOrganizationDomainVerification"
    @verify="verifyOrganizationDomain"
  />
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch, watchEffect } from 'vue'
import { useToast } from 'bootstrap-vue-next'
import DomainCreateModal from '@/modal/DomainCreateModal.vue'
import DomainVerificationModal from '@/modal/DomainVerificationModal.vue'
import ExternalIdpConfigModal from '@/modal/ExternalIdpConfigModal.vue'
import {
  prepareOrganizationDomainVerification as apiPrepareOrganizationDomainVerification,
  verifyOrganizationDomain as apiVerifyOrganizationDomain,
} from '@/api/manage/organization'
import {
  createExternalIdp as apiCreateExternalIdp,
  queryExternalIdps as apiQueryExternalIdps,
  updateExternalIdp as apiUpdateExternalIdp
} from '@/api/manage/external_idp'
import { useConsoleStore } from '@/stores/console'
import { useOrganizationStore } from '@/stores/organization'
import { BButton, BForm, BFormInput, BFormSelect } from 'bootstrap-vue-next'
import { notifyToast } from '@shared/utils/notify'

const toast = useToast()
const console = useConsoleStore()
const organizationStore = useOrganizationStore()
const formatDateTime = console.formatDateTime

function showToast(
  message: string,
  variant: 'success' | 'danger',
  options: {
    source: string
    trigger?: string
    error?: unknown
    metadata?: Record<string, unknown>
  } = {
    source: 'console/Settings'
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
const currentExternalIDPKind = ref<'google' | 'github' | 'apple'>('google')
const organizationDomainRows = ref<OrganizationDomainRow[]>([])
const domainCreateModalVisible = ref(false)
const domainVerificationModalVisible = ref(false)
const currentDomainVerificationIndex = ref(-1)
const currentDomainVerificationChallenge = ref<DomainVerificationChallenge>({
  token: '',
  fileUrl: '',
  fileContent: '',
  txtRecordName: '',
  txtRecordValue: ''
})

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
const domainCreateForm = reactive({
  host: ''
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
  allowRecoveryCode: true
})
const organizationMailSettingsSnapshot = reactive({
  provider: 'disabled' as 'disabled' | 'smtp' | 'mailgun' | 'sendgrid',
  from: '',
  smtpHost: '',
  smtpPort: 587,
  smtpUser: '',
  smtpPass: '',
  mailgunDomain: '',
  mailgunApiKey: '',
  mailgunApiBase: '',
  sendgridApiKey: ''
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
  verificationMethod: 'http_file' | 'dns_txt'
  verificationToken: string
  verifiedAt: string
}

type DomainVerificationChallenge = {
  token: string
  fileUrl: string
  fileContent: string
  txtRecordName: string
  txtRecordValue: string
}

type OrganizationConsoleSettings = {
    domains: Array<{
      host: string
      verified?: boolean
      verificationMethod: 'http_file' | 'dns_txt'
      verificationToken: string
      verifiedAt?: string
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
  }
  mail: {
    provider: 'disabled' | 'smtp' | 'mailgun' | 'sendgrid'
    from: string
    smtpHost: string
    smtpPort: number
    smtpUser: string
    smtpPass: string
    mailgunDomain: string
    mailgunApiKey: string
    mailgunApiBase: string
    sendgridApiKey: string
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

const providerKinds = ['google', 'github', 'apple'] as const

const externalIdpProviderRows = computed(() => providerKinds.map((kind) => {
  const provider = findExistingExternalIdp(kind)
  return {
    id: kind,
    label: providerLabel(kind),
    summary: provider ? `已配置 · ${provider.clientId || provider.issuer || provider.name}` : '未启用',
    enabled: Boolean(provider)
  }
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
    organizationMailSettingsSnapshot.provider = settings.mail.provider
    organizationMailSettingsSnapshot.from = settings.mail.from
    organizationMailSettingsSnapshot.smtpHost = settings.mail.smtpHost
    organizationMailSettingsSnapshot.smtpPort = settings.mail.smtpPort
    organizationMailSettingsSnapshot.smtpUser = settings.mail.smtpUser
    organizationMailSettingsSnapshot.smtpPass = settings.mail.smtpPass
    organizationMailSettingsSnapshot.mailgunDomain = settings.mail.mailgunDomain
    organizationMailSettingsSnapshot.mailgunApiKey = settings.mail.mailgunApiKey
    organizationMailSettingsSnapshot.mailgunApiBase = settings.mail.mailgunApiBase
    organizationMailSettingsSnapshot.sendgridApiKey = settings.mail.sendgridApiKey
    organizationCaptchaForm.provider = settings.captcha.provider
    organizationCaptchaForm.client_key = settings.captcha.client_key
    organizationCaptchaForm.client_secret = settings.captcha.client_secret
    organizationDomainRows.value = settings.domains.map((item) => ({
      id: createLocalRowId(),
      host: item.host,
      verified: Boolean(item.verified) || Boolean(item.verifiedAt),
      verificationMethod: item.verificationMethod,
      verificationToken: item.verificationToken,
      verifiedAt: item.verifiedAt || ''
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
  return 'Apple'
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
    showToast(successMessage, 'success', {
      source: 'console/Settings.withFeedback',
      trigger: 'withFeedback'
    })
  } catch (error) {
    showToast(String(error), 'danger', {
      source: 'console/Settings.withFeedback',
      trigger: 'withFeedback',
      error
    })
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
      allowRecoveryCode: true
    },
    mail: {
      provider: 'disabled',
      from: '',
      smtpHost: '',
      smtpPort: 587,
      smtpUser: '',
      smtpPass: '',
      mailgunDomain: '',
      mailgunApiKey: '',
      mailgunApiBase: '',
      sendgridApiKey: ''
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
      ...(parsed.mfaPolicy || {})
    },
    mail: {
      ...defaults.mail,
      ...(parsed.mail || {}),
      provider: ['disabled', 'smtp', 'mailgun', 'sendgrid'].includes(String(parsed?.mail?.provider || '').toLowerCase())
        ? String(parsed.mail.provider).toLowerCase() as OrganizationConsoleSettings['mail']['provider']
        : defaults.mail.provider,
      from: String(parsed?.mail?.from || ''),
      smtpHost: String(parsed?.mail?.smtpHost || ''),
      smtpPort: Number(parsed?.mail?.smtpPort || 587),
      smtpUser: String(parsed?.mail?.smtpUser || ''),
      smtpPass: String(parsed?.mail?.smtpPass || ''),
      mailgunDomain: String(parsed?.mail?.mailgunDomain || ''),
      mailgunApiKey: String(parsed?.mail?.mailgunApiKey || ''),
      mailgunApiBase: String(parsed?.mail?.mailgunApiBase || ''),
      sendgridApiKey: String(parsed?.mail?.sendgridApiKey || '')
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
          verificationMethod: String(item.verificationMethod || 'http_file') === 'dns_txt' ? 'dns_txt' : 'http_file',
          verificationToken: String(item.verificationToken || ''),
          verifiedAt: String(item.verifiedAt || '')
        })).filter((item: OrganizationConsoleSettings['domains'][number]) => item.host)
      : []
  }
}

function buildOrganizationConsoleSettings(): OrganizationConsoleSettings {
  return {
    domains: organizationDomainRows.value
      .map((item) => ({
        host: item.host.trim(),
        verificationMethod: item.verificationMethod,
        verificationToken: item.verificationToken
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
      allowRecoveryCode: organizationMFAPolicyForm.allowRecoveryCode
    },
    mail: {
      provider: organizationMailSettingsSnapshot.provider,
      from: organizationMailSettingsSnapshot.from.trim(),
      smtpHost: organizationMailSettingsSnapshot.smtpHost.trim(),
      smtpPort: Number(organizationMailSettingsSnapshot.smtpPort),
      smtpUser: organizationMailSettingsSnapshot.smtpUser.trim(),
      smtpPass: organizationMailSettingsSnapshot.smtpPass,
      mailgunDomain: organizationMailSettingsSnapshot.mailgunDomain.trim(),
      mailgunApiKey: organizationMailSettingsSnapshot.mailgunApiKey.trim(),
      mailgunApiBase: organizationMailSettingsSnapshot.mailgunApiBase.trim(),
      sendgridApiKey: organizationMailSettingsSnapshot.sendgridApiKey.trim()
    },
    captcha: {
      provider: organizationCaptchaForm.provider,
      client_key: organizationCaptchaForm.client_key.trim(),
      client_secret: organizationCaptchaForm.client_secret.trim()
    }
  }
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

const currentDomainVerificationRow = computed(() => {
  if (currentDomainVerificationIndex.value < 0) {
    return null
  }
  return organizationDomainRows.value[currentDomainVerificationIndex.value] || null
})

function openCreateDomainModal() {
  resetCreateDomainForm()
  domainCreateModalVisible.value = true
}

function resetCreateDomainForm() {
  domainCreateForm.host = ''
}

async function submitCreateDomain() {
  const host = domainCreateForm.host.trim()
  if (!host) {
    showToast('请填写域名', 'danger')
    return
  }
  if (organizationDomainRows.value.some((item) => item.host === host)) {
    showToast('域名已存在', 'danger')
    return
  }
  const nextRows = [
    ...organizationDomainRows.value,
    {
      id: createLocalRowId(),
      host,
      verified: false,
      verificationMethod: 'http_file' as const,
      verificationToken: '',
      verifiedAt: ''
    }
  ]
  await withFeedback(async () => {
    organizationDomainRows.value = nextRows
    await saveOrganizationConsoleSettings()
    domainCreateModalVisible.value = false
    resetCreateDomainForm()
  }, '域名已添加')
}

function removeOrganizationDomainRow(index: number) {
  if (currentDomainVerificationIndex.value === index) {
    currentDomainVerificationIndex.value = -1
    domainVerificationModalVisible.value = false
    resetDomainVerificationChallenge()
  } else if (currentDomainVerificationIndex.value > index) {
    currentDomainVerificationIndex.value -= 1
  }
  organizationDomainRows.value.splice(index, 1)
}

async function deleteOrganizationDomain(index: number) {
  const snapshot = organizationDomainRows.value.map((item) => ({ ...item }))
  removeOrganizationDomainRow(index)
  try {
    await saveOrganizationConsoleSettings()
    showToast('域名已删除', 'success', {
      source: 'console/Settings.deleteOrganizationDomain',
      trigger: 'deleteOrganizationDomain'
    })
  } catch (error) {
    organizationDomainRows.value = snapshot
    showToast(String(error), 'danger', {
      source: 'console/Settings.deleteOrganizationDomain',
      trigger: 'deleteOrganizationDomain',
      error
    })
  }
}

function openDomainVerificationModal(index: number) {
  const item = organizationDomainRows.value[index]
  if (!item || !item.host.trim()) {
    showToast('请先填写域名', 'danger')
    return
  }
  currentDomainVerificationIndex.value = index
  currentDomainVerificationChallenge.value = buildDomainVerificationChallenge(item)
  domainVerificationModalVisible.value = true
}

async function prepareOrganizationDomainVerification(method: 'http_file' | 'dns_txt') {
  const item = currentDomainVerificationRow.value
  if (!item || !item.host.trim()) {
    showToast('请先填写域名', 'danger')
    return
  }
  await withFeedback(async () => {
    const organizationId = console.currentOrganizationId || externalIDPForm.organizationId
    const challenge = await apiPrepareOrganizationDomainVerification({
      organizationId,
      host: item.host.trim(),
      method
    })
    item.host = challenge.host
    item.verificationMethod = challenge.method === 'dns_txt' ? 'dns_txt' : 'http_file'
    item.verificationToken = challenge.token
    item.verified = false
    item.verifiedAt = ''
    currentDomainVerificationChallenge.value = {
      token: challenge.token || '',
      fileUrl: challenge.fileUrl || '',
      fileContent: challenge.fileContent || '',
      txtRecordName: challenge.txtRecordName || '',
      txtRecordValue: challenge.txtRecordValue || ''
    }
  }, '已生成域名验证信息')
}

async function verifyOrganizationDomain() {
  const item = currentDomainVerificationRow.value
  if (!item || !item.host.trim()) {
    showToast('请先填写域名', 'danger')
    return
  }
  await withFeedback(async () => {
    const organizationId = console.currentOrganizationId || externalIDPForm.organizationId
    const verifiedDomain = await apiVerifyOrganizationDomain({
      organizationId,
      host: item.host.trim()
    })
    item.host = String(verifiedDomain.host || item.host).trim()
    item.verified = Boolean(verifiedDomain.verified) || Boolean(verifiedDomain.verifiedAt)
    item.verificationMethod = String(verifiedDomain.verificationMethod || item.verificationMethod || 'http_file') === 'dns_txt' ? 'dns_txt' : 'http_file'
    item.verificationToken = String(verifiedDomain.verificationToken || item.verificationToken || '')
    item.verifiedAt = String(verifiedDomain.verifiedAt || '')
    currentDomainVerificationChallenge.value = buildDomainVerificationChallenge(item)
    domainVerificationModalVisible.value = false
    currentDomainVerificationIndex.value = -1
    resetDomainVerificationChallenge()
  }, '域名验证成功')
}

function resetDomainVerificationChallenge() {
  currentDomainVerificationChallenge.value = {
    token: '',
    fileUrl: '',
    fileContent: '',
    txtRecordName: '',
    txtRecordValue: ''
  }
}

function buildDomainVerificationChallenge(item: OrganizationDomainRow): DomainVerificationChallenge {
  const host = item.host.trim()
  const token = item.verificationToken || ''
  return {
    token,
    fileUrl: host ? `https://${host}/.well-known/ppvt-domain-verification.txt` : '',
    fileContent: token,
    txtRecordName: host && !host.includes(':') ? `_ppvt-domain-verification.${host}` : '',
    txtRecordValue: token
  }
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
  kind: 'google' | 'github' | 'apple'
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
</script>
