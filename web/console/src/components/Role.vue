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
    :roles="roleStore.roles"
    :selected-role-id="roleStore.selectedRoleId"
    :policies="roleStore.policies"
    :selected-role="selectedRole"
    :selected-role-policies="selectedRolePolicies"
    :role-form="roleStore.roleForm"
    :role-type-options="roleTypeOptions"
    :policy-form="roleStore.policyForm"
    :policy-check-form="roleStore.policyCheckForm"
    :decision-result="decisionResult"
    @back="backToRoleList"
    @run-module-action="runModuleAction"
    @select-role="selectRole"
    @update-role="runWithFeedback(() => roleStore.updateRole())"
    @save-policy="runWithFeedback(() => roleStore.savePolicy())"
    @evaluate-policy-check="evaluatePolicyCheck"
    @edit-policy="roleStore.editPolicy"
    @delete-policy="runWithFeedback(() => roleStore.deletePolicy($event))"
    @reset-policy-form="roleStore.resetPolicyForm"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BButton, BForm, BFormInput, BFormSelect, useToast } from 'bootstrap-vue-next'
import RoleDetail from '@/components/RoleDetail.vue'
import { useConsoleStore } from '@/stores/console'
import { useOrganizationStore } from '@/stores/organization'
import { useRoleStore } from '@/stores/role'
import { notifyToast } from '@shared/utils/notify'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const console = useConsoleStore()
const organizationStore = useOrganizationStore()
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
    source: 'console/Role'
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

const roleTypeOptions = [
  { value: 'user', text: '用户角色' },
  { value: 'application', text: '应用角色' }
]

const roleViewMode = ref<'list' | 'detail'>('list')
const selectedRoleIds = ref<string[]>([])
const showCreateRoleForm = ref(false)
const createRoleFormType = ref<'user' | 'application'>('user')
const decisionResult = ref<unknown>(null)

const applicationAssignableRoles = computed(() => roleStore.roles.filter((item: any) => item.type === 'application'))
const userAssignableRoles = computed(() => roleStore.roles.filter((item: any) => item.type === 'user'))
const selectedUserRoleIds = computed(() => selectedRoleIds.value.filter((id) => userAssignableRoles.value.some((role) => role.id === id)))
const selectedApplicationRoleIds = computed(() => selectedRoleIds.value.filter((id) => applicationAssignableRoles.value.some((role) => role.id === id)))
const showCreateUserRoleForm = computed(() => showCreateRoleForm.value && createRoleFormType.value === 'user')
const showCreateApplicationRoleForm = computed(() => showCreateRoleForm.value && createRoleFormType.value === 'application')
const roles = computed(() => roleStore.roles)
const policies = computed(() => roleStore.policies)
const selectedRole = computed(() => roleStore.roles.find((item: any) => item.id === roleStore.selectedRoleId) || roleStore.roles[0])
const selectedRolePolicies = computed(() => roleStore.policies.filter((item: any) => item.roleId === selectedRole.value?.id))
const roleForm = roleStore.roleForm

watchEffect(() => {
  if (roleViewMode.value === 'detail') {
    console.setPageHeader('', '')
    return
  }
  console.setPageHeader('角色', '维护角色标签、策略规则与 Policy Check。')
})

watch(
  () => [console.currentOrganizationId, route.name, route.params.roleId],
  async ([organizationId, routeName, routeRoleId]) => {
    const nextOrganizationId = typeof organizationId === 'string' ? organizationId : ''
    if (!nextOrganizationId) {
      roleStore.clearRoleState()
      decisionResult.value = null
      roleViewMode.value = 'list'
      return
    }
    roleStore.roleForm.organizationId = nextOrganizationId
    roleStore.policyForm.organizationId = nextOrganizationId
    roleViewMode.value = routeName === 'console-role-detail' ? 'detail' : 'list'
    await Promise.all([roleStore.loadRoles(), roleStore.loadPolicies()])
    if (typeof routeRoleId === 'string' && routeRoleId) {
      roleStore.setSelectedRoleId(routeRoleId)
    }
    if (!roleStore.selectedRoleId && roleStore.roles.length) {
      roleStore.setSelectedRoleId(roleStore.roles[0].id)
    }
    if (selectedRole.value) {
      roleStore.syncRoleForms(selectedRole.value)
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

function toggleCreateRoleForm(type: 'user' | 'application') {
  roleStore.roleForm.type = type
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

async function runWithFeedback(fn: () => Promise<unknown>, successMessage = '操作成功') {
  try {
    await fn()
    showToast(successMessage, 'success', {
      source: 'console/Role.submitRoleMutation',
      trigger: 'submitRoleMutation'
    })
  } catch (error) {
    showToast(String(error), 'danger', {
      source: 'console/Role.submitRoleMutation',
      trigger: 'submitRoleMutation',
      error
    })
  }
}

function selectRole(role: any) {
  roleViewMode.value = 'detail'
  roleStore.setSelectedRoleId(role.id ?? '')
  void router.push({
    name: 'console-role-detail',
    params: {
      organizationId: console.currentOrganizationId || organizationStore.currentOrganization?.id || '',
      roleId: role.id ?? ''
    }
  })
}

function backToRoleList() {
  roleViewMode.value = 'list'
  void router.push({
    name: 'console-role-list',
    params: {
      organizationId: console.currentOrganizationId || organizationStore.currentOrganization?.id || ''
    }
  })
}

async function submitRoleCreateFromList() {
  await runWithFeedback(() => roleStore.createRole())
  showCreateRoleForm.value = false
}

async function deleteSelectedRolesByType(_type: 'user' | 'application', roleIds: string[]) {
  if (!roleIds.length) {
    return
  }
  await runWithFeedback(() => roleStore.deleteRoles({ roleIds }))
}

async function deleteSingleRole(roleId: string) {
  await runWithFeedback(() => roleStore.deleteRoles({ roleId }))
}

async function runModuleAction() {
  await runWithFeedback(() => Promise.all([roleStore.loadRoles(), roleStore.loadPolicies()]))
}

async function evaluatePolicyCheck() {
  try {
    decisionResult.value = await roleStore.evaluatePolicyCheck()
    showToast('操作成功', 'success', {
      source: 'console/Role.runPolicyCheck',
      trigger: 'runPolicyCheck'
    })
  } catch (error) {
    showToast(String(error), 'danger', {
      source: 'console/Role.runPolicyCheck',
      trigger: 'runPolicyCheck',
      error
    })
  }
}
</script>
