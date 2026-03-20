<template>
  <section v-if="roleViewMode === 'list'" class="card-stack">
    <div class="info-card">
      <div class="section-title">当前组织下可用的用户角色</div>
      <div class="d-flex align-items-center justify-content-between gap-3 mb-3 flex-wrap">
        <div class="d-flex align-items-center gap-2 flex-wrap">
          <BButton size="sm" variant="outline-danger" :disabled="selectedUserRoleIds.length === 0" @click="deleteSelectedRolesByType('user', selectedUserRoleIds)">删除角色</BButton>
        </div>
        <BButton size="sm" variant="primary" @click="toggleCreateRoleForm('user')">{{ showCreateUserRoleForm ? '收起添加角色' : '添加角色' }}</BButton>
      </div>
      <div v-if="showCreateUserRoleForm" class="detail-card mb-3">
        <BForm @submit.prevent="submitRoleCreateFromList">
          <BFormInput v-model="roleForm.name" placeholder="role label" class="mb-2" />
          <BFormSelect v-model="roleForm.type" :options="roleTypeOptions" class="mb-2" />
          <BFormInput v-model="roleForm.description" placeholder="description" class="mb-2" />
          <div class="d-flex gap-2">
            <BButton type="submit" variant="primary">创建角色</BButton>
            <BButton type="button" variant="outline-secondary" @click="showCreateRoleForm = false">取消</BButton>
          </div>
        </BForm>
      </div>
      <div class="table-responsive">
        <table class="table align-middle console-list-table mb-0">
          <thead>
            <tr>
              <th class="console-list-check-col">
                <input
                  class="form-check-input console-list-checkbox"
                  type="checkbox"
                  :checked="userAssignableRoles.length > 0 && userAssignableRoles.every((role) => selectedRoleIds.includes(role.id))"
                  @change="toggleRolesByType('user', ($event.target as HTMLInputElement).checked)"
                />
              </th>
              <th>角色 ID</th>
              <th>角色标签</th>
              <th>描述</th>
              <th>策略数</th>
              <th class="text-end">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="role in userAssignableRoles" :key="role.id">
              <td class="console-list-check-col">
                <input class="form-check-input console-list-checkbox" type="checkbox" :checked="selectedRoleIds.includes(role.id)" @change="toggleRoleSelection(role.id, ($event.target as HTMLInputElement).checked)" />
              </td>
              <td class="console-list-id">{{ role.id }}</td>
              <td>{{ role.name || '-' }}</td>
              <td>{{ role.description || '-' }}</td>
              <td>{{ policies.filter((item) => item.roleId === role.id).length }}</td>
              <td class="text-end">
                <div class="d-inline-flex gap-2">
                  <BButton size="sm" variant="outline-primary" @click="selectRole(role)">查看详情</BButton>
                  <BButton size="sm" variant="outline-danger" @click="deleteSingleRole(role.id)">删除</BButton>
                </div>
              </td>
            </tr>
            <tr v-if="userAssignableRoles.length === 0">
              <td colspan="6" class="text-center text-secondary py-4">当前组织下还没有用户角色。</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <div class="info-card">
      <div class="section-title">当前组织下可用的应用角色</div>
      <div class="d-flex align-items-center justify-content-between gap-3 mb-3 flex-wrap">
        <div class="d-flex align-items-center gap-2 flex-wrap">
          <BButton size="sm" variant="outline-danger" :disabled="selectedApplicationRoleIds.length === 0" @click="deleteSelectedRolesByType('application', selectedApplicationRoleIds)">删除角色</BButton>
        </div>
        <BButton size="sm" variant="primary" @click="toggleCreateRoleForm('application')">{{ showCreateApplicationRoleForm ? '收起添加角色' : '添加角色' }}</BButton>
      </div>
      <div v-if="showCreateApplicationRoleForm" class="detail-card mb-3">
        <BForm @submit.prevent="submitRoleCreateFromList">
          <BFormInput v-model="roleForm.name" placeholder="role label" class="mb-2" />
          <BFormSelect v-model="roleForm.type" :options="roleTypeOptions" class="mb-2" />
          <BFormInput v-model="roleForm.description" placeholder="description" class="mb-2" />
          <div class="d-flex gap-2">
            <BButton type="submit" variant="primary">创建角色</BButton>
            <BButton type="button" variant="outline-secondary" @click="showCreateRoleForm = false">取消</BButton>
          </div>
        </BForm>
      </div>
      <div class="table-responsive">
        <table class="table align-middle console-list-table mb-0">
          <thead>
            <tr>
              <th class="console-list-check-col">
                <input
                  class="form-check-input console-list-checkbox"
                  type="checkbox"
                  :checked="applicationAssignableRoles.length > 0 && applicationAssignableRoles.every((role) => selectedRoleIds.includes(role.id))"
                  @change="toggleRolesByType('application', ($event.target as HTMLInputElement).checked)"
                />
              </th>
              <th>角色 ID</th>
              <th>角色标签</th>
              <th>描述</th>
              <th>策略数</th>
              <th class="text-end">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="role in applicationAssignableRoles" :key="role.id">
              <td class="console-list-check-col">
                <input class="form-check-input console-list-checkbox" type="checkbox" :checked="selectedRoleIds.includes(role.id)" @change="toggleRoleSelection(role.id, ($event.target as HTMLInputElement).checked)" />
              </td>
              <td class="console-list-id">{{ role.id }}</td>
              <td>{{ role.name || '-' }}</td>
              <td>{{ role.description || '-' }}</td>
              <td>{{ policies.filter((item) => item.roleId === role.id).length }}</td>
              <td class="text-end">
                <div class="d-inline-flex gap-2">
                  <BButton size="sm" variant="outline-primary" @click="selectRole(role)">查看详情</BButton>
                  <BButton size="sm" variant="outline-danger" @click="deleteSingleRole(role.id)">删除</BButton>
                </div>
              </td>
            </tr>
            <tr v-if="applicationAssignableRoles.length === 0">
              <td colspan="6" class="text-center text-secondary py-4">当前组织下还没有应用角色。</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>

  <RoleDetail
    v-else
    :roles="roles"
    :selected-role-id="selectedRoleId"
    :policies="policies"
    :selected-role="selectedRole"
    :selected-role-policies="selectedRolePolicies"
    :role-form="roleForm"
    :role-type-options="roleTypeOptions"
    :policy-form="policyForm"
    :policy-check-form="policyCheckForm"
    :decision-result="decisionResult"
    :module-recent-changes="moduleRecentChanges"
    :format-date-time="console.formatDateTime"
    @back="backToRoleList"
    @run-module-action="runModuleAction"
    @copy-metric="copyMetricValue"
    @scroll-to-panel="scrollToPanel"
    @select-role="selectRole"
    @update-role="updateRole"
    @save-policy="savePolicy"
    @evaluate-policy-check="evaluatePolicyCheck"
    @edit-policy="editPolicy"
    @delete-policy="deletePolicy"
    @reset-policy-form="resetPolicyForm"
  />
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BButton, BForm, BFormInput, BFormSelect } from 'bootstrap-vue-next'
import { useToast } from '@shared/composables/toast'
import { checkPolicy as apiCheckPolicy, createPolicy as apiCreatePolicy, deletePolicy as apiDeletePolicy, queryPolicies as apiQueryPolicies, updatePolicy as apiUpdatePolicy } from '../api/manage/policy'
import { createRole as apiCreateRole, deleteRoles as apiDeleteRoles, queryRoles as apiQueryRoles, updateRole as apiUpdateRole } from '../api/manage/role'
import RoleDetail from '../components/RoleDetail.vue'
import { useConsoleLayout } from '../composables/useConsoleLayout'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const console = useConsoleLayout()

