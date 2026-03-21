<template>
  <div class="min-vh-100 d-flex align-items-center justify-content-center bg-body-tertiary px-3">
    <div class="card shadow-sm border-0" style="max-width: 32rem; width: 100%">
      <div class="card-body p-4 text-center">
        <div class="mb-3">
          <div class="spinner-border text-primary" role="status" aria-hidden="true"></div>
        </div>
        <h1 class="h4 mb-2">正在跳转到登录页</h1>
        <p class="text-secondary mb-0">{{ message }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { getCurrentAccessToken, startConsoleAuthorization } from '../api/auth'

const route = useRoute()
const message = ref('正在准备控制台登录流程。')

function normalizeTarget(value: string) {
  let next = value
  for (let i = 0; i < 3; i += 1) {
    try {
      const url = new URL(next, window.location.origin)
      if (url.pathname !== '/console') {
        return url.toString()
      }
      const nested = url.searchParams.get('target')
      if (!nested) {
        return `${window.location.origin}/console/dashboard`
      }
      next = nested
    } catch {
      return `${window.location.origin}/console/dashboard`
    }
  }
  return `${window.location.origin}/console/dashboard`
}

onMounted(async () => {
  if (getCurrentAccessToken()) {
    window.location.replace('/console/dashboard')
    return
  }
  try {
    const rawTarget = typeof route.query.target === 'string' && route.query.target
      ? route.query.target
      : `${window.location.origin}/console/dashboard`
    const target = normalizeTarget(rawTarget)
    await startConsoleAuthorization(target)
  } catch (error) {
    message.value = String(error)
  }
})
</script>
