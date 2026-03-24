<template>
  <div v-if="auth.context" class="auth-form">
    <div class="auth-confirm-box">
      <div class="auth-confirm-title">{{ auth.text.accountTitle }}</div>
      <div class="auth-account-summary">
        <strong>{{ auth.currentAccountLabel }}</strong>
        <span v-if="auth.currentAccountSecondary">{{ auth.currentAccountSecondary }}</span>
      </div>
    </div>

    <div>
      <button type="button" class="auth-button auth-button-primary auth-button-block" @click="auth.continueAuthorization">
        {{ auth.continueButtonText }}
      </button>
    </div>

    <div>
      <button type="button" class="auth-button auth-button-secondary auth-button-block" @click="auth.switchAccount">
        {{ auth.text.logoutAndSwitchAccount }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>
