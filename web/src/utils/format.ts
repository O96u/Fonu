export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  const digits = value >= 100 || unit === 0 ? 0 : value >= 10 ? 1 : 2
  return `${value.toFixed(digits)} ${units[unit]}`
}

export function formatRate(bytesPerSec: number): string {
  if (!Number.isFinite(bytesPerSec) || bytesPerSec <= 0) return '0 B/s'
  return `${formatBytes(bytesPerSec)}/s`
}

export function formatRateIdle(bytesPerSec: number): string {
  if (!Number.isFinite(bytesPerSec) || bytesPerSec <= 0) return '—'
  return formatRate(bytesPerSec)
}

export function formatUptime(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (h > 0) return `${h} 小时 ${m} 分钟`
  return `${m} 分钟`
}

export function formatRelativeTime(iso?: string): string {
  if (!iso) return '-'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  const diff = Date.now() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  return `${days} 天前`
}

export function formatMs(seconds: number): string {
  const ms = Math.round(seconds * 1000)
  return `${ms} ms`
}

export function formatDate(iso?: string): string {
  if (!iso) return '-'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export type FormatLogTimeOptions = {
  /** IANA 时区，默认 Asia/Shanghai（与 Fonu 设置默认一致） */
  timeZone?: string
}

const DEFAULT_LOG_TIMEZONE = 'Asia/Shanghai'

const ISO_WITH_OFFSET =
  /^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2}:\d{2}(?:\.\d+)?)(Z|[+-]\d{2}:?\d{2})$/

const NAIVE_DATETIME =
  /^(\d{4})[-/](\d{2})[-/](\d{2})[ T](\d{2}):(\d{2}):(\d{2})(?:\.\d+)?$/

function formatInTimeZone(date: Date, timeZone: string): string {
  const fmt = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
  const parts = fmt.formatToParts(date)
  const pick = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find((p) => p.type === type)?.value ?? '00'
  return `${pick('year')}-${pick('month')}-${pick('day')} ${pick('hour')}:${pick('minute')}:${pick('second')}`
}

/** Parse log timestamps; naive strings (nginx error / frpc) are treated as UTC. */
export function parseLogTimestamp(value: string): Date | null {
  const trimmed = value.trim()
  if (!trimmed) return null

  const isoTz = trimmed.match(ISO_WITH_OFFSET)
  if (isoTz) {
    const normalized = `${isoTz[1]}T${isoTz[2]}${isoTz[3].replace(/([+-]\d{2})(\d{2})$/, '$1:$2')}`
    const ms = Date.parse(normalized)
    if (!Number.isNaN(ms)) return new Date(ms)
  }

  const naive = trimmed.match(NAIVE_DATETIME)
  if (naive) {
    const y = Number(naive[1])
    const mo = Number(naive[2]) - 1
    const d = Number(naive[3])
    const h = Number(naive[4])
    const mi = Number(naive[5])
    const s = Number(naive[6])
    return new Date(Date.UTC(y, mo, d, h, mi, s))
  }

  if (/[Z+-]/.test(trimmed)) {
    const ms = Date.parse(trimmed)
    if (!Number.isNaN(ms)) return new Date(ms)
  }

  return null
}

/** Normalize log timestamps to YYYY-MM-DD HH:mm:ss in the configured timezone. */
export function formatLogTime(value?: string, options?: FormatLogTimeOptions): string {
  if (!value) return '-'

  const timeZone = options?.timeZone ?? DEFAULT_LOG_TIMEZONE
  const parsed = parseLogTimestamp(value.trim())
  if (parsed) return formatInTimeZone(parsed, timeZone)

  return value.trim()
}

/** Normalize leading timestamp in a raw log line */
export function formatLogLine(line: string, options?: FormatLogTimeOptions): string {
  const trimmed = line.trim()
  if (!trimmed) return line

  const nginx = trimmed.match(/^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2})(.*)$/)
  if (nginx) return formatLogTime(nginx[1], options) + nginx[2]

  const iso = trimmed.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:[+-]\d{2}:\d{2}|Z)?)(.*)$/)
  if (iso) return formatLogTime(iso[1], options) + iso[2]

  if (trimmed.startsWith('{')) {
    try {
      const raw = JSON.parse(trimmed) as { time?: string }
      if (raw.time) return trimmed.replace(raw.time, formatLogTime(raw.time, options))
    } catch {
      // not JSON
    }
  }

  return trimmed
}
