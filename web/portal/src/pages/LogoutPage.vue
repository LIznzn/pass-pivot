<template>
  <PageShell>
    <div class="mb-4">
      <div class="eyebrow">PPVT Authentication</div>
      <h1 class="display-title">退出登录</h1>
      <p class="text-secondary mb-0">{{ message }}</p>
    </div>
  </PageShell>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import PageShell from '@shared/components/PageShell.vue'
import { clearPortalAuthSession, getCurrentAccessToken, getCurrentRefreshToken } from '../auth'

const route = useRoute()
const message = ref('正在结束当前登录会话。')
const authBaseUrl = import.meta.env.PPVT_PORTAL_AUTH_BASE_URL ?? 'http://localhost:8091'

onMounted(() => {
  const postLogoutRedirectUri = typeof route.query.post_logout_redirect_uri === 'string' ? route.query.post_logout_redirect_uri.trim() : ''
  const accessToken = getCurrentAccessToken()
  const refreshToken = getCurrentRefreshToken()
  clearPortalAuthSession()
  const url = new URL(`${authBaseUrl}/auth/end_session`)
  if (postLogoutRedirectUri) {
    url.searchParams.set('post_logout_redirect_uri', postLogoutRedirectUri)
  }
  if (accessToken) {
    url.searchParams.set('access_token', accessToken)
  }
  if (refreshToken) {
    url.searchParams.set('refresh_token', refreshToken)
  }
  window.location.replace(url.toString())
})
</script>
