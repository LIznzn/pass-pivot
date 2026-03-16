function decodeBase64Url(value: string): ArrayBuffer {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized + '='.repeat((4 - (normalized.length % 4)) % 4)
  const binary = atob(padded)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes.buffer
}

function encodeBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (const byte of bytes) {
    binary += String.fromCharCode(byte)
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

export function normalizeCreationOptions(options: any): PublicKeyCredentialCreationOptions {
  return {
    ...options.publicKey,
    challenge: decodeBase64Url(options.publicKey.challenge),
    user: {
      ...options.publicKey.user,
      id: decodeBase64Url(options.publicKey.user.id)
    },
    excludeCredentials: (options.publicKey.excludeCredentials ?? []).map((item: any) => ({
      ...item,
      id: decodeBase64Url(item.id)
    }))
  }
}

export function normalizeRequestOptions(options: any): PublicKeyCredentialRequestOptions {
  return {
    ...options.publicKey,
    challenge: decodeBase64Url(options.publicKey.challenge),
    allowCredentials: (options.publicKey.allowCredentials ?? []).map((item: any) => ({
      ...item,
      id: decodeBase64Url(item.id)
    }))
  }
}

export function serializeCredential(credential: PublicKeyCredential): Record<string, unknown> {
  const response = credential.response as AuthenticatorAttestationResponse | AuthenticatorAssertionResponse
  const base: Record<string, unknown> = {
    id: credential.id,
    rawId: encodeBase64Url(credential.rawId),
    type: credential.type,
    response: {}
  }

  if ('attestationObject' in response) {
    base.response = {
      clientDataJSON: encodeBase64Url(response.clientDataJSON),
      attestationObject: encodeBase64Url(response.attestationObject)
    }
    return base
  }

  base.response = {
    clientDataJSON: encodeBase64Url(response.clientDataJSON),
    authenticatorData: encodeBase64Url(response.authenticatorData),
    signature: encodeBase64Url(response.signature),
    userHandle: response.userHandle ? encodeBase64Url(response.userHandle) : null
  }
  return base
}
