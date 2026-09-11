import { computed, ref, watch } from 'vue'
import { darkTheme, type GlobalThemeOverrides } from 'naive-ui'

const themeKey = 'fonu-theme'
const themeMode = ref(localStorage.getItem(themeKey) ?? 'system')

function systemDark() {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

const brandOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#10B981',
    primaryColorHover: '#059669',
    primaryColorPressed: '#047857',
    primaryColorSuppl: '#34D399',
    borderRadius: '8px',
    fontFamily: 'var(--fonu-font)',
    bodyColor: '#F8FAFC',
    cardColor: '#FFFFFF',
    borderColor: '#E5E7EB',
  },
  Card: { borderRadius: '12px' },
  Button: { borderRadiusMedium: '8px', heightMedium: '36px' },
  Input: { borderRadius: '8px' },
  DataTable: { borderRadius: '12px', thColor: '#F9FAFB' },
  Tabs: { tabGapMediumLine: '0px' },
}

export function useTheme() {
  const isDark = computed(() => {
    if (themeMode.value === 'dark') return true
    if (themeMode.value === 'light') return false
    return systemDark()
  })

  const naiveTheme = computed(() => (isDark.value ? darkTheme : null))
  const themeOverrides = computed<GlobalThemeOverrides>(() => brandOverrides)

  function toggleTheme() {
    themeMode.value = isDark.value ? 'light' : 'dark'
  }

  watch(
    themeMode,
    (value) => {
      localStorage.setItem(themeKey, value)
      const dark = value === 'dark' || (value === 'system' && systemDark())
      document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
      document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light')
    },
    { immediate: true },
  )

  if (typeof window !== 'undefined') {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (themeMode.value === 'system') {
        document.documentElement.style.colorScheme = systemDark() ? 'dark' : 'light'
        document.documentElement.setAttribute('data-theme', systemDark() ? 'dark' : 'light')
      }
    })
  }

  return { themeMode, naiveTheme, themeOverrides, isDark, toggleTheme }
}
