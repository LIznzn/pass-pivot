<template>
  <div class="auth-shell">
    <main class="auth-main">
      <div class="auth-stack">
        <Header
          :title="auth.stageTitle"
          :application-name="auth.applicationName"
          :organization-display-name="auth.organizationDisplayName"
          :text="auth.text"
        />

        <section class="auth-card">
          <DeviceCodeStep v-if="auth.stage === 'user_code'" />
          <DeviceAuthorizationReviewStep v-else-if="auth.stage === 'device_review'" />
          <LoginStep v-else-if="auth.stage === 'login'" />
          <AccountStep v-else-if="auth.stage === 'account'" />
          <ConfirmationStep v-else-if="auth.stage === 'confirmation'" />
          <MfaStep v-else-if="auth.stage === 'mfa'" />
          <DoneStep v-else />
        </section>

        <Footer
          :locale="auth.locale"
          :locale-options="auth.localeOptions"
          :text="auth.text"
          :terms-of-service-url="auth.termsOfServiceUrl"
          :privacy-policy-url="auth.privacyPolicyUrl"
          @update:locale="auth.setLocale"
        />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useToast } from 'bootstrap-vue-next'
import AccountStep from '@/components/AccountStep.vue'
import ConfirmationStep from '@/components/ConfirmationStep.vue'
import DeviceAuthorizationReviewStep from '@/components/DeviceAuthorizationReviewStep.vue'
import DeviceCodeStep from '@/components/DeviceCodeStep.vue'
import DoneStep from '@/components/DoneStep.vue'
import LoginStep from '@/components/LoginStep.vue'
import MfaStep from '@/components/MfaStep.vue'
import Footer from '@/layout/Footer.vue'
import Header from '@/layout/Header.vue'
import { useAuthStore } from '@/stores/auth'
import { notifyToast } from '@shared/utils/notify'

const toast = useToast()
const auth = useAuthStore()

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

auth.initialize()
  .catch((error) => {
    auth.setMessage(auth.formatRequestError(error), 'danger', {
      source: 'auth/MainPage.initialize',
      trigger: 'auth.initialize',
      error
    })
  })

watch(
  () => auth.locale,
  (value) => {
    document.documentElement.lang = value
    document.title = `Pass Pivot · ${auth.stageTitle}`
  },
  { immediate: true }
)

watch(
  () => auth.stageTitle,
  (value) => {
    document.title = `Pass Pivot · ${value}`
  },
  { immediate: true }
)

watch(
  () => auth.localizedContextError,
  (value, previousValue) => {
    if (!value || value === previousValue) {
      return
    }
    showToast(value, 'danger', {
      source: 'auth/MainPage.contextError',
      trigger: 'watch(auth.localizedContextError)',
      metadata: {
        stage: auth.stage
      }
    })
  },
  { immediate: true }
)

watch(
  () => auth.message,
  (value) => {
    if (!value) {
      return
    }
    if (auth.messageVariant === 'danger') {
      showToast(value, 'danger', {
        source: auth.messageSource || 'auth/MainPage.message',
        trigger: auth.messageTrigger || 'watch(auth.message)',
        error: auth.messageError,
        metadata: auth.messageMetadata
      })
    } else {
      showToast(value, 'success', {
        source: auth.messageSource || 'auth/MainPage.message',
        trigger: auth.messageTrigger || 'watch(auth.message)',
        error: auth.messageError,
        metadata: auth.messageMetadata
      })
    }
    auth.clearMessage()
  }
)
</script>

<style>
body {
  margin: 0;
  --auth-control-height: calc(2.5rem + 2px);
  --auth-control-radius: 6px;
  background: #f6f8fa;
  color: #1f2328;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
}

* {
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

.auth-footer {
  display: grid;
  justify-items: center;
  gap: 0.85rem;
  padding: 0.25rem 0 0.5rem;
}

.auth-footer-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 0.75rem 1rem;
  color: #656d76;
  font-size: 0.82rem;
  text-align: center;
}

.auth-footer-meta a {
  color: #0969da;
  text-decoration: none;
}

