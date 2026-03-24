<template>
  <div v-if="auth.context" class="auth-form">
    <div class="auth-result" :class="auth.resultStatus === 'success' ? 'auth-result-success' : 'auth-result-error'">
      <strong>{{ auth.applicationName }}</strong>
      <p>{{ auth.resultMessage || auth.stageHint }}</p>
      <p v-if="auth.currentAccountLabel">{{ auth.currentAccountLabel }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>
