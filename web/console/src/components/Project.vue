<template>
  <Application
    v-if="currentView === 'application-detail'"
    :current-application="currentApplication"
    :application-update-form="applicationUpdateForm"
    :application-type-options="applicationTypeOptions"
    :grant-type-options="grantTypeOptions"
    :token-type-options="tokenTypeOptions"
    :client-authentication-type-options="clientAuthenticationTypeOptions"
    :application-assignable-roles="applicationAssignableRoles"
    :module-recent-changes="moduleRecentChanges"
    :format-date-time="console.formatDateTime"
    :format-application-type="formatApplicationType"
    :format-application-token-type="formatApplicationTokenType"
    :format-application-grant-type="formatApplicationGrantType"
    :format-application-client-authentication-type="formatApplicationClientAuthenticationType"
    :format-role-labels="formatRoleLabels"
    @back="backToProjectDetail"
    @disable="showApplicationDisableNotice"
    @delete="showApplicationDeleteNotice"
    @copy-metric="copyMetricValue"
    @scroll-to-panel="scrollToPanel"
    @update-application="updateApplication"
    @reset-application-key="resetApplicationKey"
  />

  <section v-else-if="projectViewMode === 'list'" class="section-grid">
    <div class="info-card">
      <div class="section-title">当前组织下可用的项目</div>
      <div class="record-list project-list-records">
        <button
          v-for="project in projects"
          :key="project.id"
          type="button"
          class="record-card record-card-button"
          @click="selectProject(project)"
        >
          <div class="project-card-id mb-1">{{ project.id }}</div>
          <div class="record-head align-items-center mb-1">
            <div class="project-card-name">{{ project.name || '-' }}</div>
            <span class="badge rounded-pill" :class="project.status === 'disabled' ? 'text-bg-secondary' : 'text-bg-success'">
              {{ project.status === 'disabled' ? '停用' : '启用' }}
            </span>
          </div>
          <div class="record-meta">创建时间</div>
          <div class="project-card-value mb-1">{{ console.formatDateTime(project.createdAt) }}</div>
          <div class="record-meta">更新时间</div>
          <div class="project-card-value mb-1">{{ console.formatDateTime(project.updatedAt) }}</div>
          <div class="record-meta">应用数</div>
          <div class="project-card-value">{{ project.applications?.length ?? 0 }}</div>
        </button>
        <div class="record-card project-create-card">
          <button type="button" class="project-create-trigger" @click="openProjectCreateModal">
            <div class="project-create-plus text-secondary lh-1 mb-2">+</div>
            <div class="project-create-title">创建新项目</div>
          </button>
        </div>
      </div>
    </div>
  </section>

  <ProjectDetail
    v-else
    :current-project="currentProject"
    :applications="applications"
    :project-update-form="projectUpdateForm"
    :project-assigned-user-ids="projectAssignedUserIds"
    :users="console.users"
    :module-recent-changes="moduleRecentChanges"
    :format-date-time="console.formatDateTime"
    :format-application-token-type="formatApplicationTokenType"
    :format-application-grant-type="formatApplicationGrantType"
    :format-role-labels="formatRoleLabels"
    :format-application-client-authentication-type="formatApplicationClientAuthenticationType"
    @back="backToProjectList"
    @disable="showProjectDisableNotice"
    @delete="showProjectDeleteNotice"
    @copy-metric="copyMetricValue"
    @scroll-to-panel="scrollToPanel"
    @go-application-detail="goApplicationDetail"
    @go-application-create="openApplicationCreateModal"
    @save-project-user-assignments="saveProjectUserAssignments"
    @update-project="updateProject"
  />

  <ApplicationKeyModal
    :visible="applicationKeyModalVisible"
    :title="applicationKeyModalTitle"
    :snapshot="applicationPrivateKeySnapshot"
    @update:visible="applicationKeyModalVisible = $event"
  />

  <ProjectCreateModal
    :visible="projectCreateModalVisible"
    :project-form="projectForm"
    @update:visible="projectCreateModalVisible = $event"
    @hidden="resetProjectCreateForm"
    @submit="submitProjectCreate"
  />

  <ApplicationCreateModal
    :visible="applicationCreateModalVisible"
    :application-form="applicationForm"
    :application-assignable-roles="applicationAssignableRoles"
    :application-type-options="applicationTypeOptions"
    :grant-type-options="grantTypeOptions"
    :token-type-options="tokenTypeOptions"
    :client-authentication-type-options="clientAuthenticationTypeOptions"
    :application-protocol-templates="applicationProtocolTemplates"
    @update:visible="applicationCreateModalVisible = $event"
    @hidden="resetApplicationCreateForm"
    @submit="submitApplicationCreate"
    @validation-error="toast.error($event)"
    @toggle-role-name="toggleRoleName"
  />
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from '@shared/composables/toast'
import {
  createApplication as apiCreateApplication,
  deleteApplication as apiDeleteApplication,
  disableApplication as apiDisableApplication,
  queryApplications as apiQueryApplications,
  resetApplicationKey as apiResetApplicationKey,
  updateApplication as apiUpdateApplication
} from '../api/manage/application'
import {
  createProject as apiCreateProject,
  deleteProject as apiDeleteProject,
  disableProject as apiDisableProject,
  queryProjects as apiQueryProjects,
  updateProject as apiUpdateProject,
  updateProjectUserAssignments as apiUpdateProjectUserAssignments
} from '../api/manage/project'
import ProjectDetail from '../components/ProjectDetail.vue'
import Application from '../components/Application.vue'
import ApplicationKeyModal from '../modal/ApplicationKeyModal.vue'
import ProjectCreateModal from '../modal/ProjectCreateModal.vue'
import ApplicationCreateModal from '../modal/ApplicationCreateModal.vue'
import { useConsoleLayout } from '../composables/useConsoleLayout'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const console = useConsoleLayout()

