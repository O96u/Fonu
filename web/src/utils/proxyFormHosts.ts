import type { SelectGroupOption } from 'naive-ui'
import { hostsTextToValues, valuesToHostsText } from './ddnsHosts'

export function formHostsList(selected: string[]): string[] {
  return selected.map((h) => h.trim()).filter(Boolean)
}

export function setFormHostsFromText(selected: string[], text: string): void {
  selected.splice(0, selected.length, ...hostsTextToValues(text))
}

export function formHostsToText(selected: string[]): string {
  return valuesToHostsText(selected)
}

const EXTRA_GROUP_KEY = '__proxy_host_extras__'

/** 将当前已选但不在 DDNS 列表中的域名并入选项（编辑旧规则时） */
export function mergeHostSelectOptions(
  ddnsGroups: SelectGroupOption[],
  selected: string[],
): SelectGroupOption[] {
  const known = new Set<string>()
  for (const group of ddnsGroups) {
    if (group.type !== 'group' || !group.children) continue
    for (const child of group.children) {
      if (child && typeof child === 'object' && 'value' in child && child.value) {
        known.add(String(child.value).toLowerCase())
      }
    }
  }
  const extras: string[] = []
  for (const value of selected) {
    const v = value.trim().toLowerCase()
    if (!v || known.has(v)) continue
    known.add(v)
    extras.push(v)
  }
  if (extras.length === 0) return ddnsGroups
  return [
    ...ddnsGroups,
    {
      type: 'group',
      label: '当前规则（未在 DDNS）',
      key: EXTRA_GROUP_KEY,
      children: extras.map((h) => ({ label: h, value: h })),
    },
  ]
}
