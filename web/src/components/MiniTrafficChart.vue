<template>
  <VChart class="chart" :option="option" autoresize />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { useTheme } from '../composables/useTheme'
import { formatRate } from '../utils/format'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent])

const props = defineProps<{
  labels: string[]
  upload: number[]
  download: number[]
}>()

const { isDark } = useTheme()

function axisRate(value: number): string {
  if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(0)} MB/s`
  if (value >= 1024) return `${(value / 1024).toFixed(0)} KB/s`
  return `${Math.round(value)} B/s`
}

const option = computed(() => {
  const textColor = isDark.value ? '#94a3b8' : '#9ca3af'
  const splitColor = isDark.value ? '#334155' : '#e5e7eb'

  return {
    grid: { left: 4, right: 8, top: 8, bottom: 4, containLabel: true },
    tooltip: {
      trigger: 'axis',
      backgroundColor: isDark.value ? '#1e293b' : '#fff',
      borderColor: isDark.value ? '#334155' : '#e5e7eb',
      textStyle: { color: isDark.value ? '#f1f5f9' : '#111827', fontSize: 12 },
      formatter: (params: { seriesName: string; value: number; axisValue: string }[]) => {
        const time = params[0]?.axisValue ?? ''
        const lines = params.map((p) => `${p.seriesName} ${formatRate(p.value)}`)
        return `${time}<br/>${lines.join('<br/>')}`
      },
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: props.labels,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: textColor, fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      min: 0,
      splitNumber: 3,
      splitLine: { lineStyle: { color: splitColor, type: 'dashed' } },
      axisLabel: {
        color: textColor,
        fontSize: 11,
        formatter: (v: number) => axisRate(v),
      },
    },
    series: [
      {
        name: '上传',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: props.upload,
        lineStyle: { width: 2, color: '#10b981' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(16, 185, 129, 0.25)' },
              { offset: 1, color: 'rgba(16, 185, 129, 0.02)' },
            ],
          },
        },
      },
      {
        name: '下载',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: props.download,
        lineStyle: { width: 2, color: '#3b82f6' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(59, 130, 246, 0.25)' },
              { offset: 1, color: 'rgba(59, 130, 246, 0.02)' },
            ],
          },
        },
      },
    ],
  }
})
</script>

<style scoped>
.chart {
  width: 100%;
  height: 160px;
}
</style>