const consoleApplicationId = import.meta.env.PPVT_CONSOLE_APPLICATION_ID ?? ''
const projectViewMode = ref<'list' | 'detail'>('list')
const selectedProjectId = ref('')
const selectedApplicationId = ref('')
const projects = ref<any[]>([])
const applications = ref<any[]>([])
const projectAssignedUserIds = ref<string[]>([])
const projectCreateModalVisible = ref(false)
const applicationCreateModalVisible = ref(false)
const applicationKeyModalVisible = ref(false)
const applicationKeyModalTitle = ref('应用私钥')
const applicationPrivateKeySnapshot = ref('')

const projectQuery = reactive({ organizationId: '' })
const applicationQuery = reactive({ projectId: '' })
const projectForm = reactive({ organizationId: '', name: '', userAclEnabled: false })
const projectUpdateForm = reactive({ id: '', name: '', description: '', userAclEnabled: false })
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

const applicationTypeOptions = [
  { value: 'web', text: 'Web' },
  { value: 'native', text: 'Native' },
  { value: 'api', text: 'API' }
]
const tokenTypeOptions = [
  { value: 'access_token', text: 'access_token' },
  { value: 'id_token', text: 'id_token' }
]
const grantTypeOptions = [
  { value: 'authorization_code', text: 'authorization_code' },
  { value: 'authorization_code_pkce', text: 'authorization_code_pkce' },
  { value: 'client_credentials', text: 'client_credentials' },
  { value: 'device_code', text: 'device_code' },
  { value: 'implicit', text: 'implicit' },
  { value: 'password', text: 'password' }
]
const clientAuthenticationTypeOptions = [
  { value: 'none', text: 'none' },
  { value: 'client_secret_basic', text: 'client_secret_basic' },
  { value: 'client_secret_post', text: 'client_secret_post' },
  { value: 'client_secret_jwt', text: 'client_secret_jwt' },
  { value: 'private_key_jwt', text: 'private_key_jwt' },
  { value: 'tls_client_auth', text: 'tls_client_auth' },
  { value: 'self_signed_tls_client_auth', text: 'self_signed_tls_client_auth' }
]
const applicationProtocolTemplates: Record<string, { text: string; allowedTypes: string[]; grantType: string[]; enableRefreshToken: boolean; tokenType: string[]; clientAuthenticationType: string }> = {
  'oauth21-oidc-pkce-private-key-jwt': { text: 'OAuth2.1 + OIDC 1.0 + Private Key JWT（高安全性）', allowedTypes: ['web'], grantType: ['authorization_code_pkce'], enableRefreshToken: false, tokenType: ['access_token'], clientAuthenticationType: 'private_key_jwt' },
  'oauth21-oidc-pkce-client-secret-basic': { text: 'OAuth2.1 + OIDC 1.0 + Client Secret Basic（高安全性）', allowedTypes: ['web'], grantType: ['authorization_code_pkce'], enableRefreshToken: false, tokenType: ['access_token'], clientAuthenticationType: 'client_secret_basic' },
  'oauth21-oidc-pkce-none': { text: 'OAuth2.1 + OIDC 1.0（中高安全性）', allowedTypes: ['web', 'native'], grantType: ['authorization_code_pkce'], enableRefreshToken: false, tokenType: ['access_token'], clientAuthenticationType: 'none' },
  'oauth20-oidc-auth-code-private-key-jwt': { text: 'OAuth2.0 + OIDC 1.0 + Private Key JWT（中高安全性）', allowedTypes: ['web'], grantType: ['authorization_code'], enableRefreshToken: false, tokenType: ['access_token'], clientAuthenticationType: 'private_key_jwt' },
  'oauth20-oidc-auth-code-client-secret-basic': { text: 'OAuth2.0 + OIDC 1.0 + Client Secret Basic（中高安全性）', allowedTypes: ['web'], grantType: ['authorization_code'], enableRefreshToken: false, tokenType: ['access_token'], clientAuthenticationType: 'client_secret_basic' },
  'oauth20-device-code': { text: 'OAuth2.0 + Device Code（中安全性）', allowedTypes: ['native'], grantType: ['device_code'], enableRefreshToken: false, tokenType: ['access_token'], clientAuthenticationType: 'none' },
  'oauth20-oidc-client-credentials': { text: 'OAuth2.0 + OIDC 1.0 + Client Credentials（中安全性）', allowedTypes: ['api'], grantType: ['client_credentials'], enableRefreshToken: false, tokenType: ['access_token'], clientAuthenticationType: 'client_secret_basic' },
  'oauth20-oidc-implicit-client-secret-basic': { text: 'OAuth2.0 + OIDC 1.0 + Implicit + Client Secret Basic（低安全性）', allowedTypes: ['web'], grantType: ['implicit'], enableRefreshToken: false, tokenType: ['access_token', 'id_token'], clientAuthenticationType: 'client_secret_basic' },
  'oauth20-implicit-client-secret-basic': { text: 'OAuth2.0 + Implicit + Client Secret Basic（低安全性）', allowedTypes: ['web'], grantType: ['implicit'], enableRefreshToken: false, tokenType: ['access_token'], clientAuthenticationType: 'client_secret_basic' }
}

