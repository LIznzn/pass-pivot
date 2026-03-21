import { requestPost } from '../../util/request'

export function queryUsers(payload: any) {
  return requestPost<{ items: any[] }>('/api/manage/v1/user/query', payload)
}

export function queryUserDetail(userId: string) {
  return requestPost<any>('/api/manage/v1/user/detail/query', { userId })
}

export function createUser(payload: any) {
  return requestPost('/api/manage/v1/user/create', payload)
}

export function updateUser(payload: any) {
  return requestPost('/api/manage/v1/user/update', payload)
}

export function deleteUsers(payload: { userId?: string; userIds?: string[] }) {
  return requestPost('/api/manage/v1/user/delete', payload)
}

export function createExternalIdentityBinding(payload: any) {
  return requestPost('/api/manage/v1/external_identity_binding/create', payload)
}

export function deleteExternalIdentityBinding(userId: string, bindingId: string) {
  return requestPost('/api/manage/v1/external_identity_binding/delete', { userId, bindingId })
}

export function beginRegisterSecureKey(userId: string, purpose?: 'webauthn' | 'u2f') {
  return requestPost<{ challengeId: string; options: any }>('/api/manage/v1/user/securekey/register/begin', { userId, purpose })
}

export function finishRegisterSecureKey(challengeId: string, response: unknown) {
  return requestPost('/api/manage/v1/user/securekey/register/finish', { challengeId, response })
}

export function deleteSecureKey(userId: string, credentialId: string) {
  return requestPost('/api/manage/v1/user/securekey/delete', { userId, credentialId })
}

export function updateSecureKey(userId: string, credentialId: string, identifier: string) {
  return requestPost('/api/manage/v1/user/securekey/update', { userId, credentialId, identifier })
}

export function enrollUserTotp(userId: string, applicationId: string) {
  return requestPost('/api/manage/v1/user/totp/enroll', { userId, applicationId })
}

export function verifyUserTotp(userId: string, enrollmentId: string, code: string) {
  return requestPost('/api/manage/v1/user/totp/verify', { userId, enrollmentId, code })
}

export function deleteUserMfaEnrollment(userId: string, method: string) {
  return requestPost('/api/manage/v1/user/mfa_enrollment/delete', { userId, method })
}

export function updateUserMfaMethod(userId: string, method: string, enabled: boolean) {
  return requestPost('/api/manage/v1/user/mfa_method/update', { userId, method, enabled })
}

export function generateUserRecoveryCodes(userId: string) {
  return requestPost('/api/manage/v1/user/recovery_code/generate', { userId })
}

export function queryUserRecoveryCodes(userId: string) {
  return requestPost<{ codes: string[] }>('/api/manage/v1/user/recovery_code/query', { userId })
}

export function resetUserPassword(userId: string, password: string) {
  return requestPost('/api/manage/v1/user/reset_password', { userId, password })
}

export function resetUserUkid(userId: string) {
  return requestPost('/api/manage/v1/user/reset_ukid', { userId })
}

export function disableUser(userId: string) {
  return requestPost('/api/manage/v1/user/disable', { userId })
}

export function enableUser(userId: string) {
  return requestPost('/api/manage/v1/user/enable', { userId })
}

export function untrustUserDevice(userId: string, deviceId: string) {
  return requestPost('/api/manage/v1/user/device/untrust', { userId, deviceId })
}

export function revokeAllUserSessions(userId: string) {
  return requestPost('/api/manage/v1/user/session/revoke_all', { userId })
}

export function rotateUserToken(userId: string) {
  return requestPost('/api/manage/v1/user/token/rotate', { userId })
}
