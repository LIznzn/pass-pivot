<template>
  <div class="auth-shell">
    <main class="auth-main">
      <div class="auth-stack">
        <Header
          :title="text.passwordRecoveryTitle"
          :subtitle="pageSubtitle"
          :application-name="applicationName"
          :organization-display-name="organizationDisplayName"
          :text="text"
        />

        <section class="auth-card">
          <template v-if="target">
            <form v-if="step === 'account'" class="auth-form" @submit.prevent="goToVerifyStep">
              <label class="auth-field">
                <span>{{ text.identifier }}</span>
                <input
                  v-model="identifier"
                  name="identifier"
                  autocomplete="username"
                  :placeholder="text.identifierPlaceholder"
                />
              </label>

              <DefaultCaptcha
                v-if="captcha?.provider === 'default'"
                :text="text"
                :captcha-image-data-url="captcha.imageDataUrl || ''"
                :captcha-challenge-token="captcha.challengeToken || ''"
                :answer="captchaAnswer"
                @refresh="refreshBootstrap"
                @update:answer="captchaAnswer = $event"
              />

              <CloudflareCaptcha
                v-else-if="captcha?.provider === 'cloudflare'"
                :text="text"
                :client-key="captcha.client_key || ''"
                :reset-key="captchaResetKey"
                @update:token="captchaToken = $event"
                @error="handleCaptchaError"
              />

              <GoogleCaptcha
                v-else-if="captcha?.provider === 'google'"
                :text="text"
                :client-key="captcha.client_key || ''"
                :reset-key="captchaResetKey"
                @update:token="captchaToken = $event"
                @error="handleCaptchaError"
              />

              <button type="submit" class="auth-button auth-button-primary auth-button-block">
                {{ text.passwordRecoveryNextStep }}
              </button>
            </form>

            <form v-else-if="step === 'verify'" class="auth-form" @submit.prevent="goToPasswordStep">
              <label class="auth-field">
                <span>{{ text.verificationMethod }}</span>
                <select v-model="selectedMethod" name="method">
                  <option v-for="option in localizedResetMethods" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </option>
                </select>
              </label>

              <label class="auth-field">
                <span>{{ contactLabel }}</span>
                <input
                  v-model="contact"
                  name="contact"
                  autocomplete="email"
                  :placeholder="contactPlaceholder"
                />
              </label>

              <label class="auth-field">
                <span>{{ text.passwordRecoveryCode }}</span>
                <div class="auth-inline-field auth-recovery-code-field">
                  <input
                    v-model="recoveryCode"
                    name="recovery_code"
                    autocomplete="one-time-code"
                    inputmode="numeric"
                    maxlength="6"
                    class="auth-recovery-code-input"
                    :placeholder="text.passwordRecoveryCodePlaceholder"
                  />
                  <button type="button" class="auth-button auth-button-secondary auth-inline-action auth-recovery-code-action" :disabled="sendCodeCooldown > 0" @click="sendResetCode">
                    {{ sendCodeButtonText }}
                  </button>
                </div>
              </label>

              <button type="submit" class="auth-button auth-button-primary auth-button-block">
                {{ text.passwordRecoveryNextStep }}
              </button>

              <button type="button" class="auth-button auth-button-secondary auth-button-block" @click="goBackToAccountStep">
                {{ text.passwordRecoveryPrevStep }}
              </button>
            </form>

            <form v-else class="auth-form" @submit.prevent="submitReset">
              <label class="auth-field">
                <span>{{ text.newPassword }}</span>
                <input
                  v-model="newPassword"
                  name="new_password"
                  type="password"
                  autocomplete="new-password"
                  :placeholder="text.newPasswordPlaceholder"
                />
              </label>

              <button type="submit" class="auth-button auth-button-primary auth-button-block">
                {{ text.passwordRecoveryTitle }}
              </button>
            </form>
          </template>

          <div v-else class="auth-inline-note">
            {{ bootstrapError || text.errorFallback }}
          </div>
        </section>

        <Footer
          :locale="locale"
          :locale-options="localeOptions"
          :text="text"
          :terms-of-service-url="termsOfServiceUrl"
          :privacy-policy-url="privacyPolicyUrl"
          @update:locale="setLocale"
        />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'bootstrap-vue-next'
