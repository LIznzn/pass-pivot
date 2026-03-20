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
          <BFormInput v-model="userStore.userForm.username" placeholder="username" class="mb-2" />
          <BFormInput v-model="userStore.userForm.name" placeholder="name" class="mb-2" />
          <BFormInput v-model="userStore.userForm.email" placeholder="email" class="mb-2" />
          <div class="phone-input-group mb-2">
            <BFormSelect v-model="userPhoneInput.countryCode" :options="phoneCountryOptions" class="phone-country-select" />
            <BFormInput v-model="userPhoneInput.localNumber" placeholder="phoneNumber" class="phone-local-input" />
          </div>
          <BFormInput v-model="userStore.userForm.roleLabels" placeholder="role labels, comma separated" class="mb-2" />
          <BFormInput v-model="userStore.userForm.identifier" placeholder="login identifier" class="mb-2" />
          <BFormInput v-model="userStore.userForm.password" type="password" placeholder="initial password" class="mb-2" />
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
                  :checked="userStore.users.length > 0 && selectedUserIds.length === userStore.users.length"
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
            <tr v-for="user in userStore.users" :key="user.id">
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
            <tr v-if="userStore.users.length === 0">
              <td colspan="8" class="text-center text-secondary py-4">当前组织下还没有用户。</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>

  <template v-else>
    <UserDetail
      :user-update-form="userStore.userUpdateForm"
      :user-update-phone-input="userUpdatePhoneInput"
      :phone-country-options="phoneCountryOptions"
      :current-user-record="currentUserRecord"
      :user-detail="userStore.userDetail"
      :user-admin-form="userAdminForm"
      :external-binding-form="userStore.externalBindingForm"
      :user-assignable-roles="userAssignableRoles"
      :user-role-assignments="userStore.userRoleAssignments"
      :user-admin-result="userAdminResult"
      :selected-user-id="userStore.selectedUserId"
      @back="backToUserList"
      @run-module-action="runModuleAction"
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
      :u2f-secure-keys="u2fSecureKeys"
      :user-detail="userStore.userDetail"
      :generated-recovery-code-list="generatedRecoveryCodeList"
      @update:visible="mfaConfigModalVisible = $event"
      @delete-totp-enrollments="deleteTotpEnrollments"
      @delete-secure-key="deleteSecureKey"
      @submit="submitCurrentMFAModal"
    />
  </template>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import QRCode from 'qrcode'
import { BButton, BForm, BFormInput, BFormSelect } from 'bootstrap-vue-next'
import { normalizeCreationOptions, serializeCredential } from '@shared/api/webauthn'
import { useToast } from '@shared/composables/toast'
import UserDetail from '../components/UserDetail.vue'
import MfaConfigModal from '../modal/MfaConfigModal.vue'
import { useConsoleStore } from '../stores/console'
import { useOrganizationStore } from '../stores/organization'
import { useUserStore } from '../stores/user'

type MFAMethod = 'totp' | 'email_code' | 'sms_code' | 'u2f' | 'recovery_code'
type PhoneInputState = {
  countryCode: string
  localNumber: string
}

const router = useRouter()
const route = useRoute()
const toast = useToast()
const console = useConsoleStore()
const organizationStore = useOrganizationStore()
const userStore = useUserStore()
const consoleApplicationId = import.meta.env.PPVT_CONSOLE_APPLICATION_ID ?? ''

const userViewMode = ref<'list' | 'detail'>('list')
const mfaConfigModalVisible = ref(false)
const currentMFAMethod = ref<MFAMethod>('totp')
const totpSetup = ref<unknown>(null)
const totpQRCodeDataURL = ref('')
const selectedUserIds = ref<string[]>([])
const showCreateUserForm = ref(false)
const userAdminResult = ref<unknown>(null)
const recoveryCodes = ref<{ codes?: string[] } | null>(null)

const userAdminForm = reactive({ password: '' })
const userPhoneInput = reactive<PhoneInputState>({ countryCode: '+86', localNumber: '' })
const userUpdatePhoneInput = reactive<PhoneInputState>({ countryCode: '+86', localNumber: '' })
const mfaSettingForm = reactive({ emailEnabled: 'disabled', smsEnabled: 'disabled' })
const totpVerifyForm = reactive({ enrollmentId: '', code: '' })

const phoneCountryOptions = [
  { value: '+86', text: '+86 中国' },
  { value: '+852', text: '+852 中国香港' },
  { value: '+853', text: '+853 中国澳门' },
  { value: '+886', text: '+886 中国台湾' },
  { value: '+81', text: '+81 日本' },
  { value: '+82', text: '+82 韩国' },
  { value: '+1', text: '+1 美国/加拿大' },
  { value: '+44', text: '+44 英国' },
  { value: '+49', text: '+49 德国' },
  { value: '+33', text: '+33 法国' },
  { value: '+65', text: '+65 新加坡' },
  { value: '+60', text: '+60 马来西亚' },
  { value: '+61', text: '+61 澳大利亚' }
]

