import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import { useToast } from 'bootstrap-vue-next'
import QRCode from 'qrcode'
import { normalizeCreationOptions, serializeCredential } from '@shared/utils/webauthn'
import { notifyToast } from '@shared/utils/notify'
import {
  beginSecureKeyRegistration,
  createExternalIdentityBinding,
  deleteExternalIdentityBinding,
  deleteMFAEnrollment,
  deleteSecureKey,
  enrollTotp,
  finishSecureKeyRegistration,
  generateRecoveryCodes as requestGenerateRecoveryCodes,
  queryProfile,
  queryRecoveryCodes,
  queryUserDetail,
  queryUserSetting,
  untrustDevice as requestUntrustDevice,
  updateMFAMethod,
  updateProfile,
  updateSecureKey,
  updateUserSetting,
  verifyTotp as requestVerifyTotp
} from '@/api/user'
import { formatPortalError } from '@/utils/auth-error'
import { inferDeviceName } from '@/utils/portal'

type DetailData = {
  user: {
    id: string
    username: string
    name: string
    email: string
    phoneNumber: string
  }
  passwordCredential: boolean
  secureKeys: Array<{ id: string; identifier: string; publicKeyId: string; webauthnEnable: boolean; u2fEnable: boolean; createdAt: string }>
  bindings: Array<{ id: string; providerName: string; externalIdpId: string; issuer: string; subject: string; createdAt: string }>
  externalIdps: Array<{ id: string; name: string; issuer: string }>
  mfaEnrollments: Array<{ id: string; method: string; label: string; target: string; status: string; lastUsedAt?: string }>
  devices: Array<{ id: string; userAgent?: string; online: boolean; trusted: boolean; lastLoginIp?: string; ipLocation?: string; firstLoginAt?: string; lastLoginAt?: string; deviceFingerprint?: string }>
  recoverySummary: { available: number; lastGeneratedAt?: string }
}

type SettingData = {
  session?: { applicationId?: string }
}

type ProfileData = {
  username: string
  name: string
  email: string
  phoneNumber: string
}

