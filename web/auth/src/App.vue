<template>
  <div class="auth-shell">
    <ToastHost />

    <main class="auth-main">
      <div class="auth-stack">
        <header class="auth-header">
          <div class="auth-brand-mark" aria-hidden="true">P</div>
          <h1 class="auth-title">{{ stageTitle }}</h1>
          <p v-if="organizationDisplayName" class="auth-subnote">{{ text.developedBy(applicationName, organizationDisplayName) }}</p>
        </header>

        <section class="auth-card">
          <div v-if="localizedBootstrapError" class="auth-alert auth-alert-danger">
            {{ localizedBootstrapError }}
          </div>

          <form v-if="bootstrap.stage === 'login'" :action="bootstrap.loginAction" method="post" class="auth-form">
            <input type="hidden" name="interaction" value="login" />

            <label class="auth-field">
              <span>{{ text.identifier }}</span>
              <input
                name="identifier"
                autocomplete="username"
                :placeholder="text.identifierPlaceholder"
              />
            </label>

            <label class="auth-field">
              <span>{{ text.password }}</span>
              <input
                name="secret"
                type="password"
                autocomplete="current-password"
                :placeholder="text.passwordPlaceholder"
              />
            </label>

            <button type="submit" class="auth-button auth-button-primary auth-button-block">
              {{ text.signIn }}
            </button>

            <div v-if="externalIDPs.length" class="auth-idp-block">
              <div class="auth-idp-divider">
                <span>{{ text.or }}</span>
              </div>
              <div class="auth-idp-list">
                <button
                  v-for="provider in externalIDPs"
                  :key="provider.id"
                  type="button"
                  class="auth-idp-button"
                >
                  <span class="auth-idp-badge" aria-hidden="true">{{ resolveProviderGlyph(provider.name) }}</span>
                  <span>{{ text.continueWithProvider(provider.name) }}</span>
                </button>
              </div>
            </div>

            <button v-if="supportsWebAuthnLogin" type="button" class="auth-link-button" @click="loginWithWebAuthn">
              {{ text.signInWithPasskey }}
            </button>
          </form>

          <div v-else-if="bootstrap.stage === 'account'" class="auth-form">
            <div class="auth-confirm-box">
              <div class="auth-confirm-title">{{ text.accountTitle }}</div>
              <div class="auth-account-summary">
                <strong>{{ currentAccountLabel }}</strong>
                <span v-if="currentAccountSecondary">{{ currentAccountSecondary }}</span>
              </div>
            </div>

            <form :action="bootstrap.accountAction" method="post">
              <input type="hidden" name="interaction" value="account" />
              <input type="hidden" name="continue" value="true" />
              <button type="submit" class="auth-button auth-button-primary auth-button-block">
                {{ text.continueAsCurrentAccount }}
              </button>
            </form>

            <form :action="bootstrap.switchAccountAction" method="post">
              <input type="hidden" name="interaction" value="account" />
              <input type="hidden" name="continue" value="false" />
              <button type="submit" class="auth-button auth-button-secondary auth-button-block">
                {{ text.logoutAndSwitchAccount }}
              </button>
            </form>
          </div>

          <div v-else-if="bootstrap.stage === 'confirmation'" class="auth-form">
            <div class="auth-confirm-box">
              <div class="auth-confirm-title">{{ text.confirmationTitle }}</div>
              <ul class="auth-confirm-list">
                <li>{{ text.confirmationItems.trustedDevice }}</li>
                <li>{{ text.confirmationItems.futureSkip }}</li>
                <li>{{ text.confirmationItems.continueWithoutTrust }}</li>
              </ul>
            </div>

            <form :action="bootstrap.confirmAction" method="post">
              <input type="hidden" name="interaction" value="confirm" />
              <input type="hidden" name="accept" value="true" />
              <input type="hidden" name="trustDevice" value="true" />
              <button type="submit" class="auth-button auth-button-primary auth-button-block">
                {{ text.confirmTrustDevice }}
              </button>
            </form>

            <form :action="bootstrap.confirmAction" method="post">
              <input type="hidden" name="interaction" value="confirm" />
              <input type="hidden" name="accept" value="true" />
              <input type="hidden" name="trustDevice" value="false" />
              <button type="submit" class="auth-button auth-button-secondary auth-button-block">
                {{ text.confirmContinueWithoutTrust }}
              </button>
            </form>
          </div>

          <form v-else :action="bootstrap.mfaAction" method="post" class="auth-form">
            <input type="hidden" name="interaction" value="mfa" />

            <label class="auth-field">
              <span>{{ text.verificationMethod }}</span>
              <select v-model="selectedMethod" name="method">
                <option v-for="option in localizedMFAOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>

            <label v-if="showsCodeInput" class="auth-field">
              <span>{{ text.verificationCode }}</span>
              <div class="auth-inline-field">
                <input name="code" :placeholder="text.verificationCodePlaceholder" />
                <button
                  v-if="showsChallengeButton"
                  type="button"
                  class="auth-button auth-button-secondary auth-inline-action"
                  @click="sendVerificationChallenge"
                >
                  {{ text.sendVerificationCode }}
                </button>
              </div>
            </label>

            <button
              v-if="!isSecurityKeyMethod"
              type="submit"
              class="auth-button auth-button-primary auth-button-block"
            >
              {{ text.verifyAndContinue }}
            </button>

            <button
              v-if="isSecurityKeyMethod && supportsSessionU2F"
              type="button"
              class="auth-button auth-button-primary auth-button-block"
              @click="verifyMFAWithU2F"
            >
              {{ text.useSecurityKey }}
            </button>

            <div v-if="challengeFeedback" class="auth-alert auth-alert-muted">
              {{ challengeFeedback }}
            </div>
          </form>
        </section>

        <footer class="auth-footer">
          <label class="auth-locale-switcher">
            <span>{{ text.language }}</span>
            <select v-model="locale">
              <option v-for="item in localeOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </option>
            </select>
          </label>
          <div class="auth-footer-meta">
            <span>{{ text.securedBy }}</span>
            <a v-if="termsOfServiceUrl" :href="termsOfServiceUrl" target="_blank" rel="noreferrer">{{ text.tos }}</a>
            <a v-if="privacyPolicyUrl" :href="privacyPolicyUrl" target="_blank" rel="noreferrer">{{ text.privacyPolicy }}</a>
          </div>
        </footer>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { normalizeRequestOptions, serializeCredential } from '@shared/api/webauthn'
