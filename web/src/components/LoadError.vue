<template>
  <div class="load-error">
    <n-result status="error" title="加载失败" :description="message">
      <template #footer>
        <n-button type="primary" @click="$emit('retry')">重新加载</n-button>
      </template>
    </n-result>
  </div>
</template>

<script setup lang="ts">
import { NButton, NResult } from 'naive-ui'

withDefaults(
  defineProps<{
    message?: string
  }>(),
  { message: '请检查 Fonu 服务是否正常运行' },
)

defineEmits<{ retry: [] }>()
</script>

<style scoped>
.load-error {
  padding: var(--fonu-space-6) 0;
}
</style>
