import type { CertificateRecord } from '../api/types'

/** Matches Go certificate.RecordStorageNames / StorageNames. */
export function certStorageNames(row: CertificateRecord): string[] {
  let domains = row.domains?.length ? [...row.domains] : [row.domain]
  domains = domains.map((d) => d.trim().toLowerCase()).filter(Boolean)
  if (domains.length === 0 && row.domain) {
    const domain = row.domain.trim().toLowerCase()
    domains = row.wildcard ? [domain, `*.${domain}`] : [domain]
  }
  const seen = new Set<string>()
  const names: string[] = []
  for (const domain of domains) {
    let name = domain
    if (domain.startsWith('*.')) {
      name = `wildcard.${domain.slice(2)}`
    }
    if (seen.has(name)) continue
    seen.add(name)
    names.push(name)
  }
  return names
}

export function certSourceDir(certsRoot: string, row: CertificateRecord): string {
  const root = certsRoot.replace(/[/\\]+$/, '')
  if (!root) return ''
  const names = certStorageNames(row)
  return names[0] ? `${root}/${names[0]}` : root
}