import ToastHost from '@shared/components/ToastHost.vue'
import { useToast } from '@shared/composables/toast'

type LocaleKey = 'en-US' | 'ja-JP' | 'zh-CN'
type AuthStage = 'login' | 'account' | 'confirmation' | 'mfa'

type TranslationShape = {
  productTagline: string
  language: string
  securityLabel: string
  stageTitles: Record<AuthStage, (appName: string) => string>
  stageHints: Record<AuthStage, string>
  accountTitle: string
  continueAsCurrentAccount: string
  logoutAndSwitchAccount: string
  identifier: string
  identifierPlaceholder: string
  password: string
  passwordPlaceholder: string
  signIn: string
  or: string
  continueWithProvider: (providerName: string) => string
  signInWithPasskey: string
  confirmationTitle: string
  confirmationItems: {
    trustedDevice: string
    futureSkip: string
    continueWithoutTrust: string
  }
  confirmTrustDevice: string
  confirmContinueWithoutTrust: string
  verificationMethod: string
  verificationCode: string
  verificationCodePlaceholder: string
  verifyAndContinue: string
  sendVerificationCode: string
  useSecurityKey: string
  securedBy: string
  tos: string
  privacyPolicy: string
  developedBy: (appName: string, organizationName: string) => string
  challengeSent: string
  challengeSentWithDemoCode: (code: string) => string
  passkeyRequiresIdentifier: string
  mfaMethodLabels: Record<string, string>
  errorFallback: string
  errorTranslations: Record<string, string>
}

