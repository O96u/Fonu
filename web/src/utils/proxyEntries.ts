import type { ProxyEntry, ProxyRule } from '../api/types'

export interface ProxyEntryListen {
  listen_port: number
  https_enabled: boolean
  http_redirect: boolean
  listen_ipv4: boolean
  listen_ipv6: boolean
}

export interface ProxyEntryGroup {
  key: string
  entryId?: number
  name: string
  listen: ProxyEntryListen
  rules: ProxyRule[]
  enabledCount: number
  portHasMixedEntries: boolean
}

export const PRESET_ENTRY_443: ProxyEntryListen = {
  listen_port: 443,
  https_enabled: true,
  http_redirect: true,
  listen_ipv4: true,
  listen_ipv6: false,
}

export const PRESET_ENTRY_80: ProxyEntryListen = {
  listen_port: 80,
  https_enabled: false,
  http_redirect: false,
  listen_ipv4: true,
  listen_ipv6: false,
}

export function entryKeyFromEntry(entry: ProxyEntry): string {
  return `entry:${entry.id}`
}

export function entryListenFromRule(rule: ProxyRule): ProxyEntryListen {
  return {
    listen_port: rule.listen_port,
    https_enabled: rule.https_enabled,
    http_redirect: rule.http_redirect,
    listen_ipv4: rule.listen_ipv4 ?? true,
    listen_ipv6: rule.listen_ipv6 ?? false,
  }
}

export function entryListenFromEntry(entry: ProxyEntry): ProxyEntryListen {
  return {
    listen_port: entry.listen_port,
    https_enabled: entry.https_enabled,
    http_redirect: entry.http_redirect,
    listen_ipv4: entry.listen_ipv4 ?? true,
    listen_ipv6: entry.listen_ipv6 ?? false,
  }
}

export function entryGroupKey(listen: ProxyEntryListen): string {
  return [
    listen.listen_port,
    listen.https_enabled ? 1 : 0,
    listen.http_redirect ? 1 : 0,
    listen.listen_ipv4 ? 1 : 0,
    listen.listen_ipv6 ? 1 : 0,
  ].join('|')
}

function applyPortMixedFlags(groups: ProxyEntryGroup[]) {
  const portListenKeys = new Map<number, Set<string>>()
  for (const group of groups) {
    if (!portListenKeys.has(group.listen.listen_port)) {
      portListenKeys.set(group.listen.listen_port, new Set())
    }
    portListenKeys.get(group.listen.listen_port)!.add(entryGroupKey(group.listen))
  }
  for (const group of groups) {
    const listenKeysOnPort = portListenKeys.get(group.listen.listen_port)!
    group.portHasMixedEntries = listenKeysOnPort.size > 1
  }
}

function sortEntryGroups(groups: ProxyEntryGroup[]): ProxyEntryGroup[] {
  return [...groups].sort((a, b) => {
    if (a.listen.listen_port !== b.listen.listen_port) {
      return a.listen.listen_port - b.listen.listen_port
    }
    return a.key.localeCompare(b.key)
  })
}

export function ruleDisplayLabel(rule: ProxyRule): string {
  const name = rule.name?.trim()
  if (name) return name
  const host = rule.hosts?.[0]?.hostname?.trim()
  if (host) return host
  return rule.domain?.trim() ?? ''
}

function deriveGroupName(name: string, rules: ProxyRule[]): string {
  const trimmed = name.trim()
  if (trimmed) return trimmed
  if (rules.length === 1) {
    return ruleDisplayLabel(rules[0])
  }
  return ''
}

function sortRulesByOrder(rules: ProxyRule[]): ProxyRule[] {
  return [...rules].sort((a, b) => {
    if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order
    return a.id - b.id
  })
}

