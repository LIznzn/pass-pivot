import { RequestError } from './request'

const errorAliases: Record<string, string> = {
  'captcha is required': 'authn.captcha_required',
  'invalid captcha': 'authn.captcha_invalid',
  'invalid JSON body': 'authn.invalid_json_body',
  'invalid credentials': 'authn.invalid_credentials',
  'invalid TOTP code': 'authn.mfa_code_invalid',
  'invalid challenge code': 'authn.mfa_code_invalid',
  'invalid recovery code': 'authn.recovery_code_invalid',
  'invalid password reset code': 'authn.password_reset_code_invalid',
  'password reset contact does not match': 'authn.password_reset_contact_invalid',
  'no reachable password reset target': 'authn.password_reset_target_unavailable',
  'webauthn challenge not found': 'authn.webauthn_challenge_not_found',
  'webauthn challenge expired': 'authn.webauthn_challenge_expired'
}

function normalizeErrorKey(input: string) {
  const normalized = String(input).trim()
  return errorAliases[normalized] || normalized
}

export function localizeError(input: string | undefined, text: { errorText: Record<string, string>; errorFallback: string }) {
  if (!input) {
    return ''
  }
  const normalized = normalizeErrorKey(input)
  if (!normalized) {
    return ''
  }
  return text.errorText[normalized] || normalized || text.errorFallback
}

export function formatRequestError(error: unknown, localize: (input?: string) => string) {
  if (error instanceof RequestError) {
    return localize(error.code || error.message)
  }
  if (error instanceof Error) {
    return localize(error.message)
  }
  return localize(String(error))
}
