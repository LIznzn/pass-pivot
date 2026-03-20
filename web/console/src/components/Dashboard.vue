<template>
  <section class="console-module-shell">
    <div class="console-module-summary-card">
      <div class="console-module-hero">
        <div>
          <div class="console-module-eyebrow">仪表盘</div>
          <h2 class="console-module-title">实例概览</h2>
          <p class="console-module-subtitle">概览当前实例下的核心 IAM 统计和审计摘要。</p>
        </div>
        <BButton variant="primary" @click="console.runModuleAction">刷新概览</BButton>
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
              @click="console.copyMetricValue(item.copyValue || item.value)"
            >
              <i class="bi bi-copy" aria-hidden="true"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="console-module-workspace">
      <aside class="console-module-sidebar">
        <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="console.scrollToPanel(item.id)">{{ item.label }}</button>
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
          <BButton size="sm" variant="outline-primary" class="mb-3" @click="console.loadAudit">刷新</BButton>
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
      <RightSide :items="moduleRecentChanges" :format-date-time="formatDateTime" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { BButton } from 'bootstrap-vue-next'
import RightSide from '../layout/RightSide.vue'
import { useConsoleLayout } from '../composables/useConsoleLayout'

const console = useConsoleLayout()

const currentModulePanels = [
  { id: 'dashboard-overview', label: '平台概览' },
  { id: 'dashboard-audit', label: '审计摘要' }
]

const summaryTiles = computed(() => [
  { label: '组织', value: console.summary.organizationCount },
  { label: '项目', value: console.summary.projectCount },
  { label: '应用', value: console.summary.applicationCount },
  { label: '用户', value: console.summary.userCount },
  { label: '角色标签', value: console.summary.roleCount },
  { label: '策略', value: console.summary.policyCount }
])

const currentModuleMetrics = computed<Array<{ label: string; value: string; copyable?: boolean; copyValue?: string }>>(() =>
  summaryTiles.value.map((item) => ({ label: item.label, value: String(item.value) }))
)

const recentAuditLogs = computed(() => console.recentAuditLogs)
const moduleRecentChanges = computed(() => console.moduleRecentChanges)
const formatDateTime = console.formatDateTime
</script>