const currentRouteName = computed(() => String(route.name ?? 'console-dashboard'))
const currentView = computed(() => {
  if (currentRouteName.value === 'console-application-detail') return 'application-detail'
  return 'main'
})
const currentProject = computed(() => projects.value.find((item: any) => item.id === selectedProjectId.value) || projects.value[0])
const currentApplication = computed(() => applications.value.find((item: any) => item.id === selectedApplicationId.value) || applications.value[0])
const applicationAssignableRoles = computed(() => console.roles.filter((item: any) => item.type === 'application'))
const moduleRecentChanges = computed(() => console.recentAuditLogs.slice(0, 6))

watch(
  () => [console.currentOrganizationId, route.name, route.params.projectId, route.params.applicationId],
  async ([organizationId, routeName, routeProjectId, routeApplicationId]) => {
    const nextOrganizationId = typeof organizationId === 'string' ? organizationId : ''
    if (!nextOrganizationId) {
      clearProjectAndApplicationState()
      return
    }
    projectForm.organizationId = nextOrganizationId
    projectQuery.organizationId = nextOrganizationId
    projectViewMode.value = routeName === 'console-project-list' ? 'list' : 'detail'
    await loadProjects()
    if (typeof routeProjectId === 'string' && routeProjectId) {
      selectedProjectId.value = routeProjectId
    }
    const builtinProject = console.currentOrganization?.projects?.find((item: any) => (item.applications || []).some((application: any) => application.id === consoleApplicationId))
    if (!selectedProjectId.value) {
      selectedProjectId.value = builtinProject?.id || projects.value[0]?.id || ''
    }
    applicationQuery.projectId = selectedProjectId.value
    applicationForm.projectId = selectedProjectId.value
    await loadApplications()
    if (typeof routeApplicationId === 'string' && routeApplicationId) {
      selectedApplicationId.value = routeApplicationId
    }
  },
  { immediate: true }
)

