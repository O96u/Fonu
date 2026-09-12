<template>
  <div class="stat-card">
    <div class="stat-card__header">
      <div class="stat-card__title-row">
        <div v-if="$slots.icon" class="stat-card__icon" :class="`stat-card__icon--${tone}`">
          <slot name="icon" />
        </div>
        <span class="stat-card__label">{{ label }}</span>
      </div>
      <slot name="extra" />
    </div>
    <div v-if="$slots.value" class="stat-card__value-slot">
      <slot name="value" />
    </div>
    <div v-else-if="value !== undefined && value !== ''" class="stat-card__value" :class="{ 'stat-card__value--sm': valueSmall }">
      {{ value }}
    </div>
    <div v-if="sub" class="stat-card__sub">{{ sub }}</div>
    <slot />
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string
    value?: string | number
    sub?: string
    tone?: 'brand' | 'blue' | 'green' | 'teal'
    valueSmall?: boolean
  }>(),
  { tone: 'brand', valueSmall: false },
)
</script>

<style scoped>
.stat-card {
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  padding: var(--fonu-space-5);
  min-height: 140px;
  display: flex;
  flex-direction: column;
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
}

.stat-card:hover {
  box-shadow: var(--fonu-shadow-md);
}

.stat-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-2);
  margin-bottom: var(--fonu-space-3);
}

.stat-card__title-row {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
}

.stat-card__icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
}

.stat-card__icon--brand {
  background: var(--fonu-brand-soft);
  color: var(--fonu-brand);
}

.stat-card__icon--blue {
  background: rgba(59, 130, 246, 0.12);
  color: #3b82f6;
}

.stat-card__icon--green {
  background: rgba(16, 185, 129, 0.12);
  color: #10b981;
}

.stat-card__icon--teal {
  background: rgba(20, 184, 166, 0.12);
  color: #14b8a6;
}

.stat-card__label {
  font-size: 13px;
  font-weight: 500;
  color: var(--fonu-text-secondary);
}

.stat-card__value,
.stat-card__value-slot {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.3;
  color: var(--fonu-text);
  word-break: break-all;
  letter-spacing: -0.02em;
}

.stat-card__value--sm {
  font-size: 18px;
}

.stat-card__sub {
  margin-top: var(--fonu-space-2);
  font-size: 13px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}
</style>
