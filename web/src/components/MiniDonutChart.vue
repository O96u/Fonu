<template>
  <div class="donut-wrap">
    <svg viewBox="0 0 120 120" class="donut">
      <circle cx="60" cy="60" r="44" fill="none" stroke="#f3f4f6" stroke-width="16" />
      <circle
        v-for="(seg, i) in segments"
        :key="i"
        cx="60"
        cy="60"
        r="44"
        fill="none"
        :stroke="seg.color"
        stroke-width="16"
        :stroke-dasharray="`${seg.len} ${circumference}`"
        :stroke-dashoffset="seg.offset"
        transform="rotate(-90 60 60)"
      />
    </svg>
    <div class="donut-legend">
      <div v-for="item in legend" :key="item.label" class="legend-row">
        <span class="legend-dot" :style="{ background: item.color }" />
        <span class="legend-label">{{ item.label }}</span>
        <span class="legend-value">{{ item.pct }}%</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  segments: { label: string; value: number; color: string }[]
}>()

const circumference = 2 * Math.PI * 44

const total = computed(() => props.segments.reduce((s, x) => s + x.value, 0) || 1)

const legend = computed(() =>
  props.segments.map((s) => ({
    label: s.label,
    color: s.color,
    pct: Math.round((s.value / total.value) * 100),
  })),
)

const segments = computed(() => {
  let offset = 0
  return props.segments.map((s) => {
    const len = (s.value / total.value) * circumference
    const seg = { color: s.color, len, offset: -offset }
    offset += len
    return seg
  })
})
</script>

<style scoped>
.donut-wrap {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-5);
}

.donut {
  width: 120px;
  height: 120px;
  flex-shrink: 0;
}

.donut-legend {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.legend-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.legend-label {
  flex: 1;
  color: var(--fonu-text-secondary);
}

.legend-value {
  font-weight: 600;
  color: var(--fonu-text);
}
</style>
