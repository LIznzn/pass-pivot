import type { Locale } from './locale'
import { errorText, text } from './translation'

export type LocalizedText = Record<Locale, string>

interface LocaleTree {
  [key: string]: LocalizedText | LocaleTree
}

type LocaleNode = LocalizedText | LocaleTree
type TextKey = keyof typeof text

export type TranslationShape = {
  productTagline: string
  language: string
  securityLabel: string
  stageTitles: Record<'user_code' | 'device_review' | 'login' | 'account' | 'confirmation' | 'mfa', (appName: string) => string>
  stageHints: Record<'user_code' | 'device_review' | 'login' | 'account' | 'confirmation' | 'mfa', string>
  deviceAuthorizationCompleteTitle: string
  deviceAuthorizationCompleteHint: string
  deviceAuthorizationErrorHint: string
  accountTitle: string
  authorizeThisClient: string
  continueAsCurrentAccount: string
  logoutAndSwitchAccount: string
  userCode: string
  userCodePlaceholder: string
  submitUserCode: string
  deviceAuthorizationRequestTitle: string
  deviceAuthorizationRequestDevice: string
  deviceAuthorizationRequestIp: string
  deviceAuthorizationReviewTitle: string
  deviceAuthorizationReviewContinue: string
  deviceAuthorizationReviewCancel: string
  closePage: string
  identifier: string
  identifierPlaceholder: string
  password: string
  passwordPlaceholder: string
  signIn: string
  or: string
  continueWithProvider: (providerName: string) => string
  signInWithPasskey: string
  confirmationTitle: string
  confirmationItems: {
    trustedDevice: string
    futureSkip: string
    continueWithoutTrust: string
  }
  confirmTrustDevice: string
  confirmContinueWithoutTrust: string
  verificationMethod: string
  captcha: string
  captchaImageAlt: string
  captchaAnswer: string
  captchaAnswerPlaceholder: string
  verificationCode: string
  verificationCodePlaceholder: string
  verifyAndContinue: string
  sendVerificationCode: string
  useSecurityKey: string
  cancelLogin: string
  securedBy: string
  tos: string
  privacyPolicy: string
  developedBy: (appName: string, organizationName: string) => string
  challengeSent: string
  challengeSentWithDemoCode: (code: string) => string
  passkeyRequiresIdentifier: string
  mfaMethodLabels: Record<string, string>
  errorFallback: string
  errorText: Record<string, string>
}

function fmt(template: string, values: Record<string, string>) {
  return Object.entries(values).reduce(
    (output, [key, value]) => output.split(`{${key}}`).join(value),
    template
  )
}

function isLocalizedText(value: LocaleNode | undefined): value is LocalizedText {
  return Boolean(value) &&
    typeof value === 'object' &&
    'en' in value &&
    'ja' in value &&
    'chs' in value &&
    'cht' in value
}

function getTextNode(path: string): LocalizedText | LocaleTree | undefined {
  return path.split('.').reduce<LocaleNode | undefined>((current, segment) => {
    if (!current || isLocalizedText(current)) {
      return undefined
    }
    return current[segment]
  }, text as unknown as LocaleTree)
}

function pick(locale: Locale, value: LocalizedText) {
  return value[locale]
}

export function resolveLocaleText(key: TextKey | string, locale: Locale): string {
  const node = getTextNode(String(key))
  if (!isLocalizedText(node)) {
    throw new Error(`missing localized text: ${key}`)
  }
  return pick(locale, node)
}

export function formatLocaleText(key: TextKey | string, locale: Locale, values: Record<string, string>) {
  return fmt(resolveLocaleText(key, locale), values)
}

export function resolveLocaleRecord<T extends string>(record: Record<T, LocalizedText>, locale: Locale): Record<T, string> {
  return Object.fromEntries(
    Object.entries(record).map(([key, value]) => [key, pick(locale, value as LocalizedText)])
  ) as Record<T, string>
}

