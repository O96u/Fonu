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

/** Normalize log timestamps to YYYY-MM-DD HH:mm:ss */
export function formatLogTime(value?: string): string {
  if (!value) return '-'

  const trimmed = value.trim()

  const standard = trimmed.match(/^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})/)
  if (standard) return standard[1]

  const iso = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2}):(\d{2})/)
  if (iso) return `${iso[1]}-${iso[2]}-${iso[3]} ${iso[4]}:${iso[5]}:${iso[6]}`

  const nginx = trimmed.match(/^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2}):(\d{2})/)
  if (nginx) return `${nginx[1]}-${nginx[2]}-${nginx[3]} ${nginx[4]}:${nginx[5]}:${nginx[6]}`

  const date = new Date(trimmed)
  if (!Number.isNaN(date.getTime())) {
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
  }

  return trimmed
}

/** Normalize leading timestamp in a raw log line */
export function formatLogLine(line: string): string {
  const trimmed = line.trim()
  if (!trimmed) return line

  const nginx = trimmed.match(/^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2})(.*)$/)
  if (nginx) return formatLogTime(nginx[1]) + nginx[2]

  const iso = trimmed.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:[+-]\d{2}:\d{2}|Z)?)(.*)$/)
  if (iso) return formatLogTime(iso[1]) + iso[2]

  if (trimmed.startsWith('{')) {
    try {
      const raw = JSON.parse(trimmed) as { time?: string }
      if (raw.time) return trimmed.replace(raw.time, formatLogTime(raw.time))
    } catch {
      // not JSON
    }
  }

  return trimmed
}
