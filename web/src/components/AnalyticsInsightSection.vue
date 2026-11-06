<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { NTag } from 'naive-ui'
import { RouterLink } from 'vue-router'
import type { AnalyticsOverview, AwarenessRange } from '../api/types'
import { displayRegionName } from '../utils/geoDisplay'
import EmptyState from './EmptyState.vue'
import FonuCard from './FonuCard.vue'
import MetricStat from './MetricStat.vue'

const AnalyticsAccessMap = defineAsyncComponent(() => import('./AnalyticsAccessMap.vue'))

const props = defineProps<{
  data: AnalyticsOverview | null
}>()

const range = defineModel<AwarenessRange>('range', { default: 'today' })
const rangeOptions: { key: AwarenessRange; label: string }[] = [
  { key: 'today', label: '今天' },
  { key: '7d', label: '7 天' },
  { key: '30d', label: '30 天' },
  { key: 'all', label: '全部' },
]

const regionTableWrapRef = ref<HTMLElement | null>(null)
const visibleRowCount = ref(10)
const mapScope = ref<'world' | 'china'>('world')
const mapMetric = ref<'requests' | 'intercept'>('requests')

/** 与全球名单同一行高，国内条目多时不再把行距压扁。 */
const TABLE_HEAD_PX = 28
const TABLE_ROW_PX = 40

let tableResizeObserver: ResizeObserver | null = null

function updateVisibleRowCount() {
  const el = regionTableWrapRef.value
  if (!el) return
  const bodyH = el.clientHeight - TABLE_HEAD_PX
  visibleRowCount.value = Math.max(4, Math.floor(bodyH / TABLE_ROW_PX))
}

onMounted(() => {
  updateVisibleRowCount()
  if (typeof ResizeObserver !== 'undefined' && regionTableWrapRef.value) {
    tableResizeObserver = new ResizeObserver(() => updateVisibleRowCount())
    tableResizeObserver.observe(regionTableWrapRef.value)
  }
})

onUnmounted(() => {
  tableResizeObserver?.disconnect()
  tableResizeObserver = null
})

const sourceColumn = computed(() => (mapScope.value === 'china' ? '省份' : '国家'))
const valueColumn = computed(() => (mapMetric.value === 'intercept' ? '拦截数' : '请求数'))

const regionVisitorsAll = computed(() => {
  const metric = mapMetric.value
  const rows =
    mapScope.value === 'china'
      ? (props.data?.by_province ?? [])
          .filter((r) => r.province && r.province !== '未知' && r.province !== '海外')
          .map((r) => ({
            region: r.province,
            value: metric === 'intercept' ? (r.intercept ?? 0) : r.requests,
          }))
      : (props.data?.by_country ?? [])
          .filter((r) => r.country && r.country !== '未知')
          .map((r) => ({
            region: r.country,
            value: metric === 'intercept' ? (r.intercept ?? 0) : r.requests,
          }))
  return rows.filter((r) => r.value > 0).sort((a, b) => b.value - a.value || a.region.localeCompare(b.region, 'zh'))
})

const regionVisitors = computed(() => regionVisitorsAll.value.slice(0, visibleRowCount.value))

watch(regionVisitorsAll, () => {
  void nextTick(updateVisibleRowCount)
})

</script>

