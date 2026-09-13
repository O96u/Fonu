<template>
  <VChart class="chart" :option="option" autoresize />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { useTheme } from '../composables/useTheme'

use([CanvasRenderer, BarChart, GridComponent, TooltipComponent])

const props = defineProps<{
  values: number[]
  labels?: string[]
}>()

const { isDark } = useTheme()

const option = computed(() => {
  const labels = props.labels ?? ['00', '04', '08', '12', '16', '20']
  const textColor = isDark.value ? '#94a3b8' : '#6b7280'
  const splitColor = isDark.value ? '#334155' : '#e5e7eb'

  return {
    grid: { left: 8, right: 8, top: 12, bottom: 28, containLabel: true },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      backgroundColor: isDark.value ? '#1e293b' : '#fff',
      borderColor: isDark.value ? '#334155' : '#e5e7eb',
      textStyle: { color: isDark.value ? '#f1f5f9' : '#111827', fontSize: 12 },
      formatter: (params: { name: string; value: number }[]) => {
        const p = params[0]
        if (!p) return ''
        return `${p.name}:00<br/>请求数 ${p.value}`
      },
    },
    xAxis: {
      type: 'category',
      data: labels,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: textColor, fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: splitColor, type: 'dashed' } },
      axisLabel: { color: textColor, fontSize: 11 },
    },
    series: [
      {
        type: 'bar',
        data: props.values,
        barMaxWidth: 28,
        itemStyle: {
          borderRadius: [4, 4, 0, 0],
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: '#6ee7b7' },
              { offset: 1, color: '#10b981' },
            ],
          },
        },
        emphasis: {
          itemStyle: {
            color: {
              type: 'linear',
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: '#a7f3d0' },
                { offset: 1, color: '#059669' },
              ],
            },
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
  height: 200px;
}
</style>
