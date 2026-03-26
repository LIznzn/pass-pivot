import { createRouter, createWebHistory } from 'vue-router'
import MainPage from '@/pages/MainPage.vue'
import ForgotPasswordPage from '@/pages/ForgotPasswordPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/auth/authorize' },
    {
      path: '/auth/authorize',
      name: 'auth-authorize',
      component: MainPage
    },
    {
      path: '/auth/forgot-password',
      name: 'auth-forgot-password',
      component: ForgotPasswordPage
    }
  ]
})

export default router
