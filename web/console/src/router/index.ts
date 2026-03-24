import { createRouter, createWebHistory } from 'vue-router'
import MainPage from '@/pages/MainPage.vue'
import Dashboard from '@/components/Dashboard.vue'
import OrganizationSelect from '@/components/OrganizationSelect.vue'
import Organization from '@/components/Organization.vue'
import Project from '@/components/Project.vue'
import User from '@/components/User.vue'
import Role from '@/components/Role.vue'
import Audit from '@/components/Audit.vue'
import Settings from '@/components/Settings.vue'
import LoginCallbackPage from '@/pages/LoginCallbackPage.vue'
import LoginRedirectPage from '@/pages/LoginRedirectPage.vue'
import { getCurrentAccessToken } from '@/api/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/console' },
    { path: '/console', name: 'console-entry', component: LoginRedirectPage },
    { path: '/console/callback', name: 'console-callback', component: LoginCallbackPage },
    {
      path: '',
      component: MainPage,
      meta: { requiresAuth: true },
      children: [
        { path: '/console/organization/:organizationId/dashboard', name: 'console-dashboard', component: Dashboard },
        { path: '/console/organization/select', name: 'console-organization-manage', component: OrganizationSelect },
        { path: '/console/organization/:organizationId', name: 'console-organization', component: Organization },
        { path: '/console/organization/:organizationId/project', name: 'console-project-list', component: Project },
        { path: '/console/organization/:organizationId/project/:projectId', name: 'console-project-detail', component: Project },
        { path: '/console/organization/:organizationId/project/:projectId/application/:applicationId', name: 'console-application-detail', component: Project },
        { path: '/console/organization/:organizationId/user', name: 'console-user-list', component: User },
        { path: '/console/organization/:organizationId/user/:userId', name: 'console-user-detail', component: User },
        { path: '/console/organization/:organizationId/role', name: 'console-role-list', component: Role },
        { path: '/console/organization/:organizationId/role/:roleId', name: 'console-role-detail', component: Role },
        { path: '/console/organization/:organizationId/audit', name: 'console-audit', component: Audit },
        { path: '/console/organization/:organizationId/settings', name: 'console-settings', component: Settings }
      ]
    }
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
