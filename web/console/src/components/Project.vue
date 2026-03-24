<template>
  <Application
    v-if="currentView === 'application-detail'"
    :current-application="currentApplication"
    :application-update-form="applicationStore.applicationUpdateForm"
    :application-type-options="applicationTypeOptions"
    :grant-type-options="grantTypeOptions"
    :token-type-options="tokenTypeOptions"
    :client-authentication-type-options="clientAuthenticationTypeOptions"
    :application-assignable-roles="applicationAssignableRoles"
    :format-application-type="formatApplicationType"
    :format-application-token-type="formatApplicationTokenType"
    :format-application-grant-type="formatApplicationGrantType"
    :format-application-client-authentication-type="formatApplicationClientAuthenticationType"
    :format-role-labels="formatRoleLabels"
    @back="backToProjectDetail"
    @disable="showApplicationDisableNotice"
    @delete="showApplicationDeleteNotice"
    @update-application="updateApplication"
    @save-application-metadata="saveApplicationMetadata"
    @reset-application-key="resetApplicationKey"
  />

  <section v-else-if="projectViewMode === 'list'" class="section-grid">
    <div class="info-card">
      <div class="section-title">当前组织下可用的项目</div>
      <div class="record-list project-list-records">
        <button
          v-for="project in projectStore.projects"
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
          <div class="project-card-value mb-1">{{ consoleStore.formatDateTime(project.createdAt) }}</div>
          <div class="record-meta">更新时间</div>
          <div class="project-card-value mb-1">{{ consoleStore.formatDateTime(project.updatedAt) }}</div>
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
    :applications="applicationStore.applications"
    :project-update-form="projectStore.projectUpdateForm"
    :project-assigned-user-ids="projectStore.projectAssignedUserIds"
    :format-application-token-type="formatApplicationTokenType"
    :format-application-grant-type="formatApplicationGrantType"
    :format-role-labels="formatRoleLabels"
    :format-application-client-authentication-type="formatApplicationClientAuthenticationType"
    @back="backToProjectList"
    @disable="showProjectDisableNotice"
    @delete="showProjectDeleteNotice"
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
    :project-form="projectStore.projectForm"
    @update:visible="projectCreateModalVisible = $event"
    @hidden="projectStore.resetProjectCreateForm"
    @submit="submitProjectCreate"
  />

  <ApplicationCreateModal
    :visible="applicationCreateModalVisible"
    :application-form="applicationStore.applicationForm"
    :application-assignable-roles="applicationAssignableRoles"
    :application-type-options="applicationTypeOptions"
    :grant-type-options="grantTypeOptions"
    :token-type-options="tokenTypeOptions"
    :client-authentication-type-options="clientAuthenticationTypeOptions"
    :application-protocol-templates="applicationProtocolTemplates"
    @update:visible="applicationCreateModalVisible = $event"
    @hidden="applicationStore.resetApplicationCreateForm"
    @submit="submitApplicationCreate"
    @validation-error="showToast($event, 'danger')"
    @toggle-role-name="toggleRoleName"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'bootstrap-vue-next'
import ProjectDetail from '@/components/ProjectDetail.vue'
import Application from '@/components/Application.vue'
import ApplicationKeyModal from '@/modal/ApplicationKeyModal.vue'
import ProjectCreateModal from '@/modal/ProjectCreateModal.vue'
import ApplicationCreateModal from '@/modal/ApplicationCreateModal.vue'
import { useApplicationStore } from '@/stores/application'
import { useConsoleStore } from '@/stores/console'
import { useOrganizationStore } from '@/stores/organization'
import { useProjectStore } from '@/stores/project'
import { useRoleStore } from '@/stores/role'
import { notifyToast } from '@shared/utils/notify'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const applicationStore = useApplicationStore()
const consoleStore = useConsoleStore()
const organizationStore = useOrganizationStore()
const projectStore = useProjectStore()
const roleStore = useRoleStore()

function showToast(
  message: string,
  variant: 'success' | 'danger',
  options: {
    source: string
    trigger?: string
    error?: unknown
    metadata?: Record<string, unknown>
  } = {
    source: 'console/Project'
  }
) {
  notifyToast({
    toast,
    message,
    variant,
    source: options.source,
    trigger: options.trigger,
    error: options.error,
    metadata: options.metadata
  })
}

