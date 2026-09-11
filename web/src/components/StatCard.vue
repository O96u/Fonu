<template>
  <div class="stat-card">
    <div class="stat-card__header">
      <span class="stat-card__label">{{ label }}</span>
      <slot name="extra" />
    </div>
    <div class="stat-card__value">{{ value }}</div>
    <div v-if="sub" class="stat-card__sub">{{ sub }}</div>
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  sub?: string
}>()
</script>

<style scoped>
.stat-card {
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  padding: var(--fonu-space-5);
  min-height: 120px;
}

.stat-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-2);
  margin-bottom: var(--fonu-space-3);
}

.stat-card__label {
  font-size: 13px;
  font-weight: 500;
  color: var(--fonu-text-secondary);
}

.stat-card__value {
  font-size: 24px;
  font-weight: 600;
  line-height: 1.2;
  color: var(--fonu-text);
  word-break: break-all;
}

.stat-card__sub {
  margin-top: var(--fonu-space-2);
  font-size: 13px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}
</style>