import CloudflareCaptcha from '@/components/captcha/CloudflareCaptcha.vue'
import DefaultCaptcha from '@/components/captcha/DefaultCaptcha.vue'
import GoogleCaptcha from '@/components/captcha/GoogleCaptcha.vue'
import { bootstrapPasswordReset, finishPasswordReset, queryPasswordResetOptions, refreshCaptcha as requestRefreshCaptcha, startPasswordReset } from '@/api/auth'
import { buildLocaleText } from '@/i18n/auth'
import { localeOptions, resolveInitialLocale, type Locale } from '@/i18n/locale'
import Footer from '@/layout/Footer.vue'
import Header from '@/layout/Header.vue'
import { formatRequestError, localizeError } from '@/utils/auth-error'
import { RequestError } from '@/utils/request'
import type { AuthCaptcha, AuthTarget } from '@/vite-env'
import { notifyToast } from '@shared/utils/notify'

type ForgotPasswordBootstrapResponse = {
  clientId?: string
  target?: AuthTarget
  captcha?: AuthCaptcha
}

type PasswordResetMethodOption = {
  method: string
  maskedTarget: string
}

type PasswordResetOptionsResponse = {
  methods: PasswordResetMethodOption[]
}

type RecoveryStep = 'account' | 'verify' | 'password'

const route = useRoute()
const toast = useToast()
const locale = ref<Locale>(resolveInitialLocale())
const text = computed(() => buildLocaleText(locale.value))
const target = ref<AuthTarget | null>(null)
const captcha = ref<AuthCaptcha | null>(null)
const bootstrapError = ref('')
const captchaResetKey = ref(0)
const step = ref<RecoveryStep>('account')
const identifier = ref('')
const contact = ref('')
const resetMethods = ref<PasswordResetMethodOption[]>([])
const selectedMethod = ref('')
const recoveryCode = ref('')
const newPassword = ref('')
const captchaAnswer = ref('')
const captchaToken = ref('')
const recoveryCodeSent = ref(false)
const sendCodeCooldown = ref(0)
let sendCodeCooldownTimer: number | null = null

const clientId = computed(() => {
  const explicit = typeof route.query.clientId === 'string' ? route.query.clientId : ''
  if (explicit.trim()) {
    return explicit.trim()
  }
  const legacy = typeof route.query.client_id === 'string' ? route.query.client_id : ''
  return legacy.trim()
})

const applicationName = computed(() => {
  const value = target.value
  if (!value) {
    return 'Pass Pivot'
  }
  return value.applicationDisplayNames?.[locale.value] ||
    value.applicationDisplayNames?.default ||
    value.applicationName ||
    'Pass Pivot'
})

const organizationDisplayName = computed(() => {
  const value = target.value
  if (!value) {
    return ''
  }
  return value.organizationDisplayNames?.[locale.value] ||
    value.organizationDisplayNames?.default ||
    value.displayName ||
    value.organizationName ||
    ''
})

const pageSubtitle = computed(() => {
  if (!organizationDisplayName.value) {
    return ''
  }
  return text.value.passwordRecoveryTitleWithOrganization(organizationDisplayName.value)
})

const localizedResetMethods = computed(() =>
  resetMethods.value.map((option) => ({
    value: option.method,
    label: text.value.mfaMethodLabels[option.method] || option.method
  }))
)

const selectedMaskedTarget = computed(() =>
  resetMethods.value.find((option) => option.method === selectedMethod.value)?.maskedTarget || ''
)

const contactLabel = computed(() => {
  if (selectedMethod.value === 'sms_code') {
    return selectedMaskedTarget.value
      ? text.value.passwordRecoverySmsPrompt(selectedMaskedTarget.value)
      : text.value.passwordRecoverySmsPlaceholder
  }
  return selectedMaskedTarget.value
    ? text.value.passwordRecoveryEmailPrompt(selectedMaskedTarget.value)
    : text.value.passwordRecoveryEmailPlaceholder
})

const contactPlaceholder = computed(() =>
  selectedMethod.value === 'sms_code'
    ? text.value.passwordRecoverySmsPlaceholder
    : text.value.passwordRecoveryEmailPlaceholder
)

const sendCodeButtonText = computed(() =>
  sendCodeCooldown.value > 0
    ? `${sendCodeCooldown.value}s`
    : text.value.passwordRecoverySendCode
)

const termsOfServiceUrl = computed(() => String(target.value?.termsOfServiceUrl || '').trim())
const privacyPolicyUrl = computed(() => String(target.value?.privacyPolicyUrl || '').trim())

function localize(input?: string) {
  return localizeError(input, text.value)
}

function formatError(error: unknown) {
  return formatRequestError(error, localize)
}

