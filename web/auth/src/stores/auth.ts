import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { normalizeRequestOptions, serializeCredential } from '@shared/utils/webauthn'
import router from '@/router'
import {
  beginSessionU2F,
  beginWebAuthnLogin,
  completeDeviceAuthorization,
  confirmAuthorizeSession,
  createAuthorizeSession,
  finishSessionU2F,
  finishWebAuthnLogin,
  queryAuthContext,
  refreshCaptcha as requestRefreshCaptcha,
  sendMFAChallenge,
  verifyAuthorizeMFA
} from '@/api/auth'
import { buildLocaleText } from '@/i18n/auth'
import { localeOptions, resolveInitialLocale, type Locale } from '@/i18n/locale'
import { formatRequestError, localizeError } from '@/utils/auth-error'
import type { AuthCaptcha, AuthContextPayload, AuthMethodOption } from '@/vite-env'

function hasTurnstileAPI() {
  return typeof window !== 'undefined' &&
    typeof window.turnstile?.render === 'function' &&
    typeof window.turnstile?.reset === 'function'
}

function hasRecaptchaAPI() {
  return typeof window !== 'undefined' &&
    typeof window.grecaptcha?.render === 'function' &&
    typeof window.grecaptcha?.reset === 'function'
}

function buildAuthorizePayload() {
  const query = router.currentRoute.value.query
  return {
    sessionId: typeof query.ppvt_session_id === 'string' ? query.ppvt_session_id : '',
    flowType: typeof query.type === 'string' ? query.type : '',
    userCode: typeof query.user_code === 'string' ? query.user_code : '',
    clientId: typeof query.client_id === 'string' ? query.client_id : '',
    responseType: typeof query.response_type === 'string' ? query.response_type : '',
    redirectUri: typeof query.redirect_uri === 'string' ? query.redirect_uri : '',
    scope: typeof query.scope === 'string' ? query.scope : '',
    state: typeof query.state === 'string' ? query.state : '',
    nonce: typeof query.nonce === 'string' ? query.nonce : '',
    codeChallenge: typeof query.code_challenge === 'string' ? query.code_challenge : '',
    codeChallengeMethod: typeof query.code_challenge_method === 'string' ? query.code_challenge_method : '',
    prompt: typeof query.prompt === 'string' ? query.prompt : '',
    skipAccountSelection: false
  }
}

