/// <reference types="vite/client" />

type OAuthBootstrapTarget = {
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

type OAuthBootstrapMethodOption = {
  value: string
  label: string
}

type OAuthBootstrapCurrentUser = {
  id: string
  username: string
  name: string
  email: string
  phoneNumber: string
}

type OAuthBootstrapAPIConfig = {
  webauthnLoginBegin: string
  webauthnLoginEnd: string
  sessionU2fBegin: string
  sessionU2fFinish: string
  mfaChallenge: string
  captchaRefresh: string
}

type OAuthBootstrapCaptcha = {
  provider: string
  client_key?: string
  imageDataUrl?: string
  challengeToken?: string
}

type OAuthBootstrapPayload = {
  stage: 'login' | 'account' | 'confirmation' | 'mfa'
  title: string
  error?: string
  authorizeReturnUrl: string
  target: OAuthBootstrapTarget
  currentUser?: OAuthBootstrapCurrentUser
  applicationId: string
  loginAction: string
  accountAction: string
  switchAccountAction: string
  confirmAction: string
  mfaAction: string
  secondFactorMethod?: string
  mfaOptions: OAuthBootstrapMethodOption[]
  captcha?: OAuthBootstrapCaptcha
  api: OAuthBootstrapAPIConfig
}

type DeviceBootstrapCurrentUser = {
  id: string
  username: string
  name: string
  email: string
  phoneNumber: string
}

type DeviceBootstrapPayload = {
  title: string
  status: 'pending' | 'done' | 'error'
  error?: string
  userCode: string
  applicationName?: string
  organizationName?: string
  currentUser?: DeviceBootstrapCurrentUser
  loginAction: string
  confirmAction: string
  denied?: boolean
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
  interface Window {
    __PPVT_OAUTH_BOOTSTRAP__?: OAuthBootstrapPayload
  }
  interface Window {
    __PPVT_DEVICE_BOOTSTRAP__?: DeviceBootstrapPayload
  }
}

export {}
