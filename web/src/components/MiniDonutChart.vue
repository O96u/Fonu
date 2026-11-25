<template>
  <VChart class="chart" :option="option" autoresize />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { use } from 'echarts/core'
import { PieChart } from 'echarts/charts'
import { LegendComponent, TitleComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { useTheme } from '../composables/useTheme'

use([CanvasRenderer, PieChart, LegendComponent, TitleComponent, TooltipComponent])

const props = defineProps<{
  segments: { label: string; value: number; color: string }[]
}>()

const { palette } = useTheme()

const option = computed(() => {
  const textColor = palette.value.secondary
  const titleColor = palette.value.text
  const total = props.segments.reduce((sum, s) => sum + s.value, 0)

  return {
    title: [
      {
        text: String(total),
        left: '31%',
        top: '38%',
        textAlign: 'center',
        textStyle: { fontSize: 22, fontWeight: 700, color: titleColor },
      },
      {
        text: '总请求',
        left: '31%',
        top: '52%',
        textAlign: 'center',
        textStyle: { fontSize: 12, color: textColor },
      },
    ],
    tooltip: {
      trigger: 'item',
      backgroundColor: palette.value.surface,
      borderColor: palette.value.border,
      textStyle: { color: palette.value.text, fontSize: 12 },
      formatter: '{b}: {c} ({d}%)',
    },
    legend: {
      orient: 'vertical',
      right: 0,
      top: 'center',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 12,
      textStyle: { color: textColor, fontSize: 13 },
      formatter: (name: string) => {
        const seg = props.segments.find((s) => s.label === name)
        const total = props.segments.reduce((sum, s) => sum + s.value, 0) || 1
        const pct = seg ? Math.round((seg.value / total) * 100) : 0
        return `${name}  ${pct}% (${seg?.value ?? 0})`
      },
    },
    series: [
      {
        type: 'pie',
        radius: ['52%', '72%'],
        center: ['32%', '50%'],
        avoidLabelOverlap: false,
        label: { show: false },
        emphasis: {
          scale: true,
          scaleSize: 4,
        },
        data: props.segments.map((s) => ({
          name: s.label,
          value: s.value,
          itemStyle: { color: s.color },
        })),
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
