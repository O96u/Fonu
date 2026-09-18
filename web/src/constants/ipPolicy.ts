/** Mirrors internal/proxy.DefaultPrivateCIDRs — always exempt from china_only / global allow. */
export const DEFAULT_PRIVATE_CIDRS = [
  '10.0.0.0/8',
  '127.0.0.0/8',
  '172.16.0.0/12',
  '192.168.0.0/16',
  '::1/128',
  'fc00::/7',
]

export function mergeIPList(defaults: string[], custom: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const cidr of [...defaults, ...custom]) {
    const value = cidr.trim()
    if (!value || seen.has(value)) continue
    seen.add(value)
    out.push(value)
  }
  return out
}

export function stripDefaultIPList(defaults: string[], list: string[]): string[] {
  const defaultSet = new Set(defaults)
  return list.map((item) => item.trim()).filter((item) => item && !defaultSet.has(item))
}
