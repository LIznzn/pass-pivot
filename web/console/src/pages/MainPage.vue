<template>
  <div class="admin-classic">
    <Topbar
      :active-tab="tab"
      :organizations="organizations"
      :current-organization-id="currentOrganizationId"
      :current-organization-label="currentOrganizationLabel"
      :current-user-initials="currentUserInitials"
      :current-user-display-name="currentUserDisplayName"
      :current-user-email="currentUserEmail"
      @set-tab="setTab"
      @switch-organization="handleOrganizationSwitch"
      @manage-organization="toggleManageOrganization"
      @go-my="goMy"
      @logout="logout"
    />

    <main class="admin-content container-fluid py-4">
      <Header :title="pageHeaderTitle" :description="pageHeaderDescription" />

      <ToastHost />

      <RouterView />

      <div v-if="showBackToTopButton" class="console-back-to-top-wrap" :class="backToTopWrapClass">
        <button type="button" class="console-back-to-top" @click="scrollToTop" aria-label="回到顶部">
          <i class="bi bi-arrow-up" aria-hidden="true"></i>
        </button>
      </div>

      <CreateOrganizationModal
        :visible="createOrganizationModalVisible"
        :form="organizationForm"
        @update:visible="createOrganizationModalVisible = $event"
        @hidden="resetCreateOrganizationForm"
        @create="createOrganization"
      />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, provide, reactive, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import ToastHost from '@shared/components/ToastHost.vue'
import { useToast } from '@shared/composables/toast'
import { startConsoleLogout } from '../api/auth'
import {
  createOrganization as apiCreateOrganization,
  deleteOrganization as apiDeleteOrganization,
  disableOrganization as apiDisableOrganization,
  queryOrganizations as apiQueryOrganizations,
  updateOrganization as apiUpdateOrganization
} from '../api/manage/organization'
import { queryAuditLogs as apiQueryAuditLogs } from '../api/manage/policy'
import { queryRoles as apiQueryRoles } from '../api/manage/role'
import { queryUsers as apiQueryUsers } from '../api/manage/user'
import Header from '../layout/Header.vue'
import Topbar from '../layout/Topbar.vue'
import CreateOrganizationModal from '../modal/CreateOrganizationModal.vue'
import { consoleLayoutKey } from '../composables/useConsoleLayout'

type ConsoleTab = 'dashboard' | 'organization' | 'project' | 'user' | 'role' | 'audit' | 'setting'
type ConsoleRefreshTask = 'organizations' | 'users' | 'roles' | 'audit'

const router = useRouter()
const route = useRoute()
const toast = useToast()

const tab = ref<ConsoleTab>('dashboard')
const message = ref('')
const messageVariant = ref<'success' | 'danger'>('success')
const organizations = ref<any[]>([])
const users = ref<any[]>([])
const roles = ref<any[]>([])
const auditLogs = ref<any[]>([])
const policyCount = ref(0)
const externalIdpCount = ref(0)
const createOrganizationModalVisible = ref(false)
const currentOrganizationId = ref('')
const organizationSwitcher = ref('')
const currentLoginUser = ref('')

const organizationForm = reactive({ name: '', description: '' })
const organizationUpdateForm = reactive({ id: '', name: '', description: '', metadata: {} as Record<string, string> })
const userQuery = reactive({ organizationId: '' })
const roleQuery = reactive({ organizationId: '' })

const summary = computed(() => ({
  organizationCount: organizations.value.length,
  projectCount: currentOrganization.value?.projects?.length ?? 0,
  applicationCount: currentOrganization.value?.projects?.reduce((total: number, project: any) => total + (project.applications?.length ?? 0), 0) ?? 0,
  userCount: users.value.length,
  roleCount: roles.value.length,
  policyCount: policyCount.value,
  externalIdpCount: externalIdpCount.value,
  auditCount: auditLogs.value.length
}))

