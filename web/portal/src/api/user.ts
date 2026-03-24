import { requestPost } from '@/utils/request'

export function queryProfile(payload: any) {
  return requestPost<any>('/api/user/v1/profile/query', payload)
}

export function updateProfile(payload: any) {
  return requestPost<any>('/api/user/v1/profile/update', payload)
}

export function queryUserDetail(payload: any) {
  return requestPost<any>('/api/user/v1/detail/query', payload)
}

export function queryUserSetting(payload: any) {
  return requestPost<any>('/api/user/v1/setting/query', payload)
}

export function updateUserSetting(payload: any) {
  return requestPost<any>('/api/user/v1/setting/update', payload)
}

export function beginSecureKeyRegistration(payload: any) {
  return requestPost<any>('/api/user/v1/securekey/register/begin', payload)
}

export function finishSecureKeyRegistration(payload: any) {
  return requestPost<any>('/api/user/v1/securekey/register/finish', payload)
}

export function updateSecureKey(payload: any) {
  return requestPost<any>('/api/user/v1/securekey/update', payload)
}

export function deleteSecureKey(payload: any) {
  return requestPost<any>('/api/user/v1/securekey/delete', payload)
}

export function updateMFAMethod(payload: any) {
  return requestPost<any>('/api/user/v1/mfa_method/update', payload)
}

export function enrollTotp(payload: any) {
  return requestPost<any>('/api/user/v1/totp/enroll', payload)
}

export function verifyTotp(payload: any) {
  return requestPost<any>('/api/user/v1/totp/verify', payload)
}

export function deleteMFAEnrollment(payload: any) {
  return requestPost<any>('/api/user/v1/mfa_enrollment/delete', payload)
}

export function queryRecoveryCodes(payload: any) {
  return requestPost<any>('/api/user/v1/recovery_code/query', payload)
}

export function generateRecoveryCodes(payload: any) {
  return requestPost<any>('/api/user/v1/recovery_code/generate', payload)
}

export function createExternalIdentityBinding(payload: any) {
  return requestPost<any>('/api/user/v1/external_identity_binding/create', payload)
}

export function deleteExternalIdentityBinding(payload: any) {
  return requestPost<any>('/api/user/v1/external_identity_binding/delete', payload)
}

export function untrustDevice(payload: any) {
  return requestPost<any>('/api/user/v1/device/untrust', payload)
}
