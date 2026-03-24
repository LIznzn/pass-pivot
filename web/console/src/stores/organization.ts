import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import router from '../router'
import { useConsoleStore } from './console'
import { useUserStore } from './user'
import { useRoleStore } from './role'
import { useAuditStore } from './audit'
import {
  createOrganization as apiCreateOrganization,
  deleteOrganization as apiDeleteOrganization,
  disableOrganization as apiDisableOrganization,
  queryOrganizations as apiQueryOrganizations,
  updateOrganization as apiUpdateOrganization
} from '../api/manage/organization'

export const useOrganizationStore = defineStore('organization', () => {
  const console = useConsoleStore()
  const organizations = ref<any[]>([])
  const createOrganizationModalVisible = ref(false)

  const organizationForm = reactive({ name: '', description: '' })

  const currentOrganization = computed(() => organizations.value.find((item: any) => item.id === console.currentOrganizationId) || organizations.value[0])

  async function loadOrganizations() {
    const response = await apiQueryOrganizations()
    organizations.value = response.items
    if (response.items.length === 0) {
      console.setMessage('当前没有可用组织', 'danger')
    }
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
      resetCreateOrganizationForm()
      createOrganizationModalVisible.value = false
    })
  }

  function openCreateOrganizationModal() {
    resetCreateOrganizationForm()
    createOrganizationModalVisible.value = true
  }

  function resetCreateOrganizationForm() {
    organizationForm.name = ''
    organizationForm.description = ''
  }

  async function saveOrganizationMetadata(rows: Array<{ id?: string; key: string; value: string }>) {
    if (!currentOrganization.value?.id) {
      return
    }
    await withFeedback(async () => {
      await apiUpdateOrganization({
        id: currentOrganization.value.id,
        metadata: buildOrganizationMetadataPayload(rows)
      })
      await loadOrganizations()
    })
  }

  async function saveOrganizationConsoleSettings(consoleSettings: unknown, options: { name?: string; description?: string } = {}) {
    if (!currentOrganization.value?.id) {
      return
    }
    await apiUpdateOrganization({
      id: currentOrganization.value.id,
      name: options.name ?? '',
      description: options.description ?? '',
      consoleSettings
    })
    await loadOrganizations()
  }

  async function showOrganizationDisableNotice() {
    if (!currentOrganization.value?.id) {
      return
    }
    await withFeedback(async () => {
      await apiDisableOrganization(currentOrganization.value.id)
      await loadOrganizations()
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
      console.currentOrganizationId = organizations.value[0]?.id ?? ''
      await Promise.all([loadOrganizations(), useUserStore().loadUsers(), useRoleStore().loadRoles(), useAuditStore().loadAudit()])
      if (console.currentOrganizationId) {
        await router.push({ name: 'console-organization', params: { organizationId: console.currentOrganizationId } })
      }
    }, '组织已删除')
  }

  async function handleOrganizationSwitch(value: string) {
    console.currentOrganizationId = value
    await loadOrganizations()
    await Promise.all([useUserStore().loadUsers(), useRoleStore().loadRoles(), useAuditStore().loadAudit()])
    const route = router.currentRoute.value
    const nextRoute = buildOrganizationSwitchRoute({
      currentRouteName: String(route.name ?? ''),
      organizationId: value,
      projectId: typeof route.params.projectId === 'string' ? route.params.projectId : '',
      applicationId: typeof route.params.applicationId === 'string' ? route.params.applicationId : ''
    })
    if (nextRoute) {
      await router.push(nextRoute)
    }
  }

  async function withFeedback(fn: () => Promise<void>, successMessage = '操作成功') {
    try {
      await fn()
      console.setMessage(successMessage, 'success')
    } catch (error) {
      console.setMessage(String(error), 'danger')
    }
  }

  return {
    organizations,
    createOrganizationModalVisible,
    organizationForm,
    currentOrganization,
    loadOrganizations,
    createOrganization,
    openCreateOrganizationModal,
    resetCreateOrganizationForm,
    saveOrganizationMetadata,
    saveOrganizationConsoleSettings,
    showOrganizationDisableNotice,
    showOrganizationDeleteNotice,
    handleOrganizationSwitch
  }
})

function buildOrganizationSwitchRoute(options: {
  currentRouteName: string
  organizationId: string
  projectId: string
  applicationId: string
}) {
  if (options.currentRouteName === 'console-dashboard') {
    return { name: 'console-dashboard', params: { organizationId: options.organizationId } }
  }
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
