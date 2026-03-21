/// <reference types="vite/client" />

type OAuthBootstrapTarget = {
  organizationId: string
  organizationName: string
  displayName: string
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
  api: OAuthBootstrapAPIConfig
}

declare global {
  interface Window {
    __PPVT_OAUTH_BOOTSTRAP__?: OAuthBootstrapPayload
  }
}

export {}
