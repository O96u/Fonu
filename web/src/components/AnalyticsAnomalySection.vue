<script setup lang="ts">
import { computed } from 'vue'
import { use } from 'echarts/core'
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { RouterLink } from 'vue-router'
import type { AnalyticsAnomaly, AnalyticsOverview } from '../api/types'
import { useTheme } from '../composables/useTheme'
import EmptyState from './EmptyState.vue'
import MetricStat from './MetricStat.vue'

use([CanvasRenderer, BarChart, LineChart, GridComponent, LegendComponent, TooltipComponent])

const props = defineProps<{
  data: AnalyticsOverview | null
}>()

const { isDark } = useTheme()

const anomaly = computed<AnalyticsAnomaly>(() => {
  return (
    props.data?.anomaly ?? {
      slow_threshold_ms: 1000,
      total_requests: 0,
      status_5xx: { current: 0, previous: 0, delta: 0, change_rate: 0, share: 0 },
      status_4xx: { current: 0, previous: 0, delta: 0, change_rate: 0, share: 0 },
      avg_request_ms: { current: 0, previous: 0, delta: 0, change_rate: 0 },
      slow_rate: { current: 0, previous: 0, delta: 0, change_rate: 0 },
      trend: [],
      top_urls: [],
    }
  )
})

const topURLs = computed(() => (anomaly.value.top_urls ?? []).slice(0, 5))

function ms(n: number): string {
  if (!Number.isFinite(n)) return '0 ms'
  return `${Math.round(n)} ms`
}

const cards = computed(() => {
  const a = anomaly.value
  return [
    {
      key: '5xx',
      label: '5xx',
      hint: '服务端错误',
      value: Math.round(a.status_5xx.current).toLocaleString(),
      delta: a.status_5xx.change_rate,
      previous: Math.round(a.status_5xx.previous).toLocaleString(),
    },
    {
      key: '4xx',
      label: '4xx',
      hint: '客户端或拦截',
      value: Math.round(a.status_4xx.current).toLocaleString(),
      delta: a.status_4xx.change_rate,
      previous: Math.round(a.status_4xx.previous).toLocaleString(),
    },
    {
      key: 'latency',
      label: '平均耗时',
      hint: '不含长连接',
      value: ms(a.avg_request_ms.current),
      delta: a.avg_request_ms.change_rate,
      previous: ms(a.avg_request_ms.previous),
    },
    {
      key: 'slow',
      label: '慢请求',
      hint: `超过 ${a.slow_threshold_ms} ms`,
      value: `${a.slow_rate.current.toFixed(2)}%`,
      delta: a.slow_rate.change_rate,
      previous: `${a.slow_rate.previous.toFixed(2)}%`,
    },
  ]
})

const option = computed(() => {
  const text = isDark.value ? '#94a3b8' : '#9ca3af'
  const split = isDark.value ? '#334155' : '#f3f4f6'
  const trend = anomaly.value.trend ?? []
  return {
    grid: { left: 8, right: 12, top: 28, bottom: 28, containLabel: true },
    legend: {
      bottom: 0,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: text, fontSize: 11 },
      data: ['4xx 请求数', '5xx 请求数', '平均请求耗时'],
    },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: trend.map((p) => p.label),
      axisLabel: { color: text, fontSize: 11 },
      axisLine: { lineStyle: { color: split } },
      axisTick: { show: false },
    },
    yAxis: [
      {
        type: 'value',
        minInterval: 1,
        axisLabel: { color: text, fontSize: 11 },
        splitLine: { lineStyle: { color: split } },
      },
      {
        type: 'value',
        axisLabel: { color: text, fontSize: 11, formatter: '{value} ms' },
        splitLine: { show: false },
      },
    ],
    series: [
      {
        name: '4xx 请求数',
        type: 'bar',
        barMaxWidth: 18,
        itemStyle: { color: '#fdba74', borderRadius: [4, 4, 0, 0] },
        data: trend.map((p) => p.status_4xx),
      },
      {
        name: '5xx 请求数',
        type: 'bar',
        barMaxWidth: 18,
        itemStyle: { color: '#fb7185', borderRadius: [4, 4, 0, 0] },
        data: trend.map((p) => p.status_5xx),
      },
      {
        name: '平均请求耗时',
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        itemStyle: { color: '#60a5fa' },
        data: trend.map((p) => Math.round(p.avg_request_ms)),
      },
    ],
  }
})

