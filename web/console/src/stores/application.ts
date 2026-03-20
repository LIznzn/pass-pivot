import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  createApplication as apiCreateApplication,
  deleteApplication as apiDeleteApplication,
  disableApplication as apiDisableApplication,
  queryApplications as apiQueryApplications,
  resetApplicationKey as apiResetApplicationKey,
  updateApplication as apiUpdateApplication
} from '../api/manage/application'
import { useProjectStore } from './project'

export const useApplicationStore = defineStore('application', () => {
  const projectStore = useProjectStore()
  const applications = ref<any[]>([])
  const selectedApplicationId = ref('')

  const applicationForm = reactive({
    projectId: '',
    name: '',
    redirectUris: '',
    applicationType: 'web',
    tokenType: ['access_token'] as string[],
    enableRefreshToken: false,
    grantType: ['authorization_code_pkce'] as string[],
    clientAuthenticationType: 'none',
    roles: [] as string[],
    publicKey: '',
    accessTokenTTLMinutes: 10,
    refreshTokenTTLHours: 168
  })
  const applicationUpdateForm = reactive({
    id: '',
    name: '',
    redirectUris: '',
    applicationType: 'web',
    tokenType: ['access_token'] as string[],
    enableRefreshToken: false,
    grantType: ['authorization_code_pkce'] as string[],
    clientAuthenticationType: 'none',
    roles: [] as string[],
    publicKey: '',
    accessTokenTTLMinutes: 10,
    refreshTokenTTLHours: 168
  })

  const currentApplication = computed(() => applications.value.find((item: any) => item.id === selectedApplicationId.value) || applications.value[0])
  const currentProject = computed(() => projectStore.projects.find((item: any) => item.id === projectStore.selectedProjectId) || projectStore.projects[0])

  function syncApplicationForm(application?: any) {
    if (!application) {
      applicationUpdateForm.id = ''
      applicationUpdateForm.name = ''
      applicationUpdateForm.redirectUris = ''
      applicationUpdateForm.applicationType = 'web'
      applicationUpdateForm.grantType = ['authorization_code_pkce']
      applicationUpdateForm.clientAuthenticationType = 'none'
      applicationUpdateForm.tokenType = ['access_token']
      applicationUpdateForm.enableRefreshToken = false
      applicationUpdateForm.roles = []
      applicationUpdateForm.publicKey = ''
      applicationUpdateForm.accessTokenTTLMinutes = 10
      applicationUpdateForm.refreshTokenTTLHours = 168
      return
    }
    applicationUpdateForm.id = application.id ?? ''
    applicationUpdateForm.name = application.name ?? ''
    applicationUpdateForm.redirectUris = application.redirectUris ?? ''
    applicationUpdateForm.applicationType = application.applicationType ?? 'web'
    applicationUpdateForm.grantType = [...(application.grantType ?? ['authorization_code_pkce'])]
    applicationUpdateForm.clientAuthenticationType = application.clientAuthenticationType ?? 'none'
    applicationUpdateForm.tokenType = [...(application.tokenType ?? ['access_token'])]
    applicationUpdateForm.enableRefreshToken = Boolean(application.enableRefreshToken)
    applicationUpdateForm.roles = [...(application.roles ?? [])]
    applicationUpdateForm.publicKey = application.publicKey ?? ''
    applicationUpdateForm.accessTokenTTLMinutes = application.accessTokenTTLMinutes ?? 10
    applicationUpdateForm.refreshTokenTTLHours = application.refreshTokenTTLHours ?? 168
  }

  function clearApplicationState() {
    applications.value = []
    selectedApplicationId.value = ''
    syncApplicationForm()
  }

  function resetApplicationCreateForm(projectId = projectStore.selectedProjectId || currentProject.value?.id || '') {
    applicationForm.projectId = projectId
    applicationForm.name = ''
    applicationForm.redirectUris = ''
    applicationForm.applicationType = 'web'
    applicationForm.tokenType = ['access_token']
    applicationForm.enableRefreshToken = false
    applicationForm.grantType = ['authorization_code_pkce']
    applicationForm.clientAuthenticationType = 'none'
    applicationForm.roles = []
    applicationForm.publicKey = ''
    applicationForm.accessTokenTTLMinutes = 10
    applicationForm.refreshTokenTTLHours = 168
  }

  function setSelectedApplicationId(applicationId: string) {
    selectedApplicationId.value = applicationId
    syncApplicationForm(currentApplication.value)
  }

  async function loadApplications(projectId = projectStore.selectedProjectId) {
    applicationForm.projectId = projectId || ''
    if (!projectId) {
      clearApplicationState()
      return
    }
    const response = await apiQueryApplications({ projectId })
    applications.value = response.items
    if (!applications.value.some((item: any) => item.id === selectedApplicationId.value)) {
      selectedApplicationId.value = applications.value[0]?.id ?? ''
    }
    syncApplicationForm(currentApplication.value)
  }

  async function createApplication() {
    const created = await apiCreateApplication({
      ...applicationForm,
      roles: [...applicationForm.roles],
      accessTokenTTLMinutes: Number(applicationForm.accessTokenTTLMinutes),
      refreshTokenTTLHours: Number(applicationForm.refreshTokenTTLHours)
    })
    applicationForm.publicKey = created.publicKey ?? ''
    await loadApplications(applicationForm.projectId)
    return created
  }

  async function updateApplication() {
    const updated = await apiUpdateApplication({
      ...applicationUpdateForm,
      roles: [...applicationUpdateForm.roles],
      accessTokenTTLMinutes: Number(applicationUpdateForm.accessTokenTTLMinutes),
      refreshTokenTTLHours: Number(applicationUpdateForm.refreshTokenTTLHours)
    })
    applicationUpdateForm.publicKey = updated.publicKey ?? applicationUpdateForm.publicKey
    await loadApplications(applicationForm.projectId || projectStore.selectedProjectId)
    syncApplicationForm(currentApplication.value)
    return updated
  }

  async function resetApplicationKey() {
    if (!applicationUpdateForm.id) {
      return null
    }
    const result = await apiResetApplicationKey(applicationUpdateForm.id)
    applicationUpdateForm.publicKey = result.publicKey ?? ''
    await loadApplications(applicationForm.projectId || projectStore.selectedProjectId)
    return result
  }

  async function disableApplication() {
    if (!selectedApplicationId.value) {
      return
    }
    await apiDisableApplication(selectedApplicationId.value)
    await loadApplications(applicationForm.projectId || projectStore.selectedProjectId)
  }

  async function deleteApplication() {
    if (!selectedApplicationId.value) {
      return
    }
    await apiDeleteApplication(selectedApplicationId.value)
    selectedApplicationId.value = ''
    await loadApplications(applicationForm.projectId || projectStore.selectedProjectId)
  }

  return {
    applications,
    selectedApplicationId,
    applicationForm,
    applicationUpdateForm,
    syncApplicationForm,
    clearApplicationState,
    resetApplicationCreateForm,
    setSelectedApplicationId,
    loadApplications,
    createApplication,
    updateApplication,
    resetApplicationKey,
    disableApplication,
    deleteApplication
  }
})
