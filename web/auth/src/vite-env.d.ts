/// <reference types="vite/client" />

export type AuthTarget = {
  organizationId: string
  organizationName: string
  displayName: string
  organizationDisplayNames: Record<string, string>
  websiteUrl: string
  termsOfServiceUrl: string
  privacyPolicyUrl: string
  projectId: string
  projectName: string
  applicationId: string
  applicationName: string
  applicationDisplayNames: Record<string, string>
  externalIdps?: Array<{
    id: string
    organizationId: string
    protocol: string
    name: string
    issuer: string
  }>
}

export type AuthMethodOption = {
  value: string
  label: string
}

export type AuthCurrentUser = {
  id: string
  username: string
  name: string
  email: string
  phoneNumber: string
}

export type AuthCaptcha = {
  provider: string
  client_key?: string
  imageDataUrl?: string
  challengeToken?: string
}

export type AuthContextPayload = {
  action: string
  redirectTarget?: string
  flowType?: 'authorize' | 'device_code'
  stage: 'login' | 'account' | 'confirmation' | 'mfa' | 'done'
  resultStatus?: 'success' | 'error'
  resultMessage?: string
  error?: string
  authorizeReturnUrl: string
  target: AuthTarget
  currentUser?: AuthCurrentUser
  applicationId: string
  secondFactorMethod?: string
  mfaOptions: AuthMethodOption[]
  captcha?: AuthCaptcha
}

declare global {
  interface Window {
    turnstile?: {
      render: (container: HTMLElement, options: {
        sitekey: string
        theme?: 'light' | 'dark' | 'auto'
        callback?: (token: string) => void
        'before-interactive-callback'?: () => void
        'expired-callback'?: () => void
        'error-callback'?: () => void
      }) => string
      reset: (widgetId?: string) => void
    }
  }
  interface Window {
    grecaptcha?: {
      render: (container: HTMLElement, options: {
        sitekey: string
        callback?: (token: string) => void
        'expired-callback'?: () => void
        'error-callback'?: () => void
      }) => number
      reset: (widgetId?: number) => void
    }
  }
  interface Window {
    __ppvtRecaptchaOnload?: () => void
  }
}

export {}