export const useAuthStore = defineStore('auth', () => {
  const context = ref<AuthContextPayload | null>(null)
  const initialized = ref(false)
  const turnstileWidgetId = ref<string | null>(null)
  const recaptchaWidgetId = ref<number | null>(null)
  const locale = ref<Locale>(resolveInitialLocale())
  const selectedMethod = ref('')
  const challengeFeedback = ref('')
  const captchaImageDataUrl = ref('')
  const captchaChallengeToken = ref('')
  const captchaToken = ref('')
  const message = ref('')
  const messageVariant = ref<'success' | 'danger'>('success')
  const messageSource = ref('')
  const messageTrigger = ref('')
  const messageError = ref<unknown>(undefined)
  const messageMetadata = ref<Record<string, unknown> | undefined>(undefined)
  let turnstileScriptPromise: Promise<void> | null = null
  let recaptchaScriptPromise: Promise<void> | null = null

  const text = computed(() => buildLocaleText(locale.value))
  const stage = computed<'login' | 'account' | 'confirmation' | 'mfa' | 'done'>(() => context.value?.stage || 'login')
  const flowType = computed<'authorize' | 'device_code'>(() => context.value?.flowType || 'authorize')
  const termsOfServiceUrl = computed(() => String(context.value?.target.termsOfServiceUrl || '').trim())
  const privacyPolicyUrl = computed(() => String(context.value?.target.privacyPolicyUrl || '').trim())
  const organizationDisplayName = computed(() => {
    const target = context.value?.target
    if (!target) return ''
    const localized = target.organizationDisplayNames?.[locale.value]
    const fallbackDisplayName = target.organizationDisplayNames?.default
    return localized || fallbackDisplayName || target.displayName || target.organizationName || ''
  })
  const applicationName = computed(() => {
    const target = context.value?.target
    if (!target) return 'Pass Pivot'
    const localized = target.applicationDisplayNames?.[locale.value]
    const fallbackDisplayName = target.applicationDisplayNames?.default
    return localized || fallbackDisplayName || target.applicationName || 'Pass Pivot'
  })
  const currentAccountLabel = computed(() =>
    String(
      context.value?.currentUser?.name ||
      context.value?.currentUser?.username ||
      context.value?.currentUser?.email ||
      context.value?.currentUser?.phoneNumber ||
      ''
    ).trim()
  )
  const currentAccountSecondary = computed(() =>
    String(
      context.value?.currentUser?.email ||
      context.value?.currentUser?.phoneNumber ||
      context.value?.currentUser?.username ||
      ''
    ).trim()
  )
  const externalIdps = computed(() => Array.isArray(context.value?.target.externalIdps) ? context.value?.target.externalIdps : [])
  const stageTitle = computed(() => {
    if (stage.value === 'done' && flowType.value === 'device_code') {
      return text.value.deviceAuthorizationCompleteTitle(applicationName.value)
    }
    return text.value.stageTitles[stage.value === 'done' ? 'login' : stage.value](applicationName.value)
  })
  const stageHint = computed(() => {
    if (stage.value === 'done' && flowType.value === 'device_code') {
      return text.value.deviceAuthorizationCompleteHint
    }
    return text.value.stageHints[stage.value === 'done' ? 'login' : stage.value]
  })
  const resultStatus = computed(() => context.value?.resultStatus || '')
  const resultMessage = computed(() => String(context.value?.resultMessage || '').trim())
  const continueButtonText = computed(() =>
    flowType.value === 'device_code' ? text.value.authorizeThisClient : text.value.continueAsCurrentAccount
  )
  const localizedContextError = computed(() => localizeError(context.value?.error, text.value))
  const localizedMFAOptions = computed(() =>
    (context.value?.mfaOptions || []).map((option: AuthMethodOption) => ({
      value: option.value,
      label: text.value.mfaMethodLabels[option.value] || option.label || option.value
    }))
  )
  const supportsWebAuthnLogin = computed(() => stage.value === 'login')
  const supportsSessionU2F = computed(() => true)
  const supportsEmailChallenge = computed(() => true)
  const isSecurityKeyMethod = computed(() => selectedMethod.value === 'u2f')
  const showsChallengeButton = computed(() =>
    supportsEmailChallenge.value && (selectedMethod.value === 'email_code' || selectedMethod.value === 'sms_code')
  )
  const showsCodeInput = computed(() => !isSecurityKeyMethod.value)

  function requireContext() {
    if (!context.value) {
      throw new Error('missing auth context')
    }
    return context.value
  }

  function initializeFromContext(nextContext: AuthContextPayload) {
    const mfaOptions = Array.isArray(nextContext.mfaOptions) ? nextContext.mfaOptions : []
    if (!selectedMethod.value || !mfaOptions.some((option) => option.value === selectedMethod.value)) {
      selectedMethod.value = nextContext.secondFactorMethod || mfaOptions[0]?.value || ''
    }
    syncCaptchaBootstrap(nextContext.captcha)
  }

  function setLocale(value: Locale) {
    locale.value = value
  }

  function setSelectedMethod(value: string) {
    selectedMethod.value = value
  }

  function setChallengeFeedback(value: string) {
    challengeFeedback.value = value
  }

  function setCaptchaToken(value: string) {
    captchaToken.value = value
  }

  function syncCaptchaBootstrap(captcha?: Pick<AuthCaptcha, 'imageDataUrl' | 'challengeToken'>) {
    captchaImageDataUrl.value = captcha?.imageDataUrl || ''
    captchaChallengeToken.value = captcha?.challengeToken || ''
    captchaToken.value = ''
  }

  function resetCaptchaState() {
    captchaToken.value = ''
  }

  function setMessage(
    value: string,
    variant: 'success' | 'danger',
    options: {
      source?: string
      trigger?: string
      error?: unknown
      metadata?: Record<string, unknown>
    } = {}
  ) {
    message.value = value
    messageVariant.value = variant
    messageSource.value = options.source || ''
    messageTrigger.value = options.trigger || ''
    messageError.value = options.error
    messageMetadata.value = options.metadata
  }

  function clearMessage() {
    message.value = ''
    messageSource.value = ''
    messageTrigger.value = ''
    messageError.value = undefined
    messageMetadata.value = undefined
  }

  function localize(input?: string) {
    return localizeError(input, text.value)
  }

  function formatError(error: unknown) {
    return formatRequestError(error, localize)
  }

  async function reloadContext() {
    const nextContext = await queryAuthContext(buildAuthorizePayload())
    context.value = nextContext
    initializeFromContext(nextContext)
    if (nextContext.action === 'redirect' && nextContext.redirectTarget) {
      window.location.assign(nextContext.redirectTarget)
    }
  }

  async function initialize() {
    if (initialized.value) {
      return
    }
    initialized.value = true
    await reloadContext()
  }

  async function continueAuthorization() {
    const current = requireContext()
    if (current.flowType === 'device_code') {
      const userCode = typeof router.currentRoute.value.query.user_code === 'string'
        ? router.currentRoute.value.query.user_code
        : ''
      await completeDeviceAuthorization({ userCode })
      await reloadContext()
      return
    }
    const nextContext = await queryAuthContext({
      ...buildAuthorizePayload(),
      skipAccountSelection: true
    })
    if (nextContext.action === 'redirect' && nextContext.redirectTarget) {
      window.location.assign(nextContext.redirectTarget)
      return
    }
    context.value = nextContext
    initializeFromContext(nextContext)
  }

  async function createSession(identifier: string, secret: string, captchaAnswer: string) {
    const current = requireContext()
    try {
      const result = await createAuthorizeSession({
        organizationId: current.target.organizationId,
        applicationId: current.applicationId,
        identifier,
        secret,
        captchaProvider: current.captcha?.provider || '',
        captchaToken: current.captcha?.provider === 'default' ? '' : captchaToken.value,
        captchaChallengeToken: captchaChallengeToken.value,
        captchaAnswer
      })
      if (result?.nextStep === 'done') {
        await continueAuthorization()
        return
      }
      await reloadContext()
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.createSession',
        trigger: 'createSession',
        error
      })
      await reloadContext()
    }
  }

  async function switchAccount() {
    try {
      setChallengeFeedback('')
      const url = new URL('/auth/end_session', window.location.origin)
      url.searchParams.set('post_logout_redirect_uri', window.location.href)
      window.location.assign(url.toString())
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.switchAccount',
        trigger: 'switchAccount',
        error
      })
    }
  }

  async function confirmSession(accept: boolean, trustDevice: boolean) {
    try {
      const result = await confirmAuthorizeSession({ accept, trustDevice })
      if (result?.nextStep === 'done') {
        await continueAuthorization()
        return
      }
      await reloadContext()
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.confirmSession',
        trigger: 'confirmSession',
        error
      })
      await reloadContext()
    }
  }

  async function verifyMFA(code: string) {
    try {
      const result = await verifyAuthorizeMFA({
        method: selectedMethod.value,
        code,
        trustDevice: false
      })
      if (result?.nextStep === 'done') {
        await continueAuthorization()
        return
      }
      await reloadContext()
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.verifyMFA',
        trigger: 'verifyMFA',
        error
      })
      await reloadContext()
    }
  }

  async function loginWithWebAuthn(identifier = '') {
    const current = requireContext()
    try {
      const begin = await beginWebAuthnLogin(current.applicationId)
      const credential = await navigator.credentials.get({
        publicKey: normalizeRequestOptions(begin.options)
      })
      if (!credential) {
        return
      }
      await finishWebAuthnLogin({
        challengeId: begin.challengeId,
        response: serializeCredential(credential as PublicKeyCredential),
        applicationId: current.applicationId
      })
      await continueAuthorization()
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.loginWithWebAuthn',
        trigger: 'loginWithWebAuthn',
        error,
        metadata: {
          identifier
        }
      })
      if (identifier) {
        void identifier
      }
      await reloadContext()
    }
  }

  async function sendVerificationChallenge() {
    try {
      const result = await sendMFAChallenge(selectedMethod.value)
      setChallengeFeedback(result.demoCode
        ? text.value.challengeSentWithDemoCode(result.demoCode)
        : text.value.challengeSent)
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.sendVerificationChallenge',
        trigger: 'sendVerificationChallenge',
        error
      })
    }
  }

  async function verifyMFAWithU2F() {
    try {
      const begin = await beginSessionU2F()
      const credential = await navigator.credentials.get({
        publicKey: normalizeRequestOptions(begin.options)
      })
      if (!credential) {
        return
      }
      await finishSessionU2F({
        challengeId: begin.challengeId,
        response: serializeCredential(credential as PublicKeyCredential),
        trustDevice: true
      })
      await continueAuthorization()
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.verifyMFAWithU2F',
        trigger: 'verifyMFAWithU2F',
        error
      })
      await reloadContext()
    }
  }

  async function refreshCaptcha() {
    const current = requireContext()
    if (current.captcha?.provider !== 'default') {
      return
    }
    try {
      const nextCaptcha = await requestRefreshCaptcha()
      syncCaptchaBootstrap(nextCaptcha)
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.refreshCaptcha',
        trigger: 'refreshCaptcha',
        error
      })
    }
  }

  function loadTurnstileScript() {
    if (hasTurnstileAPI()) return Promise.resolve()
    if (turnstileScriptPromise) return turnstileScriptPromise
    turnstileScriptPromise = new Promise((resolve, reject) => {
      const existingScript = document.querySelector<HTMLScriptElement>('script[data-ppvt-turnstile]')
      if (existingScript) {
        existingScript.addEventListener('load', () => resolve(), { once: true })
        existingScript.addEventListener('error', () => reject(new Error('failed to load cloudflare captcha')), { once: true })
        return
      }
      const script = document.createElement('script')
      script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
      script.async = true
      script.defer = true
      script.dataset.ppvtTurnstile = 'true'
      script.onload = () => resolve()
      script.onerror = () => reject(new Error('failed to load cloudflare captcha'))
      document.head.appendChild(script)
    })
    return turnstileScriptPromise
  }

  function loadRecaptchaScript() {
    if (hasRecaptchaAPI()) return Promise.resolve()
    if (recaptchaScriptPromise) return recaptchaScriptPromise
    recaptchaScriptPromise = new Promise((resolve, reject) => {
      window.__ppvtRecaptchaOnload = () => resolve()
      const existingScript = document.querySelector<HTMLScriptElement>('script[data-ppvt-recaptcha]')
      if (existingScript) {
        if (hasRecaptchaAPI()) {
          resolve()
          return
        }
        existingScript.addEventListener('load', () => {
          if (hasRecaptchaAPI()) resolve()
        }, { once: true })
        existingScript.addEventListener('error', () => reject(new Error('failed to load google captcha')), { once: true })
        return
      }
      const script = document.createElement('script')
      script.src = 'https://www.google.com/recaptcha/api.js?onload=__ppvtRecaptchaOnload&render=explicit'
      script.async = true
      script.defer = true
      script.dataset.ppvtRecaptcha = 'true'
      script.onerror = () => reject(new Error('failed to load google captcha'))
      document.head.appendChild(script)
    })
    return recaptchaScriptPromise
  }

  async function renderCloudflareCaptcha(container: HTMLElement) {
    const current = requireContext()
    if (current.stage !== 'login' || current.captcha?.provider !== 'cloudflare') return
    if (!current.captcha.client_key) {
      setMessage('missing cloudflare captcha client key', 'danger', {
        source: 'auth/store.renderCloudflareCaptcha',
        trigger: 'renderCloudflareCaptcha'
      })
      return
    }
    try {
      await loadTurnstileScript()
      if (!hasTurnstileAPI() || !window.turnstile) throw new Error('cloudflare captcha is unavailable')
      resetCaptchaState()
      if (turnstileWidgetId.value) {
        window.turnstile.reset(turnstileWidgetId.value)
        return
      }
      container.innerHTML = ''
      turnstileWidgetId.value = window.turnstile.render(container, {
        sitekey: current.captcha.client_key,
        theme: 'light',
        callback: (token: string) => setCaptchaToken(String(token || '').trim()),
        'expired-callback': () => setCaptchaToken(''),
        'error-callback': () => {
          setCaptchaToken('')
          setMessage('cloudflare captcha failed to load', 'danger', {
            source: 'auth/store.renderCloudflareCaptcha',
            trigger: 'turnstile.error-callback'
          })
        }
      })
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.renderCloudflareCaptcha',
        trigger: 'renderCloudflareCaptcha',
        error
      })
    }
  }

  async function renderGoogleCaptcha(container: HTMLElement) {
    const current = requireContext()
    if (current.stage !== 'login' || current.captcha?.provider !== 'google') return
    if (!current.captcha.client_key) {
      setMessage('missing google captcha client key', 'danger', {
        source: 'auth/store.renderGoogleCaptcha',
        trigger: 'renderGoogleCaptcha'
      })
      return
    }
    try {
      await loadRecaptchaScript()
      if (!hasRecaptchaAPI() || !window.grecaptcha) throw new Error('google captcha is unavailable')
      resetCaptchaState()
      if (recaptchaWidgetId.value !== null) {
        window.grecaptcha.reset(recaptchaWidgetId.value)
        return
      }
      container.innerHTML = ''
      recaptchaWidgetId.value = window.grecaptcha.render(container, {
        sitekey: current.captcha.client_key,
        callback: (token: string) => setCaptchaToken(String(token || '').trim()),
        'expired-callback': () => setCaptchaToken(''),
        'error-callback': () => {
          setCaptchaToken('')
          setMessage('google captcha failed to load', 'danger', {
            source: 'auth/store.renderGoogleCaptcha',
            trigger: 'grecaptcha.error-callback'
          })
        }
      })
    } catch (error) {
      setMessage(formatError(error), 'danger', {
        source: 'auth/store.renderGoogleCaptcha',
        trigger: 'renderGoogleCaptcha',
        error
      })
    }
  }

  async function cancelLogin() {
    await switchAccount()
  }

  return {
    context,
    initialized,
    turnstileWidgetId,
    recaptchaWidgetId,
    locale,
    localeOptions,
    selectedMethod,
    challengeFeedback,
    captchaImageDataUrl,
    captchaChallengeToken,
    captchaToken,
    message,
    messageVariant,
    messageSource,
    messageTrigger,
    messageError,
    messageMetadata,
    text,
    stage,
    flowType,
    stageTitle,
    stageHint,
    resultStatus,
    resultMessage,
    continueButtonText,
    termsOfServiceUrl,
    privacyPolicyUrl,
    organizationDisplayName,
    applicationName,
    currentAccountLabel,
    currentAccountSecondary,
    externalIdps,
    localizedContextError,
    localizedMFAOptions,
    supportsWebAuthnLogin,
    supportsSessionU2F,
    supportsEmailChallenge,
    isSecurityKeyMethod,
    showsChallengeButton,
    showsCodeInput,
    initializeFromContext,
    initialize,
    reloadContext,
    continueAuthorization,
    createSession,
    switchAccount,
    confirmSession,
    verifyMFA,
    loginWithWebAuthn,
    sendVerificationChallenge,
    verifyMFAWithU2F,
    refreshCaptcha,
    renderCloudflareCaptcha,
    renderGoogleCaptcha,
    cancelLogin,
    setLocale,
    setSelectedMethod,
    setChallengeFeedback,
    setCaptchaToken,
    syncCaptchaBootstrap,
    resetCaptchaState,
    setMessage,
    clearMessage,
    localizeError: localize,
    formatRequestError: formatError
  }
})
