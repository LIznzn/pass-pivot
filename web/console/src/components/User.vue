<template>
  <section v-if="userViewMode === 'list'" class="section-grid">
    <div class="info-card">
      <div class="section-title">当前组织下可用的用户</div>
      <div class="d-flex align-items-center justify-content-between gap-3 mb-3 flex-wrap">
        <div class="d-flex align-items-center gap-2 flex-wrap">
          <BButton size="sm" variant="outline-danger" :disabled="selectedUserIds.length === 0" @click="deleteSelectedUsers(selectedUserIds)">删除用户</BButton>
        </div>
        <BButton size="sm" variant="primary" @click="showCreateUserForm = !showCreateUserForm">{{ showCreateUserForm ? '收起添加用户' : '添加用户' }}</BButton>
      </div>
      <div v-if="showCreateUserForm" class="detail-card mb-3">
        <BForm @submit.prevent="submitUserCreateFromList">
          <BFormInput v-model="userForm.username" placeholder="username" class="mb-2" />
          <BFormInput v-model="userForm.name" placeholder="name" class="mb-2" />
          <BFormInput v-model="userForm.email" placeholder="email" class="mb-2" />
          <div class="phone-input-group mb-2">
            <BFormSelect v-model="userPhoneInput.countryCode" :options="console.phoneCountryOptions" class="phone-country-select" />
            <BFormInput v-model="userPhoneInput.localNumber" placeholder="phoneNumber" class="phone-local-input" />
          </div>
          <BFormInput v-model="userForm.roleLabels" placeholder="role labels, comma separated" class="mb-2" />
          <BFormInput v-model="userForm.identifier" placeholder="login identifier" class="mb-2" />
          <BFormInput v-model="userForm.password" type="password" placeholder="initial password" class="mb-2" />
          <div class="d-flex gap-2">
            <BButton type="submit" variant="primary">创建用户</BButton>
            <BButton type="button" variant="outline-secondary" @click="showCreateUserForm = false">取消</BButton>
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
                  :checked="users.length > 0 && selectedUserIds.length === users.length"
                  @change="toggleAllUsers(($event.target as HTMLInputElement).checked)"
                />
              </th>
              <th>用户 ID</th>
              <th>用户名</th>
              <th>名称</th>
              <th>邮箱 / 手机号</th>
              <th>状态</th>
              <th>角色</th>
              <th class="text-end">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in users" :key="user.id">
              <td class="console-list-check-col">
                <input class="form-check-input console-list-checkbox" type="checkbox" :checked="selectedUserIds.includes(user.id)" @change="toggleUserSelection(user.id, ($event.target as HTMLInputElement).checked)" />
              </td>
              <td class="console-list-id">{{ user.id }}</td>
              <td>{{ user.username || '-' }}</td>
              <td>{{ user.name || '-' }}</td>
              <td>{{ user.email || user.phoneNumber || '-' }}</td>
              <td>
                <span class="badge rounded-pill" :class="user.status === 'disabled' ? 'text-bg-secondary' : 'text-bg-success'">
                  {{ user.status === 'disabled' ? '停用' : '启用' }}
                </span>
              </td>
              <td>{{ formatRoleLabels(user.roles) }}</td>
              <td class="text-end">
                <div class="d-inline-flex gap-2">
                  <BButton size="sm" variant="outline-primary" @click="selectUser(user)">查看详情</BButton>
                  <BButton size="sm" variant="outline-danger" @click="deleteSingleUser(user.id)">删除</BButton>
                </div>
              </td>
            </tr>
            <tr v-if="users.length === 0">
              <td colspan="8" class="text-center text-secondary py-4">当前组织下还没有用户。</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>

  <template v-else>
    <UserDetail
      :user-update-form="userUpdateForm"
      :user-update-phone-input="userUpdatePhoneInput"
      :phone-country-options="console.phoneCountryOptions"
      :current-user-record="currentUserRecord"
      :user-detail="userDetail"
      :user-admin-form="userAdminForm"
      :external-binding-form="externalBindingForm"
      :user-assignable-roles="userAssignableRoles"
      :user-role-assignments="userRoleAssignments"
      :user-admin-result="userAdminResult"
      :selected-user-id="selectedUserId"
      :module-recent-changes="moduleRecentChanges"
      :format-date-time="console.formatDateTime"
      @back="backToUserList"
      @run-module-action="runModuleAction"
      @copy-metric="copyMetricValue"
      @scroll-to-panel="scrollToPanel"
      @update-user="updateUser"
      @reset-user-password="resetUserPassword"
      @toggle-webauthn-login="toggleWebAuthnLogin"
      @register-secure-key="registerSecureKey"
      @delete-external-binding="deleteExternalBinding"
      @create-external-binding="createExternalBinding"
      @handle-inline-mfa-method-action="handleInlineMFAMethodAction"
      @open-mfa-modal="openMFAModal"
      @revoke-all-user-sessions="revokeAllUserSessions"
      @untrust-managed-device="untrustManagedDevice"
      @toggle-user-role="toggleUserRole"
      @disable-user="disableUser"
      @enable-user="enableUser"
      @reset-user-ukid="resetUserUkid"
      @rotate-user-token="rotateUserToken"
      @delete-single-user="deleteSingleUser"
    />

    <MfaConfigModal
      :visible="mfaConfigModalVisible"
      :method="currentMFAMethod"
      :active-totp-enrollments="activeTOTPEnrollments"
      :totp-setup="totpSetup"
      :totp-qr-code-data-url="totpQRCodeDataURL"
      :pending-totp-enrollment-id="pendingTotpEnrollmentId"
      :pending-totp-manual-entry-key="pendingTotpManualEntryKey"
      :totp-verify-form="totpVerifyForm"
      :current-user-record="currentUserRecord"
      :mfa-setting-form="mfaSettingForm"
      :boolean-setting-options="console.booleanSettingOptions"
      :u2f-secure-keys="u2fSecureKeys"
      :user-detail="userDetail"
      :generated-recovery-code-list="generatedRecoveryCodeList"
      :format-date-time="console.formatDateTime"
      @update:visible="mfaConfigModalVisible = $event"
      @delete-totp-enrollments="deleteTotpEnrollments"
      @delete-secure-key="deleteSecureKey"
      @submit="submitCurrentMFAModal"
    />
  </template>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import QRCode from 'qrcode'