const userAssignableRoles = computed(() => userStore.roles.filter((item: any) => item.type === 'user'))
const currentUserRecord = computed(() => {
  if (userStore.userDetail?.user?.id === userStore.selectedUserId) {
    return userStore.userDetail.user
  }
  return userStore.users.find((item: any) => item.id === userStore.selectedUserId)
})
const pendingTotpEnrollmentId = computed(() => (totpSetup.value as { enrollmentId?: string } | null)?.enrollmentId || '')
const pendingTotpProvisioningUri = computed(() => (totpSetup.value as { provisioningUri?: string } | null)?.provisioningUri || '')
const pendingTotpManualEntryKey = computed(() => (totpSetup.value as { manualEntryKey?: string } | null)?.manualEntryKey || '')
const generatedRecoveryCodeList = computed(() => recoveryCodes.value?.codes || [])
const activeTOTPEnrollments = computed(() => (userStore.userDetail?.mfaEnrollments || []).filter((item: any) => item.method === 'totp'))
const emailCodeEnrollment = computed(() => (userStore.userDetail?.mfaEnrollments || []).find((item: any) => item.method === 'email_code'))
const smsCodeEnrollment = computed(() => (userStore.userDetail?.mfaEnrollments || []).find((item: any) => item.method === 'sms_code'))
const u2fSecureKeys = computed(() => (userStore.userDetail?.secureKeys || []).filter((item: any) => item.u2fEnable))

watchEffect(() => {
  if (userViewMode.value === 'detail') {
    console.setPageHeader('', '')
    return
  }
  console.setPageHeader('用户', '管理用户、通行密钥、身份验证器、备用验证码与管理员动作。')
})

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
      userStore.clearUserState()
      userAdminResult.value = null
      recoveryCodes.value = null
      userViewMode.value = 'list'
      return
    }
    userStore.userForm.organizationId = nextOrganizationId
    userStore.externalBindingForm.organizationId = nextOrganizationId
    userViewMode.value = routeName === 'console-user-detail' ? 'detail' : 'list'
    await Promise.all([loadUsers(), userStore.loadRoles()])
    if (typeof routeUserId === 'string' && routeUserId) {
      userStore.setSelectedUserId(routeUserId)
      await loadUserDetail(routeUserId)
      return
    }
    if (!userStore.selectedUserId && userStore.users.length) {
      userStore.setSelectedUserId(userStore.users[0].id)
    }
    if (userViewMode.value === 'detail') {
      await loadUserDetail(userStore.selectedUserId)
    }
  },
  { immediate: true }
)

watch(
  () => userStore.users,
  (items) => {
    const userIds = new Set(items.map((item: any) => item.id))
    selectedUserIds.value = selectedUserIds.value.filter((id) => userIds.has(id))
  },
  { immediate: true, deep: true }
)

async function loadUsers() {
  await userStore.loadUsers()
  if (userStore.userUpdateForm.phoneNumber !== undefined) {
    syncPhoneInput(userUpdatePhoneInput, userStore.userUpdateForm.phoneNumber, phoneCountryOptions.map((item) => item.value))
  }
}

async function loadUserDetail(userId = userStore.selectedUserId) {
  await userStore.loadUserDetail(userId)
  syncPhoneInput(userUpdatePhoneInput, userStore.userUpdateForm.phoneNumber, phoneCountryOptions.map((item) => item.value))
}

