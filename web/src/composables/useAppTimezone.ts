import { onMounted, ref, type Ref } from 'vue'
import { api } from '../api/client'

const DEFAULT_TIMEZONE = 'Asia/Shanghai'

let cachedTimezone: string | null = null

/** Fonu 设置中的 IANA 时区，用于日志等与服务器时间对齐的展示。 */
export function useAppTimezone(): Ref<string> {
  const timeZone = ref(cachedTimezone ?? DEFAULT_TIMEZONE)

  onMounted(async () => {
    if (cachedTimezone) {
      timeZone.value = cachedTimezone
      return
    }
    try {
      const settings = await api.getSettings()
      const tz = settings.timezone?.trim()
      if (tz) {
        cachedTimezone = tz
        timeZone.value = tz
      }
    } catch {
      // 保持默认
    }
  })

  return timeZone
}

export function getDefaultAppTimezone(): string {
  return cachedTimezone ?? DEFAULT_TIMEZONE
}

export function invalidateAppTimezoneCache(): void {
  cachedTimezone = null
}