const phoneCountryOptions = [
  { value: '+86', text: '+86 中国' },
  { value: '+852', text: '+852 中国香港' },
  { value: '+853', text: '+853 中国澳门' },
  { value: '+886', text: '+886 中国台湾' },
  { value: '+81', text: '+81 日本' },
  { value: '+82', text: '+82 韩国' },
  { value: '+1', text: '+1 美国/加拿大' },
  { value: '+44', text: '+44 英国' },
  { value: '+49', text: '+49 德国' },
  { value: '+33', text: '+33 法国' },
  { value: '+65', text: '+65 新加坡' },
  { value: '+60', text: '+60 马来西亚' },
  { value: '+61', text: '+61 澳大利亚' }
]

const booleanSettingOptions = [
  { value: 'active', text: '开启' },
  { value: 'disabled', text: '关闭' }
]

const fieldVisibilityOptions = [
  { value: 'hidden', text: '隐藏' },
  { value: 'optional', text: '选填' },
  { value: 'required', text: '必填' }
]

const currentOrganization = computed(() => organizations.value.find((item: any) => item.id === currentOrganizationId.value) || organizations.value[0])
const currentOrganizationLabel = computed(() => currentOrganization.value?.name || currentOrganization.value?.id || '选择组织')
const currentRouteName = computed(() => String(route.name ?? 'console-dashboard'))
const recentAuditLogs = computed(() => auditLogs.value.slice(0, 12))
const moduleRecentChanges = computed(() => recentAuditLogs.value.slice(0, 6))
const currentLoginUserLabel = computed(() => currentLoginUser.value || '当前登录用户')
const currentUserDisplayName = computed(() => currentLoginUser.value || '当前登录用户')
const currentUserEmail = computed(() => currentLoginUser.value || '-')
const currentUserInitials = computed(() => {
  const source = currentUserDisplayName.value || currentUserEmail.value
  const cleaned = source.replace(/[^A-Za-z0-9\u4e00-\u9fa5 ]/g, ' ').trim()
  if (!cleaned) {
    return 'U'
  }
  const parts = cleaned.split(/\s+/).filter(Boolean)
  if (parts.length >= 2) {
    return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase()
  }
  return cleaned.slice(0, 2).toUpperCase()
})

const currentView = computed(() => {
  if (currentRouteName.value === 'console-organization-manage') return 'organization-manage'
  if (currentRouteName.value === 'console-application-detail') return 'application-detail'
  return 'main'
})

const currentTabLabel = computed(() => {
  if (currentView.value === 'organization-manage') return '组织切换'
  if (currentView.value === 'application-detail') return '应用详情'
  if (tab.value === 'dashboard') return '仪表盘'
  if (tab.value === 'organization') return '组织'
  if (tab.value === 'project') return '项目'
  if (tab.value === 'user') return '用户'
  if (tab.value === 'role') return '角色'
  if (tab.value === 'audit') return '审计'
  return '设置'
})

const currentTabDescription = computed(() => {
  if (currentView.value === 'organization-manage') return '在这里切换当前控制台所属组织，必要时可直接创建新的组织；组织基础信息调整请前往该组织的设置页。'
  if (currentView.value === 'application-detail') return '查看并维护当前应用的接入配置。'
  if (tab.value === 'dashboard') return '概览当前实例下的核心 IAM 统计和审计摘要。'
  if (tab.value === 'organization') return ''
  if (tab.value === 'project') return '管理项目与应用的结构、协议模式与接入配置。'
  if (tab.value === 'user') return '管理用户、通行密钥、身份验证器、备用验证码与管理员动作。'
  if (tab.value === 'role') return '维护角色标签、策略规则与 Policy Check。'
  if (tab.value === 'audit') return '查看平台关键事件、登录轨迹与策略变更审计。'
  return '配置外部 OAuth/OIDC 联邦与身份绑定。'
})

const pageHeaderTitle = computed(() => {
  if (tab.value === 'organization' && currentView.value === 'main') return ''
  if (currentView.value === 'application-detail') return ''
  if (currentRouteName.value === 'console-project-detail') return ''
  if (currentRouteName.value === 'console-user-detail') return ''
  if (currentRouteName.value === 'console-role-detail') return ''
  return currentTabLabel.value
})