function toggleAllUsers(checked: boolean) {
  selectedUserIds.value = checked ? userStore.users.map((item: any) => item.id) : []
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
  if (userStore.selectedUserId) {
    return {
      name: 'console-user-detail',
      params: {
        organizationId: console.currentOrganizationId,
        userId: userStore.selectedUserId
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
  userStore.setSelectedUserId(user.id)
  void loadUserDetail(user.id)
  void router.push({
    name: 'console-user-detail',
    params: {
      organizationId: console.currentOrganizationId || organizationStore.currentOrganization?.id || '',
      userId: user.id ?? ''
    }
  })
}

function backToUserList() {
  userViewMode.value = 'list'
  void router.push({
    name: 'console-user-list',
    params: {
      organizationId: console.currentOrganizationId || organizationStore.currentOrganization?.id || ''
    }
  })
}

async function deleteSelectedUsers(userIds: string[]) {
  if (!userIds.length) {
    return
  }
  await withFeedback(async () => {
    await userStore.deleteUsers({ userIds })
    await loadUsers()
    await loadUserDetail()
  })
}

async function deleteSingleUser(userId: string) {
  await withFeedback(async () => {
    await userStore.deleteUsers({ userId })
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
    await userStore.createUser({
      ...userStore.userForm,
      phoneNumber: composePhoneNumber(userPhoneInput),
      roles: splitRoleLabels(userStore.userForm.roleLabels)
    })
    resetPhoneInput(userPhoneInput)
    await loadUsers()
  })
}

async function submitUserCreateFromList() {
  userStore.userForm.phoneNumber = composePhoneNumber(userPhoneInput)
  await createUser()
  showCreateUserForm.value = false
}

async function updateUser() {
  await withFeedback(async () => {
    await userStore.updateUser({
      ...userStore.userUpdateForm,
      phoneNumber: composePhoneNumber(userUpdatePhoneInput),
      roles: userStore.userRoleAssignments
    })
    await loadUsers()
    await loadUserDetail()
  })
}

async function createExternalBinding() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    userStore.externalBindingForm.userId = userStore.externalBindingForm.userId || userStore.selectedUserId
    await userStore.createExternalBinding()
    await loadUserDetail()
  })
}

async function registerSecureKey(purpose: 'webauthn' | 'u2f' = 'webauthn') {
  const userId = userStore.selectedUserId
  if (!userId) {
    return
  }
  await withFeedback(async () => {
    const begin = await userStore.beginRegisterSecureKey(purpose)
    if (!begin) {
      return
    }
    const credential = await navigator.credentials.create({
      publicKey: normalizeCreationOptions(begin.options)
    })
    if (!credential) {
      throw new Error('Secure key registration was cancelled')
    }
    await userStore.finishRegisterSecureKey(begin.challengeId, serializeCredential(credential as PublicKeyCredential))
  })
}

async function enrollTotp() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    totpSetup.value = await userStore.enrollUserTotp(consoleApplicationId)
    totpVerifyForm.enrollmentId = pendingTotpEnrollmentId.value
  })
}

async function verifyTotpEnrollment() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    await userStore.verifyUserTotp(pendingTotpEnrollmentId.value, totpVerifyForm.code)
    totpSetup.value = null
    totpVerifyForm.enrollmentId = ''
    totpVerifyForm.code = ''
  })
}

async function generateRecoveryCodes() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    recoveryCodes.value = await userStore.generateRecoveryCodes() as { codes?: string[] } | null
  })
}

async function saveMFAEmailSetting() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    await userStore.updateUserMfaMethod('email_code', mfaSettingForm.emailEnabled === 'active')
    mfaConfigModalVisible.value = false
  })
}

async function saveMFASMSSetting() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    await userStore.updateUserMfaMethod('sms_code', mfaSettingForm.smsEnabled === 'active')
    mfaConfigModalVisible.value = false
  })
}

async function toggleInlineMFAMethod(method: MFAMethod, enabled: boolean) {
  if (method !== 'email_code' && method !== 'sms_code') {
    return
  }
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    await userStore.updateUserMfaMethod(method, enabled)
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
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    await userStore.deleteUserMfaEnrollment('totp')
    totpSetup.value = null
    totpVerifyForm.enrollmentId = ''
    totpVerifyForm.code = ''
    mfaConfigModalVisible.value = false
  })
}

async function deleteSecureKey(credentialId: string) {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    await userStore.deleteSecureKey(credentialId)
  })
}

async function toggleWebAuthnLogin(enabled: boolean) {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    await userStore.updateUserMfaMethod('webauthn', enabled)
  }, enabled ? '已启用通行密钥登录' : '已关闭通行密钥登录')
}

async function resetUserPassword() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await userStore.resetUserPassword(userAdminForm.password)
    userAdminForm.password = ''
  })
}

async function resetUserUkid() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await userStore.resetUserUkid()
  })
}

async function disableUser() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await userStore.disableUser()
  })
}

async function enableUser() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await userStore.enableUser()
  })
}

async function deleteExternalBinding(bindingId: string) {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    await userStore.deleteExternalBinding(bindingId)
    await loadUserDetail()
  })
}

async function untrustManagedDevice(deviceId: string) {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    await userStore.untrustManagedDevice(deviceId)
  })
}

async function revokeAllUserSessions() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await userStore.revokeAllUserSessions()
  })
}

async function rotateUserToken() {
  if (!userStore.selectedUserId) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await userStore.rotateUserToken()
  })
}

function toggleUserRole(roleName: string, checked: boolean) {
  if (checked) {
    if (!userStore.userRoleAssignments.includes(roleName)) {
      userStore.userRoleAssignments = [...userStore.userRoleAssignments, roleName]
    }
    return
  }
  userStore.userRoleAssignments = userStore.userRoleAssignments.filter((item) => item !== roleName)
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
