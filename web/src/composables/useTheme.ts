import { computed, ref, watch } from 'vue'
import { darkTheme, type GlobalThemeOverrides } from 'naive-ui'

export const themePresets = [
  { value: 'brand', label: '经典品牌绿（V1）', light: '#10b981', dark: '#10b981' },
  { value: 'ice', label: '冰川蓝', light: '#3869c9', dark: '#8cb4ff' },
  { value: 'teal', label: '青蓝绿', light: '#128579', dark: '#5cc7b7' },
  { value: 'forest', label: '经典翠绿', light: '#237d60', dark: '#77c6a5' },
  { value: 'blue', label: '科技蓝', light: '#2563eb', dark: '#82aaff' },
  { value: 'purple', label: '星空紫', light: '#7549be', dark: '#b49af5' },
  { value: 'orange', label: '暖橙色', light: '#a95218', dark: '#ecad72' },
  { value: 'graphite', label: '石墨灰', light: '#626971', dark: '#b5bcc5' },
] as const
export const surfaces = {
  light: { bg: '#f5f6f8', muted: '#eef0f3', surface: '#ffffff', border: '#dfe3e8', text: '#20242c', secondary: '#555e6b', subtle: '#687383' },
  dark: { bg: '#171a1f', muted: '#1d2128', surface: '#22262e', border: '#343b45', text: '#e8ecf2', secondary: '#b4bdca', subtle: '#9da9b9' },
}
function read(key: string, fallback: string) {
  try { return localStorage.getItem(key) || fallback } catch { return fallback }
}
function persist(key: string, value: string) {
  try { localStorage.setItem(key, value) } catch { /* storage unavailable */ }
}
function normalizeMode(mode: string) { return mode === 'light' || mode === 'dark' ? mode : 'system' }
function normalizeAccent(value: string) { return themePresets.find(p => p.value === value)?.value ?? 'brand' }
const themeMode = ref(normalizeMode(read('fonu-theme', 'system')))
const themeAccent = ref(normalizeAccent(read('fonu-accent', 'brand')))
const media = window.matchMedia('(prefers-color-scheme: dark)')
const systemIsDark = ref(media.matches)
const onMediaChange = (event: MediaQueryListEvent) => { systemIsDark.value = event.matches }
if (media.addEventListener) media.addEventListener('change', onMediaChange)
else media.addListener(onMediaChange)

const isDark = computed(() => themeMode.value === 'system' ? systemIsDark.value : themeMode.value === 'dark')
const naiveTheme = computed(() => isDark.value ? darkTheme : null)
const palette = computed(() => {
  const preset = themePresets.find(p => p.value === themeAccent.value) ?? themePresets[0]
  const dark = isDark.value
  const brand = dark ? preset.dark : preset.light
  const classic = preset.value === 'brand'
  const brandText = classic ? (dark ? '#6ee7b7' : '#047857') : brand
  return { ...surfaces[dark ? 'dark' : 'light'], brand, brandText, hover: classic ? '#059669' : mix(brand, dark ? '#ffffff' : '#000000', .10), pressed: classic ? '#047857' : mix(brand, dark ? '#ffffff' : '#000000', .18), soft: brand + (dark ? '2e' : '1f'), onBrand: classic || dark ? '#10251e' : '#ffffff', chartSecondary: themeAccent.value === 'purple' ? (dark ? '#55c9ef' : '#087eaa') : (dark ? '#b49af5' : '#7558c6') }
})
function mix(hex: string, other: string, amount: number) {
  return '#' + [1, 3, 5].map(i => Math.round(parseInt(hex.slice(i, i + 2), 16) * (1 - amount) + parseInt(other.slice(i, i + 2), 16) * amount).toString(16).padStart(2, '0')).join('')
}
const themeOverrides = computed<GlobalThemeOverrides>(() => {
  const p = palette.value
  return {
    common: { primaryColor: p.brand, primaryColorHover: p.hover, primaryColorPressed: p.pressed, primaryColorSuppl: p.hover, borderRadius: '6px', fontFamily: 'var(--fonu-font)', bodyColor: p.bg, cardColor: p.surface, modalColor: p.surface, popoverColor: p.surface, borderColor: p.border, dividerColor: p.border, textColor1: p.text, textColor2: p.secondary, textColor3: p.subtle, hoverColor: p.soft },
    Card: { borderRadius: '12px', color: p.surface, borderColor: p.border },
    Button: { borderRadiusMedium: '6px', heightMedium: '36px', textColorPrimary: p.onBrand, textColorHoverPrimary: p.onBrand, textColorPressedPrimary: p.onBrand, textColorFocusPrimary: p.onBrand },
    Input: { borderRadius: '6px', color: p.surface },
    DataTable: { borderRadius: '8px', thColor: p.muted, tdColor: p.surface, borderColor: p.border },
    Tabs: { tabGapMediumLine: '24px', tabPaddingMediumLine: '12px 4px' },
    Layout: { color: p.bg, siderColor: p.muted },
    Menu: { itemColorActive: p.soft, itemTextColorActive: p.brandText, itemIconColorActive: p.brand, itemColorActiveHover: p.soft },
    Radio: { buttonBorderColorActive: p.brand, buttonTextColorActive: p.brandText, buttonColorActive: p.soft, buttonBoxShadowHover: 'none', buttonBoxShadowFocus: 'none' },
  }
})
watch(themeMode, mode => persist('fonu-theme', mode), { flush: 'sync' })
watch(themeAccent, accent => persist('fonu-accent', accent), { flush: 'sync' })
watch([isDark, palette], ([dark, p]) => {
  const root = document.documentElement
  root.style.colorScheme = dark ? 'dark' : 'light'
  root.dataset.theme = dark ? 'dark' : 'light'
  root.classList.toggle('dark', dark)
  const tokens = { brand: p.brand, 'brand-hover': p.hover, 'brand-soft': p.soft, 'brand-text': p.brandText, 'brand-sidebar': p.brand, 'on-brand': p.onBrand, 'chart-secondary': p.chartSecondary, bg: p.bg, 'bg-muted': p.muted, surface: p.surface, border: p.border, text: p.text, 'text-secondary': p.secondary, 'text-muted': p.subtle }
  for (const [key, value] of Object.entries(tokens)) root.style.setProperty('--fonu-' + key, value)
}, { immediate: true, flush: 'sync' })

export function useTheme() {
  function setThemeMode(mode: string) { themeMode.value = normalizeMode(mode) }
  function setThemeAccent(accent: string) { themeAccent.value = normalizeAccent(accent) }
  function toggleTheme() { setThemeMode(isDark.value ? 'light' : 'dark') }
  return { themeMode, themeAccent, naiveTheme, themeOverrides, isDark, palette, toggleTheme, setThemeMode, setThemeAccent }
}
