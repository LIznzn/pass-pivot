import { createRouter, createWebHistory } from 'vue-router'
import MainPage from '@/pages/MainPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/auth/authorize' },
    {
      path: '/auth/authorize',
      name: 'auth-authorize',
      component: MainPage
    }
  ]
})

export default router