import { BButton, BForm, BFormInput, BFormSelect } from 'bootstrap-vue-next'
import { normalizeCreationOptions, serializeCredential } from '@shared/api/webauthn'
import { useToast } from '@shared/composables/toast'
import {
  beginRegisterSecureKey as apiBeginRegisterSecureKey,
  createExternalIdentityBinding as apiCreateExternalIdentityBinding,
  createUser as apiCreateUser,
  deleteExternalIdentityBinding as apiDeleteExternalIdentityBinding,
  deleteSecureKey as apiDeleteSecureKey,
  deleteUserMfaEnrollment as apiDeleteUserMfaEnrollment,
  deleteUsers as apiDeleteUsers,
  disableUser as apiDisableUser,
  enableUser as apiEnableUser,
  enrollUserTotp as apiEnrollUserTotp,
  finishRegisterSecureKey as apiFinishRegisterSecureKey,
  generateUserRecoveryCodes as apiGenerateUserRecoveryCodes,
  queryUserDetail as apiQueryUserDetail,
  queryUsers as apiQueryUsers,
  resetUserPassword as apiResetUserPassword,
  resetUserUkid as apiResetUserUkid,
  revokeAllUserSessions as apiRevokeAllUserSessions,
  rotateUserToken as apiRotateUserToken,
  untrustUserDevice as apiUntrustUserDevice,
  updateUser as apiUpdateUser,
  updateUserMfaMethod as apiUpdateUserMfaMethod,
  verifyUserTotp as apiVerifyUserTotp
} from '../api/manage/user'
import { queryRoles as apiQueryRoles } from '../api/manage/role'
import UserDetail from '../components/UserDetail.vue'
import MfaConfigModal from '../modal/MfaConfigModal.vue'
import { useConsoleLayout } from '../composables/useConsoleLayout'

type MFAMethod = 'totp' | 'email_code' | 'sms_code' | 'u2f' | 'recovery_code'
type PhoneInputState = {
  countryCode: string
  localNumber: string
}

const router = useRouter()
const route = useRoute()
const toast = useToast()
const console = useConsoleLayout()
const consoleApplicationId = import.meta.env.PPVT_CONSOLE_APPLICATION_ID ?? ''

