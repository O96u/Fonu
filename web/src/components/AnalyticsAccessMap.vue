<script setup lang="ts">
import { computed, nextTick, onActivated, onUnmounted, ref, watch } from 'vue'
import { use, registerMap } from 'echarts/core'
import { MapChart, LinesChart, EffectScatterChart, ScatterChart } from 'echarts/charts'
import { GeoComponent, TooltipComponent, VisualMapComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import type { AnalyticsOverview } from '../api/types'
import { useTheme } from '../composables/useTheme'
import { resolveProvinceCoord, resolveServerHub } from '../utils/chinaMapCoords'
import { normalizeCountryKey, resolveCountryGeo, mapFeatureDisplayName } from '../utils/worldMapCoords'
import chinaJson from '../assets/geo/china.json'
import worldJson from '../assets/geo/world.json'
import EmptyState from './EmptyState.vue'

use([CanvasRenderer, MapChart, LinesChart, EffectScatterChart, ScatterChart, GeoComponent, TooltipComponent, VisualMapComponent])

registerMap('china', chinaJson as unknown as Parameters<typeof registerMap>[1])
registerMap('world', worldJson as unknown as Parameters<typeof registerMap>[1])

const props = defineProps<{
  data: AnalyticsOverview | null
}>()

type MapScope = 'china' | 'world'
type MapMetric = 'requests' | 'intercept'

const mapReady = ref(true)
const mapStageRef = ref<HTMLElement | null>(null)
const layoutSizePercent = ref(100)
const chartInitOptions = { devicePixelRatio: Math.min(typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1, 2) }
const scope = defineModel<MapScope>('scope', { default: 'world' })
const metric = defineModel<MapMetric>('metric', { default: 'requests' })
const { isDark } = useTheme()

/** 国内 / 全球各自独立的布局比例（勿共用同一组参数） */
type LayoutProfile = {
  /** 未提供 bbox 时，用容器宽高比估算 layoutSize */
  mapAspect: number
  minPct: number
  maxPct: number
  gain: number
  center: [string, string]
  /**
   * 绘制宽高比 = bboxAspect × aspectScale。
   * 国内 GeoJSON 经度跨度约 1.74×纬度；0.75 会把整图压扁并裁切，飞线弧就甩进海里。
   */
  aspectScale: number
  /** 经纬度包围盒宽/高；有值时按容器完整装入，不再放大裁切 */
  bboxAspect?: number
  fill?: number
}

const LAYOUT_PROFILE: Record<MapScope, LayoutProfile> = {
  china: {
    mapAspect: 1.18,
    minPct: 80,
    maxPct: 220,
    gain: 1,
    center: ['50%', '50%'],
    // 1.739 × 0.68 ≈ 1.18，接近常见中国地图观感，且整图落在画布内
    bboxAspect: 1.739,
    aspectScale: 0.68,
    fill: 0.98,
  },
  world: {
    mapAspect: 1.72,
    minPct: 108,
    maxPct: 182,
    gain: 1.08,
    center: ['50%', '52%'],
    aspectScale: 0.74,
  },
}

const layoutCenter = ref<[string, string]>(LAYOUT_PROFILE.world.center)

let stageResizeObserver: ResizeObserver | null = null

function updateMapLayout() {
  const el = mapStageRef.value
  if (!el) return
  const w = el.clientWidth
  const h = el.clientHeight
  if (w < 16 || h < 16) return
  const cfg = LAYOUT_PROFILE[scope.value]
  let pct: number
  if (cfg.bboxAspect) {
    const geoAspect = cfg.bboxAspect * cfg.aspectScale
    const minSide = Math.min(w, h)
    const size = geoAspect >= 1 ? Math.min(w, h * geoAspect) : Math.min(h, w / geoAspect)
    pct = (size / minSide) * 100 * (cfg.fill ?? 1)
  } else {
    const ratio = w / h
    pct = ratio >= cfg.mapAspect ? (100 * ratio) / cfg.mapAspect : (100 * cfg.mapAspect) / ratio
    pct *= cfg.gain
  }
  layoutSizePercent.value = Math.min(cfg.maxPct, Math.max(cfg.minPct, Math.round(pct)))
  layoutCenter.value = cfg.center
}

function bindStageObserver() {
  stageResizeObserver?.disconnect()
  stageResizeObserver = null
  updateMapLayout()
  const el = mapStageRef.value
  if (!el || typeof ResizeObserver === 'undefined') return
  stageResizeObserver = new ResizeObserver(() => updateMapLayout())
  stageResizeObserver.observe(el)
}

onActivated(() => {
  void nextTick(bindStageObserver)
})

onUnmounted(() => {
  stageResizeObserver?.disconnect()
  stageResizeObserver = null
})

function normalizeMapName(province: string): string {
  const p = province.trim()
  const suffixes = ['特别行政区', '维吾尔自治区', '壮族自治区', '回族自治区', '自治区', '省', '市']
  for (const s of suffixes) {
    if (p.endsWith(s)) return p.slice(0, -s.length) || p
  }
  return p
}

type RegionRow = { label: string; requests: number; intercept: number; mapName: string }

const chinaRegions = computed((): RegionRow[] => {
  return (props.data?.by_province ?? [])
    .filter((r) => r.province && r.province !== '未知' && r.province !== '海外')
    .map((r) => ({
      label: r.province,
      requests: r.requests,
      intercept: r.intercept ?? 0,
      mapName: normalizeMapName(r.province),
    }))
})

const worldRegions = computed((): RegionRow[] => {
  const merged = new Map<string, RegionRow>()
  for (const r of props.data?.by_country ?? []) {
    if (!r.country || r.country === '未知') continue
    const geo = resolveCountryGeo(r.country)
    if (!geo) continue
    const display = normalizeCountryKey(r.country)
    const cur = merged.get(geo.mapName) ?? {
      label: display,
      requests: 0,
      intercept: 0,
      mapName: geo.mapName,
    }
    cur.label = display
    cur.requests += r.requests
    cur.intercept += r.intercept ?? 0
    merged.set(geo.mapName, cur)
  }
  return [...merged.values()]
})

const activeRegions = computed(() => (scope.value === 'china' ? chinaRegions.value : worldRegions.value))

const mapData = computed(() =>
  activeRegions.value.map((r) => {
    const raw = metric.value === 'intercept' ? (r.intercept ?? 0) : r.requests
    const value = Number.isFinite(raw) ? raw : 0
    return {
      name: r.mapName,
      value,
      requests: r.requests,
      intercept: r.intercept ?? 0,
    }
  }),
)

function fmtCount(n: unknown): number {
  const v = typeof n === 'number' ? n : Number(n)
  return Number.isFinite(v) ? v : 0
}

const option = computed(() => {
  if (!mapReady.value || !props.data) return {}
  const border = isDark.value ? '#334155' : '#e5e7eb'
  const area = isDark.value ? '#1e293b' : '#f1f5f9'
  const isChina = scope.value === 'china'
  const showAccessLines = metric.value === 'requests'

  const accessLines: { coords: [number, number][]; value: number; name: string }[] = []
  const interceptLines: { coords: [number, number][]; value: number; name: string }[] = []
  const hub = resolveServerHub(props.data.server)
  const hubLabel = props.data.server?.label?.trim() || '本机'

  if (hub) {
    if (isChina) {
      for (const row of props.data.by_province ?? []) {
        const from = resolveProvinceCoord(row.province)
        if (!from) continue
        const label = normalizeMapName(row.province)
        if (showAccessLines && row.requests > 0) {
          accessLines.push({ coords: [from, hub], value: row.requests, name: label })
        }
        if (!showAccessLines && (row.intercept ?? 0) > 0) {
          interceptLines.push({ coords: [from, hub], value: row.intercept, name: label })
        }
      }
    } else {
      for (const row of props.data.by_country ?? []) {
        const geo = resolveCountryGeo(row.country)
        if (!geo) continue
        const from = geo.coord
        const label = normalizeCountryKey(row.country)
        if (showAccessLines && row.requests > 0) {
          accessLines.push({ coords: [from, hub], value: row.requests, name: label })
        }
        if (!showAccessLines && (row.intercept ?? 0) > 0) {
          interceptLines.push({ coords: [from, hub], value: row.intercept, name: label })
        }
      }
    }
  }

  const maxVal = Math.max(...mapData.value.map((d) => d.value), 1)

  const colorRange =
    metric.value === 'intercept' ? ['#fecaca', '#ef4444', '#991b1b'] : ['#d1fae5', '#10b981', '#047857']

  const layoutCfg = LAYOUT_PROFILE[scope.value]

  const geoLayout = {
    layoutCenter: layoutCenter.value,
    layoutSize: `${layoutSizePercent.value}%`,
    aspectScale: layoutCfg.aspectScale,
  }

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: isDark.value ? '#1e293b' : '#fff',
      borderColor: border,
      textStyle: { color: isDark.value ? '#f1f5f9' : '#111827', fontSize: 12 },
      formatter: (p: { name?: string; value?: unknown; seriesName?: string }) => {
        const place = p.name ? mapFeatureDisplayName(p.name) : ''
        if (p.seriesName === '访问飞线') return `${place}<br/>正常访问 ${fmtCount(p.value)}`
        if (p.seriesName === '拦截飞线') return `${place}<br/>拦截/风险 ${fmtCount(p.value)}`
        const row = mapData.value.find((d) => d.name === p.name)
        const title = mapFeatureDisplayName(p.name ?? '')
        const shown = fmtCount(row ? (metric.value === 'intercept' ? row.intercept : row.requests) : p.value)
        if (row || p.seriesName === '访问量' || p.seriesName === '拦截量') {
          return `${title}<br/>${metric.value === 'intercept' ? '拦截' : '请求'} ${shown}${
            row ? `<br/>请求 ${row.requests}<br/>拦截 ${row.intercept}` : ''
          }`
        }
        return place
      },
    },
    visualMap: {
      show: false,
      seriesIndex: 0,
      min: 0,
      max: maxVal,
      calculable: false,
      inRange: { color: colorRange },
    },
    geo: {
      map: isChina ? 'china' : 'world',
      roam: true,
      scaleLimit: { min: 0.85, max: 4 },
      label: { show: false },
      ...geoLayout,
      itemStyle: {
        areaColor: area,
        borderColor: border,
        borderWidth: isChina ? 0.8 : 0.5,
      },
      emphasis: {
        itemStyle: { areaColor: isDark.value ? '#334155' : '#e2e8f0' },
        label: { show: false },
      },
    },
    series: [
      {
        name: metric.value === 'intercept' ? '拦截量' : '访问量',
        type: 'map',
        geoIndex: 0,
        label: { show: false },
        data: mapData.value,
      },
      {
        name: '访问飞线',
        type: 'lines',
        coordinateSystem: 'geo',
        geoIndex: 0,
        z: 3,
        effect: {
          show: showAccessLines && accessLines.length > 0,
          period: 4,
          trailLength: 0.15,
          symbol: 'arrow',
          symbolSize: 5,
          color: '#10b981',
        },
        lineStyle: {
          color: '#34d399',
          width: 1,
          opacity: 0.9,
          curveness: isChina ? 0.2 : 0.25,
        },
        data: accessLines.map((l) => ({
          name: l.name,
          coords: l.coords,
          value: l.value,
          lineStyle: { width: Math.min(3, 0.5 + Math.log10(l.value + 1)) },
        })),
      },
      {
        name: '拦截飞线',
        type: 'lines',
        coordinateSystem: 'geo',
        geoIndex: 0,
        z: 4,
        effect: {
          show: interceptLines.length > 0,
          period: 3,
          trailLength: 0.12,
          symbol: 'arrow',
          symbolSize: 4,
          color: '#ef4444',
        },
        lineStyle: {
          color: '#f87171',
          width: 1,
          opacity: 0.9,
          curveness: isChina ? 0.22 : 0.28,
          type: 'dashed',
        },
        data: interceptLines.map((l) => ({
          name: l.name,
          coords: l.coords,
          value: l.value,
          lineStyle: { width: Math.min(2.5, 0.4 + Math.log10(l.value + 1)) },
        })),
      },
      {
        name: '服务节点',
        type: 'effectScatter',
        coordinateSystem: 'geo',
        geoIndex: 0,
        z: 5,
        rippleEffect: { brushType: 'stroke', scale: 3 },
        symbolSize: isChina ? 10 : 8,
        itemStyle: { color: '#3b82f6', shadowBlur: 8, shadowColor: 'rgba(59,130,246,0.5)' },
        data: hub ? [{ name: hubLabel, value: [...hub, 1] }] : [],
      },
    ],
  }
})