const pageHeaderDescription = computed(() => {
  if (tab.value === 'organization' && currentView.value === 'main') return ''
  if (currentView.value === 'application-detail') return ''
  if (currentRouteName.value === 'console-project-detail') return ''
  if (currentRouteName.value === 'console-user-detail') return ''
  if (currentRouteName.value === 'console-role-detail') return ''
  return currentTabDescription.value
})

const browserPageTitle = computed(() => {
  if (currentRouteName.value === 'console-dashboard') return 'Dashboard'
  if (currentRouteName.value === 'console-organization-manage') return '组织切换'
  if (currentRouteName.value === 'console-organization') return '组织'
  if (currentRouteName.value === 'console-project-list') return '项目列表'
  if (currentRouteName.value === 'console-project-detail') return '项目详情'
  if (currentRouteName.value === 'console-application-detail') return '应用详情'
  if (currentRouteName.value === 'console-user-list') return '用户列表'
  if (currentRouteName.value === 'console-user-detail') return '用户详情'
  if (currentRouteName.value === 'console-role-list') return '角色列表'
  if (currentRouteName.value === 'console-role-detail') return '角色详情'
  if (currentRouteName.value === 'console-audit') return '审计'
  if (currentRouteName.value === 'console-settings') return '设置'
  return 'Console'
})

const consoleBreadcrumbTitle = computed(() => {
  const organizationLabel = currentOrganization.value?.name || currentOrganization.value?.id || ''
  const segments: string[] = []

  if (currentView.value === 'organization-manage') {
    if (organizationLabel) {
      segments.push(organizationLabel)
    }
  } else if (tab.value === 'organization' || tab.value === 'audit' || tab.value === 'setting') {
    if (organizationLabel) {
      segments.push(organizationLabel)
    }
  } else if (tab.value === 'project') {
    if (organizationLabel) {
      segments.push(organizationLabel)
    }
    if (typeof route.params.projectId === 'string' && route.params.projectId) {
      segments.push(route.params.projectId)
    }
    if (currentView.value === 'application-detail' && typeof route.params.applicationId === 'string' && route.params.applicationId) {
      segments.push(route.params.applicationId)
    }
  } else if (tab.value === 'user') {
    if (organizationLabel) {
      segments.push(organizationLabel)
    }
    if (currentRouteName.value === 'console-user-detail' && typeof route.params.userId === 'string' && route.params.userId) {
      segments.push(route.params.userId)
    }
  } else if (tab.value === 'role') {
    if (organizationLabel) {
      segments.push(organizationLabel)
    }
    if (currentRouteName.value === 'console-role-detail' && typeof route.params.roleId === 'string' && route.params.roleId) {
      segments.push(route.params.roleId)
    }
  } else if (organizationLabel) {
    segments.push(organizationLabel)
  }

  if (!segments.length) {
    return 'Console'
  }
  return `${segments.join(' > ')} | Console`
})

const browserDocumentTitle = computed(() => {
  const pageTitle = browserPageTitle.value
  const breadcrumb = consoleBreadcrumbTitle.value
  if (!breadcrumb || breadcrumb === 'Console') {
    return `${pageTitle} | Console`
  }
  return `${pageTitle} | ${breadcrumb}`
})

const showBackToTopButton = computed(() => {
  if (currentView.value === 'application-detail') return true
  if (tab.value === 'organization') return true
  if (tab.value === 'project' && currentRouteName.value !== 'console-project-list') return true
  if (currentRouteName.value === 'console-user-detail') return true
  if (tab.value === 'role') return true
  if (tab.value === 'audit') return true
  if (tab.value === 'setting') return true
  return false
})

const backToTopWrapClass = computed(() => 'console-back-to-top-wrap-middle')

