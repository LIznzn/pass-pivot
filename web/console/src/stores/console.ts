import { ref } from 'vue'
import { defineStore } from 'pinia'
import router from '../router'
import { startConsoleLogout } from '../api/auth'
import { requestPost } from '../util/request'
import { formatDateTime as formatSharedDateTime } from '@shared/utils/datetime'

export type ConsoleTab = 'dashboard' | 'organization' | 'project' | 'user' | 'role' | 'audit' | 'setting'

export const useConsoleStore = defineStore('console', () => {
  const tab = ref<ConsoleTab>('dashboard')
  const message = ref('')
  const messageVariant = ref<'success' | 'danger'>('success')
  const pageHeaderTitle = ref('')
  const pageHeaderDescription = ref('')
  const currentOrganizationId = ref('')
  const currentLoginUser = ref('')
  const currentLoginName = ref('')
  const currentLoginEmail = ref('')

  function syncRouteState() {
    const route = router.currentRoute.value
    const routeName = String(route.name ?? 'console-dashboard')
    if (routeName === 'console-organization') tab.value = 'organization'
    else if (routeName === 'console-project-list' || routeName === 'console-project-detail' || routeName === 'console-application-detail') tab.value = 'project'
    else if (routeName === 'console-user-list' || routeName === 'console-user-detail') tab.value = 'user'
    else if (routeName === 'console-role-list' || routeName === 'console-role-detail') tab.value = 'role'
    else if (routeName === 'console-audit') tab.value = 'audit'
    else if (routeName === 'console-settings') tab.value = 'setting'
    else tab.value = 'dashboard'
    if (typeof route.params.organizationId === 'string' && route.params.organizationId) {
      currentOrganizationId.value = route.params.organizationId
    }
  }

  function resolveOrganizationId() {
    const routeOrganizationId = router.currentRoute.value.params.organizationId
    if (typeof routeOrganizationId === 'string' && routeOrganizationId) {
      return routeOrganizationId
    }
    return currentOrganizationId.value
  }

  async function setTab(nextTab: ConsoleTab) {
    tab.value = nextTab
    const organizationId = resolveOrganizationId()
    if (!organizationId) {
      await router.push({ name: 'console-organization-manage' })
      return
    }
    if (nextTab === 'dashboard') {
      await router.push({ name: 'console-dashboard', params: { organizationId } })
      return
    }
    if (nextTab === 'organization') {
      await router.push({ name: 'console-organization', params: { organizationId } })
      return
    }
    if (nextTab === 'project') {
      await router.push({ name: 'console-project-list', params: { organizationId } })
      return
    }
    if (nextTab === 'user') {
      await router.push({ name: 'console-user-list', params: { organizationId } })
      return
    }
    if (nextTab === 'role') {
      await router.push({ name: 'console-role-list', params: { organizationId } })
      return
    }
    if (nextTab === 'audit') {
      await router.push({ name: 'console-audit', params: { organizationId } })
      return
    }
    if (nextTab === 'setting') {
      await router.push({ name: 'console-settings', params: { organizationId } })
      return
    }
  }

  async function toggleManageOrganization() {
    await router.push({ name: 'console-organization-manage' })
  }

  function goMy(hash = '') {
    const portalBaseUrl = import.meta.env.PPVT_CONSOLE_PORTAL_BASE_URL ?? 'http://localhost:8092'
    const suffix = hash.startsWith('#') ? hash : ''
    window.location.assign(`${portalBaseUrl}/portal/my${suffix}`)
  }

  function logout() {
    sessionStorage.removeItem('ppvt-login-identifier')
    sessionStorage.removeItem('ppvt-login-name')
    sessionStorage.removeItem('ppvt-login-email')
    sessionStorage.removeItem('ppvt-external-idp-application-id')
    startConsoleLogout()
  }

  function initializeCurrentLoginUser() {
    currentLoginUser.value = sessionStorage.getItem('ppvt-login-identifier') ?? ''
    currentLoginName.value = sessionStorage.getItem('ppvt-login-name') ?? ''
    currentLoginEmail.value = sessionStorage.getItem('ppvt-login-email') ?? ''
  }

  async function loadCurrentLoginUser() {
    try {
      const profile = await requestPost<{
        username?: string
        name?: string
        email?: string
      }>('/api/user/v1/profile/query', {})
      currentLoginUser.value = profile.username?.trim() || profile.email?.trim() || currentLoginUser.value
      currentLoginName.value = profile.name?.trim() || profile.username?.trim() || currentLoginName.value
      currentLoginEmail.value = profile.email?.trim() || currentLoginEmail.value
      if (currentLoginUser.value) {
        sessionStorage.setItem('ppvt-login-identifier', currentLoginUser.value)
      }
      if (currentLoginName.value) {
        sessionStorage.setItem('ppvt-login-name', currentLoginName.value)
      }
      if (currentLoginEmail.value) {
        sessionStorage.setItem('ppvt-login-email', currentLoginEmail.value)
      }
    } catch {
      // Ignore profile bootstrap failure and keep existing session snapshot.
    }
  }

  function formatDateTime(value?: string) {
    return formatSharedDateTime(value)
  }

  async function copyMetricValue(value: string) {
    if (!value || value === '-') {
      return
    }
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(value)
      } else {
        const textarea = document.createElement('textarea')
        textarea.value = value
        textarea.setAttribute('readonly', 'true')
        textarea.style.position = 'absolute'
        textarea.style.left = '-9999px'
        document.body.appendChild(textarea)
        textarea.select()
        document.execCommand('copy')
        document.body.removeChild(textarea)
      }
      setMessage('已复制到剪贴板', 'success')
    } catch (error) {
      setMessage(String(error), 'danger')
    }
  }

  function scrollToPanel(id: string) {
    const target = document.getElementById(id)
    if (!target) {
      return
    }
    const topbar = document.querySelector('.admin-topbar') as HTMLElement | null
    const offset = (topbar?.offsetHeight ?? 0) + 32
    const targetTop = target.getBoundingClientRect().top + window.scrollY - offset
    window.scrollTo({ top: Math.max(targetTop, 0), behavior: 'smooth' })
  }

  function setMessage(value: string, variant: 'success' | 'danger') {
    message.value = value
    messageVariant.value = variant
  }

  function clearMessage() {
    message.value = ''
  }

  function setPageHeader(title: string, description = '') {
    pageHeaderTitle.value = title
    pageHeaderDescription.value = description
  }

  return {
    tab,
    message,
    messageVariant,
    pageHeaderTitle,
    pageHeaderDescription,
    currentOrganizationId,
    currentLoginUser,
    currentLoginName,
    currentLoginEmail,
    syncRouteState,
    resolveOrganizationId,
    setTab,
    toggleManageOrganization,
    goMy,
    logout,
    initializeCurrentLoginUser,
    loadCurrentLoginUser,
    formatDateTime,
    copyMetricValue,
    scrollToPanel,
    setMessage,
    clearMessage,
    setPageHeader
  }
})