export const usePortalStore = defineStore('portal', () => {
  const initialized = ref(false)
  const loading = ref(false)
  const detail = ref<DetailData | null>(null)
  const setting = ref<SettingData | null>(null)
  const showMFAModal = ref(false)
  const showKeyModal = ref(false)
  const mfaModalType = ref<'totp' | 'u2f' | 'recovery_code' | ''>('')
  const totpCode = ref('')
  const totpQRCode = ref('')
  const recoveryCodes = ref<string[]>([])
  const keyNameDrafts = reactive<Record<string, string>>({})

  const profile = reactive({
    username: '',
    name: '',
    email: '',
    phoneNumber: ''
  })
  const passwordForm = reactive({
    currentPassword: '',
    newPassword: ''
  })
  const bindingForm = reactive({
    externalIdpId: '',
    issuer: '',
    subject: ''
  })
  const totpEnrollment = reactive({
    enrollmentId: '',
    provisioningUri: '',
    manualEntryKey: ''
  })

  const activeTotpEnrollment = computed(() => detail.value?.mfaEnrollments.find((item) => item.method === 'totp' && item.status === 'active') ?? null)
  const mfaEnrollment = computed(() => detail.value?.mfaEnrollments.find((item) => item.method === 'mfa' && item.status === 'active') ?? null)
  const webauthnEnrollment = computed(() => detail.value?.mfaEnrollments.find((item) => item.method === 'webauthn') ?? null)
  const loginSecureKeys = computed(() => (detail.value?.secureKeys ?? []).filter((item) => item.webauthnEnable))
  const u2fSecureKeys = computed(() => (detail.value?.secureKeys ?? []).filter((item) => item.u2fEnable))
  const allSecureKeys = computed(() => detail.value?.secureKeys ?? [])
  const deviceRows = computed(() => (detail.value?.devices ?? []).map((device) => ({
    id: device.id,
    label: inferDeviceName(device.userAgent),
    online: Boolean(device.online),
    trusted: Boolean(device.trusted),
    ipAddress: device.lastLoginIp || '',
    ipLocation: device.ipLocation || '',
    firstLoginAt: device.firstLoginAt,
    lastLoginAt: device.lastLoginAt,
    fingerprint: device.deviceFingerprint || ''
  })))
  const webauthnLoginEnabled = computed(() => webauthnEnrollment.value?.status === 'active' && loginSecureKeys.value.length > 0)
  const mfaEnabled = computed(() => Boolean(mfaEnrollment.value))
  const mfaRows = computed(() => {
    const enrollments = detail.value?.mfaEnrollments ?? []
    const u2fEnabled = enrollments.some((item) => item.method === 'u2f' && item.status === 'active') && u2fSecureKeys.value.length > 0
    const emailEnabled = enrollments.some((item) => item.method === 'email_code' && item.status === 'active')
    const smsEnabled = enrollments.some((item) => item.method === 'sms_code' && item.status === 'active')
    const totpEnabled = enrollments.some((item) => item.method === 'totp' && item.status === 'active')
    return [
      { id: 'email_code', label: '邮箱验证码', enabled: emailEnabled, summary: emailEnabled ? `目标：${profile.email || '未配置邮箱'}` : '使用邮箱接收验证码' },
      { id: 'sms_code', label: '手机验证码', enabled: smsEnabled, summary: smsEnabled ? `目标：${profile.phoneNumber || '未配置手机'}` : '使用手机接收验证码' },
      { id: 'totp', label: '身份验证器（TOTP）', enabled: totpEnabled, summary: totpEnabled ? '已配置身份验证器' : '使用身份验证器 App 生成动态验证码' },
      { id: 'u2f', label: '安全密钥', enabled: u2fEnabled, summary: u2fSecureKeys.value.length ? `已登记 ${u2fSecureKeys.value.length} 把安全密钥` : '当前没有可用于安全密钥验证的密钥' }
    ]
  })
  const configuredPrimaryMfaCount = computed(() => mfaRows.value.filter((item) => item.enabled).length)
  const recoveryCodeCount = computed(() => detail.value?.recoverySummary?.available ?? 0)
  const mfaSummaryText = computed(() => {
    if (!mfaEnabled.value) {
      return '开启后可配置主验证方式，并自动准备备用验证码。'
    }
    if (configuredPrimaryMfaCount.value === 0) {
      return `已生成 ${recoveryCodeCount.value} 个备用验证码，但尚未配置其他验证方式；当前登录不会触发多因素验证。`
    }
    return `已配置 ${configuredPrimaryMfaCount.value} 种主验证方式，备用验证码剩余 ${recoveryCodeCount.value} 个。`
  })
  const mfaModalTitle = computed(() => {
    if (mfaModalType.value === 'totp') return '身份验证器（TOTP）'
    if (mfaModalType.value === 'u2f') return '安全密钥'
    if (mfaModalType.value === 'recovery_code') return '备用验证码'
    return '多因素验证'
  })

  function showToast(
    message: string,
    variant: 'success' | 'danger',
    options: {
      source: string
      trigger?: string
      error?: unknown
      metadata?: Record<string, unknown>
    }
  ) {
    try {
      const toast = useToast()
      notifyToast({
        toast,
        message,
        variant,
        source: options.source,
        trigger: options.trigger,
        error: options.error,
        metadata: options.metadata
      })
    } catch (error) {
      const payload = {
        source: options.source,
        trigger: options.trigger,
        variant,
        message,
        error: options.error,
        metadata: options.metadata,
        toastError: error
      }
      if (variant === 'danger') {
        console.error('[portal-toast-fallback]', payload)
        return
      }
      console.info('[portal-toast-fallback]', payload)
    }
  }

  function resetKeyNameDrafts() {
    Object.keys(keyNameDrafts).forEach((key) => delete keyNameDrafts[key])
    for (const item of allSecureKeys.value) {
      keyNameDrafts[item.id] = item.identifier || ''
    }
  }

  function syncBindingIssuer() {
    const current = detail.value?.externalIdps.find((item) => item.id === bindingForm.externalIdpId)
    bindingForm.issuer = current?.issuer ?? ''
  }

  function openMFAModal(type: 'totp' | 'u2f' | 'recovery_code') {
    mfaModalType.value = type
    showMFAModal.value = true
  }

  function openKeyModal() {
    showKeyModal.value = true
  }

  function closeMFAModal() {
    showMFAModal.value = false
  }

  function closeKeyModal() {
    showKeyModal.value = false
  }

  async function loadPortalData() {
    loading.value = true
    try {
      const [profileResponse, detailResponse, settingResponse] = await Promise.all([
        queryProfile({}),
        queryUserDetail({}),
        queryUserSetting({})
      ])
      profile.username = profileResponse.username || ''
      profile.name = profileResponse.name || ''
      profile.email = profileResponse.email || ''
      profile.phoneNumber = profileResponse.phoneNumber || ''
      detail.value = detailResponse
      setting.value = settingResponse
      if (!bindingForm.externalIdpId && detailResponse.externalIdps.length) {
        bindingForm.externalIdpId = detailResponse.externalIdps[0].id
        bindingForm.issuer = detailResponse.externalIdps[0].issuer
      }
      resetKeyNameDrafts()
      initialized.value = true
    } catch (error) {
      showToast(formatPortalError(error), 'danger', {
        source: 'portal/store.loadPortalData',
        trigger: 'loadPortalData',
        error
      })
      throw error
    } finally {
      loading.value = false
    }
  }

  async function initialize() {
    if (initialized.value) {
      return
    }
    await loadPortalData()
  }

  function reset() {
    initialized.value = false
    loading.value = false
    detail.value = null
    setting.value = null
    showMFAModal.value = false
    showKeyModal.value = false
    mfaModalType.value = ''
    totpCode.value = ''
    totpQRCode.value = ''
    recoveryCodes.value = []
    profile.username = ''
    profile.name = ''
    profile.email = ''
    profile.phoneNumber = ''
    passwordForm.currentPassword = ''
    passwordForm.newPassword = ''
    bindingForm.externalIdpId = ''
    bindingForm.issuer = ''
    bindingForm.subject = ''
    totpEnrollment.enrollmentId = ''
    totpEnrollment.provisioningUri = ''
    totpEnrollment.manualEntryKey = ''
    Object.keys(keyNameDrafts).forEach((key) => delete keyNameDrafts[key])
  }

  async function saveProfileAction() {
    await updateProfile({ ...profile })
    showToast('基本信息已保存', 'success', {
      source: 'portal/store.saveProfile',
      trigger: 'saveProfile'
    })
    await loadPortalData()
  }

  async function savePasswordAction() {
    await updateUserSetting({ ...passwordForm })
    passwordForm.currentPassword = ''
    passwordForm.newPassword = ''
    showToast('密码已更新', 'success', {
      source: 'portal/store.savePassword',
      trigger: 'savePassword'
    })
  }

  async function registerSecureKeyAction(purpose: 'webauthn' | 'u2f') {
    const begin = await beginSecureKeyRegistration({ purpose })
    const credential = await navigator.credentials.create({
      publicKey: normalizeCreationOptions(begin.options)
    })
    if (!credential) {
      return
    }
    await finishSecureKeyRegistration({
      challengeId: begin.challengeId,
      response: serializeCredential(credential as PublicKeyCredential)
    })
    showToast(purpose === 'webauthn' ? '通行密钥已注册' : '安全密钥已注册', 'success', {
      source: 'portal/store.registerSecureKey',
      trigger: 'registerSecureKey',
      metadata: { purpose }
    })
    await loadPortalData()
  }

  async function deleteSecureKeyAction(credentialId: string) {
    await deleteSecureKey({ credentialId })
    showToast('密钥已删除', 'success', {
      source: 'portal/store.deleteSecureKey',
      trigger: 'deleteSecureKey',
      metadata: { credentialId }
    })
    await loadPortalData()
  }

  async function updateSecureKeyAction(credentialId: string, identifier: string) {
    await updateSecureKey({ credentialId, identifier })
    showToast('密钥名称已更新', 'success', {
      source: 'portal/store.updateSecureKey',
      trigger: 'updateSecureKey',
      metadata: { credentialId }
    })
    await loadPortalData()
  }

  async function toggleWebAuthnLoginAction(enabled: boolean) {
    if (enabled && loginSecureKeys.value.length === 0) {
      openKeyModal()
      return
    }
    await updateMFAMethod({ method: 'webauthn', enabled })
    showToast(enabled ? '已启用通行密钥登录' : '已关闭通行密钥登录', 'success', {
      source: 'portal/store.toggleWebAuthnLogin',
      trigger: 'toggleWebAuthnLogin',
      metadata: { enabled }
    })
    await loadPortalData()
  }

  async function createBindingAction() {
    await createExternalIdentityBinding({ ...bindingForm })
    bindingForm.subject = ''
    showToast('账号绑定已新增', 'success', {
      source: 'portal/store.createBinding',
      trigger: 'createBinding'
    })
    await loadPortalData()
  }

  async function deleteBindingAction(bindingId: string) {
    await deleteExternalIdentityBinding({ bindingId })
    showToast('账号绑定已删除', 'success', {
      source: 'portal/store.deleteBinding',
      trigger: 'deleteBinding',
      metadata: { bindingId }
    })
    await loadPortalData()
  }

  async function toggleSimpleMFAAction(method: string, enabled: boolean) {
    if (method === 'u2f' && enabled && u2fSecureKeys.value.length === 0) {
      openKeyModal()
      return
    }
    await updateMFAMethod({ method, enabled })
    showToast(enabled ? '已开启' : '已关闭', 'success', {
      source: 'portal/store.toggleSimpleMFA',
      trigger: 'toggleSimpleMFA',
      metadata: { method, enabled }
    })
    await loadPortalData()
  }

  async function toggleMFAEnabledAction(enabled: boolean) {
    await updateMFAMethod({ method: 'mfa', enabled })
    if (enabled) {
      const result = await queryRecoveryCodes({})
      recoveryCodes.value = result.codes
      openMFAModal('recovery_code')
    } else {
      recoveryCodes.value = []
    }
    showToast(enabled ? '已更新多因素验证主开关，并已准备备用验证码' : '已关闭多因素验证', 'success', {
      source: 'portal/store.toggleMFAEnabled',
      trigger: 'toggleMFAEnabled',
      metadata: { enabled }
    })
    await loadPortalData()
  }

  async function loadRecoveryCodes() {
    const result = await queryRecoveryCodes({})
    recoveryCodes.value = result.codes
  }

  async function prepareMFAModal(type: 'totp' | 'u2f' | 'recovery_code') {
    openMFAModal(type)
    if (type === 'totp') {
      totpCode.value = ''
      totpEnrollment.enrollmentId = ''
      totpEnrollment.provisioningUri = ''
      totpEnrollment.manualEntryKey = ''
      totpQRCode.value = ''
    }
    if (type === 'recovery_code') {
      await loadRecoveryCodes()
    }
  }

  async function generateTotpAction() {
    const applicationId = setting.value?.session?.applicationId ?? ''
    const result = await enrollTotp({ applicationId })
    totpEnrollment.enrollmentId = result.enrollmentId
    totpEnrollment.provisioningUri = result.provisioningUri
    totpEnrollment.manualEntryKey = result.manualEntryKey
    totpQRCode.value = await QRCode.toDataURL(result.provisioningUri, { width: 180, margin: 1 })
  }

  async function verifyTotpAction() {
    if (!totpEnrollment.enrollmentId) {
      await generateTotpAction()
    }
    await requestVerifyTotp({
      enrollmentId: totpEnrollment.enrollmentId,
      code: totpCode.value
    })
    showToast('身份验证器已启用', 'success', {
      source: 'portal/store.verifyTotp',
      trigger: 'verifyTotp'
    })
    closeMFAModal()
    await loadPortalData()
  }

  async function disableTotpAction() {
    await deleteMFAEnrollment({ method: 'totp' })
    showToast('身份验证器已关闭', 'success', {
      source: 'portal/store.disableTotp',
      trigger: 'disableTotp'
    })
    closeMFAModal()
    await loadPortalData()
  }

  async function generateRecoveryCodesAction() {
    const result = await requestGenerateRecoveryCodes({})
    recoveryCodes.value = result.codes
    showToast('已重新生成备用验证码', 'success', {
      source: 'portal/store.generateRecoveryCodes',
      trigger: 'generateRecoveryCodes'
    })
    await loadPortalData()
  }

  async function untrustDeviceAction(deviceId: string) {
    await requestUntrustDevice({ deviceId })
    showToast('设备已取消可信', 'success', {
      source: 'portal/store.untrustDevice',
      trigger: 'untrustDevice',
      metadata: { deviceId }
    })
    await loadPortalData()
  }

  return {
    initialized,
    loading,
    detail,
    setting,
    profile,
    passwordForm,
    bindingForm,
    showMFAModal,
    showKeyModal,
    mfaModalType,
    totpEnrollment,
    totpCode,
    totpQRCode,
    recoveryCodes,
    keyNameDrafts,
    activeTotpEnrollment,
    allSecureKeys,
    deviceRows,
    webauthnLoginEnabled,
    mfaEnabled,
    mfaRows,
    mfaSummaryText,
    mfaModalTitle,
    initialize,
    reset,
    loadPortalData,
    saveProfileAction,
    savePasswordAction,
    registerSecureKeyAction,
    deleteSecureKeyAction,
    updateSecureKeyAction,
    toggleWebAuthnLoginAction,
    syncBindingIssuer,
    createBindingAction,
    deleteBindingAction,
    toggleSimpleMFAAction,
    toggleMFAEnabledAction,
    openKeyModal,
    closeKeyModal,
    prepareMFAModal,
    closeMFAModal,
    generateTotpAction,
    verifyTotpAction,
    disableTotpAction,
    generateRecoveryCodesAction,
    untrustDeviceAction
  }
})