.auth-footer-meta a:hover {
  color: #0550ae;
  text-decoration: underline;
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

.auth-card {
  background: #fff;
  border: 1px solid #d0d7de;
  border-radius: 12px;
  padding: 1rem;
}

.auth-result {
  display: grid;
  gap: 0.5rem;
}

.auth-result-title,
.auth-result-text {
  margin: 0;
}

.auth-result-title {
  font-size: 1rem;
  font-weight: 600;
  color: #1f2328;
}

.auth-result-text {
  color: #656d76;
  line-height: 1.6;
}

.auth-form {
  display: grid;
  gap: 1rem;
}

.auth-step-hint {
  margin: 0;
  color: #656d76;
  font-size: 0.9rem;
  line-height: 1.5;
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
  height: var(--auth-control-height);
  min-height: var(--auth-control-height);
  border-radius: var(--auth-control-radius);
  border: 1px solid #d0d7de;
  background: #fff;
  padding: 0.58rem 0.85rem;
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

.auth-button {
  height: var(--auth-control-height);
  min-height: var(--auth-control-height);
  width: 100%;
  border-radius: var(--auth-control-radius);
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

.auth-button-secondary {
  background: #f6f8fa;
  border-color: #d0d7de;
  color: #1f2328;
}

.auth-button-secondary:hover {
  background: #eef2f6;
}

.auth-button-block {
  display: block;
}

.auth-link-button {
  border: none;
  background: transparent;
  color: #0969da;
  cursor: pointer;
  font: inherit;
  font-size: 0.92rem;
  font-weight: 600;
  padding: 0;
}

.auth-link-button:hover {
  color: #0550ae;
}

.auth-idp-block {
  display: grid;
  gap: 0.85rem;
}

.auth-idp-divider {
  display: flex;
  align-items: center;
  gap: 0.8rem;
  color: #656d76;
  font-size: 0.82rem;
}

.auth-idp-divider::before,
.auth-idp-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: #d8dee4;
}

.auth-idp-list {
  display: grid;
  gap: 0.75rem;
}

.auth-idp-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.7rem;
  width: 100%;
  min-height: var(--auth-control-height);
  border-radius: var(--auth-control-radius);
  border: 1px solid #d0d7de;
  background: #fff;
  color: #1f2328;
  font: inherit;
  font-size: 0.9rem;
  font-weight: 600;
  cursor: pointer;
}

.auth-idp-button:hover {
  background: #f6f8fa;
}

.auth-idp-badge {
  display: inline-grid;
  place-items: center;
  width: 1.55rem;
  height: 1.55rem;
  border-radius: 999px;
  background: #24292f;
  color: #fff;
  font-size: 0.82rem;
}

.auth-inline-field {
  display: grid;
  gap: 0.75rem;
}

.auth-inline-action {
  width: auto;
}

.auth-alert {
  border-radius: 10px;
  padding: 0.75rem 0.85rem;
  font-size: 0.86rem;
  line-height: 1.5;
}

.auth-alert-muted {
  background: #f6f8fa;
  border: 1px solid #d8dee4;
  color: #57606a;
}

.auth-confirm-box {
  display: grid;
  gap: 0.75rem;
}

.auth-confirm-title {
  font-size: 0.95rem;
  font-weight: 600;
}

.auth-confirm-list {
  margin: 0;
  padding-left: 1.2rem;
  color: #57606a;
  display: grid;
  gap: 0.4rem;
}

.auth-account-summary {
  display: grid;
  gap: 0.15rem;
  padding: 0.85rem 0.9rem;
  border-radius: 10px;
  background: #f6f8fa;
  border: 1px solid #d8dee4;
}

.auth-account-summary strong {
  font-size: 0.94rem;
}

.auth-account-summary span {
  color: #656d76;
  font-size: 0.82rem;
}

.auth-captcha-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.75rem;
  align-items: center;
}

.auth-captcha-input {
  display: block;
}

.auth-captcha-image-frame {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 6.5rem;
  min-height: var(--auth-control-height);
  padding: 0.25rem 0.4rem;
  border-radius: 8px;
  border: 1px solid #d0d7de;
  background: #fff;
}

.auth-captcha-image {
  max-width: 100%;
  max-height: 2.1rem;
  display: block;
  cursor: pointer;
}

.auth-turnstile-shell {
  min-height: calc(var(--auth-control-height) + 8px);
  display: flex;
  align-items: center;
}

.auth-turnstile-widget {
  width: 100%;
}

@media (min-width: 640px) {
  .auth-inline-field {
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
  }
}
</style>
