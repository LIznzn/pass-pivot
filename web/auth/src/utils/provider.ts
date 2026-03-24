export function resolveProviderGlyph(name?: string) {
  const value = String(name || '').trim()
  if (!value) {
    return 'I'
  }
  return value.slice(0, 1).toUpperCase()
}