function isPasswordResetContactMismatch(error: unknown) {
  if (error instanceof RequestError) {
    return error.code === 'authn.password_reset_contact_invalid' || error.message === 'password reset contact does not match'
  }
  if (error instanceof Error) {
    return error.message === 'authn.password_reset_contact_invalid' || error.message === 'password reset contact does not match'
  }
  return false
}

function showToast(
  message: string,
  variant: 'success' | 'danger',
  options: {
    source: string
    trigger?: string
    error?: unknown
    metadata?: Record<string, unknown>
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

function resetCaptchaState() {
  captchaAnswer.value = ''
  captchaToken.value = ''
  captchaResetKey.value += 1
}

async function refreshPasswordResetCaptcha() {
  if (captcha.value?.provider === 'default') {
    try {
      const nextCaptcha = await requestRefreshCaptcha()
      captcha.value = {
        ...captcha.value,
        imageDataUrl: nextCaptcha.imageDataUrl || '',
        challengeToken: nextCaptcha.challengeToken || ''
      }
    } catch {
      // Keep the current step stable even if captcha refresh fails.
    }
  }
  resetCaptchaState()
}

function setLocale(value: Locale) {
  locale.value = value
}

function resetRecoveryState() {
  step.value = 'account'
  contact.value = ''
  resetMethods.value = []
  selectedMethod.value = ''
  recoveryCode.value = ''
  newPassword.value = ''
  recoveryCodeSent.value = false
  stopSendCodeCooldown()
  resetCaptchaState()
}

function stopSendCodeCooldown() {
  if (sendCodeCooldownTimer !== null) {
    window.clearInterval(sendCodeCooldownTimer)
    sendCodeCooldownTimer = null
  }
  sendCodeCooldown.value = 0
}

function startSendCodeCooldown() {
  stopSendCodeCooldown()
  sendCodeCooldown.value = 60
  sendCodeCooldownTimer = window.setInterval(() => {
    if (sendCodeCooldown.value <= 1) {
      stopSendCodeCooldown()
      return
    }
    sendCodeCooldown.value -= 1
  }, 1000)
}

async function goToVerifyStep() {
  if (!identifier.value.trim()) {
    showToast(text.value.identifierPlaceholder, 'danger', {
      source: 'auth/ForgotPasswordPage.goToVerifyStep',
      trigger: 'goToVerifyStep'
    })
    return
  }
  try {
    const response = await queryPasswordResetOptions({
      clientId: clientId.value,
      identifier: identifier.value
    }) as PasswordResetOptionsResponse
    resetMethods.value = Array.isArray(response.methods) ? response.methods : []
    if (!resetMethods.value.length) {
      showToast(localize('authn.password_reset_target_unavailable'), 'danger', {
        source: 'auth/ForgotPasswordPage.goToVerifyStep',
        trigger: 'queryPasswordResetOptions.empty'
      })
      return
    }
    selectedMethod.value = resetMethods.value[0]?.method || ''
    contact.value = ''
    recoveryCode.value = ''
    recoveryCodeSent.value = false
    step.value = 'verify'
  } catch (error) {
    showToast(formatError(error), 'danger', {
      source: 'auth/ForgotPasswordPage.goToVerifyStep',
      trigger: 'queryPasswordResetOptions',
      error
    })
  }
}

async function refreshBootstrap() {
  try {
    const response = await bootstrapPasswordReset({ clientId: clientId.value }) as ForgotPasswordBootstrapResponse
    target.value = response.target || null
    captcha.value = response.captcha || null
    bootstrapError.value = ''
    resetRecoveryState()
  } catch (error) {
    target.value = null
    captcha.value = null
    bootstrapError.value = formatError(error)
    showToast(bootstrapError.value, 'danger', {
      source: 'auth/ForgotPasswordPage.bootstrap',
      trigger: 'bootstrapPasswordReset',
      error
    })
  }
}

async function sendResetCode() {
  if (sendCodeCooldown.value > 0) {
    return
  }
  if (!selectedMethod.value.trim()) {
    showToast(localize('authn.password_reset_method_required'), 'danger', {
      source: 'auth/ForgotPasswordPage.sendResetCode',
      trigger: 'validate.method'
    })
    return
  }
  if (!contact.value.trim()) {
    showToast(text.value.passwordRecoveryContactPlaceholder, 'danger', {
      source: 'auth/ForgotPasswordPage.sendResetCode',
      trigger: 'validate.contact'
    })
    return
  }
  try {
    await startPasswordReset({
      clientId: clientId.value,
      identifier: identifier.value,
      method: selectedMethod.value,
      contact: contact.value,
      captchaProvider: captcha.value?.provider || '',
      captchaToken: captcha.value?.provider === 'default' ? '' : captchaToken.value,
      captchaChallengeToken: captcha.value?.challengeToken || '',
      captchaAnswer: captchaAnswer.value
    })
    showToast(text.value.passwordRecoverySent, 'success', {
      source: 'auth/ForgotPasswordPage.sendResetCode',
      trigger: 'startPasswordReset'
    })
    recoveryCodeSent.value = true
    startSendCodeCooldown()
  } catch (error) {
    if (isPasswordResetContactMismatch(error)) {
      showToast(
        selectedMethod.value === 'sms_code'
          ? text.value.passwordRecoverySmsSentIfMatched
          : text.value.passwordRecoveryEmailSentIfMatched,
        'success',
        {
          source: 'auth/ForgotPasswordPage.sendResetCode',
          trigger: 'startPasswordReset.contactMismatch',
          error
        }
      )
      startSendCodeCooldown()
      return
    }
    showToast(formatError(error), 'danger', {
      source: 'auth/ForgotPasswordPage.sendResetCode',
      trigger: 'startPasswordReset',
      error
    })
    await refreshPasswordResetCaptcha()
  }
}

function goToPasswordStep() {
  if (!selectedMethod.value.trim()) {
    showToast(localize('authn.password_reset_method_required'), 'danger', {
      source: 'auth/ForgotPasswordPage.goToPasswordStep',
      trigger: 'validate.method'
    })
    return
  }
  if (!contact.value.trim()) {
    showToast(text.value.passwordRecoveryContactPlaceholder, 'danger', {
      source: 'auth/ForgotPasswordPage.goToPasswordStep',
      trigger: 'validate.contact'
    })
    return
  }
  if (!recoveryCodeSent.value) {
    showToast(text.value.passwordRecoveryStepVerifyHint, 'danger', {
      source: 'auth/ForgotPasswordPage.goToPasswordStep',
      trigger: 'validate.recoveryCodeSent'
    })
    return
  }
  if (!recoveryCode.value.trim()) {
    showToast(text.value.passwordRecoveryCodePlaceholder, 'danger', {
      source: 'auth/ForgotPasswordPage.goToPasswordStep',
      trigger: 'goToPasswordStep'
    })
    return
  }
  step.value = 'password'
}

async function submitReset() {
  try {
    await finishPasswordReset({
      clientId: clientId.value,
      identifier: identifier.value,
      code: recoveryCode.value,
      newPassword: newPassword.value
    })
    showToast(text.value.passwordRecoveryUpdated, 'success', {
      source: 'auth/ForgotPasswordPage.submitReset',
      trigger: 'finishPasswordReset'
    })
    recoveryCode.value = ''
    newPassword.value = ''
    step.value = 'verify'
  } catch (error) {
    showToast(formatError(error), 'danger', {
      source: 'auth/ForgotPasswordPage.submitReset',
      trigger: 'finishPasswordReset',
      error
    })
  }
}

function handleCaptchaError(error: Error) {
  showToast(formatError(error), 'danger', {
    source: 'auth/ForgotPasswordPage.captcha',
    trigger: 'captcha.render',
    error
  })
}

function cancelRecovery() {
  window.close()
}

function goBackToAccountStep() {
  step.value = 'account'
  contact.value = ''
  recoveryCode.value = ''
  recoveryCodeSent.value = false
  stopSendCodeCooldown()
}

watch(
  () => locale.value,
  (value) => {
    document.documentElement.lang = value
    document.title = text.value.passwordRecoveryTitle
  },
  { immediate: true }
)

watch(
  () => clientId.value,
  () => {
    void refreshBootstrap()
  },
  { immediate: true }
)
</script>

<style scoped>
.auth-recovery-code-field {
  display: grid;
  gap: 0.75rem;
}

.auth-recovery-code-input {
  width: 100%;
  min-width: 0;
}

.auth-recovery-code-action {
  min-width: 8.5rem;
  justify-self: end;
  white-space: nowrap;
}

.auth-button-primary {
  background: #1f883d;
  border-color: rgba(31, 35, 40, 0.15);
  color: #fff;
}

.auth-button-primary:hover {
  background: #1a7f37;
}

@media (min-width: 640px) {
  .auth-recovery-code-field {
    grid-template-columns: minmax(0, 1fr) max-content;
    align-items: center;
  }
}
</style>
