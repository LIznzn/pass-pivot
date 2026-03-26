<template>
  <header class="auth-header">
    <div class="auth-brand-mark" aria-hidden="true">P</div>
    <h1 class="auth-title">{{ title }}</h1>
    <p v-if="subtitle || organizationDisplayName" class="auth-subnote">{{ subtitle || text.developedBy(applicationName, organizationDisplayName) }}</p>
  </header>
</template>

<script setup lang="ts">
import type { TranslationShape } from '@/i18n/auth'

defineProps<{
  title: string
  subtitle?: string
  applicationName: string
  organizationDisplayName: string
  text: TranslationShape
}>()
</script>