const users = ref<any[]>([])
const roles = ref<any[]>([])
const userDetail = ref<any | null>(null)
const selectedUserId = ref('')
const userViewMode = ref<'list' | 'detail'>('list')
const mfaConfigModalVisible = ref(false)
const currentMFAMethod = ref<MFAMethod>('totp')
const totpSetup = ref<unknown>(null)
const totpQRCodeDataURL = ref('')
const recoveryCodes = ref<unknown>(null)
const userAdminResult = ref<unknown>(null)
const userRoleAssignments = ref<string[]>([])
const selectedUserIds = ref<string[]>([])
const showCreateUserForm = ref(false)

const userForm = reactive({ organizationId: '', applicationId: '', username: '', name: '', email: '', phoneNumber: '', roleLabels: '', identifier: '', password: '' })
const userUpdateForm = reactive({ id: '', username: '', name: '', email: '', phoneNumber: '', roleLabels: '', status: '' })
const userAdminForm = reactive({ password: '' })
const externalBindingForm = reactive({ organizationId: '', userId: '', externalIdpId: '', issuer: '', subject: '' })
const userPhoneInput = reactive<PhoneInputState>({ countryCode: '+86', localNumber: '' })
const userUpdatePhoneInput = reactive<PhoneInputState>({ countryCode: '+86', localNumber: '' })
const mfaSettingForm = reactive({ emailEnabled: 'disabled', smsEnabled: 'disabled' })
const totpVerifyForm = reactive({ enrollmentId: '', code: '' })

const selectedUser = computed(() => users.value.find((item: any) => item.id === selectedUserId.value))
const currentUserRecord = computed(() => {
  if (userDetail.value?.user?.id === selectedUserId.value) {
    return userDetail.value.user
  }
  return selectedUser.value
})
const userAssignableRoles = computed(() => roles.value.filter((item: any) => item.type === 'user'))
const moduleRecentChanges = computed(() => {
  if (userDetail.value?.recentAuditLogs?.length) {
    return userDetail.value.recentAuditLogs.slice(0, 6)
  }
  return console.recentAuditLogs.slice(0, 6)
})
const pendingTotpEnrollmentId = computed(() => (totpSetup.value as { enrollmentId?: string } | null)?.enrollmentId || '')
const pendingTotpProvisioningUri = computed(() => (totpSetup.value as { provisioningUri?: string } | null)?.provisioningUri || '')
const pendingTotpManualEntryKey = computed(() => (totpSetup.value as { manualEntryKey?: string } | null)?.manualEntryKey || '')
const generatedRecoveryCodeList = computed(() => (recoveryCodes.value as { codes?: string[] } | null)?.codes || [])
const activeTOTPEnrollments = computed(() => (userDetail.value?.mfaEnrollments || []).filter((item: any) => item.method === 'totp'))
const emailCodeEnrollment = computed(() => (userDetail.value?.mfaEnrollments || []).find((item: any) => item.method === 'email_code'))
const smsCodeEnrollment = computed(() => (userDetail.value?.mfaEnrollments || []).find((item: any) => item.method === 'sms_code'))
const u2fSecureKeys = computed(() => (userDetail.value?.secureKeys || []).filter((item: any) => item.u2fEnable))

watch(
  () => pendingTotpProvisioningUri.value,
  async (value) => {
    if (!value) {
      totpQRCodeDataURL.value = ''
      return
    }
    try {
      totpQRCodeDataURL.value = await QRCode.toDataURL(value, { margin: 1, width: 192 })
    } catch {
      totpQRCodeDataURL.value = ''
    }
  },
  { immediate: true }
)

watch(
  () => mfaConfigModalVisible.value,
  (visible) => {
    if (visible || currentMFAMethod.value !== 'totp') {
      return
    }
    totpSetup.value = null
    totpVerifyForm.enrollmentId = ''
    totpVerifyForm.code = ''
  }
)

