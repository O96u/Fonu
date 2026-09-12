import { computed, ref, watch } from 'vue'
import { darkTheme, type GlobalThemeOverrides } from 'naive-ui'

const themeKey = 'fonu-theme'
const themeMode = ref(localStorage.getItem(themeKey) ?? 'system')

function systemDark() {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

const shared = {
  primaryColor: '#10B981',
  primaryColorHover: '#059669',
  primaryColorPressed: '#047857',
  primaryColorSuppl: '#34D399',
  borderRadius: '8px',
  fontFamily: 'var(--fonu-font)',
}

const lightOverrides: GlobalThemeOverrides = {
  common: {
    ...shared,
    bodyColor: '#F8FAFC',
    cardColor: '#FFFFFF',
    modalColor: '#FFFFFF',
    popoverColor: '#FFFFFF',
    borderColor: '#E5E7EB',
    dividerColor: '#E5E7EB',
    textColor1: '#111827',
    textColor2: '#6B7280',
    textColor3: '#9CA3AF',
    hoverColor: 'rgba(16, 185, 129, 0.08)',
  },
  Card: { borderRadius: '12px', color: '#FFFFFF' },
  Button: { borderRadiusMedium: '8px', heightMedium: '36px' },
  Input: { borderRadius: '8px' },
  DataTable: { borderRadius: '12px', thColor: '#F9FAFB', tdColor: '#FFFFFF' },
  Tabs: { tabGapMediumLine: '32px', tabPaddingMediumLine: '12px 4px' },
  Layout: { color: '#FFFFFF', siderColor: '#FFFFFF' },
  Menu: {
    itemColorActive: 'rgba(16, 185, 129, 0.12)',
    itemTextColorActive: '#047857',
    itemIconColorActive: '#10B981',
    itemColorActiveHover: 'rgba(16, 185, 129, 0.16)',
  },
  Radio: {
    buttonBorderColorActive: '#10B981',
    buttonTextColorActive: '#047857',
    buttonColorActive: 'rgba(16, 185, 129, 0.12)',
    buttonBoxShadowHover: 'none',
    buttonBoxShadowFocus: 'none',
  },
}

const darkOverrides: GlobalThemeOverrides = {
  common: {
    ...shared,
    bodyColor: '#0F172A',
    cardColor: '#1E293B',
    modalColor: '#1E293B',
    popoverColor: '#1E293B',
    borderColor: '#334155',
    dividerColor: '#334155',
    textColor1: '#F1F5F9',
    textColor2: '#94A3B8',
    textColor3: '#64748B',
    hoverColor: 'rgba(16, 185, 129, 0.12)',
  },
  Card: { borderRadius: '12px', color: '#1E293B', borderColor: '#334155' },
  Button: { borderRadiusMedium: '8px', heightMedium: '36px' },
  Input: { borderRadius: '8px', color: '#0F172A' },
  DataTable: { borderRadius: '12px', thColor: '#0F172A', tdColor: '#1E293B', borderColor: '#334155' },
  Tabs: { tabGapMediumLine: '32px', tabPaddingMediumLine: '12px 4px' },
  Layout: { color: '#0F172A', siderColor: '#1E293B' },
  Menu: {
    itemColorActive: 'rgba(16, 185, 129, 0.18)',
    itemTextColorActive: '#6EE7B7',
    itemIconColorActive: '#34D399',
    itemColorActiveHover: 'rgba(16, 185, 129, 0.24)',
  },
  Radio: {
    buttonBorderColorActive: '#34D399',
    buttonTextColorActive: '#6EE7B7',
    buttonColorActive: 'rgba(16, 185, 129, 0.22)',
    buttonBoxShadowHover: 'none',
    buttonBoxShadowFocus: 'none',
  },
}

function resolveDark(mode: string) {
  if (mode === 'dark') return true
  if (mode === 'light') return false
  return systemDark()
}

function applyDocumentTheme(dark: boolean) {
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light')
  document.documentElement.classList.toggle('dark', dark)
}

export function useTheme() {
  const isDark = computed(() => resolveDark(themeMode.value))

  const naiveTheme = computed(() => (isDark.value ? darkTheme : null))
  const themeOverrides = computed<GlobalThemeOverrides>(() =>
    isDark.value ? darkOverrides : lightOverrides,
  )

  function setThemeMode(mode: string) {
    themeMode.value = mode
  }

  function toggleTheme() {
    themeMode.value = isDark.value ? 'light' : 'dark'
  }

  watch(
    themeMode,
    (value) => {
      localStorage.setItem(themeKey, value)
      applyDocumentTheme(resolveDark(value))
    },
    { immediate: true },
  )

  if (typeof window !== 'undefined') {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (themeMode.value === 'system') {
        applyDocumentTheme(systemDark())
      }
    })
  }

  return { themeMode, naiveTheme, themeOverrides, isDark, toggleTheme, setThemeMode }
}