watch(() => currentProject.value, (value) => {
  if (!value) {
    return
  }
  projectUpdateForm.id = value.id ?? ''
  projectUpdateForm.name = value.name ?? ''
  projectUpdateForm.description = value.description ?? ''
  projectUpdateForm.userAclEnabled = Boolean(value.userAclEnabled)
  projectAssignedUserIds.value = Array.isArray(value.assignedUserIds) ? [...value.assignedUserIds] : []
})

watch(() => currentApplication.value, (value) => {
  if (!value) {
    return
  }
  applicationUpdateForm.id = value.id ?? ''
  applicationUpdateForm.name = value.name ?? ''
  applicationUpdateForm.redirectUris = value.redirectUris ?? ''
  applicationUpdateForm.applicationType = value.applicationType ?? 'web'
  applicationUpdateForm.grantType = [...(value.grantType ?? ['authorization_code_pkce'])]
  applicationUpdateForm.clientAuthenticationType = value.clientAuthenticationType ?? 'none'
  applicationUpdateForm.tokenType = [...(value.tokenType ?? ['access_token'])]
  applicationUpdateForm.enableRefreshToken = Boolean(value.enableRefreshToken)
  applicationUpdateForm.roles = [...(value.roles ?? [])]
  applicationUpdateForm.publicKey = value.publicKey ?? ''
  applicationUpdateForm.accessTokenTTLMinutes = value.accessTokenTTLMinutes ?? 10
  applicationUpdateForm.refreshTokenTTLHours = value.refreshTokenTTLHours ?? 168
})

async function loadProjects() {
  const response = await apiQueryProjects(projectQuery)
  projects.value = response.items
  if (!projects.value.some((item: any) => item.id === selectedProjectId.value)) {
    selectedProjectId.value = projects.value[0]?.id ?? ''
  }
}

