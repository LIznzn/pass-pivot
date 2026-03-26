<template>
  <div class="min-vh-100 d-flex align-items-center justify-content-center bg-body-tertiary px-3">
    <div class="card shadow-sm border-0" style="max-width: 32rem; width: 100%">
      <div class="card-body p-4 text-center">
        <div class="mb-3">
          <div class="spinner-border text-primary" role="status" aria-hidden="true"></div>
        </div>
        <h1 class="h4 mb-2">正在完成登录</h1>
        <p class="text-secondary mb-0">{{ message }}</p>
        <button v-if="showRetry" type="button" class="btn btn-primary mt-3" @click="restartLogin">重新登录</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { clearConsoleAuthSession, clearConsoleOAuthHandshake, finishConsoleAuthorization, startConsoleAuthorization } from '@/api/auth'

const route = useRoute()
const message = ref('正在交换授权码并建立控制台会话。')
const showRetry = ref(false)

onMounted(async () => {
  const error = typeof route.query.error === 'string' ? route.query.error : ''
  const errorDescription = typeof route.query.error_description === 'string' ? route.query.error_description : ''
  const code = typeof route.query.code === 'string' ? route.query.code : ''
  const state = typeof route.query.state === 'string' ? route.query.state : ''

  if (!error && !code && !state) {
    window.location.replace('/console')
    return
  }

  if (error) {
    message.value = errorDescription || error
    clearConsoleAuthSession()
    clearConsoleOAuthHandshake()
    showRetry.value = true
    return
  }

  try {
    const target = await finishConsoleAuthorization(code, state)
    window.location.replace(target)
  } catch (err) {
    message.value = String(err)
    clearConsoleAuthSession()
    clearConsoleOAuthHandshake()
    showRetry.value = true
  }
})

function restartLogin() {
  void startConsoleAuthorization(`${window.location.origin}/console`)
}
</script>