watch(
  () => [console.currentOrganizationId, route.name, route.params.userId],
  async ([organizationId, routeName, routeUserId]) => {
    const nextOrganizationId = typeof organizationId === 'string' ? organizationId : ''
    if (!nextOrganizationId) {
      users.value = []
      roles.value = []
      userDetail.value = null
      selectedUserId.value = ''
      userViewMode.value = 'list'
      return
    }
    userForm.organizationId = nextOrganizationId
    externalBindingForm.organizationId = nextOrganizationId
    userViewMode.value = routeName === 'console-user-detail' ? 'detail' : 'list'
    await Promise.all([loadUsers(), loadRoles()])
    if (typeof routeUserId === 'string' && routeUserId) {
      selectedUserId.value = routeUserId
      await loadUserDetail(routeUserId)
      return
    }
    if (!selectedUserId.value && users.value.length) {
      selectedUserId.value = users.value[0].id
      syncCurrentUser(users.value[0])
    }
    if (userViewMode.value === 'detail') {
      await loadUserDetail(selectedUserId.value)
    }
  },
  { immediate: true }
)

watch(
  () => users.value,
  (items) => {
    const userIds = new Set(items.map((item: any) => item.id))
    selectedUserIds.value = selectedUserIds.value.filter((id) => userIds.has(id))
  },
  { immediate: true, deep: true }
)

function syncCurrentUser(user?: any) {
  if (!user) {
    return
  }
  userUpdateForm.id = user.id ?? ''
  userUpdateForm.username = user.username ?? ''
  userUpdateForm.name = user.name ?? ''
  userUpdateForm.email = user.email ?? ''
  userUpdateForm.phoneNumber = user.phoneNumber ?? ''
  syncPhoneInput(userUpdatePhoneInput, userUpdateForm.phoneNumber, console.phoneCountryOptions.map((item: any) => item.value))
  userUpdateForm.roleLabels = (user.roles ?? []).join(',')
  userUpdateForm.status = user.status ?? ''
  userRoleAssignments.value = [...(user.roles ?? [])]
}

async function loadUsers() {
  const response = await apiQueryUsers({ organizationId: console.currentOrganizationId })
  users.value = response.items
  await console.loadUsers()
  if (!users.value.some((item: any) => item.id === selectedUserId.value)) {
    selectedUserId.value = users.value[0]?.id ?? ''
  }
  if (selectedUserId.value) {
    syncCurrentUser(users.value.find((item: any) => item.id === selectedUserId.value) || users.value[0])
  }
}

async function loadRoles() {
  const response = await apiQueryRoles({ organizationId: console.currentOrganizationId })
  roles.value = response.items
}

async function loadUserDetail(userId = selectedUserId.value) {
  if (!userId) {
    userDetail.value = null
    return
  }
  const detail = await apiQueryUserDetail(userId)
  userDetail.value = detail
  syncCurrentUser(detail.user)
  externalBindingForm.organizationId = detail.user?.organizationId ?? console.currentOrganizationId
  externalBindingForm.userId = detail.user?.id ?? userId
  externalBindingForm.externalIdpId = detail.externalIdps?.[0]?.id ?? externalBindingForm.externalIdpId
  externalBindingForm.issuer = detail.externalIdps?.find((item: any) => item.id === externalBindingForm.externalIdpId)?.issuer ?? externalBindingForm.issuer
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

function toggleAllUsers(checked: boolean) {
  selectedUserIds.value = checked ? users.value.map((item: any) => item.id) : []
}

function toggleUserSelection(userId: string, checked: boolean) {
  if (checked) {
    if (!selectedUserIds.value.includes(userId)) {
      selectedUserIds.value = [...selectedUserIds.value, userId]
    }
    return
  }
  selectedUserIds.value = selectedUserIds.value.filter((id) => id !== userId)
}

function buildUserRouteAfterDelete() {
  if (route.name !== 'console-user-detail') {
    return null
  }
  if (selectedUserId.value) {
    return {
      name: 'console-user-detail',
      params: {
        organizationId: console.currentOrganizationId,
        userId: selectedUserId.value
      }
    }
  }
  return {
    name: 'console-user-list',
    params: {
      organizationId: console.currentOrganizationId
    }
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

function selectUser(user: any) {
  userViewMode.value = 'detail'
  selectedUserId.value = user.id
  syncCurrentUser(user)
  void loadUserDetail(user.id)
  void router.push({
    name: 'console-user-detail',
    params: {
      organizationId: console.currentOrganizationId || console.currentOrganization?.id || '',
      userId: user.id ?? ''
    }
  })
}

function backToUserList() {
  userViewMode.value = 'list'
  void router.push({
    name: 'console-user-list',
    params: {
      organizationId: console.currentOrganizationId || console.currentOrganization?.id || ''
    }
  })
}

async function deleteSelectedUsers(userIds: string[]) {
  if (!userIds.length) {
    return
  }
  await withFeedback(async () => {
    await apiDeleteUsers({ userIds })
    await loadUsers()
    await loadUserDetail()
  })
}

async function deleteSingleUser(userId: string) {
  await withFeedback(async () => {
    await apiDeleteUsers({ userId })
    await loadUsers()
    await loadUserDetail()
    const nextRoute = buildUserRouteAfterDelete()
    if (nextRoute) {
      await router.replace(nextRoute)
    }
  })
}

async function createUser() {
  await withFeedback(async () => {
    await apiCreateUser({
      ...userForm,
      phoneNumber: composePhoneNumber(userPhoneInput),
      roles: splitRoleLabels(userForm.roleLabels)
    })
    resetPhoneInput(userPhoneInput)
    await loadUsers()
  })
}

async function submitUserCreateFromList() {
  userForm.phoneNumber = composePhoneNumber(userPhoneInput)
  await createUser()
  showCreateUserForm.value = false
}

async function updateUser() {
  await withFeedback(async () => {
    await apiUpdateUser({
      ...userUpdateForm,
      phoneNumber: composePhoneNumber(userUpdatePhoneInput),
      roles: userRoleAssignments.value
    })
    await loadUsers()
    await loadUserDetail()
  })
}

async function createExternalBinding() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    externalBindingForm.userId = externalBindingForm.userId || selectedUserId.value
    await apiCreateExternalIdentityBinding(externalBindingForm)
    await loadUserDetail()
  })
}