const roleTypeOptions = [
  { value: 'user', text: '用户角色' },
  { value: 'application', text: '应用角色' }
]

const roles = ref<any[]>([])
const policies = ref<any[]>([])
const decisionResult = ref<unknown>(null)
const selectedRoleId = ref('')
const roleViewMode = ref<'list' | 'detail'>('list')
const selectedRoleIds = ref<string[]>([])
const showCreateRoleForm = ref(false)
const createRoleFormType = ref<'user' | 'application'>('user')

const roleForm = reactive({ organizationId: '', name: '', type: 'user', description: '' })
const policyForm = reactive({ id: '', organizationId: '', roleId: '', name: '', effect: 'allow', priority: 10, apiRulesText: '[\n  {\n    "method": "POST",\n    "path": "/api/manage/v1/example/query"\n  }\n]' })
const policyCheckForm = reactive({ subjectType: 'application', subjectId: '', method: 'POST', path: '/api/manage/v1/organization/query' })

const selectedRole = computed(() => roles.value.find((item: any) => item.id === selectedRoleId.value) || roles.value[0])
const selectedRolePolicies = computed(() => policies.value.filter((item: any) => item.roleId === selectedRole.value?.id))
const applicationAssignableRoles = computed(() => roles.value.filter((item: any) => item.type === 'application'))
const userAssignableRoles = computed(() => roles.value.filter((item: any) => item.type === 'user'))
const moduleRecentChanges = computed(() => console.recentAuditLogs.slice(0, 6))
const selectedUserRoleIds = computed(() => selectedRoleIds.value.filter((id) => userAssignableRoles.value.some((role) => role.id === id)))
const selectedApplicationRoleIds = computed(() => selectedRoleIds.value.filter((id) => applicationAssignableRoles.value.some((role) => role.id === id)))
const showCreateUserRoleForm = computed(() => showCreateRoleForm.value && createRoleFormType.value === 'user')
const showCreateApplicationRoleForm = computed(() => showCreateRoleForm.value && createRoleFormType.value === 'application')

