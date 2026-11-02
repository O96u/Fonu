import { normalizeCountryKey } from './worldMapCoords'

/** Dashboard / analytics: foreign geo labels show country only; legacy labels are trimmed. */
export function displayGeoLabel(geo: string): string {
  const raw = geo.trim()
  if (!raw) return ''

  const headRaw = raw.split('·')[0]?.trim() ?? raw
  const normalizedHead = normalizeCountryKey(headRaw)
  if (normalizedHead !== headRaw && /^[\u4e00-\u9fff]/.test(normalizedHead)) {
    return normalizedHead
  }

  const parts = raw.split('·').map((s) => s.trim()).filter(Boolean)
  if (parts.length <= 1) return normalizeCountryKey(raw)

  const head = normalizeCountryKey(parts[0])
  if (/[A-Za-z]/.test(raw)) return head

  const foreignCountryHints = ['国', '香港', '澳门', '台湾']
  if (foreignCountryHints.some((h) => head.includes(h)) && head !== '中国') {
    return head
  }

  return raw
}

export function displayRegionName(region: string): string {
  const p = normalizeCountryKey(region.trim())
  if (!p) return '—'
  const suffixes = ['特别行政区', '维吾尔自治区', '壮族自治区', '回族自治区', '自治区', '省', '市']
  for (const s of suffixes) {
    if (p.endsWith(s)) return p.slice(0, -s.length) || p
  }
  return p
}
