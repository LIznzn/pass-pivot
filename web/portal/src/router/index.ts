import { createRouter, createWebHistory } from 'vue-router'
import UserCenterPage from '@/pages/UserCenterPage.vue'
import PortalCallbackPage from '@/pages/PortalCallbackPage.vue'
import LogoutPage from '@/pages/LogoutPage.vue'
import { getCurrentAccessToken } from '@/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/portal/my' },
    { path: '/portal/my', name: 'portal-my', component: UserCenterPage, meta: { requiresAuth: true } },
    { path: '/portal/callback', name: 'portal-callback', component: PortalCallbackPage },
    { path: '/portal/logout', name: 'portal-logout', component: LogoutPage },
  ]
})

router.beforeEach(async (to) => {
  if (to.meta.requiresAuth && !getCurrentAccessToken()) {
    return {
      name: 'portal-callback',
      query: {
        target: new URL(to.fullPath, window.location.origin).toString()
      }
    }
  }
  return true
})

export default router
