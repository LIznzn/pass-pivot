<template>
  <section class="console-module-shell">
    <div class="console-module-summary-card">
      <div class="console-module-hero">
        <div class="console-module-hero-copy">
          <button type="button" class="console-back-button" @click="roleConsole.backToRoleList()" aria-label="返回角色列表">
            <i class="bi bi-arrow-left console-back-button-icon" aria-hidden="true"></i>
          </button>
          <div>
            <div class="console-module-eyebrow">角色</div>
            <h2 class="console-module-title">{{ selectedRole?.name || '角色' }}</h2>
            <p class="console-module-subtitle">{{ selectedRole?.name ? '从角色列表选择条目后，在详情区维护角色元信息、策略列表与 Policy Check。' : '维护角色标签、策略规则与 Policy Check。' }}</p>
          </div>
        </div>
        <BButton variant="primary" @click="roleConsole.runModuleAction()">刷新角色</BButton>
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
        <div id="role-list" class="info-card">
          <div class="section-title">角色标签列表</div>
          <div class="record-meta mb-3">角色是策略组。用户只能分配用户角色，应用只能分配应用角色。</div>
          <div class="record-list">
            <div v-for="role in roles" :key="role.id" class="record-card" :class="{ 'record-card-active': role.id === selectedRoleId }">
              <div class="record-head">
                <strong>{{ role.name }}</strong>
                <code>{{ role.type === 'application' ? '应用角色' : '用户角色' }}</code>
              </div>
              <div class="record-meta">{{ role.description || 'no description' }}</div>
              <div class="record-meta">Role ID: {{ role.id }}</div>
              <div class="record-meta">策略数：{{ policies.filter((item) => item.roleId === role.id).length }}</div>
              <div class="record-actions">
                <BButton size="sm" variant="outline-primary" @click="roleConsole.selectRole(role)">查看详情</BButton>
              </div>
            </div>
          </div>
        </div>
        <div id="role-detail" class="info-card">
          <div class="section-title">角色详情</div>
          <div class="record-meta mb-3">当前详情区只展示一个角色，点击左侧列表即可切换。</div>
          <div v-if="selectedRole" class="detail-card mb-3">
            <div class="record-meta">角色 ID：{{ selectedRole.id }}</div>
            <div class="record-meta">角色标签：{{ selectedRole.name || '-' }}</div>
            <div class="record-meta">角色类型：{{ selectedRole.type === 'application' ? '应用角色' : '用户角色' }}</div>
            <div class="record-meta">描述：{{ selectedRole.description || '-' }}</div>
            <div class="record-meta">策略数：{{ selectedRolePolicies.length }}</div>
            <div class="record-meta">创建时间：{{ formatDateTime(selectedRole.createdAt) }}</div>
            <div class="record-meta">更新时间：{{ formatDateTime(selectedRole.updatedAt) }}</div>
          </div>
          <BForm v-if="selectedRole" @submit.prevent="updateRole()">
            <BFormInput v-model="roleForm.name" placeholder="role label" class="mb-2" />
            <BFormSelect v-model="roleForm.type" :options="roleTypeOptions" class="mb-2" />
            <BFormInput v-model="roleForm.description" placeholder="description" class="mb-2" />
            <BButton type="submit" variant="outline-primary">保存角色</BButton>
          </BForm>
        </div>
        <div id="policy-list" class="info-card">
          <div class="section-title">策略列表</div>
          <div class="record-list">
            <div v-for="policy in selectedRolePolicies" :key="policy.id" class="record-card">
              <div class="record-head">
                <strong>{{ policy.name }}</strong>
                <code>{{ policy.effect }} · {{ policy.priority }}</code>
              </div>
              <div class="record-meta">Policy ID：{{ policy.id }}</div>
              <div class="record-meta">API Rules：{{ formatPolicyRules(policy.apiRules) }}</div>
              <div class="record-actions">
                <BButton size="sm" variant="outline-primary" @click="roleStore.editPolicy(policy)">编辑</BButton>
                <BButton size="sm" variant="outline-danger" @click="deletePolicy(policy.id)">删除</BButton>
              </div>
            </div>
            <div v-if="selectedRolePolicies.length === 0" class="detail-card">
              <div class="record-meta">当前角色还没有策略。</div>
            </div>
          </div>
        </div>
        <div id="policy-editor" class="info-card">
          <div class="section-title">策略编辑</div>
          <div class="record-meta mb-3">策略直接挂在角色下，`apiRules.path` 支持 `keyMatch2`。</div>
          <BForm @submit.prevent="savePolicy()">
            <BFormInput v-model="policyForm.name" placeholder="policy name" class="mb-2" />
            <div class="row g-2 mb-2">
              <div class="col-md-4">
                <BFormSelect v-model="policyForm.effect" :options="effectOptions" />
              </div>
              <div class="col-md-4">
                <BFormInput v-model="policyForm.priority" type="number" placeholder="priority" />
              </div>
            </div>
            <textarea v-model="policyForm.apiRulesText" class="form-control mb-2" rows="8" />
            <div class="d-flex gap-2">
              <BButton type="submit" variant="primary">{{ policyForm.id ? '保存策略' : '创建策略' }}</BButton>
              <BButton type="button" variant="outline-secondary" @click="roleStore.resetPolicyForm()">重置</BButton>
            </div>
          </BForm>
        </div>
        <div id="role-decision" class="info-card">
          <div class="section-title">Policy Check</div>
          <BForm @submit.prevent="roleConsole.evaluatePolicyCheck()">
            <BFormSelect v-model="policyCheckForm.subjectType" :options="subjectTypeOptions" class="mb-2" />
            <BFormInput v-model="policyCheckForm.subjectId" placeholder="subjectId" class="mb-2" />
            <BFormInput v-model="policyCheckForm.method" placeholder="POST" class="mb-2" />
            <BFormInput v-model="policyCheckForm.path" placeholder="/api/manage/v1/organization/query" class="mb-2" />
            <BButton type="submit" variant="primary">检查</BButton>
          </BForm>
          <pre class="json-block mt-3">{{ JSON.stringify(decisionResult, null, 2) }}</pre>
        </div>
      </div>
      <RightSide :items="moduleRecentChanges" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
