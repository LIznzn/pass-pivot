<template>
  <div class="auth-form">
    <div class="auth-confirm-box">
      <div class="auth-confirm-title">{{ auth.text.deviceAuthorizationReviewTitle }}</div>
      <div v-if="auth.deviceAuthorizationSummary.length" class="auth-account-summary">
        <strong>{{ auth.text.deviceAuthorizationRequestTitle }}</strong>
        <span v-for="item in auth.deviceAuthorizationSummary" :key="item.label">{{ item.label }}: {{ item.value }}</span>
      </div>
    </div>

    <div>
      <button type="button" class="auth-button auth-button-primary auth-button-block" @click="auth.confirmDeviceAuthorizationReview">
        {{ auth.text.deviceAuthorizationReviewContinue }}
      </button>
    </div>

    <div>
      <button type="button" class="auth-button auth-button-secondary auth-button-block" @click="auth.cancelDeviceAuthorizationReview">
        {{ auth.text.deviceAuthorizationReviewCancel }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>
