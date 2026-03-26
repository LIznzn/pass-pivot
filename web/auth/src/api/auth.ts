import { requestPost } from '@/utils/request'

export function queryAuthContext(payload: any) {
  return requestPost<any>('/auth/api/context/query', payload)
}

export function createAuthorizeSession(payload: any) {
  return requestPost<any>('/auth/api/session/create', payload)
}

export function bootstrapPasswordReset(payload: any) {
  return requestPost<any>('/auth/api/password/reset/bootstrap', payload)
}

export function queryPasswordResetOptions(payload: any) {
  return requestPost<any>('/auth/api/password/reset/options', payload)
}

export function startPasswordReset(payload: any) {
  return requestPost<any>('/auth/api/password/reset/start', payload)
}

export function finishPasswordReset(payload: any) {
  return requestPost<any>('/auth/api/password/reset/finish', payload)
}

export function completeDeviceAuthorization(payload: any) {
  return requestPost<any>('/auth/api/device/complete', payload)
}

export function confirmAuthorizeSession(payload: any) {
  return requestPost<any>('/auth/api/session/confirm', payload)
}

export function verifyAuthorizeMFA(payload: any) {
  return requestPost<any>('/auth/api/session/verify_mfa', payload)
}

export function beginWebAuthnLogin(applicationId: string) {
  return requestPost<{ challengeId: string; options: unknown }>('/auth/api/webauthn/login/begin', { applicationId })
}

export function finishWebAuthnLogin(payload: { challengeId: string; response: unknown; applicationId: string }) {
  return requestPost<any>('/auth/api/webauthn/login/finish', payload)
}

export function sendMFAChallenge(method: string) {
  return requestPost<{ demoCode?: string }>('/auth/api/session/mfa_challenge/create', { method })
}

export function beginSessionU2F() {
  return requestPost<{ challengeId: string; options: unknown }>('/auth/api/session/u2f/begin', {})
}

export function finishSessionU2F(payload: { challengeId: string; response: unknown; trustDevice: boolean }) {
  return requestPost<any>('/auth/api/session/u2f/finish', payload)
}

export function refreshCaptcha() {
  return requestPost<{ imageDataUrl?: string; challengeToken?: string }>('/auth/api/captcha/refresh', {})
}