watch(
  () => [console.currentOrganizationId, route.name, route.params.roleId],
  async ([organizationId, routeName, routeRoleId]) => {
    const nextOrganizationId = typeof organizationId === 'string' ? organizationId : ''
    if (!nextOrganizationId) {
      roles.value = []
      policies.value = []
      selectedRoleId.value = ''
      roleViewMode.value = 'list'
      return
    }
    roleForm.organizationId = nextOrganizationId
    policyForm.organizationId = nextOrganizationId
    roleViewMode.value = routeName === 'console-role-detail' ? 'detail' : 'list'
    await Promise.all([loadRoles(), loadPolicies()])
    if (typeof routeRoleId === 'string' && routeRoleId) {
      selectedRoleId.value = routeRoleId
    }
    if (!selectedRoleId.value && roles.value.length) {
      selectedRoleId.value = roles.value[0].id
    }
    if (selectedRole.value) {
      syncRoleForms(selectedRole.value)
    }
  },
  { immediate: true }
)

watch(
  () => [userAssignableRoles.value, applicationAssignableRoles.value],
  () => {
    const roleIds = new Set([...userAssignableRoles.value, ...applicationAssignableRoles.value].map((item: any) => item.id))
    selectedRoleIds.value = selectedRoleIds.value.filter((id) => roleIds.has(id))
  },
  { immediate: true, deep: true }
)

function syncRoleForms(role?: any) {
  if (!role) {
    return
  }
  roleForm.organizationId = role.organizationId ?? console.currentOrganizationId
  roleForm.name = role.name ?? ''
  roleForm.type = role.type ?? 'user'
  roleForm.description = role.description ?? ''
  policyForm.roleId = role.id ?? ''
  policyCheckForm.subjectType = role.type === 'application' ? 'application' : 'user'
}

function toggleCreateRoleForm(type: 'user' | 'application') {
  roleForm.type = type
  if (showCreateRoleForm.value && createRoleFormType.value === type) {
    showCreateRoleForm.value = false
    return
  }
  createRoleFormType.value = type
  showCreateRoleForm.value = true
}

function toggleRolesByType(type: 'user' | 'application', checked: boolean) {
  const targetIds = (type === 'user' ? userAssignableRoles.value : applicationAssignableRoles.value).map((item: any) => item.id)
  if (checked) {
    selectedRoleIds.value = Array.from(new Set([...selectedRoleIds.value, ...targetIds]))
    return
  }
  selectedRoleIds.value = selectedRoleIds.value.filter((id) => !targetIds.includes(id))
}

function toggleRoleSelection(roleId: string, checked: boolean) {
  if (checked) {
    if (!selectedRoleIds.value.includes(roleId)) {
      selectedRoleIds.value = [...selectedRoleIds.value, roleId]
    }
    return
  }
  selectedRoleIds.value = selectedRoleIds.value.filter((id) => id !== roleId)
}

async function loadRoles() {
  const response = await apiQueryRoles({ organizationId: console.currentOrganizationId })
  roles.value = response.items
  await console.loadRoles()
  if (!roles.value.some((item: any) => item.id === selectedRoleId.value)) {
    selectedRoleId.value = roles.value[0]?.id ?? ''
  }
  if (selectedRole.value) {
    syncRoleForms(selectedRole.value)
  }
}

