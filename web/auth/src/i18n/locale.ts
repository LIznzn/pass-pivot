export type Locale = 'en' | 'ja' | 'chs' | 'cht'

export const localeOptions = [
  { value: 'en', label: 'English' },
  { value: 'ja', label: '日本語' },
  { value: 'chs', label: '简体中文' },
  { value: 'cht', label: '繁體中文' }
] as const

export function resolveInitialLocale(): Locale {
  if (typeof navigator !== 'undefined') {
    const language = navigator.language.toLowerCase()
    if (language.startsWith('ja')) return 'ja'
    if (language.startsWith('zh-hant') || language.startsWith('zh-tw') || language.startsWith('zh-hk') || language.startsWith('zh-mo')) return 'cht'
    if (language.startsWith('zh')) return 'chs'
  }
  return 'en'
}
