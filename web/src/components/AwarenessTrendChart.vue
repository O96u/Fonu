<template>
  <VChart class="chart" :option="option" autoresize />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { useTheme } from '../composables/useTheme'

use([CanvasRenderer, LineChart, GridComponent, LegendComponent, TooltipComponent])

const props = defineProps<{
  labels: string[]
  requests: number[]
  uniqueIPs: number[]
  currentBucket: number
}>()

const { palette } = useTheme()

const option = computed(() => {
  const textColor = palette.value.secondary
  const splitColor = palette.value.border
  const end = Math.min(props.currentBucket + 1, props.labels.length)
  const labels = props.labels.slice(0, end)
  const requests = props.requests.slice(0, end)
  const uniqueIPs = props.uniqueIPs.slice(0, end)

  return {
    grid: { left: 4, right: 8, top: 28, bottom: 4, containLabel: true },
    legend: {
      top: 0,
      right: 0,
      textStyle: { color: textColor, fontSize: 11 },
    },
    tooltip: {
      trigger: 'axis',
      backgroundColor: palette.value.surface,
      borderColor: palette.value.border,
      textStyle: { color: palette.value.text, fontSize: 12 },
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: labels,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: textColor, fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      min: 0,
      splitNumber: 3,
      splitLine: { lineStyle: { color: splitColor, type: 'dashed' } },
      axisLabel: { color: textColor, fontSize: 11 },
    },
    series: [
      {
        name: '访问次数',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: requests,
        lineStyle: { width: 2, color: palette.value.chartSecondary },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: palette.value.chartSecondary + '38' },
              { offset: 1, color: palette.value.chartSecondary + '05' },
            ],
          },
        },
      },
      {
        name: '独立 IP',
        type: 'line',
        smooth: true,
        showSymbol: false,
        data: uniqueIPs,
        color: palette.value.brand,
        lineStyle: { width: 2, color: palette.value.brand },
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