const hasMapData = computed(() => {
  if (scope.value === 'china') {
    return chinaRegions.value.length > 0
  }
  return (props.data?.by_country ?? []).some((r) => r.country && r.country !== '未知' && resolveCountryGeo(r.country))
})

watch(
  () => mapReady.value && hasMapData.value,
  (visible) => {
    if (visible) void nextTick(bindStageObserver)
  },
  { immediate: true },
)

watch(scope, () => {
  updateMapLayout()
  void nextTick(bindStageObserver)
})
</script>

<template>
  <div class="access-map">
    <div class="access-map__toolbar">
      <div class="access-map__seg" role="group" aria-label="地图范围">
        <button type="button" class="access-map__seg-btn" :class="{ 'access-map__seg-btn--active': scope === 'world' }" @click="scope = 'world'">
          全球
        </button>
        <button type="button" class="access-map__seg-btn" :class="{ 'access-map__seg-btn--active': scope === 'china' }" @click="scope = 'china'">
          国内
        </button>
      </div>
      <div class="access-map__seg" role="group" aria-label="着色指标">
        <button
          type="button"
          class="access-map__seg-btn"
          :class="{ 'access-map__seg-btn--active': metric === 'requests' }"
          @click="metric = 'requests'"
        >
          请求数
        </button>
        <button
          type="button"
          class="access-map__seg-btn"
          :class="{ 'access-map__seg-btn--active': metric === 'intercept' }"
          @click="metric = 'intercept'"
        >
          拦截数
        </button>
      </div>
      <div class="access-map__legend">
        <span class="access-map__legend-item access-map__legend-item--access"><i /> 访问</span>
        <span class="access-map__legend-item access-map__legend-item--block"><i /> 拦截</span>
        <span class="access-map__legend-item access-map__legend-item--hub"><i /> 本机</span>
        <span class="access-map__legend-hint">滚轮缩放 · 拖动平移</span>
      </div>
    </div>
    <div v-if="mapReady && hasMapData" ref="mapStageRef" class="access-map__stage">
      <VChart :key="scope" class="access-map__chart" :option="option" :init-options="chartInitOptions" autoresize />
      <div class="access-map__density" aria-hidden="true">
        <span>高</span>
        <span
          class="access-map__density-bar"
          :class="metric === 'intercept' ? 'access-map__density-bar--intercept' : 'access-map__density-bar--requests'"
        />
        <span>低</span>
      </div>
    </div>
    <EmptyState v-else-if="mapReady" title="暂无地图数据" description="产生带归属地的访问后将显示飞线。" />
  </div>
