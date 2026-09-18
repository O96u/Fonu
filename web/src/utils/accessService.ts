import type { AccessLogEntry, ProxyHost, ProxyRule } from '../api/types'

export type ProxyBinding = {
  name: string
  upstream: string
  host: string
  port: number
}

function bindingKey(host: string, port: number): string {
  return `${host.toLowerCase()}:${port}`
}

function normalizeAccessHost(domain: string): string {
  const raw = domain.trim().toLowerCase()
  if (!raw) return ''
  if (raw.startsWith('[')) {
    const end = raw.indexOf(']')
    if (end > 0) return raw.slice(0, end + 1)
  }
  if (raw.includes(':') && !raw.includes(']')) {
    const idx = raw.lastIndexOf(':')
    const maybePort = Number.parseInt(raw.slice(idx + 1), 10)
    if (idx > 0 && maybePort > 0 && maybePort <= 65535) {
      return raw.slice(0, idx)
    }
  }
  return raw
}

export function buildProxyBindingIndex(rules: ProxyRule[]): Map<string, ProxyBinding> {
  const index = new Map<string, ProxyBinding>()
  for (const rule of rules) {
    const fallbackName = rule.domain?.trim() || rule.upstream?.trim() || `规则 #${rule.id}`
    const name = rule.name?.trim() || fallbackName
    const hosts: ProxyHost[] = rule.hosts?.length
      ? rule.hosts
      : rule.domain
        ? [{ id: 0, hostname: rule.domain }]
        : []
    for (const host of hosts) {
      const hostname = host.hostname?.trim()
      if (!hostname) continue
      const port = host.listen_port ?? rule.listen_port
      if (port <= 0) continue
      index.set(bindingKey(hostname, port), {
        name,
        upstream: rule.upstream,
        host: hostname.toLowerCase(),
        port,
      })
    }
  }
  return index
}

function formatUpstreamShort(upstream: string): string {
  return upstream.replace(/^https?:\/\//i, '')
}

/** Raw host:port / domain / IP shown when no proxy rule matches. */
export function formatAccessFallback(entry: AccessLogEntry): string {
  const rawDomain = entry.domain?.trim()
  const host = normalizeAccessHost(entry.domain)
  const port = entry.server_port ?? 0

  if (rawDomain?.includes(':') && (rawDomain.includes(']') || rawDomain.lastIndexOf(':') > rawDomain.lastIndexOf(']'))) {
    return rawDomain
  }
  if (rawDomain?.includes(':') && !rawDomain.includes(']')) {
    const idx = rawDomain.lastIndexOf(':')
    const maybePort = Number.parseInt(rawDomain.slice(idx + 1), 10)
    if (idx > 0 && maybePort > 0 && maybePort <= 65535) {
      return rawDomain
    }
  }

  if (host && port > 0) return `${host}:${port}`
  if (rawDomain) return rawDomain
  if (host) return host
  return entry.client_ip?.trim() || '—'
}

export function resolveAccessServiceLabel(
  entry: AccessLogEntry,
  index: Map<string, ProxyBinding>,
): string {
  const host = normalizeAccessHost(entry.domain)
  const port = entry.server_port ?? 0

  // 域名 + 端口（同域多端口）
  if (host && port > 0) {
    const hit = index.get(bindingKey(host, port))
    if (hit) return hit.name
  }

  // 仅域名（单规则或同域单端口）
  if (host) {
    const matches = [...index.values()].filter((binding) => binding.host === host)
    if (matches.length === 1) return matches[0].name
    if (matches.length > 1 && port > 0) {
      const byPort = matches.find((binding) => binding.port === port)
      if (byPort) return byPort.name
    }
  }

  // 仅 IP / 域名直连、无 Host 匹配时走 upstream
  const upstream = entry.upstream?.trim()
  if (upstream && upstream !== '-') {
    return formatUpstreamShort(upstream)
  }

  return formatAccessFallback(entry)
}

export function accessServiceTooltip(entry: AccessLogEntry, index: Map<string, ProxyBinding>): string {
  const host = normalizeAccessHost(entry.domain)
  const port = entry.server_port ?? 0
  const label = resolveAccessServiceLabel(entry, index)
  const parts: string[] = [label]

  const requestHost = formatAccessFallback(entry)
  if (requestHost !== label) {
    parts.push(`请求 ${requestHost}`)
  } else if (host) {
    parts.push(port > 0 ? `${host}:${port}` : host)
  }

  const binding = host && port > 0 ? index.get(bindingKey(host, port)) : undefined
  if (binding?.upstream) {
    parts.push(`→ ${binding.upstream}`)
  } else if (entry.upstream && entry.upstream !== '-') {
    parts.push(`→ ${entry.upstream}`)
  }
  if (entry.client_ip) parts.push(`来自 ${entry.client_ip}`)
  if (entry.path) parts.push(entry.path)
  return parts.join(' · ')
}
