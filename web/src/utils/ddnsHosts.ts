import { h } from 'vue'
import type { SelectGroupOption, SelectOption } from 'naive-ui'
import type { DDNSConfig } from '../api/types'
import { ddnsProviderLabel } from './dnsProviders'

export type DdnsHostSelectOption = SelectOption & {
  providerLabel?: string
  taskLabel?: string
}

function renderDdnsHostGroupLabel(taskTitle: string, providerLabel: string) {
  return () =>
    h('span', { class: 'ddns-host-group-label' }, [
      h('span', { class: 'ddns-host-group-label__task' }, taskTitle),
      h('span', { class: 'ddns-host-group-label__dash' }, ' - '),
      h('span', { class: 'ddns-host-group-label__provider' }, providerLabel),
    ])
}

export function fqdnFromRecord(name: string, root?: string): string {
  const n = name.trim().toLowerCase()
  if (!n) return ''
  if (n.includes('.')) return n
  const rootDom = root?.trim().toLowerCase()
  if (!rootDom) return n
  return `${n}.${rootDom}`
}

export function collectDdnsHostOptions(configs: DDNSConfig[]): SelectGroupOption[] {
  const groups: SelectGroupOption[] = []
  for (const cfg of configs) {
    const remark = cfg.remark?.trim() ?? ''
    const providerLabel = ddnsProviderLabel(cfg.provider)
    const rootDomain = cfg.root_domain?.trim().toLowerCase() ?? ''
    // 有名称用名称；无名称用根域名 + 服务商（与 DDNS 任务卡片逻辑一致）
    const taskTitle = remark || rootDomain || providerLabel || `DDNS #${cfg.id}`
    const items: DdnsHostSelectOption[] = []
    const seen = new Set<string>()
    const add = (host: string) => {
      const hostValue = host.trim().toLowerCase()
      if (!hostValue || seen.has(hostValue)) return
      seen.add(hostValue)
      items.push({
        label: hostValue,
        value: hostValue,
        providerLabel,
        taskLabel: taskTitle,
      })
    }
    for (const rec of cfg.domain_records ?? []) {
      add(rec.domain)
    }
    if (items.length === 0) {
      for (const name of cfg.record_names ?? []) {
        add(fqdnFromRecord(name, cfg.root_domain))
      }
      if (cfg.record_name) add(fqdnFromRecord(cfg.record_name, cfg.root_domain))
    }
    if (items.length === 0 && cfg.root_domain) {
      add(cfg.root_domain)
    }
    if (items.length > 0) {
      items.sort((a, b) => String(a.value).localeCompare(String(b.value)))
      groups.push({
        type: 'group',
        key: `ddns-${cfg.id}`,
        label: renderDdnsHostGroupLabel(taskTitle, providerLabel),
        children: items,
      })
    }
  }
  return groups
}

export function flattenDdnsOptionValues(groups: SelectGroupOption[]): Set<string> {
  const out = new Set<string>()
  for (const group of groups) {
    if (group.type !== 'group' || !group.children) continue
    for (const child of group.children) {
      if (child && typeof child === 'object' && 'value' in child && child.value) {
        out.add(String(child.value).toLowerCase())
      }
    }
  }
  return out
}

export function hostsTextToValues(text: string): string[] {
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

export function valuesToHostsText(values: string[]): string {
  return values.map((v) => v.trim()).filter(Boolean).join('\n')
}

/** 与 internal/validate/validate.go Upstream 保持一致 */
export function validateUpstreamInput(raw: string): string {
  const text = raw.trim()
  if (!text) {
    throw new Error('目标地址不能为空')
  }
  if (/\r|\n/.test(raw)) {
    throw new Error('目标地址只能填写一行')
  }
  let u: URL
  try {
    u = new URL(text)
  } catch {
    throw new Error('目标地址格式无效，需以 http:// 或 https:// 开头')
  }
  if (u.protocol !== 'http:' && u.protocol !== 'https:') {
    throw new Error('目标地址仅支持 http:// 或 https://')
  }
  if (u.username || u.password) {
    throw new Error('目标地址不能包含用户名或密码')
  }
  if (u.pathname !== '' && u.pathname !== '/') {
    throw new Error('目标地址不能包含路径')
  }
  if (u.search || u.hash) {
    throw new Error('目标地址不能包含查询参数或片段')
  }
  const host = u.hostname
  if (!host) {
    throw new Error('目标地址主机名无效')
  }
  const isIPv4 = /^\d{1,3}(\.\d{1,3}){3}$/.test(host)
  if (isIPv4 && !u.port) {
    throw new Error('IP 地址目标必须指定端口')
  }
  return `${u.protocol}//${u.host}`
}
