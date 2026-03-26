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
      <div class="auth-inline-field auth-mfa-code-field">
        <input
          v-model="code"
          name="code"
          inputmode="numeric"
          maxlength="6"
          class="auth-mfa-code-input"
          :placeholder="auth.text.verificationCodePlaceholder"
        />
        <button
          v-if="auth.showsChallengeButton"
          type="button"
          class="auth-button auth-button-secondary auth-inline-action auth-mfa-code-action"
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

<style scoped>
.auth-mfa-code-field {
  display: grid;
  gap: 0.75rem;
}

.auth-mfa-code-input {
  width: 100%;
  max-width: 10rem;
}

.auth-mfa-code-action {
  min-width: 8.5rem;
  white-space: nowrap;
}

@media (min-width: 640px) {
  .auth-mfa-code-field {
    grid-template-columns: minmax(0, 10rem) minmax(8.5rem, max-content);
    align-items: center;
  }
}
</style>
