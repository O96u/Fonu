import type { SystemLogEntry } from '../api/types'
import { formatLogTime } from './format'

export type LogTagType = 'success' | 'info' | 'warning' | 'error' | 'default'

export interface DisplaySystemLogEntry extends SystemLogEntry {
  displayLevel: string
  displayModule: string
}

const SYSTEM_LEVEL_LABELS: Record<string, string> = {
  INFO: '信息',
  WARN: '警告',
  ERROR: '错误',
  DEBUG: '调试',
}

const SYSTEM_MODULE_LABELS: Record<string, string> = {
  SYSTEM: '系统',
  NGINX: 'Nginx',
  DDNS: '域名解析',
  ACME: '证书',
  CERT: '证书',
  CHINA_CIDR: '中国 IP 段',
  API: 'API',
  HTTP: 'HTTP',
  FRP: '内网穿透',
}

export function systemLevelLabel(level: string): string {
  const upper = level.trim().toUpperCase()
  return SYSTEM_LEVEL_LABELS[upper] ?? level
}

export function systemModuleLabel(module: string): string {
  const trimmed = module.trim()
  if (!trimmed) return '系统'
  const upper = trimmed.toUpperCase()
  return SYSTEM_MODULE_LABELS[upper] ?? trimmed
}

export function displaySystemLog(entry: SystemLogEntry): DisplaySystemLogEntry {
  const message = entry.message.trim()
  const msgLower = message.toLowerCase()
  let level = entry.level.trim().toUpperCase()
  let displayModule = systemModuleLabel(entry.module)

  if (
    level === 'INFO' &&
    (msgLower.includes('server error') ||
      msgLower.includes(' failed') ||
      msgLower.startsWith('error') ||
      msgLower.includes('bind:'))
  ) {
    level = 'ERROR'
  }

  return {
    ...entry,
    displayLevel: systemLevelLabel(level),
    displayModule,
  }
}

export interface ParsedNginxError {
  time: string
  level: string
  message: string
  raw: string
}

export function parseNginxErrorLine(line: string): ParsedNginxError {
  const raw = line.trim()
  const match = raw.match(/^(\d{4}[-/]\d{2}[-/]\d{2} \d{2}:\d{2}:\d{2})\s+\[([^\]]+)\]\s+(.+)$/)
  if (!match) {
    return { time: '', level: 'unknown', message: raw, raw }
  }
  return {
    time: formatLogTime(match[1]),
    level: match[2].toLowerCase(),
    message: match[3],
    raw,
  }
}

export function nginxLevelTagType(level: string): LogTagType {
  switch (level.toLowerCase()) {
    case 'emerg':
    case 'alert':
    case 'crit':
    case 'error':
      return 'error'
    case 'warn':
    case 'warning':
      return 'warning'
    case 'notice':
    case 'info':
      return 'info'
    default:
      return 'default'
  }
}

export function systemLevelTagType(level: string): LogTagType {
  const upper = level.trim().toUpperCase()
  if (upper === 'ERROR' || level === '错误') return 'error'
  if (upper === 'WARN' || level === '警告') return 'warning'
  if (upper === 'INFO' || level === '信息') return 'info'
  return 'default'
}

export function systemModuleTagType(module: string): LogTagType {
  const upper = module.trim().toUpperCase()
  switch (upper) {
    case 'NGINX':
    case 'Nginx':
      return 'warning'
    case 'SYSTEM':
    case '系统':
      return 'success'
    case 'CHINA_CIDR':
    case '中国 IP 段':
      return 'info'
    case 'DDNS':
    case '域名解析':
      return 'info'
    case 'CERT':
    case 'ACME':
    case '证书':
      return 'warning'
    default:
      return 'default'
  }
}

export function latencyClass(seconds: number): string {
  const ms = seconds * 1000
  if (ms >= 3000) return 'log-duration log-duration--critical'
  if (ms >= 1000) return 'log-duration log-duration--slow'
  return 'log-duration'
}

export function streamLineClass(line: string): string {
  const parsed = parseNginxErrorLine(line)
  if (parsed.level !== 'unknown') {
    const type = nginxLevelTagType(parsed.level)
    if (type === 'error') return 'level-error'
    if (type === 'warning') return 'level-warn'
    if (type === 'info') return 'level-info'
  }

  const lower = line.toLowerCase()
  if (lower.includes('[error]') || lower.includes(' emerg ') || lower.includes(' alert ') || lower.includes(' crit ')) {
    return 'level-error'
  }
  if (lower.includes('[warn]') || lower.includes(' warning ')) return 'level-warn'

  const statusMatch = line.match(/\s(\d{3})\s+\d/)
  if (statusMatch) {
    const code = Number(statusMatch[1])
    if (code >= 500) return 'level-error'
    if (code >= 400) return 'level-warn'
  }
  return ''
}
