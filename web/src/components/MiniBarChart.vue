<template>
  <div class="bar-chart">
    <div v-for="(bar, i) in bars" :key="i" class="bar-col">
      <div class="bar-track">
        <div class="bar-fill" :style="{ height: `${bar.pct}%` }" />
      </div>
      <span class="bar-label">{{ bar.label }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  values: number[]
  labels?: string[]
}>()

const bars = computed(() => {
  const max = Math.max(...props.values, 1)
  const defaultLabels = ['00', '04', '08', '12', '16', '20']
  return props.values.map((v, i) => ({
    pct: Math.max(8, (v / max) * 100),
    label: props.labels?.[i] ?? defaultLabels[i] ?? '',
  }))
})
</script>

<style scoped>
.bar-chart {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  height: 140px;
  padding-top: 8px;
}

.bar-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.bar-track {
  width: 100%;
  height: 120px;
  display: flex;
  align-items: flex-end;
}

.bar-fill {
  width: 100%;
  border-radius: 4px 4px 0 0;
  background: linear-gradient(180deg, #34d399 0%, #10b981 100%);
  min-height: 4px;
  transition: height 0.3s ease;
}

.bar-label {
  font-size: 11px;
  color: var(--fonu-text-muted);
}
</style>
