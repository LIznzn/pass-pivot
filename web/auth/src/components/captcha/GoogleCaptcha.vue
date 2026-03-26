<template>
  <label class="auth-field">
    <span>{{ text.captcha }}</span>
    <div class="auth-turnstile-shell">
      <div ref="container" class="auth-turnstile-widget"></div>
    </div>
  </label>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import type { TranslationShape } from '@/i18n/auth'

const props = defineProps<{
  text: TranslationShape
  clientKey: string
  resetKey?: string | number
}>()

const emit = defineEmits<{
  (event: 'update:token', value: string): void
  (event: 'error', error: Error): void
}>()

const container = ref<HTMLDivElement | null>(null)
const widgetId = ref<number | null>(null)
let loadScriptPromise: Promise<void> | null = null

function hasRecaptchaAPI() {
  return typeof window !== 'undefined' &&
    typeof window.grecaptcha?.render === 'function' &&
    typeof window.grecaptcha?.reset === 'function'
}

function loadRecaptchaScript() {
  if (hasRecaptchaAPI()) return Promise.resolve()
  if (loadScriptPromise) return loadScriptPromise
  loadScriptPromise = new Promise((resolve, reject) => {
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
  return loadScriptPromise
}

async function renderCaptcha() {
  if (!props.clientKey || !container.value) {
    return
  }
  try {
    await loadRecaptchaScript()
    await nextTick()
    if (!container.value || !hasRecaptchaAPI() || !window.grecaptcha) {
      throw new Error('google captcha is unavailable')
    }
    emit('update:token', '')
    if (widgetId.value !== null) {
      window.grecaptcha.reset(widgetId.value)
      return
    }
    container.value.innerHTML = ''
    widgetId.value = window.grecaptcha.render(container.value, {
      sitekey: props.clientKey,
      callback: (token: string) => emit('update:token', String(token || '').trim()),
      'expired-callback': () => emit('update:token', ''),
      'error-callback': () => {
        emit('update:token', '')
        emit('error', new Error('google captcha failed to load'))
      }
    })
  } catch (error) {
    emit('error', error instanceof Error ? error : new Error(String(error)))
  }
}

onMounted(() => {
  void renderCaptcha()
})

watch(
  () => [props.clientKey, props.resetKey],
  () => {
    void renderCaptcha()
  }
)
</script>
