<template>
  <section class="audit-page">
    <div id="audit-filter" class="info-card">
      <div class="audit-toolbar">
        <div>
          <div class="section-title">查询条件</div>
          <div class="record-meta mt-2">按时间范围和模块筛选当前组织的审计日志。</div>
        </div>
      </div>
      <div class="audit-filter-grid">
        <div class="audit-filter-field">
          <label class="form-label">开始时间</label>
          <BFormInput v-model="filters.from" type="datetime-local" />
        </div>
        <div class="audit-filter-field">
          <label class="form-label">结束时间</label>
          <BFormInput v-model="filters.to" type="datetime-local" />
        </div>
        <div class="audit-filter-field">
          <label class="form-label">模块</label>
          <BFormSelect v-model="filters.module" :options="moduleOptions" />
        </div>
      </div>
      <div class="audit-preset-row">
        <button
          v-for="item in presetOptions"
          :key="item.value"
          type="button"
          class="audit-preset-chip"
          :class="{ 'audit-preset-chip-active': filters.preset === item.value }"
          @click="applyPreset(item.value)"
        >
          {{ item.label }}
        </button>
      </div>
      <div class="detail-card mt-3">
        <div class="record-meta">当前组织：{{ currentOrganization?.name || currentOrganization?.id || '-' }}</div>
        <div class="record-meta">已匹配日志：{{ filteredAuditLogs.length }}</div>
      </div>
      <div class="audit-action-row">
        <BButton variant="outline-secondary" @click="resetFilters">清空筛选条件</BButton>
        <BButton variant="primary" @click="auditStore.loadAudit">查询</BButton>
      </div>
    </div>

    <div id="audit-list" class="info-card mt-4">
      <div class="section-title">审计日志</div>
      <div v-if="filteredAuditLogs.length" class="audit-log-list">
        <article v-for="item in filteredAuditLogs" :key="item.id" class="audit-log-card">
          <div class="audit-log-topline">
            <div class="audit-log-main">
              <span class="audit-log-module">{{ resolveModuleLabel(item) }}</span>
              <strong>{{ item.eventType }}</strong>
            </div>
            <div class="audit-log-side">
              <span class="audit-log-time">{{ formatDateTime(item.createdAt) }}</span>
              <code>{{ item.result }}</code>
            </div>
          </div>
          <div class="audit-log-grid">
            <div class="audit-log-item">
              <span>Actor</span>
              <strong>{{ item.actorType }} / {{ item.actorId || '-' }}</strong>
            </div>
            <div class="audit-log-item">
              <span>Target</span>
              <strong>{{ item.targetType }} / {{ item.targetId || '-' }}</strong>
            </div>
            <div class="audit-log-item">
              <span>IP</span>
              <strong>{{ formatIpLine(item.ipAddress, item.ipLocation) }}</strong>
            </div>
            <div class="audit-log-item">
              <span>组织</span>
              <strong>{{ item.organizationId || '-' }}</strong>
            </div>
          </div>
        </article>
      </div>
      <div v-else class="detail-card">
        <div class="record-meta">当前筛选条件下没有审计日志。</div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, watchEffect } from 'vue'
import { BButton, BFormInput, BFormSelect } from 'bootstrap-vue-next'
import { useAuditStore } from '@/stores/audit'
import { useConsoleStore } from '@/stores/console'
import { useOrganizationStore } from '@/stores/organization'

const auditStore = useAuditStore()
const consoleStore = useConsoleStore()
const organizationStore = useOrganizationStore()

const presetOptions = [
  { value: '24h', label: '最近 24 小时' },
  { value: '7d', label: '最近 7 天' },
  { value: '30d', label: '最近 30 天' },
  { value: 'all', label: '全部时间' }
] as const

const filters = reactive({
  from: '',
  to: '',
  module: 'all',
  preset: '7d'
})

watchEffect(() => {
  consoleStore.setPageHeader('审计', '查看平台关键事件、登录轨迹与策略变更审计。')
})

const moduleOptions = computed(() => {
  const values = new Set<string>()
  for (const item of auditStore.auditLogs) {
    values.add(resolveModuleValue(item))
  }
  return [
    { value: 'all', text: '全部模块' },
    ...Array.from(values)
      .filter(Boolean)
      .sort((a, b) => a.localeCompare(b))
      .map((value) => ({
        value,
        text: resolveModuleText(value)
      }))
  ]
})