function deltaText(n: number, unit: string): string {
  const sign = n > 0 ? '+' : ''
  const shown = unit === 'ms' ? Math.round(n) : n
  return unit ? `${sign}${shown} ${unit}` : `${sign}${shown}`
}
</script>

<template>
  <section class="anomaly">
    <div class="anomaly__head">
      <div>
        <h2 class="anomaly__title">异常</h2>
        <p class="anomaly__sub">只看今天，和昨天比。图是最近 7 天。</p>
      </div>
    </div>

    <div v-if="data" class="anomaly__cards">
      <MetricStat
        v-for="card in cards"
        :key="card.key"
        :label="card.label"
        :hint="card.hint"
        :value="card.value"
        compare
        :delta="card.delta"
        :previous="card.previous"
      />
    </div>
    <p v-else class="anomaly__wait">正在汇总今天的异常。</p>

    <div v-if="data" class="anomaly__grid">
      <div class="anomaly__panel">
        <div class="anomaly__panel-head">
          <h3>最近 7 天</h3>
          <p>柱是 4xx / 5xx，线是平均耗时。</p>
        </div>
        <VChart class="anomaly__chart" :option="option" autoresize />
      </div>

      <div class="anomaly__panel">
        <div class="anomaly__panel-head">
          <h3>问题 URL</h3>
          <p>5xx 或慢请求较多的路径。</p>
        </div>
        <div v-if="topURLs.length" class="anomaly-table">
          <div class="anomaly-table__head">
            <span>访问</span>
            <span>5xx</span>
            <span>平均耗时</span>
            <span>慢请求</span>
            <span>对比昨日</span>
          </div>
          <div v-for="row in topURLs" :key="row.path" class="anomaly-table__row">
            <div class="anomaly-table__path">
              <span class="mono" :title="row.path">{{ row.path }}</span>
              <small>请求数 {{ row.requests.toLocaleString() }}</small>
              <RouterLink class="anomaly-table__link" :to="{ name: 'logs', query: { tab: 'access', keyword: row.path } }">
                查看 URL 日志
              </RouterLink>
            </div>
            <span>{{ row.errors_5xx.toLocaleString() }}</span>
            <span>{{ ms(row.avg_request_ms) }}</span>
            <span>{{ row.slow.toLocaleString() }}</span>
            <span class="anomaly-table__delta">
              5xx {{ deltaText(row.errors_5xx_delta, '') }} / 耗时 {{ deltaText(row.avg_request_delta_ms, 'ms') }}
            </span>
          </div>
        </div>
        <EmptyState v-else title="暂无问题 URL" description="出现 5xx、超过 1 秒，或平均耗时偏高的路径后会列在这里。" />
      </div>
    </div>
  </section>
</template>

<style scoped>
.anomaly {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.anomaly__head {
  display: flex;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  align-items: flex-start;
  flex-wrap: wrap;
}

.anomaly__title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--fonu-text);
}

.anomaly__sub {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.anomaly__wait {
  margin: 0;
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.anomaly__cards {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.anomaly__grid {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.9fr);
  gap: var(--fonu-space-4);
}

.anomaly__panel {
  min-width: 0;
  padding: 16px 18px 12px;
  border-radius: 16px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
}

.anomaly__panel-head h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 650;
  color: var(--fonu-text);
}

.anomaly__panel-head p {
  margin: 4px 0 8px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.anomaly__chart {
  height: 280px;
}

.anomaly-table__head,
.anomaly-table__row {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) 52px 72px 64px minmax(120px, 1fr);
  gap: 8px;
  align-items: center;
}

.anomaly-table__head {
  font-size: 12px;
  color: var(--fonu-text-muted);
  padding-bottom: 8px;
}

.anomaly-table__row {
  padding: 10px 0;
  border-top: 1px solid var(--fonu-border);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.anomaly-table__path {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.anomaly-table__path .mono {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}

.anomaly-table__path small,
.anomaly-table__delta {
  color: var(--fonu-text-muted);
  font-size: 12px;
}

.anomaly-table__link {
  color: var(--fonu-brand);
  font-size: 12px;
  text-decoration: none;
}

.anomaly-table__link:hover {
  text-decoration: underline;
}

@media (max-width: 1100px) {
  .anomaly__cards,
  .anomaly__grid {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 720px) {
  .anomaly__cards,
  .anomaly__grid {
    grid-template-columns: 1fr;
  }

  .anomaly-table__head {
    display: none;
  }

  .anomaly-table__row {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
