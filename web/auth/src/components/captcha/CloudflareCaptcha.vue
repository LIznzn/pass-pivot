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
const widgetId = ref<string | null>(null)
let loadScriptPromise: Promise<void> | null = null

function hasTurnstileAPI() {
  return typeof window !== 'undefined' &&
    typeof window.turnstile?.render === 'function' &&
    typeof window.turnstile?.reset === 'function'
}

function loadTurnstileScript() {
  if (hasTurnstileAPI()) return Promise.resolve()
  if (loadScriptPromise) return loadScriptPromise
  loadScriptPromise = new Promise((resolve, reject) => {
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
  return loadScriptPromise
}

async function renderCaptcha() {
  if (!props.clientKey || !container.value) {
    return
  }
  try {
    await loadTurnstileScript()
    await nextTick()
    if (!container.value || !hasTurnstileAPI() || !window.turnstile) {
      throw new Error('cloudflare captcha is unavailable')
    }
    emit('update:token', '')
    if (widgetId.value) {
      window.turnstile.reset(widgetId.value)
      return
    }
    container.value.innerHTML = ''
    widgetId.value = window.turnstile.render(container.value, {
      sitekey: props.clientKey,
      theme: 'light',
      callback: (token: string) => emit('update:token', String(token || '').trim()),
      'expired-callback': () => emit('update:token', ''),
      'error-callback': () => {
        emit('update:token', '')
        emit('error', new Error('cloudflare captcha failed to load'))
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
