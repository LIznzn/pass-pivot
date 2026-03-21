<template>
  <section class="console-module-shell">
    <div class="console-module-summary-card">
      <div class="console-module-hero">
        <div>
          <div class="console-module-eyebrow">仪表盘</div>
          <h2 class="console-module-title">实例概览</h2>
          <p class="console-module-subtitle">概览当前实例下的核心 IAM 统计和审计摘要。</p>
        </div>
        <BButton variant="primary" @click="refreshDashboard">刷新概览</BButton>
      </div>
      <div class="console-module-metrics">
        <div v-for="item in currentModuleMetrics" :key="item.label" class="console-module-metric">
          <span class="console-module-metric-label">{{ item.label }}</span>
          <div class="console-module-metric-value-row">
            <strong class="console-module-metric-value">{{ item.value }}</strong>
            <button
              v-if="item.copyable"
              type="button"
              class="console-module-metric-copy"
              :aria-label="`复制${item.label}`"
              @click="consoleStore.copyMetricValue(item.copyValue || item.value)"
            >
              <i class="bi bi-copy" aria-hidden="true"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="console-module-workspace">
      <aside class="console-module-sidebar">
        <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="consoleStore.scrollToPanel(item.id)">{{ item.label }}</button>
      </aside>
      <div class="console-module-main">
        <div id="dashboard-overview" class="info-card">
          <div class="section-title">平台概览</div>
          <div class="small text-secondary mb-3">组织、项目、应用、用户与策略概览</div>
          <div class="summary-grid">
            <div class="summary-tile" v-for="item in summaryTiles" :key="item.label">
              <span class="summary-label">{{ item.label }}</span>
              <strong class="summary-value">{{ item.value }}</strong>
            </div>
          </div>
        </div>
        <div id="dashboard-audit" class="info-card">
          <div class="section-title">审计摘要</div>
          <BButton size="sm" variant="outline-primary" class="mb-3" @click="auditStore.loadAudit">刷新</BButton>
          <div class="record-list">
            <div v-for="item in recentAuditLogs" :key="item.id" class="record-row">
              <div>
                <strong>{{ item.eventType }}</strong>
                <div class="record-meta">{{ item.actorType }} · {{ item.result }}</div>
              </div>
              <code>{{ formatDateTime(item.createdAt) }}</code>
            </div>
          </div>
        </div>
      </div>
      <RightSide :items="moduleRecentChanges" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { BButton } from 'bootstrap-vue-next'
import RightSide from '../layout/RightSide.vue'
import { useApplicationStore } from '../stores/application'
import { useAuditStore } from '../stores/audit'
import { useConsoleStore } from '../stores/console'
import { useOrganizationStore } from '../stores/organization'
import { useProjectStore } from '../stores/project'
import { useRoleStore } from '../stores/role'
import { useUserStore } from '../stores/user'

const applicationStore = useApplicationStore()
const auditStore = useAuditStore()
const consoleStore = useConsoleStore()
const organizationStore = useOrganizationStore()
const projectStore = useProjectStore()
const roleStore = useRoleStore()
const userStore = useUserStore()

const currentModulePanels = [
  { id: 'dashboard-overview', label: '平台概览' },
  { id: 'dashboard-audit', label: '审计摘要' }
]

const projectCount = computed(() => organizationStore.currentOrganization?.projects?.length ?? 0)
const applicationCount = computed(() => organizationStore.currentOrganization?.projects?.reduce((total: number, project: any) => total + (project.applications?.length ?? 0), 0) ?? 0)
const policyCount = computed(() => roleStore.policies.length)

watchEffect(() => {
  consoleStore.setPageHeader('仪表盘', '概览当前实例下的核心 IAM 统计和审计摘要。')
})

const summaryTiles = computed(() => [
  { label: '组织', value: organizationStore.organizations.length },
  { label: '项目', value: projectCount.value },
  { label: '应用', value: applicationCount.value },
  { label: '用户', value: userStore.users.length },
  { label: '角色标签', value: roleStore.roles.length },
  { label: '策略', value: policyCount.value }
])

const currentModuleMetrics = computed<Array<{ label: string; value: string; copyable?: boolean; copyValue?: string }>>(() =>
  summaryTiles.value.map((item) => ({ label: item.label, value: String(item.value) }))
)

async function refreshDashboard() {
  await Promise.all([
    organizationStore.loadOrganizations(),
    userStore.loadUsers(),
    roleStore.loadRoles(),
    auditStore.loadAudit()
  ])
}

const recentAuditLogs = computed(() => auditStore.recentAuditLogs)
const moduleRecentChanges = computed(() => auditStore.moduleRecentChanges)
const formatDateTime = consoleStore.formatDateTime
</script>