async function loadApplications() {
  const response = await apiQueryApplications(applicationQuery)
  applications.value = response.items
  if (!applications.value.some((item: any) => item.id === selectedApplicationId.value)) {
    selectedApplicationId.value = applications.value[0]?.id ?? ''
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

async function copyMetricValue(value: string) {
  if (!value || value === '-') return
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
    toast.success('已复制到剪贴板')
  } catch (error) {
    toast.error(String(error))
  }
}

async function withFeedback(fn: () => Promise<void>, successMessage = '操作成功') {
  try {
    await fn()
    toast.success(successMessage)
  } catch (error) {
    toast.error(String(error))
  }
}

function validateApplicationProtocolInput(target: { tokenType: string[]; enableRefreshToken: boolean; grantType: string[]; clientAuthenticationType: string }) {
  if (!target.grantType.length) return '至少需要选择一个 Grant Type。'
  if (!target.tokenType.length) return '至少需要选择一个 Token Type。'
  if (target.grantType.includes('client_credentials') && !(target.tokenType.length === 1 && target.tokenType[0] === 'access_token')) return 'client_credentials 只允许 token_type=access_token。'
  if (target.grantType.includes('client_credentials') && target.enableRefreshToken) return 'client_credentials 不允许启用 Refresh Token。'
  if (target.grantType.includes('implicit') && target.tokenType.some((item) => !['access_token', 'id_token'].includes(item))) return 'implicit 只允许 access_token 和/或 id_token。'
  if (target.grantType.includes('implicit') && target.enableRefreshToken) return 'implicit 不允许启用 Refresh Token。'
  if (target.clientAuthenticationType === 'none' && target.grantType.some((item) => item !== 'authorization_code_pkce' && item !== 'device_code' && item !== 'password')) return 'client_authentication_type=none 只允许用于 authorization_code_pkce、device_code 或 password。'
  if (!target.tokenType.includes('access_token') && target.enableRefreshToken) return '未签发 access_token 时不能启用 Refresh Token。'
  return ''
}

function formatRoleLabels(value?: string[]) {
  if (!value || value.length === 0) {
    return 'none'
  }
  return value.join(', ')
}

function toggleRoleName(target: string[], value: string, checked: boolean) {
  const index = target.indexOf(value)
  if (checked && index < 0) {
    target.push(value)
    target.sort()
    return
  }
  if (!checked && index >= 0) target.splice(index, 1)
}

function formatApplicationType(value?: string) {
  if (value === 'native') return 'Native'
  if (value === 'api') return 'API'
  return 'Web'
}

function formatApplicationTokenType(value?: string | string[]) {
  const values = Array.isArray(value) ? value : value ? [value] : []
  return values.length ? values.join(' + ') : '-'
}

function formatApplicationGrantType(value?: string | string[]) {
  const values = Array.isArray(value) ? value : value ? [value] : []
  return values.length ? values.join(' + ') : '-'
}

function formatApplicationClientAuthenticationType(value?: string) {
  return value && clientAuthenticationTypeOptions.some((item) => item.value === value) ? value : '-'
}

function clearProjectAndApplicationState() {
  projects.value = []
  applications.value = []
  selectedProjectId.value = ''
  selectedApplicationId.value = ''
}

function selectApplication(application: any) {
  selectedApplicationId.value = application.id ?? ''
  if (!application) {
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

async function selectProject(project: any) {
  selectedProjectId.value = project.id ?? ''
  if (project) {
    projectUpdateForm.id = project.id ?? ''
    projectUpdateForm.name = project.name ?? ''
    projectUpdateForm.description = project.description ?? ''
    projectUpdateForm.userAclEnabled = Boolean(project.userAclEnabled)
  }
  projectAssignedUserIds.value = Array.isArray(project?.assignedUserIds) ? [...project.assignedUserIds] : []
  applicationQuery.projectId = project.id ?? ''
  applicationForm.projectId = project.id ?? ''
  await loadApplications()
  projectViewMode.value = 'detail'
  await router.push({ name: 'console-project-detail', params: { organizationId: console.currentOrganizationId || console.currentOrganization?.id || '', projectId: project.id ?? '' } })
}

async function goApplicationDetail(application: any) {
  selectApplication(application)
  await router.push({
    name: 'console-application-detail',
    params: {
      organizationId: console.currentOrganizationId,
      projectId: selectedProjectId.value || currentProject.value?.id || '',
      applicationId: application.id ?? ''
    }
  })
}

function backToProjectList() {
  projectViewMode.value = 'list'
  void router.push({ name: 'console-project-list', params: { organizationId: console.currentOrganizationId || console.currentOrganization?.id || '' } })
}

async function backToProjectDetail() {
  await router.push({ name: 'console-project-detail', params: { organizationId: console.currentOrganizationId || console.currentOrganization?.id || '', projectId: selectedProjectId.value || currentProject.value?.id || '' } })
}

function openProjectCreateModal() {
  resetProjectCreateForm()
  projectCreateModalVisible.value = true
}

function openApplicationCreateModal() {
  if (!selectedProjectId.value && !currentProject.value?.id) {
    toast.error('请先选择项目')
    return
  }
  resetApplicationCreateForm()
  applicationForm.projectId = selectedProjectId.value || currentProject.value?.id || ''
  applicationCreateModalVisible.value = true
}

async function createProject() {
  await withFeedback(async () => {
    await apiCreateProject(projectForm)
    await loadProjects()
  })
}

function resetProjectCreateForm() {
  projectForm.name = ''
}

async function submitProjectCreate() {
  await createProject()
  resetProjectCreateForm()
  projectCreateModalVisible.value = false
}

async function updateProject() {
  await withFeedback(async () => {
    await apiUpdateProject(projectUpdateForm)
    await loadProjects()
  })
}

async function createApplication() {
  let createdApplicationId = ''
  await withFeedback(async () => {
    const created = await apiCreateApplication({
      ...applicationForm,
      roles: [...applicationForm.roles],
      accessTokenTTLMinutes: Number(applicationForm.accessTokenTTLMinutes),
      refreshTokenTTLHours: Number(applicationForm.refreshTokenTTLHours)
    })
    createdApplicationId = created.id ?? ''
    applicationForm.publicKey = created.publicKey ?? ''
    if (created.generatedPrivateKey) showApplicationPrivateKey(created.generatedPrivateKey, '应用私钥')
    await loadApplications()
  })
  return createdApplicationId
}

function resetApplicationCreateForm() {
  applicationForm.projectId = selectedProjectId.value || currentProject.value?.id || ''
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

async function submitApplicationCreate() {
  const createdApplicationId = await createApplication()
  if (!createdApplicationId) return
  resetApplicationCreateForm()
  applicationCreateModalVisible.value = false
  selectedApplicationId.value = createdApplicationId
  await router.push({
    name: 'console-application-detail',
    params: {
      organizationId: console.currentOrganizationId || console.currentOrganization?.id || '',
      projectId: selectedProjectId.value || currentProject.value?.id || '',
      applicationId: createdApplicationId
    }
  })
}

async function updateApplication() {
  const updateProtocolError = validateApplicationProtocolInput(applicationUpdateForm)
  if (updateProtocolError) {
    toast.error(updateProtocolError)
    return
  }
  await withFeedback(async () => {
    const updated = await apiUpdateApplication({
      ...applicationUpdateForm,
      roles: [...applicationUpdateForm.roles],
      accessTokenTTLMinutes: Number(applicationUpdateForm.accessTokenTTLMinutes),
      refreshTokenTTLHours: Number(applicationUpdateForm.refreshTokenTTLHours)
    })
    applicationUpdateForm.publicKey = updated.publicKey ?? applicationUpdateForm.publicKey
    if (updated.generatedPrivateKey) showApplicationPrivateKey(updated.generatedPrivateKey, '应用私钥')
    await loadApplications()
  })
}

async function resetApplicationKey() {
  if (!applicationUpdateForm.id) return
  await withFeedback(async () => {
    const result = await apiResetApplicationKey(applicationUpdateForm.id)
    applicationUpdateForm.publicKey = result.publicKey ?? ''
    if (result.generatedPrivateKey) showApplicationPrivateKey(result.generatedPrivateKey, '重置后的应用私钥')
    await loadApplications()
  })
}

function showApplicationPrivateKey(privateKey: string, title: string) {
  applicationPrivateKeySnapshot.value = privateKey
  applicationKeyModalTitle.value = title
  applicationKeyModalVisible.value = true
}

async function saveProjectUserAssignments(userIds: string[]) {
  if (!selectedProjectId.value) return
  await withFeedback(async () => {
    const response = await apiUpdateProjectUserAssignments(selectedProjectId.value, userIds)
    projectAssignedUserIds.value = [...(response.userIds ?? [])]
    await loadProjects()
  }, '用户分配已保存')
}

async function showProjectDisableNotice() {
  if (!selectedProjectId.value) return
  await withFeedback(async () => {
    await apiDisableProject(selectedProjectId.value)
    await Promise.all([loadProjects(), loadApplications()])
  }, '项目已停用')
}

async function showProjectDeleteNotice() {
  if (!selectedProjectId.value) return
  await withFeedback(async () => {
    await apiDeleteProject(selectedProjectId.value)
    selectedProjectId.value = ''
    selectedApplicationId.value = ''
    await Promise.all([loadProjects(), loadApplications()])
    backToProjectList()
  }, '项目已删除')
}

async function showApplicationDisableNotice() {
  if (!selectedApplicationId.value) return
  await withFeedback(async () => {
    await apiDisableApplication(selectedApplicationId.value)
    await loadApplications()
  }, '应用已停用')
}

async function showApplicationDeleteNotice() {
  if (!selectedApplicationId.value) return
  await withFeedback(async () => {
    await apiDeleteApplication(selectedApplicationId.value)
    selectedApplicationId.value = ''
    await loadApplications()
    await backToProjectDetail()
  }, '应用已删除')
}
</script>