async function loadPolicies() {
  const response = await apiQueryPolicies({ organizationId: console.currentOrganizationId })
  policies.value = response.items
  console.policyCount = policies.value.length
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
  if (!value || value === '-') {
    return
  }
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

function selectRole(role: any) {
  roleViewMode.value = 'detail'
  selectedRoleId.value = role.id ?? ''
  syncRoleForms(role)
  void router.push({
    name: 'console-role-detail',
    params: {
      organizationId: console.currentOrganizationId || console.currentOrganization?.id || '',
      roleId: role.id ?? ''
    }
  })
}

function backToRoleList() {
  roleViewMode.value = 'list'
  void router.push({
    name: 'console-role-list',
    params: {
      organizationId: console.currentOrganizationId || console.currentOrganization?.id || ''
    }
  })
}

async function createRole() {
  await withFeedback(async () => {
    await apiCreateRole(roleForm)
    await Promise.all([loadRoles(), loadPolicies()])
  })
}

async function submitRoleCreateFromList() {
  await createRole()
  showCreateRoleForm.value = false
}

async function deleteSelectedRolesByType(_type: 'user' | 'application', roleIds: string[]) {
  if (!roleIds.length) {
    return
  }
  await withFeedback(async () => {
    await apiDeleteRoles({ roleIds })
    await loadRoles()
  })
}

async function deleteSingleRole(roleId: string) {
  await withFeedback(async () => {
    await apiDeleteRoles({ roleId })
    await loadRoles()
  })
}

async function updateRole() {
  if (!selectedRole.value?.id) {
    return
  }
  await withFeedback(async () => {
    await apiUpdateRole({
      id: selectedRole.value.id,
      name: roleForm.name,
      type: roleForm.type,
      description: roleForm.description
    })
    await Promise.all([loadRoles(), loadPolicies()])
  })
}

async function savePolicy() {
  if (!selectedRole.value?.id) {
    throw new Error('请先选择角色')
  }
  const payload = {
    id: policyForm.id || undefined,
    organizationId: console.currentOrganizationId || console.currentOrganization?.id || '',
    roleId: selectedRole.value.id,
    name: policyForm.name.trim(),
    effect: policyForm.effect,
    priority: Number(policyForm.priority),
    apiRules: JSON.parse(policyForm.apiRulesText || '[]')
  }
  await withFeedback(async () => {
    if (policyForm.id) {
      await apiUpdatePolicy(payload)
    } else {
      await apiCreatePolicy(payload)
    }
    resetPolicyForm()
    await loadPolicies()
  })
}

function editPolicy(policy: any) {
  policyForm.id = policy.id ?? ''
  policyForm.organizationId = policy.organizationId ?? console.currentOrganizationId
  policyForm.roleId = policy.roleId ?? selectedRole.value?.id ?? ''
  policyForm.name = policy.name ?? ''
  policyForm.effect = policy.effect ?? 'allow'
  policyForm.priority = Number(policy.priority ?? 10)
  policyForm.apiRulesText = JSON.stringify(policy.apiRules ?? [], null, 2)
}

function resetPolicyForm() {
  policyForm.id = ''
  policyForm.organizationId = console.currentOrganizationId || console.currentOrganization?.id || ''
  policyForm.roleId = selectedRole.value?.id ?? ''
  policyForm.name = ''
  policyForm.effect = 'allow'
  policyForm.priority = 10
  policyForm.apiRulesText = '[\n  {\n    "method": "POST",\n    "path": "/api/manage/v1/example/query"\n  }\n]'
}

async function deletePolicy(policyId: string) {
  await withFeedback(async () => {
    await apiDeletePolicy(policyId)
    if (policyForm.id === policyId) {
      resetPolicyForm()
    }
    await loadPolicies()
  })
}

async function evaluatePolicyCheck() {
  await withFeedback(async () => {
    decisionResult.value = await apiCheckPolicy({
      subjectType: policyCheckForm.subjectType,
      subjectId: policyCheckForm.subjectId.trim(),
      method: policyCheckForm.method.trim() || 'POST',
      path: policyCheckForm.path.trim()
    })
  })
}

async function runModuleAction() {
  await Promise.all([loadRoles(), loadPolicies()])
}
</script>
