<template>
  <div v-if="auth.context" class="auth-form">
    <div class="auth-confirm-box">
      <div class="auth-confirm-title">{{ auth.text.confirmationTitle }}</div>
      <ul class="auth-confirm-list">
        <li>{{ auth.text.confirmationItems.trustedDevice }}</li>
        <li>{{ auth.text.confirmationItems.futureSkip }}</li>
        <li>{{ auth.text.confirmationItems.continueWithoutTrust }}</li>
      </ul>
    </div>

    <div>
      <button type="button" class="auth-button auth-button-primary auth-button-block" @click="auth.confirmSession(true, true)">
        {{ auth.text.confirmTrustDevice }}
      </button>
    </div>

    <div>
      <button type="button" class="auth-button auth-button-secondary auth-button-block" @click="auth.confirmSession(true, false)">
        {{ auth.text.confirmContinueWithoutTrust }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>
