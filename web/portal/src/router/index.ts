import { createRouter, createWebHistory } from 'vue-router'
import MainPage from '@/pages/MainPage.vue'
import LoginRedirectPage from '@/pages/LoginRedirectPage.vue'
import LoginCallbackPage from '@/pages/LoginCallbackPage.vue'
import { usePortalAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/portal/my' },
    { path: '/portal/my', name: 'portal-my', component: MainPage, meta: { requiresAuth: true } },
    { path: '/portal/login', name: 'portal-login', component: LoginRedirectPage },
    { path: '/portal/callback', name: 'portal-callback', component: LoginCallbackPage },
  ]
})

router.beforeEach(async (to) => {
  const authStore = usePortalAuthStore()
  authStore.syncSession()
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    return {
      name: 'portal-login',
      query: {
        target: new URL(to.fullPath, window.location.origin).toString()
      }
    }
  }
  return true
})

export default router
