import { formatDateTime as formatSharedDateTime } from '@shared/utils/datetime'

export const portalSections = [
  { id: 'profile-basic', label: '基本信息' },
  { id: 'profile-login', label: '登录设置' },
  { id: 'profile-binding', label: '账号绑定' },
  { id: 'profile-mfa', label: '多因素验证' },
  { id: 'profile-device', label: '会话管理' }
] as const

export function formatDateTime(value?: string) {
  return formatSharedDateTime(value)
}

export function formatIPLine(ipAddress?: string, ipLocation?: string) {
  const ip = String(ipAddress || '').trim() || '-'
  const location = String(ipLocation || '').trim()
  return location ? `${ip} (${location})` : ip
}

export function inferDeviceName(userAgent?: string) {
  const source = String(userAgent || '').trim()
  if (!source) return '未知设备'
  const browser = source.includes('Edg/') ? 'Edge'
    : source.includes('Chrome/') && !source.includes('Edg/') ? 'Chrome'
    : source.includes('Firefox/') ? 'Firefox'
    : source.includes('Safari/') && !source.includes('Chrome/') ? 'Safari'
    : source.includes('MSIE') || source.includes('Trident/') ? 'Internet Explorer'
    : ''
  const os = source.includes('Windows NT') ? 'Windows'
    : source.includes('Mac OS X') || source.includes('Macintosh') ? 'macOS'
    : source.includes('Android') ? 'Android'
    : source.includes('iPhone') || source.includes('iPad') || source.includes('iOS') ? 'iOS'
    : source.includes('Linux') ? 'Linux'
    : ''
  if (!browser && !os) return source
  return browser && os ? `${browser} (${os})` : (browser || os)
}

export function keyCapabilityLabel(secureKey: {
  webauthnEnable: boolean
  u2fEnable: boolean
}) {
  if (secureKey.webauthnEnable && secureKey.u2fEnable) {
    return '支持通行密钥登录与安全密钥验证'
  }
  if (secureKey.webauthnEnable) {
    return '仅支持通行密钥登录'
  }
  if (secureKey.u2fEnable) {
    return '仅支持安全密钥验证'
  }
  return '能力未识别'
}
