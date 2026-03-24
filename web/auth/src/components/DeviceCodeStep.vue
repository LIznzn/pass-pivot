<template>
  <form class="auth-form" @submit.prevent="submitUserCode">
    <p class="auth-step-hint">{{ auth.stageHint }}</p>

    <label class="auth-field">
      <span>{{ auth.text.userCode }}</span>
      <input
        v-model="userCode"
        name="user_code"
        autocomplete="one-time-code"
        autocapitalize="characters"
        :placeholder="auth.text.userCodePlaceholder"
      />
    </label>

    <button type="submit" class="auth-button auth-button-primary auth-button-block">
      {{ auth.text.submitUserCode }}
    </button>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const userCode = ref(typeof route.query.user_code === 'string' ? route.query.user_code : '')

function submitUserCode() {
  void auth.submitDeviceUserCode(userCode.value)
}
</script>