const translations: Record<LocaleKey, TranslationShape> = {
  'en-US': {
    productTagline: 'Secure authorization workspace',
    language: 'Language',
    securityLabel: 'Authorization',
    stageTitles: {
      login: (appName: string) => `Sign in to ${appName}`,
      account: () => 'Choose an account',
      confirmation: () => 'Device trust',
      mfa: () => 'Multi-factor authentication'
    },
    stageHints: {
      login: 'Authenticate with your Pass Pivot account.',
      account: 'Review the current signed-in account before continuing.',
      confirmation: 'Choose whether to add this browser as a trusted device.',
      mfa: 'Choose an available verification method for this session.'
    },
    accountTitle: 'You are already signed in',
    continueAsCurrentAccount: 'Continue with this account',
    logoutAndSwitchAccount: 'Sign out and use another account',
    identifier: 'Account',
    identifierPlaceholder: 'Enter your account',
    password: 'Password',
    passwordPlaceholder: 'Enter your password',
    signIn: 'Sign in',
    or: 'or',
    continueWithProvider: (providerName: string) => `Continue with ${providerName}`,
    signInWithPasskey: 'Sign in with a passkey',
    confirmationTitle: 'Add this browser as a trusted device?',
    confirmationItems: {
      trustedDevice: 'Trust this browser for future sign-ins on this account.',
      futureSkip: 'Trusted devices can skip MFA in later sign-ins.',
      continueWithoutTrust: 'You can also continue this sign-in without trusting this browser.'
    },
    confirmTrustDevice: 'Add as trusted device and continue',
    confirmContinueWithoutTrust: 'Continue without trusting this device',
    verificationMethod: 'Verification method',
    verificationCode: 'Verification code',
    verificationCodePlaceholder: 'Enter your verification code',
    verifyAndContinue: 'Verify and continue',
    sendVerificationCode: 'Send code',
    useSecurityKey: 'Use security key',
    securedBy: 'Secured by PassPivot',
    tos: 'Terms of Service',
    privacyPolicy: 'Privacy Policy',
    developedBy: (appName: string, organizationName: string) => `${appName} by ${organizationName}`,
    challengeSent: 'A verification code has been sent.',
    challengeSentWithDemoCode: (code: string) => `A verification code has been generated. Demo code: ${code}`,
    passkeyRequiresIdentifier: 'Enter your account identifier before using a passkey.',
    mfaMethodLabels: {
      totp: 'Authenticator app (TOTP)',
      email_code: 'Email code',
      sms_code: 'SMS code',
      recovery_code: 'Recovery code',
      u2f: 'Security key',
      webauthn: 'Passkey'
    },
    errorFallback: 'The request could not be completed.',
    errorTranslations: {
      'authn.invalid_credentials': 'The account identifier or password is incorrect.',
      'authn.user_inactive': 'This account is not active.',
      'authn.organization_disabled': 'This organization is currently disabled.',
      'authn.application_disabled': 'This application is currently disabled.',
      'authn.application_access_denied': 'This account is not assigned to the target application.',
      'authn.session_state_invalid': 'This session is no longer waiting for multi-factor verification.',
      'authn.mfa_code_invalid': 'The verification code is invalid.',
      'authn.mfa_challenge_expired': 'The verification challenge has expired.',
      'authn.mfa_challenge_not_found': 'No active verification challenge was found.',
      'authn.webauthn_login_disabled': 'Passkey sign-in is not enabled for this account.',
      'authn.webauthn_challenge_not_found': 'The passkey challenge was not found.',
      'authn.webauthn_challenge_expired': 'The passkey challenge has expired.',
      'authn.external_idp_identity_unbound': 'This external identity is not bound to an existing account.',
      'invalid JSON body': 'The request body is invalid.',
      'invalid credentials': 'The account identifier or password is incorrect.',
      'invalid TOTP code': 'The verification code is invalid.',
      'invalid challenge code': 'The verification code is invalid.',
      'invalid recovery code': 'The recovery code is invalid.',
      'webauthn challenge not found': 'The passkey challenge was not found.',
      'webauthn challenge expired': 'The passkey challenge has expired.'
    }
  },
  'zh-CN': {
    productTagline: '安全授权工作台',
    language: '语言',
    securityLabel: '授权流程',
    stageTitles: {
      login: (appName: string) => `登录到 ${appName}`,
      account: () => '选择账号',
      confirmation: () => '设备信任',
      mfa: () => '多因素验证'
    },
    stageHints: {
      login: '使用你的 Pass Pivot 账号完成身份验证。',
      account: '当前已存在登录会话，请确认是否继续使用该账号。',
      confirmation: '选择是否将当前浏览器加入可信设备。',
      mfa: '为当前会话选择一种可用的验证方式。'
    },
    accountTitle: '当前已登录账号',
    continueAsCurrentAccount: '使用当前账号继续登录',
    logoutAndSwitchAccount: '退出当前账号并切换登录',
    identifier: '账号',
    identifierPlaceholder: '请输入账号',
    password: '密码',
    passwordPlaceholder: '请输入密码',
    signIn: '登录',
    or: '或',
    continueWithProvider: (providerName: string) => `使用 ${providerName} 继续`,
    signInWithPasskey: '使用通行密钥登录',
    confirmationTitle: '是否将当前浏览器加入可信设备？',
    confirmationItems: {
      trustedDevice: '加入可信后，后续在此浏览器登录可跳过 MFA。',
      futureSkip: '仅建议在你本人长期使用的浏览器上启用。',
      continueWithoutTrust: '你也可以不加入可信设备，直接继续本次登录。'
    },
    confirmTrustDevice: '加入可信设备并继续',
    confirmContinueWithoutTrust: '不加入可信设备，继续登录',
    verificationMethod: '验证方式',
    verificationCode: '验证码',
    verificationCodePlaceholder: '请输入验证码',
    verifyAndContinue: '验证并继续',
    sendVerificationCode: '发送验证码',
    useSecurityKey: '使用安全密钥',
    securedBy: '由 PassPivot 提供安全防护',
    tos: '服务条款',
    privacyPolicy: '隐私策略',
    developedBy: (appName: string, organizationName: string) => `${appName} 由 ${organizationName} 开发`,
    challengeSent: '验证码已发送。',
    challengeSentWithDemoCode: (code: string) => `验证码已生成，演示码：${code}`,
    passkeyRequiresIdentifier: '请先输入账号，再使用通行密钥登录。',
    mfaMethodLabels: {
      totp: '身份验证器（TOTP）',
      email_code: '邮箱验证码',
      sms_code: '手机验证码',
      recovery_code: '备用验证码',
      u2f: '安全密钥',
      webauthn: '通行密钥'
    },
    errorFallback: '请求暂时无法完成。',
    errorTranslations: {
      'authn.invalid_credentials': '账号或密码不正确。',
      'authn.user_inactive': '当前账号未处于可登录状态。',
      'authn.organization_disabled': '当前组织已被禁用。',
      'authn.application_disabled': '当前应用已被禁用。',
      'authn.application_access_denied': '当前账号未被分配到目标应用。',
      'authn.session_state_invalid': '当前会话已经不再等待两步验证。',
      'authn.mfa_code_invalid': '验证码不正确。',
      'authn.mfa_challenge_expired': '验证码挑战已过期。',
      'authn.mfa_challenge_not_found': '未找到有效的验证码挑战。',
      'authn.webauthn_login_disabled': '当前账号未启用通行密钥登录。',
      'authn.webauthn_challenge_not_found': '未找到通行密钥挑战。',
      'authn.webauthn_challenge_expired': '通行密钥挑战已过期。',
      'authn.external_idp_identity_unbound': '该外部身份尚未绑定到现有账号。',
      'invalid JSON body': '请求体格式不正确。',
      'invalid credentials': '账号或密码不正确。',
      'invalid TOTP code': '验证码不正确。',
      'invalid challenge code': '验证码不正确。',
      'invalid recovery code': '备用验证码不正确。',
      'webauthn challenge not found': '未找到通行密钥挑战。',
      'webauthn challenge expired': '通行密钥挑战已过期。'
    }
  },
  'ja-JP': {
    productTagline: 'セキュア認可ワークスペース',
    language: '言語',
    securityLabel: '認可フロー',
    stageTitles: {
      login: (appName: string) => `${appName} にサインイン`,
      account: () => 'アカウントを選択',
      confirmation: () => '端末の信頼',
      mfa: () => '多要素認証'
    },
    stageHints: {
      login: 'Pass Pivot アカウントで認証します。',
      account: '現在サインイン中のアカウントを確認してから続行してください。',
      confirmation: 'このブラウザを信頼済み端末に追加するか選択してください。',
      mfa: 'このセッションで利用できる認証方法を選択してください。'
    },
    accountTitle: '現在サインイン中のアカウント',
    continueAsCurrentAccount: 'このアカウントで続行',
    logoutAndSwitchAccount: 'サインアウトして別のアカウントを使用',
    identifier: 'アカウント',
    identifierPlaceholder: 'アカウントを入力',
    password: 'パスワード',
    passwordPlaceholder: 'パスワードを入力',
    signIn: 'サインイン',
    or: 'または',
    continueWithProvider: (providerName: string) => `${providerName} で続行`,
    signInWithPasskey: 'パスキーでサインイン',
    confirmationTitle: 'このブラウザを信頼済み端末に追加しますか？',
    confirmationItems: {
      trustedDevice: '信頼済み端末に追加すると、今後このブラウザでは MFA を省略できます。',
      futureSkip: '継続利用する自分のブラウザでのみ有効化してください。',
      continueWithoutTrust: '信頼済み端末に追加せず、そのままログインを続行することもできます。'
    },
    confirmTrustDevice: '信頼済み端末に追加して続行',
    confirmContinueWithoutTrust: '追加せずに続行',
    verificationMethod: '認証方法',
    verificationCode: '認証コード',
    verificationCodePlaceholder: '認証コードを入力',
    verifyAndContinue: '認証して続行',
    sendVerificationCode: '認証コードを送信',
    useSecurityKey: 'セキュリティキーを使用',
    securedBy: 'PassPivot により保護',
    tos: '利用規約',
    privacyPolicy: 'プライバシーポリシー',
    developedBy: (appName: string, organizationName: string) => `${appName} は ${organizationName} が提供`,
    challengeSent: '認証コードを送信しました。',
    challengeSentWithDemoCode: (code: string) => `認証コードが生成されました。デモコード: ${code}`,
    passkeyRequiresIdentifier: 'パスキーを使用する前にアカウント識別子を入力してください。',
    mfaMethodLabels: {
      totp: '認証アプリ (TOTP)',
      email_code: 'メールコード',
      sms_code: 'SMS コード',
      recovery_code: 'リカバリーコード',
      u2f: 'セキュリティキー',
      webauthn: 'パスキー'
    },
    errorFallback: 'リクエストを完了できませんでした。',
    errorTranslations: {
      'authn.invalid_credentials': 'アカウント識別子またはパスワードが正しくありません。',
      'authn.user_inactive': 'このアカウントは現在利用できません。',
      'authn.organization_disabled': 'この組織は現在無効です。',
      'authn.application_disabled': 'このアプリケーションは現在無効です。',
      'authn.application_access_denied': 'このアカウントには対象アプリケーションへのアクセス権がありません。',
      'authn.mfa_code_invalid': '認証コードが正しくありません。',
      'authn.webauthn_login_disabled': 'このアカウントではパスキーサインインが有効ではありません。',
      'invalid credentials': 'アカウント識別子またはパスワードが正しくありません。'
    }
  },
}