function buildVirtualProxyEntryGroups(rules: ProxyRule[]): ProxyEntryGroup[] {
  const map = new Map<string, ProxyEntryGroup>()

  for (const rule of rules) {
    const listen = entryListenFromRule(rule)
    const key = entryGroupKey(listen)

    let group = map.get(key)
    if (!group) {
      group = {
        key,
        name: '',
        listen,
        rules: [],
        enabledCount: 0,
        portHasMixedEntries: false,
      }
      map.set(key, group)
    }
    group.rules.push(rule)
    if (rule.enabled) group.enabledCount++
  }

  for (const group of map.values()) {
    group.rules = sortRulesByOrder(group.rules)
    group.name = deriveGroupName(group.name, group.rules)
  }

  const groups = sortEntryGroups([...map.values()])
  applyPortMixedFlags(groups)
  return groups
}

export function buildProxyEntryGroups(rules: ProxyRule[], entries: ProxyEntry[] = []): ProxyEntryGroup[] {
  if (entries.length === 0) {
    return buildVirtualProxyEntryGroups(rules)
  }

  const rulesByEntry = new Map<number, ProxyRule[]>()
  const unassigned: ProxyRule[] = []

  for (const rule of rules) {
    if (rule.entry_id) {
      const list = rulesByEntry.get(rule.entry_id) ?? []
      list.push(rule)
      rulesByEntry.set(rule.entry_id, list)
    } else {
      unassigned.push(rule)
    }
  }

  const groups: ProxyEntryGroup[] = entries.map((entry) => {
    const listen = entryListenFromEntry(entry)
    const groupRules = sortRulesByOrder(rulesByEntry.get(entry.id) ?? [])
    return {
      key: entryKeyFromEntry(entry),
      entryId: entry.id,
      name: deriveGroupName(entry.name ?? '', groupRules),
      listen,
      rules: groupRules,
      enabledCount: groupRules.filter((rule) => rule.enabled).length,
      portHasMixedEntries: false,
    }
  })

  if (unassigned.length > 0) {
    groups.push(...buildVirtualProxyEntryGroups(unassigned))
  }

  applyPortMixedFlags(groups)
  return groups
}

export function entryGroupTitle(group: ProxyEntryGroup): string {
  const parts: string[] = [entryGroupDisplayName(group)]
  parts.push(`${group.listen.listen_port}`)
  parts.push(group.listen.https_enabled ? 'HTTPS' : 'HTTP')
  const stacks: string[] = []
  if (group.listen.listen_ipv4) stacks.push('IPv4')
  if (group.listen.listen_ipv6) stacks.push('IPv6')
  if (stacks.length > 0) parts.push(stacks.join('+'))
  parts.push(`${group.rules.length} 条规则`)
  if (group.enabledCount > 0) {
    parts.push(`${group.enabledCount} 运行`)
  }
  return parts.join(' · ')
}

export function entryGroupDisplayName(group: ProxyEntryGroup): string {
  if (group.name.trim()) return group.name.trim()
  const stacks: string[] = []
  if (group.listen.listen_ipv4) stacks.push('IPv4')
  if (group.listen.listen_ipv6) stacks.push('IPv6')
  const stackLabel = stacks.length > 0 ? ` · ${stacks.join('+')}` : ''
  return `${group.listen.listen_port} · ${group.listen.https_enabled ? 'HTTPS' : 'HTTP'}${stackLabel}`
}

export function entryGroupRuleNames(group: ProxyEntryGroup, label: (rule: ProxyRule) => string): string {
  return group.rules.map((rule) => label(rule)).filter(Boolean).join('、')
}

export function applyEntryListen(form: ProxyEntryListen, listen: ProxyEntryListen) {
  form.listen_port = listen.listen_port
  form.https_enabled = listen.https_enabled
  form.http_redirect = listen.http_redirect
  form.listen_ipv4 = listen.listen_ipv4
  form.listen_ipv6 = listen.listen_ipv6
}

export function entryListenEqual(a: ProxyEntryListen, b: ProxyEntryListen): boolean {
  return entryGroupKey(a) === entryGroupKey(b)
}
