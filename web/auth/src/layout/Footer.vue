<template>
  <footer class="auth-footer">
    <label class="auth-locale-switcher">
      <span>{{ text.language }}</span>
      <select :value="locale" @change="$emit('update:locale', ($event.target as HTMLSelectElement).value as Locale)">
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
</template>

<script setup lang="ts">
import type { Locale } from '@/i18n/locale'
import type { TranslationShape } from '@/i18n/auth'

defineProps<{
  locale: Locale
  localeOptions: ReadonlyArray<{ value: Locale; label: string }>
  text: TranslationShape
  termsOfServiceUrl: string
  privacyPolicyUrl: string
}>()

defineEmits<{
  (event: 'update:locale', value: Locale): void
}>()
</script>