async function registerSecureKey(purpose: 'webauthn' | 'u2f' = 'webauthn') {
  const userId = selectedUserId.value
  if (!userId) {
    return
  }
  await withFeedback(async () => {
    const begin = await apiBeginRegisterSecureKey(userId, purpose)
    const credential = await navigator.credentials.create({
      publicKey: normalizeCreationOptions(begin.options)
    })
    if (!credential) {
      throw new Error('Secure key registration was cancelled')
    }
    await apiFinishRegisterSecureKey(begin.challengeId, serializeCredential(credential as PublicKeyCredential))
    await loadUserDetail()
  })
}

async function enrollTotp() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    totpSetup.value = await apiEnrollUserTotp(selectedUserId.value, consoleApplicationId)
    totpVerifyForm.enrollmentId = pendingTotpEnrollmentId.value
    await loadUserDetail()
  })
}

async function verifyTotpEnrollment() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiVerifyUserTotp(selectedUserId.value, pendingTotpEnrollmentId.value, totpVerifyForm.code)
    totpSetup.value = null
    totpVerifyForm.enrollmentId = ''
    totpVerifyForm.code = ''
    await loadUserDetail()
  })
}

async function generateRecoveryCodes() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    recoveryCodes.value = await apiGenerateUserRecoveryCodes(selectedUserId.value)
    await loadUserDetail()
  })
}

async function saveMFAEmailSetting() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiUpdateUserMfaMethod(selectedUserId.value, 'email_code', mfaSettingForm.emailEnabled === 'active')
    await loadUserDetail()
    mfaConfigModalVisible.value = false
  })
}

async function saveMFASMSSetting() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiUpdateUserMfaMethod(selectedUserId.value, 'sms_code', mfaSettingForm.smsEnabled === 'active')
    await loadUserDetail()
    mfaConfigModalVisible.value = false
  })
}

async function toggleInlineMFAMethod(method: MFAMethod, enabled: boolean) {
  if (method !== 'email_code' && method !== 'sms_code') {
    return
  }
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiUpdateUserMfaMethod(selectedUserId.value, method, enabled)
    await loadUserDetail()
  })
}

async function handleInlineMFAMethodAction(item: { id: MFAMethod; enabled: boolean; disabled?: boolean }) {
  if (item.id !== 'email_code' && item.id !== 'sms_code') {
    return
  }
  if (item.disabled) {
    toast.error(item.id === 'email_code' ? '请先在基本信息中配置邮箱' : '请先在基本信息中配置手机')
    return
  }
  await toggleInlineMFAMethod(item.id, !item.enabled)
}

