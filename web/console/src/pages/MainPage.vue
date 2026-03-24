<template>
  <div class="admin-classic">
    <Topbar
    />

    <main class="admin-content container-fluid py-4">
      <Header :title="pageHeaderTitle" :description="pageHeaderDescription" />

      <RouterView />

      <div v-if="showBackToTopButton" class="console-back-to-top-wrap" :class="backToTopWrapClass">
        <button type="button" class="console-back-to-top" @click="scrollToTop" aria-label="回到顶部">
          <i class="bi bi-arrow-up" aria-hidden="true"></i>
        </button>
      </div>

      <CreateOrganizationModal
        :visible="organizationStore.createOrganizationModalVisible"
        :form="organizationStore.organizationForm"
        @update:visible="organizationStore.createOrganizationModalVisible = $event"
        @hidden="organizationStore.resetCreateOrganizationForm"
        @create="organizationStore.createOrganization"
      />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useToast } from 'bootstrap-vue-next'
import Header from '@/layout/Header.vue'
import Topbar from '@/layout/Topbar.vue'
import CreateOrganizationModal from '@/modal/CreateOrganizationModal.vue'
import { useAuditStore } from '@/stores/audit'
import { useConsoleStore } from '@/stores/console'
import { useOrganizationStore } from '@/stores/organization'
import { useRoleStore } from '@/stores/role'
import { useUserStore } from '@/stores/user'
import { notifyToast } from '@shared/utils/notify'

const route = useRoute()
const toast = useToast()
const auditStore = useAuditStore()
const console = useConsoleStore()
const organizationStore = useOrganizationStore()
const roleStore = useRoleStore()
const userStore = useUserStore()

function showToast(
  message: string,
  variant: 'success' | 'danger',
  options: {
    source: string
    trigger?: string
    error?: unknown
    metadata?: Record<string, unknown>
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

const currentRouteName = computed(() => String(route.name ?? 'console-dashboard'))
const currentView = computed(() => {
  if (currentRouteName.value === 'console-organization-manage') return 'organization-manage'
  if (currentRouteName.value === 'console-application-detail') return 'application-detail'
  return 'main'
})

const pageHeaderTitle = computed(() => {
  return console.pageHeaderTitle
})

const pageHeaderDescription = computed(() => {
  return console.pageHeaderDescription
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
  const organizationLabel = organizationStore.currentOrganization?.name || organizationStore.currentOrganization?.id || ''
  const segments: string[] = []

  if (currentView.value === 'organization-manage') {
    if (organizationLabel) {
      segments.push(organizationLabel)
    }
  } else if (console.tab === 'organization' || console.tab === 'audit' || console.tab === 'setting') {
    if (organizationLabel) {
      segments.push(organizationLabel)
    }
  } else if (console.tab === 'project') {
    if (organizationLabel) {
      segments.push(organizationLabel)
    }
    if (typeof route.params.projectId === 'string' && route.params.projectId) {
      segments.push(route.params.projectId)
    }
    if (currentView.value === 'application-detail' && typeof route.params.applicationId === 'string' && route.params.applicationId) {
      segments.push(route.params.applicationId)
    }
  } else if (console.tab === 'user') {
    if (organizationLabel) {
      segments.push(organizationLabel)
    }
    if (currentRouteName.value === 'console-user-detail' && typeof route.params.userId === 'string' && route.params.userId) {
      segments.push(route.params.userId)
    }
  } else if (console.tab === 'role') {
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
  if (console.tab === 'organization') return true
  if (console.tab === 'project' && currentRouteName.value !== 'console-project-list') return true
  if (currentRouteName.value === 'console-user-detail') return true
  if (console.tab === 'role') return true
  if (console.tab === 'audit') return true
  if (console.tab === 'setting') return true
  return false
})

const backToTopWrapClass = computed(() => 'console-back-to-top-wrap-middle')

watch(
  () => [currentRouteName.value, route.params.organizationId, route.params.projectId, route.params.applicationId, route.params.userId, route.params.roleId],
  () => {
    console.syncRouteState()
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

watch(() => console.message, (value) => {
  if (!value) {
    return
  }
  if (console.messageVariant === 'danger') {
    showToast(value, 'danger', {
      source: console.messageSource || 'console/MainPage.message',
      trigger: console.messageTrigger || 'watch(console.message)',
      error: console.messageError,
      metadata: console.messageMetadata
    })
  } else {
    showToast(value, 'success', {
      source: console.messageSource || 'console/MainPage.message',
      trigger: console.messageTrigger || 'watch(console.message)',
      error: console.messageError,
      metadata: console.messageMetadata
    })
  }
  console.clearMessage()
})

onMounted(async () => {
  console.initializeCurrentLoginUser()
  await console.loadCurrentLoginUser()
  console.syncRouteState()
  await organizationStore.loadOrganizations()
  const fallbackOrganization = organizationStore.organizations.find((item: any) => item.id === console.currentOrganizationId) || organizationStore.organizations[0]
  if (!console.currentOrganizationId || !organizationStore.organizations.some((item: any) => item.id === console.currentOrganizationId)) {
    console.currentOrganizationId = fallbackOrganization?.id ?? ''
  }
  await Promise.allSettled([userStore.loadUsers(), roleStore.loadRoles(), auditStore.loadAudit()])
})

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
</script>
