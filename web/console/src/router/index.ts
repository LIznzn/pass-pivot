import { createRouter, createWebHistory } from 'vue-router'
import ConsolePage from '../pages/ConsolePage.vue'
import ConsoleCallbackPage from '../pages/ConsoleCallbackPage.vue'
import ConsoleLoginRedirectPage from '../pages/ConsoleLoginRedirectPage.vue'
import { getCurrentAccessToken } from '../auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/console' },
    { path: '/console', name: 'console-entry', component: ConsoleLoginRedirectPage },
    { path: '/console/callback', name: 'console-callback', component: ConsoleCallbackPage },
    { path: '/console/dashboard', name: 'console-dashboard', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/organization', name: 'console-organization', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/project', name: 'console-project-list', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/project/create', name: 'console-project-create', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/project/:projectId', name: 'console-project-detail', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/project/:projectId/application/create', name: 'console-application-create', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/project/:projectId/application/:applicationId', name: 'console-application-detail', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/user', name: 'console-user-list', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/user/:userId', name: 'console-user-detail', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/role', name: 'console-role-list', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/role/:roleId', name: 'console-role-detail', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/audit', name: 'console-audit', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/:organizationId/settings', name: 'console-settings', component: ConsolePage, meta: { requiresAuth: true } },
    { path: '/console/organization/select', name: 'console-organization-manage', component: ConsolePage, meta: { requiresAuth: true } }
  ]
})

router.beforeEach(async (to) => {
  if (to.meta.requiresAuth && !getCurrentAccessToken()) {
    return {
      name: 'console-entry',
      query: {
        target: new URL(to.fullPath, window.location.origin).toString()
      }
    }
  }
  return true
})

export default router
