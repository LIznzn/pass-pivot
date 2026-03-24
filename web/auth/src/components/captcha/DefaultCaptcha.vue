<template>
  <label class="auth-field">
    <input type="hidden" name="captcha_challenge_token" :value="captchaChallengeToken" />
    <span>{{ text.captcha }}</span>
    <div class="auth-captcha-row">
      <label class="auth-captcha-input">
        <input
          name="captcha_answer"
          :value="answer"
          autocomplete="off"
          :placeholder="text.captchaAnswerPlaceholder"
          @input="$emit('update:answer', ($event.target as HTMLInputElement).value)"
        />
      </label>
      <div v-if="captchaImageDataUrl" class="auth-captcha-image-frame">
        <img
          class="auth-captcha-image"
          :src="captchaImageDataUrl"
          :alt="text.captchaImageAlt"
          @click="$emit('refresh')"
        />
      </div>
    </div>
  </label>
</template>

<script setup lang="ts">
import type { TranslationShape } from '@/i18n/auth'

defineProps<{
  text: TranslationShape
  captchaImageDataUrl: string
  captchaChallengeToken: string
  answer: string
}>()

defineEmits<{
  (event: 'refresh'): void
  (event: 'update:answer', value: string): void
}>()
</script>