const localeOptions = [
  { value: 'en-US', label: 'English' },
  { value: 'ja-JP', label: '日本語' },
  { value: 'zh-CN', label: '简体中文' }
] as const

const toast = useToast()
const bootstrapPayload = window.__PPVT_OAUTH_BOOTSTRAP__

if (!bootstrapPayload) {
  throw new Error('missing oauth bootstrap payload')
}

const bootstrap = bootstrapPayload
const locale = ref<LocaleKey>(resolveInitialLocale())
const challengeFeedback = ref('')
const supportsWebAuthnLogin = Boolean(bootstrap.api.webauthnLoginBegin && bootstrap.api.webauthnLoginEnd)
const supportsSessionU2F = Boolean(bootstrap.api.sessionU2fBegin && bootstrap.api.sessionU2fFinish)
const supportsEmailChallenge = Boolean(bootstrap.api.mfaChallenge)
const termsOfServiceUrl = computed(() => String(bootstrap.target.termsOfServiceUrl || '').trim())
const privacyPolicyUrl = computed(() => String(bootstrap.target.privacyPolicyUrl || '').trim())
const organizationDisplayName = computed(() =>
  String(bootstrap.target.displayName || bootstrap.target.organizationName || '').trim()
)
const applicationName = computed(() => {
  const localized = bootstrap.target.applicationDisplayNames?.[toApplicationLocaleKey(locale.value)]
  const fallbackDisplayName = bootstrap.target.applicationDisplayNames?.default
  return localized || fallbackDisplayName || bootstrap.target.applicationName || 'Pass Pivot'
})
const currentAccountLabel = computed(() =>
  String(bootstrap.currentUser?.name || bootstrap.currentUser?.username || bootstrap.currentUser?.email || bootstrap.currentUser?.phoneNumber || '').trim()
)
const currentAccountSecondary = computed(() =>
  String(bootstrap.currentUser?.email || bootstrap.currentUser?.phoneNumber || bootstrap.currentUser?.username || '').trim()
)
const externalIDPs = computed(() => Array.isArray(bootstrap.target.externalIdps) ? bootstrap.target.externalIdps : [])

