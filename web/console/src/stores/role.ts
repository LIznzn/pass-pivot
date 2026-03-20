import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import { checkPolicy as apiCheckPolicy, createPolicy as apiCreatePolicy, deletePolicy as apiDeletePolicy, queryPolicies as apiQueryPolicies, updatePolicy as apiUpdatePolicy } from '../api/manage/policy'
import { createRole as apiCreateRole, deleteRoles as apiDeleteRoles, queryRoles as apiQueryRoles, updateRole as apiUpdateRole } from '../api/manage/role'
import { useConsoleStore } from './console'
import { useOrganizationStore } from './organization'

export const useRoleStore = defineStore('role', () => {
  const console = useConsoleStore()
  const organizationStore = useOrganizationStore()
  const roles = ref<any[]>([])
  const policies = ref<any[]>([])
  const selectedRoleId = ref('')

  const roleForm = reactive({ organizationId: '', name: '', type: 'user', description: '' })
  const policyForm = reactive({ id: '', organizationId: '', roleId: '', name: '', effect: 'allow', priority: 10, apiRulesText: '[\n  {\n    "method": "POST",\n    "path": "/api/manage/v1/example/query"\n  }\n]' })
  const policyCheckForm = reactive({ subjectType: 'application', subjectId: '', method: 'POST', path: '/api/manage/v1/organization/query' })

  const selectedRole = computed(() => roles.value.find((item: any) => item.id === selectedRoleId.value) || roles.value[0])
  const selectedRolePolicies = computed(() => policies.value.filter((item: any) => item.roleId === selectedRole.value?.id))

  async function loadRoles() {
    const response = await apiQueryRoles({ organizationId: console.currentOrganizationId })
    roles.value = response.items
    if (!roles.value.some((item: any) => item.id === selectedRoleId.value)) {
      selectedRoleId.value = roles.value[0]?.id ?? ''
    }
    syncRoleForms(selectedRole.value)
  }

  async function loadPolicies() {
    const response = await apiQueryPolicies({ organizationId: console.currentOrganizationId })
    policies.value = response.items
  }

  function clearRoleState() {
    roles.value = []
    policies.value = []
    selectedRoleId.value = ''
    resetRoleForm()
    resetPolicyForm()
  }

  function resetRoleForm() {
    roleForm.organizationId = console.currentOrganizationId || organizationStore.currentOrganization?.id || ''
    roleForm.name = ''
    roleForm.type = 'user'
    roleForm.description = ''
  }

  function syncRoleForms(role?: any) {
    if (!role) {
      resetRoleForm()
      policyForm.roleId = ''
      return
    }
    roleForm.organizationId = role.organizationId ?? console.currentOrganizationId
    roleForm.name = role.name ?? ''
    roleForm.type = role.type ?? 'user'
    roleForm.description = role.description ?? ''
    policyForm.roleId = role.id ?? ''
    policyCheckForm.subjectType = role.type === 'application' ? 'application' : 'user'
  }

  function setSelectedRoleId(roleId: string) {
    selectedRoleId.value = roleId
    syncRoleForms(selectedRole.value)
  }

  async function createRole() {
    const response = await apiCreateRole(roleForm)
    await Promise.all([loadRoles(), loadPolicies()])
    return response
  }

  async function deleteRoles(payload: { roleId?: string; roleIds?: string[] }) {
    const response = await apiDeleteRoles(payload)
    await loadRoles()
    return response
  }

  async function updateRole() {
    if (!selectedRole.value?.id) {
      return null
    }
    const response = await apiUpdateRole({
      id: selectedRole.value.id,
      name: roleForm.name,
      type: roleForm.type,
      description: roleForm.description
    })
    await Promise.all([loadRoles(), loadPolicies()])
    return response
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
    policyForm.organizationId = console.currentOrganizationId || organizationStore.currentOrganization?.id || ''
    policyForm.roleId = selectedRole.value?.id ?? ''
    policyForm.name = ''
    policyForm.effect = 'allow'
    policyForm.priority = 10
    policyForm.apiRulesText = '[\n  {\n    "method": "POST",\n    "path": "/api/manage/v1/example/query"\n  }\n]'
  }

  async function savePolicy() {
    if (!selectedRole.value?.id) {
      throw new Error('请先选择角色')
    }
    const payload = {
      id: policyForm.id || undefined,
      organizationId: console.currentOrganizationId || organizationStore.currentOrganization?.id || '',
      roleId: selectedRole.value.id,
      name: policyForm.name.trim(),
      effect: policyForm.effect,
      priority: Number(policyForm.priority),
      apiRules: JSON.parse(policyForm.apiRulesText || '[]')
    }
    if (policyForm.id) {
      await apiUpdatePolicy(payload)
    } else {
      await apiCreatePolicy(payload)
    }
    resetPolicyForm()
    await loadPolicies()
  }

  async function deletePolicy(policyId: string) {
    await apiDeletePolicy(policyId)
    if (policyForm.id === policyId) {
      resetPolicyForm()
    }
    await loadPolicies()
  }

  async function evaluatePolicyCheck() {
    return apiCheckPolicy({
      subjectType: policyCheckForm.subjectType,
      subjectId: policyCheckForm.subjectId.trim(),
      method: policyCheckForm.method.trim() || 'POST',
      path: policyCheckForm.path.trim()
    })
  }

  return {
    roles,
    policies,
    selectedRoleId,
    roleForm,
    policyForm,
    policyCheckForm,
    loadRoles,
    loadPolicies,
    clearRoleState,
    resetRoleForm,
    syncRoleForms,
    setSelectedRoleId,
    createRole,
    deleteRoles,
    updateRole,
    editPolicy,
    resetPolicyForm,
    savePolicy,
    deletePolicy,
    evaluatePolicyCheck
  }
})