watch(
  () => [currentRouteName.value, route.params.organizationId, route.params.projectId, route.params.applicationId, route.params.userId, route.params.roleId],
  ([routeName, organizationId]) => {
    if (routeName === 'console-organization') tab.value = 'organization'
    else if (routeName === 'console-project-list' || routeName === 'console-project-detail' || routeName === 'console-application-detail') tab.value = 'project'
    else if (routeName === 'console-user-list' || routeName === 'console-user-detail') tab.value = 'user'
    else if (routeName === 'console-role-list' || routeName === 'console-role-detail') tab.value = 'role'
    else if (routeName === 'console-audit') tab.value = 'audit'
    else if (routeName === 'console-settings') tab.value = 'setting'
    else tab.value = 'dashboard'
    if (typeof organizationId === 'string' && organizationId) {
      currentOrganizationId.value = organizationId
      organizationSwitcher.value = organizationId
    }
  },
  { immediate: true }
)

watch(
  browserDocumentTitle,
  (value) => {
    document.title = value
  },
  { immediate: true }
)

watch(message, (value) => {
  if (!value) {
    return
  }
  if (messageVariant.value === 'danger') {
    toast.error(value)
  } else {
    toast.success(value)
  }
  message.value = ''
})

onMounted(async () => {
  currentLoginUser.value = sessionStorage.getItem('ppvt-login-identifier') ?? ''
  await loadAll()
})

async function loadAll() {
  await loadOrganizations()
  const routeOrganizationId = typeof route.params.organizationId === 'string' ? route.params.organizationId : ''
  const fallbackOrganization = organizations.value.find((item: any) => item.id === routeOrganizationId) || organizations.value[0]
  if (!currentOrganizationId.value || !organizations.value.some((item: any) => item.id === currentOrganizationId.value)) {
    currentOrganizationId.value = fallbackOrganization?.id ?? ''
  }
  if (!organizationSwitcher.value && currentOrganizationId.value) {
    organizationSwitcher.value = currentOrganizationId.value
  }
  const currentOrg = organizations.value.find((item: any) => item.id === currentOrganizationId.value) || fallbackOrganization
  organizationUpdateForm.id = currentOrg?.id ?? organizationUpdateForm.id
  organizationUpdateForm.name = currentOrg?.name ?? organizationUpdateForm.name
  organizationUpdateForm.metadata = normalizeMetadataMap(currentOrg?.metadata)
  roleQuery.organizationId = currentOrg?.id ?? roleQuery.organizationId
  userQuery.organizationId = currentOrg?.id ?? userQuery.organizationId
  const results = await Promise.allSettled([loadUsers(), loadRoles(), loadAudit()])
  const rejected = results.find((item) => item.status === 'rejected') as PromiseRejectedResult | undefined
  if (rejected) {
    message.value = String(rejected.reason)
    messageVariant.value = 'danger'
  }
}

async function loadOrganizations() {
  const response = await apiQueryOrganizations()
  organizations.value = response.items
  const currentOrg = response.items.find((item: any) => item.id === currentOrganizationId.value) || response.items[0]
  if (currentOrg) {
    organizationUpdateForm.id = currentOrg.id ?? ''
    organizationUpdateForm.name = currentOrg.name ?? ''
    organizationUpdateForm.metadata = normalizeMetadataMap(currentOrg.metadata)
  }
  if (response.items.length === 0) {
    message.value = '当前没有可用组织'
    messageVariant.value = 'danger'
  }
}

async function loadUsers() {
  const response = await apiQueryUsers(userQuery)
  users.value = response.items
}

async function loadRoles() {
  const response = await apiQueryRoles(roleQuery)
  roles.value = response.items
}