const text = computed(() => translations[locale.value])
const stageTitle = computed(() => text.value.stageTitles[bootstrap.stage](applicationName.value))
const stageHint = computed(() => text.value.stageHints[bootstrap.stage])
const localizedBootstrapError = computed(() => localizeError(bootstrap.error))
const localizedMFAOptions = computed(() =>
  bootstrap.mfaOptions.map((option) => ({
    value: option.value,
    label: text.value.mfaMethodLabels[option.value] || option.label || option.value
  }))
)
const selectedMethod = ref(bootstrap.secondFactorMethod || bootstrap.mfaOptions?.[0]?.value || '')
const isSecurityKeyMethod = computed(() => selectedMethod.value === 'u2f')
const showsChallengeButton = computed(() =>
  supportsEmailChallenge && (selectedMethod.value === 'email_code' || selectedMethod.value === 'sms_code')
)
const showsCodeInput = computed(() => !isSecurityKeyMethod.value)

watch(
  localizedMFAOptions,
  (options) => {
    if (!options.some((option) => option.value === selectedMethod.value)) {
      selectedMethod.value = options[0]?.value || ''
    }
  },
  { immediate: true }
)

watch(
  () => locale.value,
  (value) => {
    document.documentElement.lang = value
    document.title = `Pass Pivot · ${translations[value].stageTitles[bootstrap.stage](applicationName.value)}`
  },
  { immediate: true }
)

