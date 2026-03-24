<template>
  <div class="min-vh-100 d-flex align-items-center justify-content-center bg-body-tertiary px-3">
    <div class="card shadow-sm border-0" style="max-width: 32rem; width: 100%">
      <div class="card-body p-4 text-center">
        <div class="mb-3">
          <div class="spinner-border text-primary" role="status" aria-hidden="true"></div>
        </div>
        <h1 class="h4 mb-2">正在跳转登录</h1>
        <p class="text-secondary mb-0">{{ message }}</p>
        <button v-if="showRetry" type="button" class="btn btn-primary mt-3" @click="restartLogin">重新登录</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { formatPortalError } from '@/utils/auth-error'
import { usePortalAuthStore } from '@/stores/auth'

const route = useRoute()
const authStore = usePortalAuthStore()
const message = ref('正在前往统一登录页。')
const showRetry = ref(false)

onMounted(async () => {
  const target = typeof route.query.target === 'string' ? route.query.target : `${window.location.origin}/portal/my`
  try {
    await authStore.startAuthorization(target)
  } catch (error) {
    message.value = formatPortalError(error)
    showRetry.value = true
  }
})

function restartLogin() {
  const target = typeof route.query.target === 'string' ? route.query.target : `${window.location.origin}/portal/my`
  void authStore.startAuthorization(target)
}
</script>
