<template>
  <section class="console-module-shell">
    <div class="console-module-summary-card">
      <div class="console-module-hero">
        <div>
          <div class="console-module-eyebrow">审计</div>
          <h2 class="console-module-title">{{ currentOrganization?.name || '审计' }}</h2>
          <p class="console-module-subtitle">查看平台关键事件、登录轨迹与策略变更审计。</p>
        </div>
        <BButton variant="primary" @click="console.runModuleAction">刷新审计</BButton>
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
        <div class="info-card">
          <div class="section-title">模块基础信息</div>
          <div class="detail-card">
            <div class="record-meta">当前组织：{{ currentOrganization?.name || currentOrganization?.id || '-' }}</div>
            <div class="record-meta">审计事件数量：{{ auditLogs.length }}</div>
            <div class="record-meta">这里聚合登录、失败、令牌签发/吊销、策略变更、发现导入、UKID 重置等关键事件。</div>
          </div>
        </div>
        <div id="audit-list" class="info-card">
          <div class="section-title">审计日志</div>
          <BButton size="sm" variant="outline-primary" class="mb-3" @click="console.loadAudit">刷新</BButton>
          <div class="record-list">
            <div v-for="item in auditLogs" :key="item.id" class="record-card">
              <div class="record-head">
                <strong>{{ item.eventType }}</strong>
                <code>{{ item.result }}</code>
              </div>
              <div class="record-meta">Actor: {{ item.actorType }} / {{ item.actorId || '-' }}</div>
              <div class="record-meta">Target: {{ item.targetType }} / {{ item.targetId || '-' }}</div>
              <div class="record-meta">组织: {{ item.organizationId || '-' }}</div>
              <div class="record-meta">IP: {{ formatIpLine(item.ipAddress, item.ipLocation) }}</div>
              <div class="record-meta">时间: {{ formatDateTime(item.createdAt) }}</div>
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
  { id: 'audit-list', label: '审计日志' }
]

const currentModuleMetrics = computed<Array<{ label: string; value: string; copyable?: boolean; copyValue?: string }>>(() => [
  { label: '组织 ID', value: console.currentOrganization?.id || '-' },
  { label: '创建时间', value: console.formatDateTime(console.currentOrganization?.createdAt) },
  { label: '审计数', value: String(console.auditLogs.length) },
  { label: '最近登录用户', value: console.currentLoginUserLabel || '-' }
])

function formatIpLine(ipAddress?: string, ipLocation?: string) {
  const ip = String(ipAddress || '').trim()
  const location = String(ipLocation || '').trim()
  if (ip && location) return `${ip} (${location})`
  return ip || location || '-'
}

const currentOrganization = computed(() => console.currentOrganization)
const auditLogs = computed(() => console.auditLogs)
const moduleRecentChanges = computed(() => console.moduleRecentChanges)
const currentLoginUserLabel = computed(() => console.currentLoginUserLabel)
const formatDateTime = console.formatDateTime
</script>