const consoleApplicationId = import.meta.env.PPVT_CONSOLE_APPLICATION_ID ?? ''
const projectViewMode = ref<'list' | 'detail'>('list')
const projectCreateModalVisible = ref(false)
const applicationCreateModalVisible = ref(false)
const applicationKeyModalVisible = ref(false)
const applicationKeyModalTitle = ref('应用私钥')
const applicationPrivateKeySnapshot = ref('')

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
  { value: 'private_key_jwt', text: 'private_key_jwt' }
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
const currentProject = computed(() => projectStore.projects.find((item: any) => item.id === projectStore.selectedProjectId) || projectStore.projects[0])
const currentApplication = computed(() => applicationStore.applications.find((item: any) => item.id === applicationStore.selectedApplicationId) || applicationStore.applications[0])
const applicationAssignableRoles = computed(() => roleStore.roles.filter((item: any) => item.type === 'application'))

watchEffect(() => {
  if (currentView.value === 'application-detail') {
    consoleStore.setPageHeader('', '')
    return
  }
  if (projectViewMode.value === 'detail') {
    consoleStore.setPageHeader('', '')
    return
  }
  consoleStore.setPageHeader('项目', '管理项目与应用的结构、协议模式与接入配置。')
})

watch(
  () => [consoleStore.currentOrganizationId, route.name, route.params.projectId, route.params.applicationId, route.query.create, route.query.projectId],
  async ([organizationId, routeName, routeProjectId, routeApplicationId]) => {
    const nextOrganizationId = typeof organizationId === 'string' ? organizationId : ''
    if (!nextOrganizationId) {
      clearProjectAndApplicationState()
      return
    }
    projectViewMode.value = routeName === 'console-project-list' ? 'list' : 'detail'
    await projectStore.loadProjects(nextOrganizationId)
    if (typeof routeProjectId === 'string' && routeProjectId) {
      projectStore.setSelectedProjectId(routeProjectId)
    }
    const builtinProject = organizationStore.currentOrganization?.projects?.find((item: any) => (item.applications || []).some((application: any) => application.id === consoleApplicationId))
    if (!projectStore.selectedProjectId) {
      projectStore.setSelectedProjectId(builtinProject?.id || projectStore.projects[0]?.id || '')
    }
    await applicationStore.loadApplications(projectStore.selectedProjectId)
    if (typeof routeApplicationId === 'string' && routeApplicationId) {
      applicationStore.setSelectedApplicationId(routeApplicationId)
    }
    await handleRouteCreateAction()
  },
  { immediate: true }
)

