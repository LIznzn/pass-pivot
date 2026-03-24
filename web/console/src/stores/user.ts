import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  beginRegisterSecureKey as apiBeginRegisterSecureKey,
  createExternalIdentityBinding as apiCreateExternalIdentityBinding,
  createUser as apiCreateUser,
  deleteExternalIdentityBinding as apiDeleteExternalIdentityBinding,
  deleteSecureKey as apiDeleteSecureKey,
  deleteUserMfaEnrollment as apiDeleteUserMfaEnrollment,
  deleteUsers as apiDeleteUsers,
  disableUser as apiDisableUser,
  enrollUserTotp as apiEnrollUserTotp,
  enableUser as apiEnableUser,
  finishRegisterSecureKey as apiFinishRegisterSecureKey,
  generateUserRecoveryCodes as apiGenerateUserRecoveryCodes,
  queryUserRecoveryCodes as apiQueryUserRecoveryCodes,
  queryUserDetail as apiQueryUserDetail,
  queryUsers as apiQueryUsers,
  resetUserPassword as apiResetUserPassword,
  resetUserUkid as apiResetUserUkid,
  revokeAllUserSessions as apiRevokeAllUserSessions,
  rotateUserToken as apiRotateUserToken,
  untrustUserDevice as apiUntrustUserDevice,
  updateUser as apiUpdateUser,
  updateSecureKey as apiUpdateSecureKey,
  updateUserMfaMethod as apiUpdateUserMfaMethod,
  verifyUserTotp as apiVerifyUserTotp
} from '@/api/manage/user'
import { useConsoleStore } from './console'
import { useOrganizationStore } from './organization'
import { queryRoles as apiQueryRoles } from '@/api/manage/role'

