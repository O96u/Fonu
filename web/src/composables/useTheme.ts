import { computed, ref, watch } from 'vue'
import { darkTheme, type GlobalThemeOverrides } from 'naive-ui'

export const themePresets = [
  { value: 'brand', label: '经典品牌绿', light: '#10b981', dark: '#10b981' },
  { value: 'ice', label: '冰川蓝', light: '#3869c9', dark: '#8cb4ff' },
  { value: 'teal', label: '青蓝绿', light: '#128579', dark: '#5cc7b7' },
  { value: 'forest', label: '经典翠绿', light: '#237d60', dark: '#77c6a5' },
  { value: 'blue', label: '科技蓝', light: '#2563eb', dark: '#82aaff' },
  { value: 'purple', label: '星空紫', light: '#7549be', dark: '#b49af5' },
  { value: 'orange', label: '暖橙色', light: '#a95218', dark: '#ecad72' },
  { value: 'graphite', label: '石墨灰', light: '#626971', dark: '#b5bcc5' },
] as const
export const surfaces = {
  light: { bg: '#f8fafc', muted: '#f3f4f6', surface: '#ffffff', border: '#e5e7eb', text: '#111827', secondary: '#6b7280', subtle: '#9ca3af' },
  dark: { bg: '#0f172a', muted: '#1e293b', surface: '#1e293b', border: '#334155', text: '#f1f5f9', secondary: '#94a3b8', subtle: '#64748b' },
}
// Complete theme surfaces. Keep semantic status colors independent of accent.
export const themeSurfaces = {
  brand: { light: { bg: '#edf5f1', muted: '#e4efe9', surface: '#fcfefd', chrome: '#f5faf7', border: '#d5e5dd', brandText: '#047857', chartSecondary: '#7558c6' }, dark: { bg: '#101e19', muted: '#172b23', surface: '#1c3028', chrome: '#14251e', border: '#304a3e', brandText: '#6ee7b7', chartSecondary: '#b49af5' } },
  ice: { light: { bg: '#eaf0f8', muted: '#e0e9f5', surface: '#fcfdff', chrome: '#f3f7fc', border: '#d4e0ef', brandText: '#3869c9', chartSecondary: '#128579' }, dark: { bg: '#111e30', muted: '#192a40', surface: '#203149', chrome: '#162439', border: '#344966', brandText: '#8cb4ff', chartSecondary: '#5cc7b7' } },
  teal: { light: { bg: '#e7f3f2', muted: '#daecea', surface: '#fcfefe', chrome: '#f2f9f8', border: '#cfe4e1', brandText: '#0e7469', chartSecondary: '#3869c9' }, dark: { bg: '#102624', muted: '#173430', surface: '#1b3935', chrome: '#142e2b', border: '#34534e', brandText: '#5cc7b7', chartSecondary: '#8cb4ff' } },
  forest: { light: { bg: '#eaf1e9', muted: '#e0ebdf', surface: '#fcfefc', chrome: '#f3f8f2', border: '#d5e3d2', brandText: '#237d60', chartSecondary: '#a95218' }, dark: { bg: '#17241b', muted: '#203127', surface: '#27392e', chrome: '#1c2c21', border: '#3b5040', brandText: '#77c6a5', chartSecondary: '#ecad72' } },
  blue: { light: { bg: '#e9eefb', muted: '#dee6fa', surface: '#fcfdff', chrome: '#f3f5fe', border: '#d5dff5', brandText: '#2563eb', chartSecondary: '#128579' }, dark: { bg: '#101a34', muted: '#192646', surface: '#202e50', chrome: '#15213d', border: '#35466d', brandText: '#82aaff', chartSecondary: '#5cc7b7' } },
  purple: { light: { bg: '#f0ecf8', muted: '#e8e0f3', surface: '#fefcff', chrome: '#f8f5fc', border: '#e2d9ef', brandText: '#7549be', chartSecondary: '#087eaa' }, dark: { bg: '#21182e', muted: '#30233f', surface: '#382a49', chrome: '#291e36', border: '#514062', brandText: '#b49af5', chartSecondary: '#55c9ef' } },
  orange: { light: { bg: '#f7eee5', muted: '#f0e3d5', surface: '#fffdf9', chrome: '#fcf8f3', border: '#eadbcc', brandText: '#a95218', chartSecondary: '#128579' }, dark: { bg: '#2b211b', muted: '#3b2e24', surface: '#43352a', chrome: '#34271f', border: '#584638', brandText: '#ecad72', chartSecondary: '#5cc7b7' } },
  graphite: { light: { bg: '#e9ebee', muted: '#e0e3e7', surface: '#fcfcfd', chrome: '#f3f4f5', border: '#d7dade', brandText: '#626971', chartSecondary: '#3869c9' }, dark: { bg: '#191c21', muted: '#24282e', surface: '#2b3037', chrome: '#20242a', border: '#434a54', brandText: '#b5bcc5', chartSecondary: '#8cb4ff' } },
} as const
export function previewTheme(accent: keyof typeof themeSurfaces, dark: boolean) {
  const theme = themeSurfaces[accent][dark ? 'dark' : 'light']
  return { ...theme, ...surfaces[dark ? 'dark' : 'light'], chrome: dark ? '#1e293b' : '#ffffff' }
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
  const theme = previewTheme(preset.value, dark)
  const brandText = theme.brandText
  const action = classic ? '#10b981' : themeSurfaces[preset.value].light.brandText
  return { ...surfaces[dark ? 'dark' : 'light'], ...theme, brand, brandText, action, hover: classic ? '#059669' : mix(brand, dark ? '#ffffff' : '#000000', .10), pressed: classic ? '#047857' : mix(brand, dark ? '#ffffff' : '#000000', .18), soft: brand + (dark ? '2e' : '1f'), onBrand: '#ffffff', chartSecondary: classic ? (dark ? '#60a5fa' : '#3b82f6') : theme.chartSecondary }
})
function mix(hex: string, other: string, amount: number) {
  return '#' + [1, 3, 5].map(i => Math.round(parseInt(hex.slice(i, i + 2), 16) * (1 - amount) + parseInt(other.slice(i, i + 2), 16) * amount).toString(16).padStart(2, '0')).join('')
}
const themeOverrides = computed<GlobalThemeOverrides>(() => {
  const p = palette.value
  return {
    common: { primaryColor: p.action, primaryColorHover: themeAccent.value==='brand'?'#059669':mix(p.action, '#000000', .08), primaryColorPressed: themeAccent.value==='brand'?'#047857':mix(p.action, '#000000', .16), primaryColorSuppl: themeAccent.value==='brand'?'#34d399':p.action, borderRadius: '8px', fontFamily: 'var(--fonu-font)', bodyColor: p.bg, cardColor: p.surface, modalColor: p.surface, popoverColor: p.surface, borderColor: p.border, dividerColor: p.border, textColor1: p.text, textColor2: p.secondary, textColor3: p.subtle, hoverColor: p.soft },
    Card: { borderRadius: '12px', color: p.surface, borderColor: p.border },
    Button: { borderRadiusMedium: '8px', heightMedium: '36px', textColorPrimary: p.onBrand, textColorHoverPrimary: p.onBrand, textColorPressedPrimary: p.onBrand, textColorFocusPrimary: p.onBrand },
    Input: { borderRadius: '8px', color: isDark.value?p.bg:p.surface },
    DataTable: { borderRadius: '12px', thColor: themeAccent.value==='brand'?(isDark.value?p.bg:'#f9fafb'):p.muted, tdColor: p.surface, borderColor: p.border },
    Tabs: { tabGapMediumLine: '32px', tabPaddingMediumLine: '12px 4px' },
    Layout: { color: p.bg, siderColor: p.chrome, headerColor: p.chrome },
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
  root.dataset.accent = themeAccent.value
  root.classList.toggle('dark', dark)
  const tokens = { brand: p.brand, 'brand-hover': p.hover, 'brand-soft': p.soft, 'brand-text': p.brandText, 'brand-sidebar': p.brand, 'on-brand': p.onBrand, action: p.action, 'chart-secondary': p.chartSecondary, bg: p.bg, 'bg-muted': p.muted, surface: p.surface, chrome: p.chrome, border: p.border, text: p.text, 'text-secondary': p.secondary, 'text-muted': p.subtle }
  root.style.setProperty('--boot-bg', p.bg)
  for (const [key, value] of Object.entries(tokens)) root.style.setProperty('--fonu-' + key, value)
}, { immediate: true, flush: 'sync' })

export function useTheme() {
  function setThemeMode(mode: string) { themeMode.value = normalizeMode(mode) }
  function setThemeAccent(accent: string) { themeAccent.value = normalizeAccent(accent) }
  function toggleTheme() { setThemeMode(isDark.value ? 'light' : 'dark') }
  return { themeMode, themeAccent, naiveTheme, themeOverrides, isDark, palette, toggleTheme, setThemeMode, setThemeAccent }
}
