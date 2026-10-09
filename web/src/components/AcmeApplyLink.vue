<template>
  <a
    class="acme-apply-link"
    :href="url"
    target="_blank"
    rel="noopener noreferrer"
  >
    <n-icon :component="OpenOutline" class="acme-apply-link__icon" />
    <span>{{ linkText }}</span>
  </a>
</template>

<script setup lang="ts">
import { NIcon } from 'naive-ui'
import { OpenOutline } from '@vicons/ionicons5'

defineProps<{
  url: string
  linkText: string
}>()
</script>

<style scoped>
.acme-apply-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  font-weight: 500;
  color: var(--fonu-primary);
  text-decoration: none;
}

.acme-apply-link:hover {
  text-decoration: underline;
}

.acme-apply-link__icon {
  font-size: 14px;
  flex-shrink: 0;
}
</style>