function resolveInitialLocale(): LocaleKey {
  if (typeof navigator === 'undefined') {
    return 'en-US'
  }
  const candidate = (navigator.language || 'en-US').toLowerCase()
  if (candidate.startsWith('ja')) return 'ja-JP'
  if (candidate.startsWith('zh')) return 'zh-CN'
  return 'en-US'
}

function toApplicationLocaleKey(locale: LocaleKey): 'en' | 'ja' | 'zhs' {
  if (locale === 'ja-JP') return 'ja'
  if (locale === 'zh-CN') return 'zhs'
  return 'en'
}

function localizeError(input?: string): string {
  if (!input) {
    return ''
  }
  const normalized = String(input).trim()
  if (!normalized) {
    return ''
  }
  return text.value.errorTranslations[normalized] || normalized || text.value.errorFallback
}

function resolveProviderGlyph(name?: string): string {
  const value = String(name || '').trim()
  if (!value) {
    return 'I'
  }
  return value.slice(0, 1).toUpperCase()
}

async function readJSON<T>(response: Response): Promise<T> {
  const responseText = await response.text()
  const contentType = response.headers.get('content-type') || ''
  let parsed: any = null

  if (responseText && contentType.includes('application/json')) {
    try {
      parsed = JSON.parse(responseText)
    } catch {
      parsed = null
    }
  }

  if (!response.ok) {
    const code = typeof parsed?.code === 'string' ? parsed.code : ''
    const message = typeof parsed?.message === 'string' ? parsed.message : responseText
    throw new Error(localizeError(code || message))
  }

  if (parsed !== null) {
    return parsed as T
  }
  return JSON.parse(responseText) as T
}

async function loginWithWebAuthn() {
  try {
    const begin = await readJSON<{ challengeId: string; options: any }>(
      await fetch(bootstrap.api.webauthnLoginBegin, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ applicationId: bootstrap.applicationId })
      })
    )
    const credential = await navigator.credentials.get({
      publicKey: normalizeRequestOptions(begin.options)
    })
    if (!credential) {
      return
    }
    await readJSON(
      await fetch(bootstrap.api.webauthnLoginEnd, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          challengeId: begin.challengeId,
          response: serializeCredential(credential as PublicKeyCredential),
          applicationId: bootstrap.applicationId
        })
      })
    )
    window.location.assign(bootstrap.authorizeReturnUrl)
  } catch (error) {
    toast.error(String(error))
  }
}

async function sendVerificationChallenge() {
  try {
    const result = await readJSON<{ demoCode?: string }>(
      await fetch(bootstrap.api.mfaChallenge, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ method: selectedMethod.value })
      })
    )
    challengeFeedback.value = result.demoCode
      ? text.value.challengeSentWithDemoCode(result.demoCode)
      : text.value.challengeSent
  } catch (error) {
    toast.error(String(error))
  }
}