const filteredAuditLogs = computed(() => {
  const fromTime = parseLocalDateTime(filters.from)
  const toTime = parseLocalDateTime(filters.to)
  return auditStore.auditLogs.filter((item) => {
    const createdAt = Date.parse(String(item.createdAt || ''))
    if (Number.isNaN(createdAt)) {
      return false
    }
    if (fromTime !== null && createdAt < fromTime) {
      return false
    }
    if (toTime !== null && createdAt > toTime) {
      return false
    }
    if (filters.module !== 'all' && resolveModuleValue(item) !== filters.module) {
      return false
    }
    return true
  })
})

function formatIpLine(ipAddress?: string, ipLocation?: string) {
  const ip = String(ipAddress || '').trim()
  const location = String(ipLocation || '').trim()
  if (ip && location) return `${ip} (${location})`
  return ip || location || '-'
}

function resolveModuleValue(item: any) {
  const eventType = String(item?.eventType || '').trim()
  if (!eventType) {
    return 'other'
  }
  const [prefix] = eventType.split('.')
  return prefix || 'other'
}

function resolveModuleText(value: string) {
  if (value === 'auth') return '认证'
  if (value === 'token') return '令牌'
  if (value === 'user') return '用户'
  if (value === 'role') return '角色'
  if (value === 'policy') return '策略'
  if (value === 'project') return '项目'
  if (value === 'application') return '应用'
  if (value === 'organization') return '组织'
  if (value === 'external') return '外部身份'
  if (value === 'session') return '会话'
  return value
}

function resolveModuleLabel(item: any) {
  return resolveModuleText(resolveModuleValue(item))
}

function parseLocalDateTime(input: string) {
  const value = String(input || '').trim()
  if (!value) {
    return null
  }
  const timestamp = new Date(value).getTime()
  return Number.isNaN(timestamp) ? null : timestamp
}

function applyPreset(preset: typeof presetOptions[number]['value']) {
  filters.preset = preset
  if (preset === 'all') {
    filters.from = ''
    filters.to = ''
    return
  }
  const now = new Date()
  const start = new Date(now)
  if (preset === '24h') {
    start.setHours(now.getHours() - 24)
  } else if (preset === '7d') {
    start.setDate(now.getDate() - 7)
  } else {
    start.setDate(now.getDate() - 30)
  }
  filters.from = toDateTimeLocalValue(start)
  filters.to = toDateTimeLocalValue(now)
}

function toDateTimeLocalValue(value: Date) {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, '0')
  const day = String(value.getDate()).padStart(2, '0')
  const hour = String(value.getHours()).padStart(2, '0')
  const minute = String(value.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hour}:${minute}`
}

function resetFilters() {
  filters.module = 'all'
  applyPreset('7d')
}

applyPreset('7d')

const currentOrganization = computed(() => organizationStore.currentOrganization)
const formatDateTime = consoleStore.formatDateTime
</script>

<style scoped>
.audit-page {
  width: 100%;
}

.audit-toolbar {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.audit-action-row {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1rem;
}

.audit-filter-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
}

.audit-filter-field {
  display: grid;
}

.audit-preset-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 1rem;
}

.audit-preset-chip {
  border: 1px solid #d0d7de;
  background: #fff;
  color: #57606a;
  border-radius: 999px;
  padding: 0.45rem 0.85rem;
  font-size: 0.88rem;
  font-weight: 600;
}

.audit-preset-chip-active {
  background: #1f6feb;
  border-color: #1f6feb;
  color: #fff;
}

.audit-log-list {
  display: grid;
  gap: 1rem;
}

.audit-log-card {
  border: 1px solid #d8dee4;
  border-radius: 14px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
  padding: 1rem 1.1rem;
}

.audit-log-topline {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.audit-log-main {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.audit-log-module {
  display: inline-flex;
  align-items: center;
  min-height: 1.8rem;
  padding: 0 0.75rem;
  border-radius: 999px;
  background: #eaf2ff;
  color: #1f6feb;
  font-size: 0.82rem;
  font-weight: 700;
}

.audit-log-side {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.audit-log-time {
  color: #656d76;
  font-size: 0.9rem;
}

.audit-log-grid {
  margin-top: 0.9rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.85rem;
}

.audit-log-item {
  border-radius: 10px;
  border: 1px solid #d8dee4;
  background: #fff;
  padding: 0.75rem 0.85rem;
  display: grid;
  gap: 0.28rem;
}

.audit-log-item span {
  color: #656d76;
  font-size: 0.8rem;
}

.audit-log-item strong {
  font-size: 0.92rem;
  word-break: break-word;
}

@media (max-width: 1100px) {
  .audit-toolbar,
  .audit-filter-grid,
  .audit-log-grid {
    grid-template-columns: 1fr;
  }

  .audit-toolbar {
    flex-direction: column;
  }

  .audit-action-row {
    justify-content: stretch;
  }
}
</style>
