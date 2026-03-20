<template>
  <section class="console-module-shell">
    <div class="console-module-summary-card">
      <div class="console-module-hero">
        <div>
          <div class="console-module-eyebrow">组织</div>
          <h2 class="console-module-title">{{ currentOrganization?.name || '组织' }}</h2>
          <p class="console-module-subtitle">{{ currentOrganization?.name ? `当前组织 ${currentOrganization.name} 的基础配置、登录方式和接入边界。` : '当前组织的基础配置、登录方式和接入边界。' }}</p>
        </div>
        <div class="console-action-menu" role="group" aria-label="组织操作">
          <button type="button" class="btn btn-primary console-action-menu-toggle">
            操作
            <i class="bi bi-chevron-down" aria-hidden="true"></i>
          </button>
          <div class="console-action-menu-list">
            <button type="button" class="console-action-menu-item" @click="console.showOrganizationDisableNotice">停用</button>
            <button type="button" class="console-action-menu-item console-action-menu-item-danger" @click="console.showOrganizationDeleteNotice">删除</button>
          </div>
        </div>
      </div>
      <div class="console-module-metrics console-module-metrics-inline">
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
        <div id="organization-metadata" class="info-card">
          <div class="section-title">维护元信息</div>
          <div class="record-meta mb-3">这些元信息会作为可用变量，用于自定义登录页等组织级展示场景。</div>
          <div v-if="currentOrganization" class="detail-card">
            <div class="metadata-table-wrap">
              <table class="table table-sm align-middle mb-0">
                <thead>
                  <tr>
                    <th class="metadata-col-key">键</th>
                    <th>值</th>
                    <th class="metadata-col-action"></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(item, index) in organizationMetadataRows" :key="item.id">
                    <td>
                      <BFormInput v-model="item.key" placeholder="例如 login_title" />
                    </td>
                    <td>
                      <BFormInput v-model="item.value" placeholder="例如 PPVT 控制台" />
                    </td>
                    <td class="text-end">
                      <BButton size="sm" variant="outline-danger" @click="removeOrganizationMetadataRow(index)">删除</BButton>
                    </td>
                  </tr>
                  <tr v-if="organizationMetadataRows.length === 0">
                    <td colspan="3" class="text-center text-secondary py-4">当前还没有元信息，新增后可作为组织级变量使用。</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="d-flex gap-2 mt-3">
              <BButton variant="outline-secondary" @click="addOrganizationMetadataRow">新增条目</BButton>
              <BButton variant="primary" @click="console.saveOrganizationMetadata(organizationMetadataRows)">保存元信息</BButton>
            </div>
          </div>
        </div>
      </div>
      <RightSide :items="moduleRecentChanges" :format-date-time="formatDateTime" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { BButton, BFormInput } from 'bootstrap-vue-next'
import RightSide from '../layout/RightSide.vue'
import { useConsoleLayout } from '../composables/useConsoleLayout'

const console = useConsoleLayout()

const organizationMetadataRows = ref<Array<{ id: string; key: string; value: string }>>([])

const currentModulePanels = [
  { id: 'organization-metadata', label: '维护元信息' }
]

const currentModuleMetrics = computed(() => {
  const projectCount = console.currentOrganization?.projects?.length ?? 0
  const applicationCount = (console.currentOrganization?.projects ?? []).reduce((count: number, project: any) => count + (project.applications?.length ?? 0), 0)
  return [
    { label: '组织 ID', value: console.currentOrganization?.id || '-', copyable: Boolean(console.currentOrganization?.id), copyValue: console.currentOrganization?.id || '' },
    { label: '项目数', value: String(projectCount) },
    { label: '应用数', value: String(applicationCount) },
    { label: '用户数', value: String(console.users.length) },
    { label: '角色数', value: String(console.roles.length) },
    { label: '创建时间', value: console.formatDateTime(console.currentOrganization?.createdAt) },
    { label: '更新时间', value: console.formatDateTime(console.currentOrganization?.updatedAt) }
  ]
})

watch(
  () => console.currentOrganization?.metadata,
  (metadata) => {
    const normalized = (!metadata || typeof metadata !== 'object' || Array.isArray(metadata))
      ? {}
      : Object.fromEntries(Object.entries(metadata as Record<string, unknown>).map(([key, value]) => [key, String(value ?? '')]))
    organizationMetadataRows.value = Object.entries(normalized).map(([key, value]) => ({
      id: createLocalRowId(),
      key,
      value
    }))
  },
  { immediate: true, deep: true }
)

function createLocalRowId() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function addOrganizationMetadataRow() {
  organizationMetadataRows.value.push({
    id: createLocalRowId(),
    key: '',
    value: ''
  })
}

function removeOrganizationMetadataRow(index: number) {
  organizationMetadataRows.value.splice(index, 1)
}

const currentOrganization = computed(() => console.currentOrganization)
const moduleRecentChanges = computed(() => console.moduleRecentChanges)
const formatDateTime = console.formatDateTime
</script>