import { BButton, BForm, BFormInput, BFormSelect } from 'bootstrap-vue-next'
import RightSide from '@/layout/RightSide.vue'
import { useAuditStore } from '@/stores/audit'
import { useConsoleStore } from '@/stores/console'
import { roleConsoleContextKey } from '@/components/Role.vue'

const effectOptions = [
  { value: 'allow', text: 'allow' },
  { value: 'deny', text: 'deny' }
]

const subjectTypeOptions = [
  { value: 'application', text: 'application' },
  { value: 'user', text: 'user' }
]

const roleConsole = inject(roleConsoleContextKey)
if (!roleConsole) {
  throw new Error('missing role console context')
}

const auditStore = useAuditStore()
const consoleStore = useConsoleStore()
const roleStore = roleConsole.roleStore
const moduleRecentChanges = computed(() => auditStore.moduleRecentChanges)
const formatDateTime = consoleStore.formatDateTime
const roles = computed(() => roleConsole.roles.value)
const selectedRoleId = computed(() => roleStore.selectedRoleId)
const policies = computed(() => roleConsole.policies.value)
const selectedRole = computed(() => roleConsole.selectedRole.value)
const selectedRolePolicies = computed(() => roleConsole.selectedRolePolicies.value)
const roleForm = roleStore.roleForm
const roleTypeOptions = roleConsole.roleTypeOptions
const policyForm = roleStore.policyForm
const policyCheckForm = roleStore.policyCheckForm
const decisionResult = computed(() => roleConsole.decisionResult.value)

const currentModulePanels = [
  { id: 'role-list', label: '角色列表' },
  { id: 'role-detail', label: '角色详情' },
  { id: 'policy-list', label: '策略列表' },
  { id: 'policy-editor', label: '策略编辑' },
  { id: 'role-decision', label: 'Policy Check' }
]

const currentModuleMetrics = computed<Array<{ label: string; value: string; copyable?: boolean; copyValue?: string }>>(() => [
  { label: '角色 ID', value: selectedRole.value?.id || '-' },
  { label: '角色类型', value: selectedRole.value?.type || '-' },
  { label: '角色数', value: String(roles.value.length) },
  { label: '关联策略', value: String(selectedRolePolicies.value.length) },
  { label: '策略总数', value: String(policies.value.length) },
  { label: '最近变更', value: formatDateTime(selectedRole.value?.updatedAt) }
])

function formatPolicyRules(rules?: Array<{ method?: string; path?: string }>) {
  if (!Array.isArray(rules) || rules.length === 0) {
    return '[]'
  }
  return rules.map((rule) => `${rule.method || '*'} ${rule.path || '*'}`).join(', ')
}

function updateRole() {
  void roleConsole!.runWithFeedback(() => roleStore.updateRole())
}

function savePolicy() {
  void roleConsole!.runWithFeedback(() => roleStore.savePolicy())
}

function deletePolicy(policyId: string) {
  void roleConsole!.runWithFeedback(() => roleStore.deletePolicy(policyId))
}
</script>