async function verifyMFAWithU2F() {
  try {
    const begin = await readJSON<{ challengeId: string; options: any }>(
      await fetch(bootstrap.api.sessionU2fBegin, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({})
      })
    )
    const credential = await navigator.credentials.get({
      publicKey: normalizeRequestOptions(begin.options)
    })
    if (!credential) {
      return
    }
    await readJSON(
      await fetch(bootstrap.api.sessionU2fFinish, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          challengeId: begin.challengeId,
          response: serializeCredential(credential as PublicKeyCredential),
          trustDevice: true
        })
      })
    )
    window.location.assign(bootstrap.authorizeReturnUrl)
  } catch (error) {
    toast.error(String(error))
  }
}
</script>

<style scoped>
:global(body) {
  margin: 0;
  background: #f6f8fa;
  color: #1f2328;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
}

:global(*) {
  box-sizing: border-box;
}

.auth-shell {
  min-height: 100vh;
  background: #f6f8fa;
}

.auth-brand-mark {
  width: 3rem;
  height: 3rem;
  border-radius: 0.8rem;
  display: grid;
  place-items: center;
  background: #24292f;
  color: #fff;
  font-size: 1.15rem;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.auth-locale-switcher {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  color: #57606a;
  font-size: 0.85rem;
  justify-content: center;
}

.auth-locale-switcher select {
  border: 1px solid #d0d7de;
  border-radius: 6px;
  background: #fff;
  padding: 0.45rem 0.75rem;
  color: #1f2328;
}

.auth-main {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem 1rem 3rem;
}

.auth-stack {
  width: min(100%, 24rem);
  display: flex;
  flex-direction: column;
  gap: 1.15rem;
}

.auth-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.9rem;
  text-align: center;
}

.auth-title {
  margin: 0;
  font-size: 2rem;
  line-height: 1.15;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.auth-subnote {
  margin: -0.25rem 0 0;
  color: #656d76;
  font-size: 0.88rem;
  line-height: 1.5;
}

.auth-summary {
  margin: 0;
  color: #656d76;
  font-size: 0.95rem;
  line-height: 1.55;
}

.auth-checklist {
  margin: 0;
  padding-left: 1.15rem;
  color: #57606a;
  display: grid;
  gap: 0.7rem;
  line-height: 1.6;
}

.auth-card {
  background: #fff;
  border: 1px solid #d0d7de;
  border-radius: 12px;
  padding: 1rem;
}

.auth-form {
  display: grid;
  gap: 1rem;
}

.auth-field {
  display: grid;
  gap: 0.45rem;
}

.auth-field span {
  font-size: 0.87rem;
  font-weight: 600;
}

.auth-field input,
.auth-field select {
  width: 100%;
  min-height: 2.75rem;
  border-radius: 6px;
  border: 1px solid #d0d7de;
  background: #fff;
  padding: 0.72rem 0.85rem;
  font: inherit;
  color: #1f2328;
  transition: border-color 140ms ease, box-shadow 140ms ease;
}

.auth-field input:focus,
.auth-field select:focus,
.auth-locale-switcher select:focus {
  outline: none;
  border-color: #0969da;
  box-shadow: 0 0 0 3px rgba(9, 105, 218, 0.15);
}

.auth-checkbox {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  color: #57606a;
  font-size: 0.9rem;
}

.auth-checkbox input {
  width: 1rem;
  height: 1rem;
  accent-color: #1f883d;
}

.auth-button {
  min-height: 2.75rem;
  width: 100%;
  border-radius: 6px;
  border: 1px solid transparent;
  font: inherit;
  font-size: 0.92rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 140ms ease, border-color 140ms ease, transform 140ms ease;
}

.auth-button:hover {
  transform: translateY(-1px);
}

.auth-button-primary {
  background: #1f883d;
  border-color: rgba(31, 35, 40, 0.15);
  color: #fff;
}

.auth-button-primary:hover {
  background: #1a7f37;
}

.auth-button-danger {
  background: #fff;
  border-color: #cf222e;
  color: #cf222e;
}

.auth-button-danger:hover {
  background: #fff5f5;
}

.auth-alert {
  border-radius: 8px;
  padding: 0.85rem 0.95rem;
  font-size: 0.9rem;
  line-height: 1.55;
}

.auth-alert-danger {
  background: #ffebe9;
  border: 1px solid #ff818266;
  color: #cf222e;
}

.auth-alert-muted {
  background: #f6f8fa;
  border: 1px solid #d8dee4;
  color: #57606a;
}

.auth-link-button {
  border: 0;
  background: transparent;
  color: #0969da;
  cursor: pointer;
  font: inherit;
  font-size: 0.95rem;
  padding: 0;
  margin-top: 0.15rem;
}

.auth-link-button:hover {
  text-decoration: underline;
}

.auth-inline-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.75rem;
  align-items: stretch;
}