async function withFeedback(fn: () => Promise<void>, successMessage = '操作成功') {
  try {
    await fn()
    showToast(successMessage, 'success', {
      source: 'console/Project.withFeedback',
      trigger: 'withFeedback'
    })
  } catch (error) {
    showToast(String(error), 'danger', {
      source: 'console/Project.withFeedback',
      trigger: 'withFeedback',
      error
    })
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
  projectStore.clearProjectState()
  applicationStore.clearApplicationState()
}

function selectApplication(application: any) {
  applicationStore.setSelectedApplicationId(application?.id ?? '')
}

async function selectProject(project: any) {
  projectStore.setSelectedProjectId(project?.id ?? '')
  await applicationStore.loadApplications(project?.id ?? '')
  projectViewMode.value = 'detail'
  await router.push({ name: 'console-project-detail', params: { organizationId: consoleStore.currentOrganizationId || organizationStore.currentOrganization?.id || '', projectId: project.id ?? '' } })
}

async function goApplicationDetail(application: any) {
  selectApplication(application)
  await router.push({
    name: 'console-application-detail',
    params: {
      organizationId: consoleStore.currentOrganizationId,
      projectId: projectStore.selectedProjectId || currentProject.value?.id || '',
      applicationId: application.id ?? ''
    }
  })
}

function backToProjectList() {
  projectViewMode.value = 'list'
  void router.push({ name: 'console-project-list', params: { organizationId: consoleStore.currentOrganizationId || organizationStore.currentOrganization?.id || '' } })
}

async function backToProjectDetail() {
  await router.push({ name: 'console-project-detail', params: { organizationId: consoleStore.currentOrganizationId || organizationStore.currentOrganization?.id || '', projectId: projectStore.selectedProjectId || currentProject.value?.id || '' } })
}

function openProjectCreateModal() {
  projectStore.resetProjectCreateForm()
  projectCreateModalVisible.value = true
}

function openApplicationCreateModal() {
  if (!projectStore.selectedProjectId && !currentProject.value?.id) {
    showToast('请先选择项目', 'danger', {
      source: 'console/Project.openApplicationCreateModal',
      trigger: 'openApplicationCreateModal'
    })
    return
  }
  applicationStore.resetApplicationCreateForm(projectStore.selectedProjectId || currentProject.value?.id || '')
  applicationCreateModalVisible.value = true
}

async function handleRouteCreateAction() {
  const createAction = typeof route.query.create === 'string' ? route.query.create : ''
  if (!createAction) {
    return
  }
  if (createAction === 'project') {
    openProjectCreateModal()
    await clearRouteCreateAction()
    return
  }
  if (createAction === 'application') {
    const routeProjectId = typeof route.query.projectId === 'string' ? route.query.projectId : ''
    if (routeProjectId && routeProjectId !== projectStore.selectedProjectId) {
      projectStore.setSelectedProjectId(routeProjectId)
      await applicationStore.loadApplications(routeProjectId)
    }
    openApplicationCreateModal()
    await clearRouteCreateAction()
  }
}

async function clearRouteCreateAction() {
  await router.replace({
    name: String(route.name ?? ''),
    params: route.params,
    query: {}
  })
}

async function submitProjectCreate() {
  await withFeedback(async () => {
    await projectStore.createProject()
  })
  projectStore.resetProjectCreateForm()
  projectCreateModalVisible.value = false
}

async function updateProject() {
  await withFeedback(async () => {
    await projectStore.updateProject()
  })
}

async function submitApplicationCreate() {
  let createdApplicationId = ''
  await withFeedback(async () => {
    const created = await applicationStore.createApplication()
    createdApplicationId = created.id ?? ''
    showApplicationPrivateKey(created.generatedPrivateKey, '应用私钥')
  })
  if (!createdApplicationId) return
  applicationStore.resetApplicationCreateForm(projectStore.selectedProjectId || currentProject.value?.id || '')
  applicationCreateModalVisible.value = false
  applicationStore.setSelectedApplicationId(createdApplicationId)
  await router.push({
    name: 'console-application-detail',
    params: {
      organizationId: consoleStore.currentOrganizationId || organizationStore.currentOrganization?.id || '',
      projectId: projectStore.selectedProjectId || currentProject.value?.id || '',
      applicationId: createdApplicationId
    }
  })
}

async function updateApplication() {
  const updateProtocolError = validateApplicationProtocolInput(applicationStore.applicationUpdateForm)
  if (updateProtocolError) {
    showToast(updateProtocolError, 'danger', {
      source: 'console/Project.updateApplication',
      trigger: 'validateApplicationProtocolInput'
    })
    return
  }
  await withFeedback(async () => {
    const updated = await applicationStore.updateApplication()
    showApplicationPrivateKey(updated?.generatedPrivateKey, '应用私钥')
  })
}

async function saveApplicationMetadata(rows: Array<{ key: string; value: string }>) {
  await withFeedback(async () => {
    await applicationStore.saveApplicationMetadata(rows)
  }, '应用元信息已保存')
}

async function resetApplicationKey() {
  if (!applicationStore.applicationUpdateForm.id) return
  await withFeedback(async () => {
    const result = await applicationStore.resetApplicationKey()
    showApplicationPrivateKey(result?.generatedPrivateKey, '重置后的应用私钥')
  })
}

async function saveProjectUserAssignments(userIds: string[]) {
  if (!projectStore.selectedProjectId) return
  await withFeedback(async () => {
    await projectStore.saveProjectUserAssignments(userIds)
  }, '用户分配已保存')
}

async function showProjectDisableNotice() {
  if (!projectStore.selectedProjectId) return
  await withFeedback(async () => {
    await projectStore.disableProject()
    await applicationStore.loadApplications(projectStore.selectedProjectId)
  }, '项目已停用')
}

async function showProjectDeleteNotice() {
  if (!projectStore.selectedProjectId) return
  await withFeedback(async () => {
    await projectStore.deleteProject()
    applicationStore.clearApplicationState()
    backToProjectList()
  }, '项目已删除')
}

async function showApplicationDisableNotice() {
  if (!applicationStore.selectedApplicationId) return
  await withFeedback(async () => {
    await applicationStore.disableApplication()
  }, '应用已停用')
}

async function showApplicationDeleteNotice() {
  if (!applicationStore.selectedApplicationId) return
  await withFeedback(async () => {
    await applicationStore.deleteApplication()
    await backToProjectDetail()
  }, '应用已删除')
}

function showApplicationPrivateKey(privateKey?: string, title = '应用私钥') {
  if (!privateKey) {
    return
  }
  applicationPrivateKeySnapshot.value = privateKey
  applicationKeyModalTitle.value = title
  applicationKeyModalVisible.value = true
}
</script>
