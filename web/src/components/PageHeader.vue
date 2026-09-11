<template>
  <div class="page-header">
    <div class="page-header__main">
      <h1 class="page-header__title">{{ title }}</h1>
      <p v-if="description" class="page-header__desc">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="page-header__actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-5);
  flex-wrap: wrap;
}

.page-header__title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.3;
  color: var(--fonu-text);
  letter-spacing: -0.02em;
}

.page-header__desc {
  margin: var(--fonu-space-2) 0 0;
  font-size: 14px;
  color: var(--fonu-text-secondary);
  line-height: 1.5;
}

.page-header__actions {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  flex-wrap: wrap;
}
</style>
