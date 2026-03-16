/// <reference types="vite/client" />

type OAuthBootstrapTarget = {
  organizationId: string
  organizationName: string
  projectId: string
  projectName: string
  applicationId: string
  applicationName: string
}

type OAuthBootstrapMethodOption = {
  value: string
  label: string
}

type OAuthBootstrapAPIConfig = {
  passkeyLoginBegin: string
  passkeyLoginEnd: string
  sessionU2fBegin: string
  sessionU2fFinish: string
  mfaChallenge: string
}

type OAuthBootstrapPayload = {
  stage: 'login' | 'confirmation' | 'mfa'
  title: string
  error?: string
  authorizeReturnUrl: string
  target: OAuthBootstrapTarget
  applicationId: string
  loginAction: string
  confirmAction: string
  mfaAction: string
  secondFactorMethod?: string
  mfaOptions: OAuthBootstrapMethodOption[]
  api: OAuthBootstrapAPIConfig
}

declare global {
  interface Window {
    __PPVT_OAUTH_BOOTSTRAP__?: OAuthBootstrapPayload
  }
}

export {}
