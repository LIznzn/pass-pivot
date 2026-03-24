<template>
  <form v-if="auth.context" class="auth-form" @submit.prevent="submitMFA">
    <label class="auth-field">
      <span>{{ auth.text.verificationMethod }}</span>
      <select :value="auth.selectedMethod" name="method" @change="auth.setSelectedMethod(($event.target as HTMLSelectElement).value)">
        <option v-for="option in auth.localizedMFAOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </label>

    <label v-if="auth.showsCodeInput" class="auth-field">
      <span>{{ auth.text.verificationCode }}</span>
      <div class="auth-inline-field">
        <input v-model="code" name="code" :placeholder="auth.text.verificationCodePlaceholder" />
        <button
          v-if="auth.showsChallengeButton"
          type="button"
          class="auth-button auth-button-secondary auth-inline-action"
          @click="auth.sendVerificationChallenge"
        >
          {{ auth.text.sendVerificationCode }}
        </button>
      </div>
    </label>

    <button
      v-if="!auth.isSecurityKeyMethod"
      type="submit"
      class="auth-button auth-button-primary auth-button-block"
    >
      {{ auth.text.verifyAndContinue }}
    </button>

    <button
      v-if="auth.isSecurityKeyMethod && auth.supportsSessionU2F"
      type="button"
      class="auth-button auth-button-primary auth-button-block"
      @click="auth.verifyMFAWithU2F"
    >
      {{ auth.text.useSecurityKey }}
    </button>

    <div v-if="auth.challengeFeedback" class="auth-alert auth-alert-muted">
      {{ auth.challengeFeedback }}
    </div>

    <button
      type="button"
      class="auth-button auth-button-secondary auth-button-block"
      @click="auth.cancelLogin"
    >
      {{ auth.text.cancelLogin }}
    </button>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const code = ref('')

function submitMFA() {
  void auth.verifyMFA(code.value)
}
</script>