async function handleOrganizationSwitch(value: string) {
  organizationSwitcher.value = value
  currentOrganizationId.value = value
  await loadAll()
  const nextRoute = buildOrganizationSwitchRoute({
    currentRouteName: currentRouteName.value,
    organizationId: value,
    projectId: typeof route.params.projectId === 'string' ? route.params.projectId : '',
    applicationId: typeof route.params.applicationId === 'string' ? route.params.applicationId : ''
  })
  if (nextRoute) {
    await router.push(nextRoute)
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

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function getConsoleRefreshTasks(tabName: ConsoleTab): ConsoleRefreshTask[] {
  if (tabName === 'organization') return ['organizations']
  if (tabName === 'audit') return ['audit']
  return ['organizations', 'users', 'roles', 'audit']
}

function buildOrganizationSwitchRoute(options: {
  currentRouteName: string
  organizationId: string
  projectId: string
  applicationId: string
}) {
  if (options.currentRouteName === 'console-organization') return { name: 'console-organization', params: { organizationId: options.organizationId } }
  if (options.currentRouteName === 'console-organization-manage') return { name: 'console-organization-manage' }
  if (options.currentRouteName === 'console-project-list') return { name: options.currentRouteName, params: { organizationId: options.organizationId } }
  if (options.currentRouteName === 'console-project-detail') {
    return { name: 'console-project-detail', params: { organizationId: options.organizationId, projectId: options.projectId } }
  }
  if (options.currentRouteName === 'console-application-detail') {
    return { name: 'console-application-detail', params: { organizationId: options.organizationId, projectId: options.projectId, applicationId: options.applicationId } }
  }
  if (options.currentRouteName === 'console-user-list' || options.currentRouteName === 'console-user-detail') return { name: 'console-user-list', params: { organizationId: options.organizationId } }
  if (options.currentRouteName === 'console-role-list' || options.currentRouteName === 'console-role-detail') return { name: 'console-role-list', params: { organizationId: options.organizationId } }
  return null
}

function buildTabRoute(options: {
  nextTab: ConsoleTab
  organizationId: string
}) {
  if (options.nextTab === 'organization') return { name: 'console-organization', params: { organizationId: options.organizationId } }
  if (options.nextTab === 'project') return { name: 'console-project-list', params: { organizationId: options.organizationId } }
  if (options.nextTab === 'user') return { name: 'console-user-list', params: { organizationId: options.organizationId } }
  if (options.nextTab === 'role') return { name: 'console-role-list', params: { organizationId: options.organizationId } }
  if (options.nextTab === 'audit') return { name: 'console-audit', params: { organizationId: options.organizationId } }
  if (options.nextTab === 'setting') return { name: 'console-settings', params: { organizationId: options.organizationId } }
  return { name: 'console-dashboard' }
}

async function runModuleAction() {
  await refreshConsoleTasks(getConsoleRefreshTasks(tab.value))
}

async function refreshConsoleTasks(tasks: ConsoleRefreshTask[]) {
  const taskMap: Record<ConsoleRefreshTask, () => Promise<void>> = {
    organizations: loadOrganizations,
    users: loadUsers,
    roles: loadRoles,
    audit: loadAudit
  }
  await Promise.all(tasks.map((task) => taskMap[task]()))
}

async function setTab(nextTab: ConsoleTab) {
  tab.value = nextTab
  await router.push(buildTabRoute({
    nextTab,
    organizationId: currentOrganizationId.value || currentOrganization.value?.id || ''
  }))
}

async function toggleManageOrganization() {
  await router.push({ name: 'console-organization-manage' })
}

async function goMy(hash = '') {
  const portalBaseUrl = import.meta.env.PPVT_CONSOLE_PORTAL_BASE_URL ?? 'http://localhost:8092'
  const suffix = hash.startsWith('#') ? hash : ''
  window.location.assign(`${portalBaseUrl}/portal/my${suffix}`)
}

async function logout() {
  sessionStorage.removeItem('ppvt-login-identifier')
  sessionStorage.removeItem('ppvt-external-idp-application-id')
  startConsoleLogout()
}

async function loadAudit() {
  const response = await apiQueryAuditLogs()
  auditLogs.value = response.items
}

async function createOrganization() {
  await withFeedback(async () => {
    const name = organizationForm.name.trim()
    const description = organizationForm.description.trim()
    if (!name) {
      throw new Error('organization name is required')
    }
    if (!/^[A-Za-z0-9-]+$/.test(name)) {
      throw new Error('organization name must contain only letters, numbers, and hyphens')
    }
    await apiCreateOrganization({ name, description })
    await loadOrganizations()
    organizationForm.name = ''
    organizationForm.description = ''
    createOrganizationModalVisible.value = false
  })
}

function openCreateOrganizationModal() {
  organizationForm.name = ''
  organizationForm.description = ''
  createOrganizationModalVisible.value = true
}

function resetCreateOrganizationForm() {
  organizationForm.name = ''
  organizationForm.description = ''
}

async function saveOrganizationMetadata(rows: Array<{ id: string; key: string; value: string }>) {
  if (!organizationUpdateForm.id) {
    return
  }
  await withFeedback(async () => {
    organizationUpdateForm.metadata = buildOrganizationMetadataPayload(rows)
    await apiUpdateOrganization({
      id: organizationUpdateForm.id,
      metadata: organizationUpdateForm.metadata
    })
    await loadOrganizations()
  })
}

async function withFeedback(fn: () => Promise<void>, successMessage = '操作成功') {
  try {
    await fn()
    message.value = successMessage
    messageVariant.value = 'success'
  } catch (error) {
    message.value = String(error)
    messageVariant.value = 'danger'
  }
}

function syncCurrentOrganizationContext(organizationId: string) {
  currentOrganizationId.value = organizationId
  organizationSwitcher.value = organizationId
  userQuery.organizationId = organizationId
  roleQuery.organizationId = organizationId
}

async function showOrganizationDisableNotice() {
  if (!currentOrganization.value?.id) {
    return
  }
  await withFeedback(async () => {
    await apiDisableOrganization(currentOrganization.value.id)
    await refreshConsoleTasks(['organizations'])
  }, '组织已停用')
}

async function showOrganizationDeleteNotice() {
  if (!currentOrganization.value?.id) {
    return
  }
  const deletedOrganizationId = currentOrganization.value.id
  await withFeedback(async () => {
    await apiDeleteOrganization(deletedOrganizationId)
    organizations.value = organizations.value.filter((item: any) => item.id !== deletedOrganizationId)
    const fallbackOrganization = organizations.value[0]
    syncCurrentOrganizationContext(fallbackOrganization?.id ?? '')
    await refreshConsoleTasks(['organizations', 'users', 'roles', 'audit'])
    if (currentOrganizationId.value) {
      await router.push({ name: 'console-organization', params: { organizationId: currentOrganizationId.value } })
    }
  }, '组织已删除')
}

function formatDateTime(value?: string) {
  if (!value) {
    return '-'
  }
  return new Date(value).toLocaleString()
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
    message.value = '已复制到剪贴板'
    messageVariant.value = 'success'
  } catch (error) {
    message.value = String(error)
    messageVariant.value = 'danger'
  }
}

function normalizeMetadataMap(value: unknown) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {} as Record<string, string>
  }
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>).map(([key, entryValue]) => [key, String(entryValue ?? '')])
  )
}

function buildOrganizationMetadataPayload(rows: Array<{ key: string; value: string }>) {
  const metadata: Record<string, string> = {}
  for (const item of rows) {
    const key = item.key.trim()
    if (!key) {
      continue
    }
    if (metadata[key] !== undefined) {
      throw new Error(`duplicate metadata key: ${key}`)
    }
    metadata[key] = item.value
  }
  return metadata
}

provide(consoleLayoutKey, reactive({
  tab,
  organizations,
  users,
  roles,
  auditLogs,
  policyCount,
  externalIdpCount,
  currentOrganizationId,
  currentOrganization,
  currentOrganizationLabel,
  currentLoginUserLabel,
  summary,
  recentAuditLogs,
  moduleRecentChanges,
  phoneCountryOptions,
  booleanSettingOptions,
  fieldVisibilityOptions,
  formatDateTime,
  loadOrganizations,
  loadUsers,
  loadRoles,
  loadAudit,
  runModuleAction,
  copyMetricValue,
  scrollToPanel,
  handleOrganizationSwitch,
  openCreateOrganizationModal,
  saveOrganizationMetadata,
  showOrganizationDisableNotice,
  showOrganizationDeleteNotice
}))
</script>
