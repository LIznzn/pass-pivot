<template>
  <form v-if="auth.context" class="auth-form" @submit.prevent="submitLogin">
    <label class="auth-field">
      <span>{{ auth.text.identifier }}</span>
      <input
        v-model="identifier"
        name="identifier"
        autocomplete="username"
        :placeholder="auth.text.identifierPlaceholder"
      />
    </label>

    <label class="auth-field">
      <span class="auth-field-label-row">
        <span>{{ auth.text.password }}</span>
        <button type="button" class="auth-inline-link" @click="goToForgotPassword">
          {{ auth.text.forgotPassword }}
        </button>
      </span>
      <input
        v-model="secret"
        name="secret"
        type="password"
        autocomplete="current-password"
        :placeholder="auth.text.passwordPlaceholder"
      />
    </label>

    <DefaultCaptcha
      v-if="auth.context.captcha?.provider === 'default'"
      :text="auth.text"
      :captcha-image-data-url="auth.captchaImageDataUrl"
      :captcha-challenge-token="auth.captchaChallengeToken"
      :answer="captchaAnswer"
      @refresh="auth.refreshCaptcha"
      @update:answer="captchaAnswer = $event"
    />

    <CloudflareCaptcha
      v-else-if="auth.context.captcha?.provider === 'cloudflare'"
      :text="auth.text"
      :client-key="auth.context.captcha.client_key || ''"
      :reset-key="auth.stage"
      @update:token="auth.setCaptchaToken"
      @error="handleCaptchaError"
    />
    <GoogleCaptcha
      v-else-if="auth.context.captcha?.provider === 'google'"
      :text="auth.text"
      :client-key="auth.context.captcha.client_key || ''"
      :reset-key="auth.stage"
      @update:token="auth.setCaptchaToken"
      @error="handleCaptchaError"
    />

    <button type="submit" class="auth-button auth-button-primary auth-button-block">
      {{ auth.text.signIn }}
    </button>

    <div v-if="auth.externalIdps.length" class="auth-idp-block">
      <div class="auth-idp-divider">
        <span>{{ auth.text.or }}</span>
      </div>
      <div class="auth-idp-list">
        <button
          v-for="provider in auth.externalIdps"
          :key="provider.id"
          type="button"
          class="auth-idp-button"
        >
          <span class="auth-idp-badge" aria-hidden="true">{{ resolveProviderGlyph(provider.name) }}</span>
          <span>{{ auth.text.continueWithProvider(provider.name) }}</span>
        </button>
      </div>
    </div>

    <button
      v-if="auth.supportsWebAuthnLogin"
      type="button"
      class="auth-link-button"
      @click="auth.loginWithWebAuthn(identifier)"
    >
      {{ auth.text.signInWithPasskey }}
    </button>

  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import CloudflareCaptcha from './captcha/CloudflareCaptcha.vue'
import DefaultCaptcha from './captcha/DefaultCaptcha.vue'
import GoogleCaptcha from './captcha/GoogleCaptcha.vue'
import { useAuthStore } from '@/stores/auth'
import { resolveProviderGlyph } from '@/utils/provider'

const auth = useAuthStore()
const identifier = ref('')
const secret = ref('')
const captchaAnswer = ref('')

if (auth.context?.captcha?.provider === 'default' && !auth.captchaImageDataUrl) {
  void auth.refreshCaptcha()
}

function submitLogin() {
  void auth.createSession(identifier.value, secret.value, captchaAnswer.value)
}

function goToForgotPassword() {
  const url = new URL('/auth/forgot-password', window.location.origin)
  const clientId = new URLSearchParams(window.location.search).get('client_id')
  if (clientId) {
    url.searchParams.set('clientId', clientId)
  }
  window.open(url.toString(), '_blank', 'noopener,noreferrer')
}

function handleCaptchaError(error: Error) {
  auth.setMessage(auth.formatRequestError(error), 'danger', {
    source: 'auth/LoginStep.captcha',
    trigger: 'captcha.render',
    error
  })
}
</script>
