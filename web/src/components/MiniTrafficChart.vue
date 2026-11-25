<template>
  <VChart class="chart" :class="{ 'chart--compact': compact }" :option="option" autoresize />
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

const props = withDefaults(
  defineProps<{
    labels: string[]
    upload: number[]
    download: number[]
    compact?: boolean
  }>(),
  { compact: false },
)

const { palette } = useTheme()

function axisRate(value: number): string {
  if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(0)} MB/s`
  if (value >= 1024) return `${(value / 1024).toFixed(0)} KB/s`
  return `${Math.round(value)} B/s`
}

const option = computed(() => {
  const textColor = palette.value.secondary
  const splitColor = palette.value.border
  const peak = Math.max(0, ...props.upload, ...props.download)
  const idle = peak <= 1

  return {
    grid: { left: 4, right: 8, top: props.compact ? 4 : 8, bottom: props.compact ? 0 : 4, containLabel: true },
    tooltip: {
      trigger: 'axis',
      backgroundColor: palette.value.surface,
      borderColor: palette.value.border,
      textStyle: { color: palette.value.text, fontSize: 12 },
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
      max: idle ? 1024 : undefined,
      interval: idle ? 512 : undefined,
      splitNumber: 2,
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
        lineStyle: { width: 2, color: palette.value.brand },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: palette.value.brand + '40' },
              { offset: 1, color: palette.value.brand + '05' },
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
        lineStyle: { width: 2, color: palette.value.chartSecondary },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: palette.value.chartSecondary + '40' },
              { offset: 1, color: palette.value.chartSecondary + '05' },
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
  height: 100%;
}

.chart--compact {
  min-height: 128px;
}
</style>
