<template>
  <label class="auth-field">
    <span>{{ text.captcha }}</span>
    <div class="auth-turnstile-shell">
      <div ref="container" class="auth-turnstile-widget"></div>
    </div>
  </label>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import type { TranslationShape } from '@/i18n/auth'
import { useAuthStore } from '@/stores/auth'

defineProps<{
  text: TranslationShape
}>()

const auth = useAuthStore()
const container = ref<HTMLDivElement | null>(null)

onMounted(async () => {
  await nextTick()
  if (container.value) {
    void auth.renderGoogleCaptcha(container.value)
  }
})
</script>
