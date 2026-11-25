import { computed, ref, watch } from 'vue'
import { darkTheme, type GlobalThemeOverrides } from 'naive-ui'

export const themePresets = [
  { value: 'forest', label: '森林绿', light: '#237d60', dark: '#77c6a5' },
  { value: 'blue', label: '经典蓝', light: '#285ec9', dark: '#82aaff' },
  { value: 'purple', label: '星空紫', light: '#7549be', dark: '#b49af5' },
  { value: 'orange', label: '暖橙色', light: '#a95218', dark: '#ecad72' },
  { value: 'graphite', label: '石墨灰', light: '#626971', dark: '#b5bcc5' },
] as const
export const surfaces = {
  light: { bg: '#f8f9f7', muted: '#edf0eb', surface: '#ffffff', border: '#dfe5df', text: '#202923', secondary: '#59665d', subtle: '#68756c' },
  dark: { bg: '#171c1a', muted: '#1e2521', surface: '#242c27', border: '#354139', text: '#e8eee9', secondary: '#b1bdb4', subtle: '#98a69c' },
}
function read(key: string, fallback: string) {
  try { return localStorage.getItem(key) || fallback } catch { return fallback }
}
function persist(key: string, value: string) {
  try { localStorage.setItem(key, value) } catch { /* storage unavailable */ }
}
function normalizeMode(mode: string) { return mode === 'light' || mode === 'dark' ? mode : 'system' }
function normalizeAccent(value: string) { return themePresets.find(p => p.value === value)?.value ?? 'forest' }
const themeMode = ref(normalizeMode(read('fonu-theme', 'system')))
const themeAccent = ref(normalizeAccent(read('fonu-accent', 'forest')))
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
  return { ...surfaces[dark ? 'dark' : 'light'], brand, hover: mix(brand, dark ? '#ffffff' : '#000000', .10), pressed: mix(brand, dark ? '#ffffff' : '#000000', .18), soft: brand + (dark ? '33' : '1c'), onBrand: dark ? '#171c1a' : '#ffffff', chartSecondary: themeAccent.value === 'blue' ? (dark ? '#77c6a5' : '#237d60') : (dark ? '#82aaff' : '#285ec9') }
})
function mix(hex: string, other: string, amount: number) {
  return '#' + [1, 3, 5].map(i => Math.round(parseInt(hex.slice(i, i + 2), 16) * (1 - amount) + parseInt(other.slice(i, i + 2), 16) * amount).toString(16).padStart(2, '0')).join('')
}
const themeOverrides = computed<GlobalThemeOverrides>(() => {
  const p = palette.value
  return {
    common: { primaryColor: p.brand, primaryColorHover: p.hover, primaryColorPressed: p.pressed, primaryColorSuppl: p.hover, borderRadius: '6px', fontFamily: 'var(--fonu-font)', bodyColor: p.bg, cardColor: p.surface, modalColor: p.surface, popoverColor: p.surface, borderColor: p.border, dividerColor: p.border, textColor1: p.text, textColor2: p.secondary, textColor3: p.subtle, hoverColor: p.soft },
    Card: { borderRadius: '8px', color: p.surface, borderColor: p.border },
    Button: { borderRadiusMedium: '6px', heightMedium: '36px', textColorPrimary: p.onBrand, textColorHoverPrimary: p.onBrand, textColorPressedPrimary: p.onBrand, textColorFocusPrimary: p.onBrand },
    Input: { borderRadius: '6px', color: p.surface },
    DataTable: { borderRadius: '8px', thColor: p.muted, tdColor: p.surface, borderColor: p.border },
    Tabs: { tabGapMediumLine: '24px', tabPaddingMediumLine: '12px 4px' },
    Layout: { color: p.bg, siderColor: p.muted },
    Menu: { itemColorActive: p.soft, itemTextColorActive: p.brand, itemIconColorActive: p.brand, itemColorActiveHover: p.soft },
    Radio: { buttonBorderColorActive: p.brand, buttonTextColorActive: p.brand, buttonColorActive: p.soft, buttonBoxShadowHover: 'none', buttonBoxShadowFocus: 'none' },
  }
})
watch(themeMode, mode => persist('fonu-theme', mode), { flush: 'sync' })
watch(themeAccent, accent => persist('fonu-accent', accent), { flush: 'sync' })
watch([isDark, palette], ([dark, p]) => {
  const root = document.documentElement
  root.style.colorScheme = dark ? 'dark' : 'light'
  root.dataset.theme = dark ? 'dark' : 'light'
  root.classList.toggle('dark', dark)
  const tokens = { brand: p.brand, 'brand-hover': p.hover, 'brand-soft': p.soft, 'brand-text': p.brand, 'brand-sidebar': p.brand, 'on-brand': p.onBrand, 'chart-secondary': p.chartSecondary, bg: p.bg, 'bg-muted': p.muted, surface: p.surface, border: p.border, text: p.text, 'text-secondary': p.secondary, 'text-muted': p.subtle }
  for (const [key, value] of Object.entries(tokens)) root.style.setProperty('--fonu-' + key, value)
}, { immediate: true, flush: 'sync' })

export function useTheme() {
  function setThemeMode(mode: string) { themeMode.value = normalizeMode(mode) }
  function setThemeAccent(accent: string) { themeAccent.value = normalizeAccent(accent) }
  function toggleTheme() { setThemeMode(isDark.value ? 'light' : 'dark') }
  return { themeMode, themeAccent, naiveTheme, themeOverrides, isDark, palette, toggleTheme, setThemeMode, setThemeAccent }
}