.auth-inline-action {
  width: auto;
  min-width: 8rem;
  padding: 0 1rem;
  white-space: nowrap;
}

.auth-idp-block {
  display: grid;
  gap: 1rem;
}

.auth-idp-divider {
  position: relative;
  text-align: center;
  color: #57606a;
  font-size: 0.9rem;
}

.auth-idp-divider::before {
  content: "";
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  border-top: 1px solid #d8dee4;
  transform: translateY(-50%);
}

.auth-idp-divider span {
  position: relative;
  display: inline-block;
  padding: 0 0.9rem;
  background: #fff;
}

.auth-idp-list {
  display: grid;
  gap: 0.8rem;
}

.auth-idp-button {
  min-height: 3rem;
  width: 100%;
  border-radius: 10px;
  border: 1px solid #d0d7de;
  background: #fff;
  color: #24292f;
  font: inherit;
  font-size: 0.98rem;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.8rem;
  cursor: default;
}

.auth-idp-badge {
  width: 1.7rem;
  height: 1.7rem;
  border-radius: 999px;
  display: inline-grid;
  place-items: center;
  background: #f6f8fa;
  border: 1px solid #d8dee4;
  color: #57606a;
  font-size: 0.82rem;
  font-weight: 700;
}

.auth-confirm-box {
  border-radius: 10px;
  border: 1px solid #d8dee4;
  background: #f6f8fa;
  padding: 1rem;
}

.auth-confirm-title {
  font-weight: 600;
  margin-bottom: 0.75rem;
}

.auth-account-summary {
  display: grid;
  gap: 0.25rem;
  color: #57606a;
}

.auth-account-summary strong {
  color: #1f2328;
  font-size: 0.98rem;
}

.auth-confirm-list {
  margin: 0;
  padding-left: 1.15rem;
  color: #57606a;
  display: grid;
  gap: 0.55rem;
}

.auth-context-card,
.auth-note-card {
  background: #fff;
  border: 1px solid #d0d7de;
  border-radius: 12px;
  padding: 1rem;
}

.auth-context-title,
.auth-note-title {
  font-size: 0.92rem;
  font-weight: 600;
}

.auth-context-subtitle {
  margin-top: 0.3rem;
  color: #656d76;
  font-size: 0.82rem;
}

.auth-context-grid {
  margin-top: 0.9rem;
  display: grid;
  gap: 0.7rem;
}

.auth-context-item {
  border-radius: 8px;
  background: #f6f8fa;
  border: 1px solid #d8dee4;
  padding: 0.75rem 0.8rem;
  display: flex;
  flex-direction: column;
  gap: 0.28rem;
}

.auth-context-item span {
  color: #656d76;
  font-size: 0.78rem;
}

.auth-context-item strong {
  font-size: 0.92rem;
  word-break: break-word;
}

.auth-footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.85rem;
  color: #656d76;
  font-size: 0.8rem;
}

.auth-footer-meta {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.75rem;
  line-height: 1.5;
}

.auth-footer-meta a {
  color: #0969da;
  text-decoration: none;
}

.auth-footer-meta a:hover {
  text-decoration: underline;
}

@media (max-width: 940px) {
  .auth-main {
    padding-top: 1.5rem;
  }
}

@media (max-width: 640px) {
  .auth-title {
    font-size: 1.7rem;
  }

  .auth-main {
    padding: 1.2rem 0.9rem 2rem;
  }

  .auth-card,
  .auth-context-card,
  .auth-note-card {
    border-radius: 10px;
  }

  .auth-card {
    padding: 1.1rem;
  }

  .auth-stack {
    width: 100%;
  }
}
</style>
