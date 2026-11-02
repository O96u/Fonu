<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    label: string
    value?: string
    hint?: string
    /** 相对昨日的变化，单位是百分比。没有数据时显示 —。 */
    delta?: number | null
    previous?: string
    compare?: boolean
  }>(),
  { compare: false, delta: null },
)

const deltaText = computed(() => {
  const n = props.delta
  if (n == null || Number.isNaN(n)) return '—'
  const abs = Math.abs(n)
  const digits = abs >= 10 || Number.isInteger(n) ? 0 : 1
  const sign = n > 0 ? '+' : ''
  return `${sign}${n.toFixed(digits)}%`
})

const deltaClass = computed(() => {
  const n = props.delta
  if (n == null || Number.isNaN(n) || n === 0) return ''
  return n > 0 ? 'metric__delta--up' : 'metric__delta--down'
})
</script>

<template>
  <article class="metric">
    <div class="metric__label">
      {{ label }}
      <span v-if="hint" class="metric__hint">{{ hint }}</span>
    </div>
    <div class="metric__value">
      <slot>{{ value }}</slot>
    </div>
    <div v-if="compare" class="metric__foot">
      <span>较昨日</span>
      <span class="metric__delta" :class="deltaClass">{{ deltaText }}</span>
      <span v-if="previous" class="metric__prev">昨日 {{ previous }}</span>
    </div>
  </article>
</template>

<style scoped>
.metric {
  min-width: 0;
  min-height: 108px;
  padding: 14px 16px;
  border-radius: 12px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.metric__label {
  display: flex;
  align-items: baseline;
  gap: 6px;
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.metric__hint {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.metric__value {
  font-size: 26px;
  font-weight: 700;
  line-height: 1.15;
  font-variant-numeric: tabular-nums;
  color: var(--fonu-text);
}

.metric__value :deep(a) {
  color: inherit;
  text-decoration: none;
}

.metric__value :deep(a:hover) {
  text-decoration: underline;
}

.metric__foot {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: auto;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.metric__delta--up {
  color: #ef4444;
  font-weight: 600;
}

.metric__delta--down {
  color: #10b981;
  font-weight: 600;
}

.metric__prev {
  margin-left: auto;
}
</style>
