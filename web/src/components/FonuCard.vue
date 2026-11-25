<template>
  <div class="fonu-card" :class="{ 'fonu-card--flush': flush }">
    <div v-if="title || $slots.title || $slots.header" class="fonu-card__header">
      <div>
        <h3 v-if="title" class="fonu-card__title">{{ title }}</h3>
        <div v-else-if="$slots.title" class="fonu-card__title fonu-card__title-row">
          <slot name="title" />
        </div>
        <p v-if="subtitle" class="fonu-card__subtitle">{{ subtitle }}</p>
      </div>
      <div v-if="$slots.header" class="fonu-card__header-extra">
        <slot name="header" />
      </div>
    </div>
    <div class="fonu-card__body">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title?: string
  subtitle?: string
  flush?: boolean
}>()
</script>

<style scoped>
.fonu-card {
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  overflow: hidden;
}

.fonu-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--fonu-space-4);
  padding: var(--fonu-space-4) var(--fonu-space-4) 0;
}

.fonu-card__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--fonu-text);
}

.fonu-card__title-row {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-2);
}

.fonu-card__subtitle {
  margin: var(--fonu-space-1) 0 0;
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.fonu-card__body {
  padding: var(--fonu-space-4);
}

.fonu-card--flush .fonu-card__body {
  padding-top: 0;
}
.fonu-card { min-width: 0; }
.fonu-card__header > div:first-child { min-width: 0; }
.fonu-card__header-extra { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
@media (max-width: 760px) { .fonu-card__header, .fonu-card__body { padding-inline: 12px; } }
</style>
