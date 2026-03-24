import { createRouter, createWebHistory } from 'vue-router'
import MainPage from '@/pages/MainPage.vue'
import Dashboard from '@/components/Dashboard.vue'
import OrganizationSelect from '@/components/OrganizationSelect.vue'
import Organization from '@/components/Organization.vue'
import Project from '@/components/Project.vue'
import ProjectDetail from '@/components/ProjectDetail.vue'
import Application from '@/components/Application.vue'
import User from '@/components/User.vue'
import UserDetail from '@/components/UserDetail.vue'
import Role from '@/components/Role.vue'
import RoleDetail from '@/components/RoleDetail.vue'
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
        {
          path: '/console/organization/:organizationId/project',
          component: Project,
          children: [
            { path: ':projectId', name: 'console-project-detail', component: ProjectDetail },
            { path: ':projectId/application/:applicationId', name: 'console-application-detail', component: Application }
          ]
        },
        {
          path: '/console/organization/:organizationId/user',
          component: User,
          children: [
            { path: ':userId', name: 'console-user-detail', component: UserDetail }
          ]
        },
        {
          path: '/console/organization/:organizationId/role',
          component: Role,
          children: [
            { path: ':roleId', name: 'console-role-detail', component: RoleDetail }
          ]
        },
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