export const useUserStore = defineStore('user', () => {
  const console = useConsoleStore()
  const organizationStore = useOrganizationStore()
  const users = ref<any[]>([])
  const roles = ref<any[]>([])
  const userDetail = ref<any | null>(null)
  const selectedUserId = ref('')

  const userForm = reactive({ organizationId: '', applicationId: '', username: '', name: '', email: '', phoneNumber: '', roleLabels: '', identifier: '', password: '' })
  const userUpdateForm = reactive({ id: '', username: '', name: '', email: '', phoneNumber: '', roleLabels: '', status: '' })
  const externalBindingForm = reactive({ organizationId: '', userId: '', externalIdpId: '', issuer: '', subject: '' })
  const userRoleAssignments = ref<string[]>([])

  const selectedUser = computed(() => users.value.find((item: any) => item.id === selectedUserId.value))
  const currentUserRecord = computed(() => {
    if (userDetail.value?.user?.id === selectedUserId.value) {
      return userDetail.value.user
    }
    return selectedUser.value
  })

  function clearUserState() {
    users.value = []
    roles.value = []
    userDetail.value = null
    selectedUserId.value = ''
    userRoleAssignments.value = []
    resetUserForm()
    resetUserUpdateForm()
  }

  function resetUserForm() {
    userForm.organizationId = console.currentOrganizationId || organizationStore.currentOrganization?.id || ''
    userForm.applicationId = ''
    userForm.username = ''
    userForm.name = ''
    userForm.email = ''
    userForm.phoneNumber = ''
    userForm.roleLabels = ''
    userForm.identifier = ''
    userForm.password = ''
  }

  function resetUserUpdateForm() {
    userUpdateForm.id = ''
    userUpdateForm.username = ''
    userUpdateForm.name = ''
    userUpdateForm.email = ''
    userUpdateForm.phoneNumber = ''
    userUpdateForm.roleLabels = ''
    userUpdateForm.status = ''
  }

  function syncCurrentUser(user?: any) {
    if (!user) {
      resetUserUpdateForm()
      userRoleAssignments.value = []
      return
    }
    userUpdateForm.id = user.id ?? ''
    userUpdateForm.username = user.username ?? ''
    userUpdateForm.name = user.name ?? ''
    userUpdateForm.email = user.email ?? ''
    userUpdateForm.phoneNumber = user.phoneNumber ?? ''
    userUpdateForm.roleLabels = (user.roles ?? []).join(',')
    userUpdateForm.status = user.status ?? ''
    userRoleAssignments.value = [...(user.roles ?? [])]
  }

  function setSelectedUserId(userId: string) {
    selectedUserId.value = userId
    syncCurrentUser(selectedUser.value)
  }

  async function loadUsers() {
    const organizationId = console.currentOrganizationId || organizationStore.currentOrganization?.id || ''
    userForm.organizationId = organizationId
    externalBindingForm.organizationId = organizationId
    if (!organizationId) {
      clearUserState()
      return
    }
    const response = await apiQueryUsers({ organizationId })
    users.value = response.items
    if (!users.value.some((item: any) => item.id === selectedUserId.value)) {
      selectedUserId.value = users.value[0]?.id ?? ''
    }
    syncCurrentUser(selectedUser.value || users.value[0])
  }

  async function loadRoles() {
    const organizationId = console.currentOrganizationId || organizationStore.currentOrganization?.id || ''
    if (!organizationId) {
      roles.value = []
      return
    }
    const response = await apiQueryRoles({ organizationId })
    roles.value = response.items
  }

  async function loadUserDetail(userId = selectedUserId.value) {
    if (!userId) {
      userDetail.value = null
      return null
    }
    const detail = await apiQueryUserDetail(userId)
    userDetail.value = detail
    selectedUserId.value = detail.user?.id ?? userId
    syncCurrentUser(detail.user)
    externalBindingForm.organizationId = detail.user?.organizationId ?? console.currentOrganizationId
    externalBindingForm.userId = detail.user?.id ?? userId
    externalBindingForm.externalIdpId = detail.externalIdps?.[0]?.id ?? externalBindingForm.externalIdpId
    externalBindingForm.issuer = detail.externalIdps?.find((item: any) => item.id === externalBindingForm.externalIdpId)?.issuer ?? externalBindingForm.issuer
    return detail
  }

  async function createUser(payload: any) {
    const response = await apiCreateUser(payload)
    await loadUsers()
    return response
  }

  async function updateUser(payload: any) {
    const response = await apiUpdateUser(payload)
    await loadUsers()
    await loadUserDetail(payload.id || selectedUserId.value)
    return response
  }

  async function deleteUsers(payload: { userId?: string; userIds?: string[] }) {
    const response = await apiDeleteUsers(payload)
    await loadUsers()
    await loadUserDetail()
    return response
  }

  async function createExternalBinding() {
    if (!selectedUserId.value) {
      return null
    }
    externalBindingForm.userId = externalBindingForm.userId || selectedUserId.value
    const response = await apiCreateExternalIdentityBinding(externalBindingForm)
    await loadUserDetail()
    return response
  }

  async function beginRegisterSecureKey(purpose?: 'webauthn' | 'u2f') {
    if (!selectedUserId.value) {
      return null
    }
    return apiBeginRegisterSecureKey(selectedUserId.value, purpose)
  }

  async function finishRegisterSecureKey(challengeId: string, credential: unknown) {
    const response = await apiFinishRegisterSecureKey(challengeId, credential)
    await loadUserDetail()
    return response
  }

  async function deleteExternalBinding(bindingId: string) {
    if (!selectedUserId.value) {
      return null
    }
    const response = await apiDeleteExternalIdentityBinding(selectedUserId.value, bindingId)
    await loadUserDetail()
    return response
  }

  async function deleteSecureKey(credentialId: string) {
    if (!selectedUserId.value) {
      return null
    }
    const response = await apiDeleteSecureKey(selectedUserId.value, credentialId)
    await loadUserDetail()
    return response
  }

  async function updateSecureKey(credentialId: string, identifier: string) {
    if (!selectedUserId.value) {
      return null
    }
    const response = await apiUpdateSecureKey(selectedUserId.value, credentialId, identifier)
    await loadUserDetail()
    return response
  }

  async function updateUserMfaMethod(method: string, enabled: boolean) {
    if (!selectedUserId.value) {
      return null
    }
    const response = await apiUpdateUserMfaMethod(selectedUserId.value, method, enabled)
    await loadUserDetail()
    return response
  }

  async function verifyUserTotp(enrollmentId: string, code: string) {
    if (!selectedUserId.value) {
      return null
    }
    const response = await apiVerifyUserTotp(selectedUserId.value, enrollmentId, code)
    await loadUserDetail()
    return response
  }

  async function deleteUserMfaEnrollment(method: string) {
    if (!selectedUserId.value) {
      return null
    }
    const response = await apiDeleteUserMfaEnrollment(selectedUserId.value, method)
    await loadUserDetail()
    return response
  }

  async function generateRecoveryCodes() {
    if (!selectedUserId.value) {
      return null
    }
    const recoveryCodes = await apiGenerateUserRecoveryCodes(selectedUserId.value)
    await loadUserDetail()
    return recoveryCodes
  }

  async function queryRecoveryCodes() {
    if (!selectedUserId.value) {
      return null
    }
    return apiQueryUserRecoveryCodes(selectedUserId.value)
  }

  async function enrollUserTotp(consoleApplicationId: string) {
    if (!selectedUserId.value) {
      return null
    }
    const response = await apiEnrollUserTotp(selectedUserId.value, consoleApplicationId)
    await loadUserDetail()
    return response
  }

  async function resetUserPassword(password: string) {
    if (!selectedUserId.value) {
      return null
    }
    return apiResetUserPassword(selectedUserId.value, password)
  }

  async function resetUserUkid() {
    if (!selectedUserId.value) {
      return null
    }
    const result = await apiResetUserUkid(selectedUserId.value)
    await loadUserDetail()
    return result
  }

  async function disableUser() {
    if (!selectedUserId.value) {
      return null
    }
    const result = await apiDisableUser(selectedUserId.value)
    await loadUsers()
    await loadUserDetail()
    return result
  }

  async function enableUser() {
    if (!selectedUserId.value) {
      return null
    }
    const result = await apiEnableUser(selectedUserId.value)
    await loadUsers()
    await loadUserDetail()
    return result
  }

  async function untrustManagedDevice(deviceId: string) {
    if (!selectedUserId.value) {
      return null
    }
    const response = await apiUntrustUserDevice(selectedUserId.value, deviceId)
    await loadUserDetail()
    return response
  }

  async function revokeAllUserSessions() {
    if (!selectedUserId.value) {
      return null
    }
    const result = await apiRevokeAllUserSessions(selectedUserId.value)
    await loadUserDetail()
    return result
  }

  async function rotateUserToken() {
    if (!selectedUserId.value) {
      return null
    }
    const result = await apiRotateUserToken(selectedUserId.value)
    await loadUserDetail()
    return result
  }

  return {
    users,
    roles,
    userDetail,
    selectedUserId,
    userForm,
    userUpdateForm,
    externalBindingForm,
    userRoleAssignments,
    clearUserState,
    resetUserForm,
    resetUserUpdateForm,
    syncCurrentUser,
    setSelectedUserId,
    loadUsers,
    loadRoles,
    loadUserDetail,
    createUser,
    updateUser,
    deleteUsers,
    createExternalBinding,
    beginRegisterSecureKey,
    finishRegisterSecureKey,
    deleteExternalBinding,
    deleteSecureKey,
    updateSecureKey,
    updateUserMfaMethod,
    deleteUserMfaEnrollment,
    generateRecoveryCodes,
    queryRecoveryCodes,
    enrollUserTotp,
    verifyUserTotp,
    resetUserPassword,
    resetUserUkid,
    disableUser,
    enableUser,
    untrustManagedDevice,
    revokeAllUserSessions,
    rotateUserToken
  }
})