export function buildLocaleText(locale: Locale): TranslationShape {
  return {
    productTagline: resolveLocaleText('productTagline', locale),
    language: resolveLocaleText('language', locale),
    securityLabel: resolveLocaleText('securityLabel', locale),
    stageTitles: {
      user_code: () => resolveLocaleText('stageTitleUserCode', locale),
      device_review: () => resolveLocaleText('stageTitleDeviceReview', locale),
      login: (appName: string) => formatLocaleText('stageTitleLogin', locale, { appName }),
      account: () => resolveLocaleText('stageTitleAccount', locale),
      confirmation: () => resolveLocaleText('stageTitleConfirmation', locale),
      mfa: () => resolveLocaleText('stageTitleMfa', locale)
    },
    stageHints: {
      user_code: resolveLocaleText('stageHintUserCode', locale),
      device_review: resolveLocaleText('stageHintDeviceReview', locale),
      login: resolveLocaleText('stageHintLogin', locale),
      account: resolveLocaleText('stageHintAccount', locale),
      confirmation: resolveLocaleText('stageHintConfirmation', locale),
      mfa: resolveLocaleText('stageHintMfa', locale)
    },
    deviceAuthorizationCompleteTitle: resolveLocaleText('deviceAuthorizationCompleteTitle', locale),
    deviceAuthorizationCompleteHint: resolveLocaleText('deviceAuthorizationCompleteHint', locale),
    deviceAuthorizationErrorHint: resolveLocaleText('deviceAuthorizationErrorHint', locale),
    accountTitle: resolveLocaleText('accountTitle', locale),
    authorizeThisClient: resolveLocaleText('authorizeThisClient', locale),
    continueAsCurrentAccount: resolveLocaleText('continueAsCurrentAccount', locale),
    logoutAndSwitchAccount: resolveLocaleText('logoutAndSwitchAccount', locale),
    userCode: resolveLocaleText('userCode', locale),
    userCodePlaceholder: resolveLocaleText('userCodePlaceholder', locale),
    submitUserCode: resolveLocaleText('submitUserCode', locale),
    deviceAuthorizationRequestTitle: resolveLocaleText('deviceAuthorizationRequestTitle', locale),
    deviceAuthorizationRequestDevice: resolveLocaleText('deviceAuthorizationRequestDevice', locale),
    deviceAuthorizationRequestIp: resolveLocaleText('deviceAuthorizationRequestIp', locale),
    deviceAuthorizationReviewTitle: resolveLocaleText('deviceAuthorizationReviewTitle', locale),
    deviceAuthorizationReviewContinue: resolveLocaleText('deviceAuthorizationReviewContinue', locale),
    deviceAuthorizationReviewCancel: resolveLocaleText('deviceAuthorizationReviewCancel', locale),
    closePage: resolveLocaleText('closePage', locale),
    identifier: resolveLocaleText('identifier', locale),
    identifierPlaceholder: resolveLocaleText('identifierPlaceholder', locale),
    password: resolveLocaleText('password', locale),
    passwordPlaceholder: resolveLocaleText('passwordPlaceholder', locale),
    signIn: resolveLocaleText('signIn', locale),
    or: resolveLocaleText('or', locale),
    continueWithProvider: (providerName: string) => formatLocaleText('continueWithProvider', locale, { providerName }),
    signInWithPasskey: resolveLocaleText('signInWithPasskey', locale),
    confirmationTitle: resolveLocaleText('confirmationTitle', locale),
    confirmationItems: {
      trustedDevice: resolveLocaleText('confirmationTrustedDevice', locale),
      futureSkip: resolveLocaleText('confirmationFutureSkip', locale),
      continueWithoutTrust: resolveLocaleText('confirmationContinueWithoutTrust', locale)
    },
    confirmTrustDevice: resolveLocaleText('confirmTrustDevice', locale),
    confirmContinueWithoutTrust: resolveLocaleText('confirmContinueWithoutTrust', locale),
    verificationMethod: resolveLocaleText('verificationMethod', locale),
    captcha: resolveLocaleText('captcha', locale),
    captchaImageAlt: resolveLocaleText('captchaImageAlt', locale),
    captchaAnswer: resolveLocaleText('captchaAnswer', locale),
    captchaAnswerPlaceholder: resolveLocaleText('captchaAnswerPlaceholder', locale),
    verificationCode: resolveLocaleText('verificationCode', locale),
    verificationCodePlaceholder: resolveLocaleText('verificationCodePlaceholder', locale),
    verifyAndContinue: resolveLocaleText('verifyAndContinue', locale),
    sendVerificationCode: resolveLocaleText('sendVerificationCode', locale),
    useSecurityKey: resolveLocaleText('useSecurityKey', locale),
    cancelLogin: resolveLocaleText('cancelLogin', locale),
    securedBy: resolveLocaleText('securedBy', locale),
    tos: resolveLocaleText('tos', locale),
    privacyPolicy: resolveLocaleText('privacyPolicy', locale),
    developedBy: (appName: string, organizationName: string) => formatLocaleText('developedBy', locale, { appName, organizationName }),
    challengeSent: resolveLocaleText('challengeSent', locale),
    challengeSentWithDemoCode: (code: string) => formatLocaleText('challengeSentWithDemoCode', locale, { code }),
    passkeyRequiresIdentifier: resolveLocaleText('passkeyRequiresIdentifier', locale),
    mfaMethodLabels: resolveLocaleRecord(text.mfaMethodLabels, locale),
    errorFallback: resolveLocaleText('errorFallback', locale),
    errorText: resolveLocaleRecord(errorText, locale)
  }
}
