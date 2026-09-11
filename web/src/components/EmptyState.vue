<template>
  <div class="empty-state">
    <div class="empty-state__icon">{{ icon }}</div>
    <h3 class="empty-state__title">{{ title }}</h3>
    <p class="empty-state__desc">{{ description }}</p>
    <div v-if="$slots.action" class="empty-state__action">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    description: string
    icon?: string
  }>(),
  { icon: '○' },
)
</script>

<style scoped>
.empty-state {
  padding: var(--fonu-space-6) var(--fonu-space-5);
  text-align: center;
}

.empty-state__icon {
  width: 48px;
  height: 48px;
  margin: 0 auto var(--fonu-space-4);
  border-radius: 50%;
  background: var(--fonu-brand-soft);
  color: var(--fonu-brand);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 600;
}

.empty-state__title {
  margin: 0 0 var(--fonu-space-2);
  font-size: 16px;
  font-weight: 600;
  color: var(--fonu-text);
}

.empty-state__desc {
  margin: 0 auto;
  max-width: 360px;
  font-size: 14px;
  line-height: 1.6;
  color: var(--fonu-text-secondary);
}

.empty-state__action {
  margin-top: var(--fonu-space-5);
}
</style>