<template>
  <section class="insight-section">
    <div class="insight-section__head">
      <div>
        <h2 class="insight-section__title">访问</h2>
        <p class="insight-section__sub">数字、地图和下面的趋势用同一个时间范围。</p>
      </div>
      <div class="insight-section__tools">
        <div class="insight-range" role="tablist" aria-label="时间范围">
          <button
            v-for="opt in rangeOptions"
            :key="opt.key"
            type="button"
            class="insight-range__btn"
            :class="{ 'insight-range__btn--active': range === opt.key }"
            :aria-selected="range === opt.key"
            @click="range = opt.key"
          >
            {{ opt.label }}
          </button>
        </div>
        <n-tag v-if="data?.backfill_active" size="small" type="warning" round :bordered="false">历史导入中</n-tag>
      </div>
    </div>

    <div class="insight-kpi-row">
      <MetricStat
        label="请求"
        :value="data ? (data.kpi.total_requests ?? 0).toLocaleString() : '—'"
        :compare="!!data && range === 'today'"
        :delta="data?.kpi.request_trend"
        :previous="data ? (data.kpi.yesterday_total ?? 0).toLocaleString() : undefined"
      />
      <MetricStat
        label="独立 IP"
        :value="data ? (data.kpi.unique_ips ?? 0).toLocaleString() : '—'"
        :compare="!!data && range === 'today'"
        :delta="data?.kpi.unique_ip_trend"
        :previous="data ? (data.kpi.yesterday_unique_ips ?? 0).toLocaleString() : undefined"
      />
      <MetricStat
        label="拦截"
        :compare="!!data && range === 'today'"
        :delta="data?.kpi.risk_trend"
        :previous="data ? (data.kpi.yesterday_risk ?? 0).toLocaleString() : undefined"
      >
        <RouterLink :to="{ name: 'logs', query: { tab: 'access', kind: 'intercept' } }">
          {{ data ? (data.kpi.risk_requests ?? 0).toLocaleString() : '—' }}
        </RouterLink>
      </MetricStat>
      <MetricStat
        label="疑似扫描"
        :value="data ? (data.funnel.scan_suspected ?? 0).toLocaleString() : '—'"
        :compare="!!data && range === 'today'"
        :delta="data?.funnel.scan_trend"
        :previous="data ? (data.funnel.yesterday_scan ?? 0).toLocaleString() : undefined"
      />
    </div>

    <FonuCard v-if="data" flush class="insight-panel">
      <div class="insight-panel__grid">
        <div class="insight-panel__block insight-panel__block--map">
          <h4 class="insight-panel__heading">地图</h4>
          <div class="insight-panel__geo-stack">
            <AnalyticsAccessMap v-model:scope="mapScope" v-model:metric="mapMetric" :data="data" />
          </div>
        </div>

        <div class="insight-panel__block insight-panel__block--side">
          <h4 class="insight-panel__heading">来源</h4>
          <div
            v-if="regionVisitorsAll.length"
            ref="regionTableWrapRef"
            class="insight-panel__geo-stack insight-region-fill"
          >
            <div class="insight-region-list">
              <div class="insight-region-list__head">
                <span>{{ sourceColumn }}</span>
                <span class="insight-region-list__num">{{ valueColumn }}</span>
              </div>
              <div class="insight-region-list__body">
                <div v-for="(row, idx) in regionVisitors" :key="`${row.region}-${idx}`" class="insight-region-list__row">
                  <span class="insight-region-list__name">{{ displayRegionName(row.region) }}</span>
                  <span class="insight-region-list__num">{{ row.value.toLocaleString() }}</span>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="insight-panel__geo-stack insight-region-fill insight-region-fill--empty">
            <EmptyState
              :title="mapMetric === 'intercept' ? '暂无拦截' : '暂无访问'"
              :description="mapScope === 'china' ? '当前范围内没有可展示的省份。' : '当前范围内没有可展示的国家。'"
            />
          </div>
        </div>
      </div>
    </FonuCard>
    <FonuCard v-else flush class="insight-panel">
      <p class="insight-wait">正在统计访问，地图随后加载。</p>
    </FonuCard>
  </section>
</template>

<style scoped>
.insight-section {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.insight-section__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  flex-wrap: wrap;
}

.insight-section__title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--fonu-text);
}

.insight-section__sub {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--fonu-text-secondary);
  line-height: 1.5;
}

.insight-section__tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.insight-range {
  display: inline-flex;
  padding: 3px;
  border-radius: 10px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  gap: 2px;
}

.insight-range__btn {
  border: none;
  background: transparent;
  padding: 6px 12px;
  font-size: 13px;
  border-radius: 8px;
  color: var(--fonu-text-secondary);
  cursor: pointer;
  line-height: 1.2;
}

.insight-range__btn--active {
  background: var(--fonu-brand);
  color: #fff;
  font-weight: 600;
}

.insight-range__btn:not(.insight-range__btn--active):hover {
  background: rgba(16, 185, 129, 0.08);
  color: var(--fonu-text);
}

.insight-wait {
  margin: 0;
  padding: 28px 8px;
  text-align: center;
  color: var(--fonu-text-muted);
  font-size: 13px;
}

.insight-kpi-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

@media (max-width: 900px) {
  .insight-kpi-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.insight-panel :deep(.fonu-card__body) {
  padding: var(--fonu-space-4) var(--fonu-space-5) var(--fonu-space-5);
}


.insight-panel__grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(160px, 240px);
  gap: var(--fonu-space-4);
  align-items: stretch;
  --insight-geo-stack-h: clamp(360px, 50vh, 540px);
}

.insight-panel__block--map,
.insight-panel__block--side {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.insight-panel__geo-stack {
  height: var(--insight-geo-stack-h);
  min-height: var(--insight-geo-stack-h);
}

.insight-panel__block--map {
  min-width: 0;
}

.insight-panel__heading {
  margin: 0 0 var(--fonu-space-3);
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.insight-panel__block--map .insight-panel__heading {
  margin-bottom: 8px;
}

.insight-panel__block--side .insight-panel__heading {
  margin-bottom: 8px;
}

.insight-region-fill {
  min-height: 0;
}

.insight-region-fill--empty {
  display: flex;
  align-items: center;
  justify-content: center;
}

.insight-region-list {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.insight-region-list__head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 0 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--fonu-text-muted);
}

.insight-region-list__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.insight-region-list__row {
  flex: 0 0 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 13px;
  color: var(--fonu-text);
}

.insight-region-list__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.insight-region-list__num {
  flex-shrink: 0;
  width: 56px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.insight-panel__block--side {
  min-width: 0;
}

@media (max-width: 960px) {
  .insight-panel__grid {
    grid-template-columns: 1fr;
    --insight-geo-stack-h: clamp(300px, 42vw, 400px);
  }

  .insight-panel__block--map .insight-panel__geo-stack {
    height: var(--insight-geo-stack-h);
    min-height: 300px;
  }

  .insight-panel__block--side .insight-panel__geo-stack,
  .insight-region-fill {
    height: auto;
    min-height: 220px;
    max-height: min(360px, 40vh);
  }
}
</style>