</template>

<style scoped>
.access-map {
  position: relative;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.access-map__stage {
  position: relative;
  flex: 1;
  min-height: 0;
  width: 100%;
  display: flex;
  flex-direction: column;
}

.access-map__chart {
  flex: 1;
  min-height: 0;
  width: 100%;
  height: 100%;
}

.access-map :deep(.empty-state) {
  flex: 1;
  min-height: 0;
}

.access-map__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  margin-bottom: 0;
  flex-shrink: 0;
}

.access-map__seg {
  display: inline-flex;
  padding: 2px;
  border-radius: 8px;
  background: var(--fonu-border);
  gap: 2px;
}

.access-map__seg-btn {
  border: none;
  background: transparent;
  padding: 5px 14px;
  font-size: 12px;
  font-weight: 500;
  color: var(--fonu-text-muted);
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.access-map__seg-btn--active {
  background: var(--fonu-brand, #10b981);
  color: #fff;
}

.access-map__legend {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 14px;
  margin-left: auto;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.access-map__density {
  position: absolute;
  left: 10px;
  bottom: 10px;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--fonu-text-muted);
  pointer-events: none;
}

.access-map__density-bar {
  width: 8px;
  height: 64px;
  border-radius: 4px;
}

.access-map__density-bar--requests {
  background: linear-gradient(180deg, #047857, #10b981, #d1fae5);
}

.access-map__density-bar--intercept {
  background: linear-gradient(180deg, #991b1b, #ef4444, #fecaca);
}

.access-map__legend-hint {
  font-size: 11px;
  color: var(--fonu-text-muted);
  opacity: 0.85;
}

@media (max-width: 720px) {
  .access-map__legend {
    margin-left: 0;
    width: 100%;
  }

  .access-map__legend-hint {
    width: 100%;
  }
}

@media (min-width: 721px) {
  .access-map__legend-hint {
    margin-left: 4px;
  }
}

.access-map__legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.access-map__legend-item i {
  display: inline-block;
  width: 14px;
  height: 3px;
  border-radius: 2px;
}

.access-map__legend-item--access i {
  background: #10b981;
}

.access-map__legend-item--block i {
  background: #ef4444;
  background-image: repeating-linear-gradient(90deg, #ef4444 0 4px, transparent 4px 7px);
}

.access-map__legend-item--hub i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #3b82f6;
}
</style>
