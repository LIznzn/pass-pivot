<template>
  <div v-if="auth.context" class="auth-form">
    <strong class="auth-result-title">{{ auth.applicationName }}</strong>
    <p class="auth-result-text">{{ auth.localizedResultMessage }}</p>
    <p v-if="auth.currentAccountLabel" class="auth-result-text">{{ auth.currentAccountLabel }}</p>
    <button type="button" class="auth-button auth-button-secondary auth-button-block" @click="closePage">
      {{ auth.text.closePage }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

function closePage() {
  window.close()
  window.location.assign('/')
}
</script>