async function submitCurrentMFAModal() {
  if (currentMFAMethod.value === 'totp') {
    await verifyTotpEnrollment()
    return
  }
  if (currentMFAMethod.value === 'email_code') {
    await saveMFAEmailSetting()
    return
  }
  if (currentMFAMethod.value === 'sms_code') {
    await saveMFASMSSetting()
    return
  }
  if (currentMFAMethod.value === 'u2f') {
    await registerSecureKey('u2f')
    return
  }
  if (currentMFAMethod.value === 'recovery_code') {
    await generateRecoveryCodes()
  }
}

async function deleteTotpEnrollments() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiDeleteUserMfaEnrollment(selectedUserId.value, 'totp')
    totpSetup.value = null
    totpVerifyForm.enrollmentId = ''
    totpVerifyForm.code = ''
    await loadUserDetail()
    mfaConfigModalVisible.value = false
  })
}

async function deleteSecureKey(credentialId: string) {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiDeleteSecureKey(selectedUserId.value, credentialId)
    await loadUserDetail()
  })
}

async function toggleWebAuthnLogin(enabled: boolean) {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiUpdateUserMfaMethod(selectedUserId.value, 'webauthn', enabled)
    await loadUserDetail()
  }, enabled ? '已启用通行密钥登录' : '已关闭通行密钥登录')
}

async function resetUserPassword() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiResetUserPassword(selectedUserId.value, userAdminForm.password)
    userAdminForm.password = ''
  })
}

async function resetUserUkid() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiResetUserUkid(selectedUserId.value)
    await loadUserDetail()
  })
}

async function disableUser() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiDisableUser(selectedUserId.value)
    await loadUsers()
    await loadUserDetail()
  })
}

async function enableUser() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiEnableUser(selectedUserId.value)
    await loadUsers()
    await loadUserDetail()
  })
}

async function deleteExternalBinding(bindingId: string) {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiDeleteExternalIdentityBinding(selectedUserId.value, bindingId)
    await loadUserDetail()
  })
}

async function untrustManagedDevice(deviceId: string) {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiUntrustUserDevice(selectedUserId.value, deviceId)
    await loadUserDetail()
  })
}

async function revokeAllUserSessions() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiRevokeAllUserSessions(selectedUserId.value)
    await loadUserDetail()
  })
}

async function rotateUserToken() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiRotateUserToken(selectedUserId.value)
    await loadUserDetail()
  })
}

function toggleUserRole(roleName: string, checked: boolean) {
  if (checked) {
    if (!userRoleAssignments.value.includes(roleName)) {
      userRoleAssignments.value = [...userRoleAssignments.value, roleName]
    }
    return
  }
  userRoleAssignments.value = userRoleAssignments.value.filter((item) => item !== roleName)
}

async function openMFAModal(method: MFAMethod) {
  currentMFAMethod.value = method
  mfaSettingForm.emailEnabled = emailCodeEnrollment.value?.status === 'active' ? 'active' : 'disabled'
  mfaSettingForm.smsEnabled = smsCodeEnrollment.value?.status === 'active' ? 'active' : 'disabled'
  mfaConfigModalVisible.value = true
  if (method === 'totp' && activeTOTPEnrollments.value.length === 0 && !pendingTotpProvisioningUri.value) {
    await enrollTotp()
  }
}

async function runModuleAction() {
  await Promise.all([loadUsers(), loadUserDetail()])
}

function syncPhoneInput(
  target: PhoneInputState,
  value: string | undefined,
  phoneCountryCodes: string[]
) {
  const normalized = String(value || '').trim()
  if (!normalized) {
    target.countryCode = '+86'
    target.localNumber = ''
    return
  }
  const matched = [...phoneCountryCodes]
    .sort((left, right) => right.length - left.length)
    .find((code) => normalized.startsWith(code))
  if (matched) {
    target.countryCode = matched
    target.localNumber = normalized.slice(matched.length).replace(/^[\s-]+/, '')
    return
  }
  target.countryCode = '+86'
  target.localNumber = normalized.replace(/^\+?86[\s-]*/, '')
}

function composePhoneNumber(source: PhoneInputState) {
  const localNumber = source.localNumber.trim()
  if (!localNumber) {
    return ''
  }
  return `${source.countryCode}${localNumber}`
}

function resetPhoneInput(target: PhoneInputState) {
  target.countryCode = '+86'
  target.localNumber = ''
}

function splitRoleLabels(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function formatRoleLabels(value?: string[]) {
  if (!value || value.length === 0) {
    return 'none'
  }
  return value.join(', ')
}
</script>
