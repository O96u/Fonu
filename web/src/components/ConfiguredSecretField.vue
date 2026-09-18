<template>
  <div class="settings-field">
    <div class="settings-field__label configured-secret-label">
      <span>{{ label }}</span>
      <n-tag v-if="configured" size="small" type="success" :bordered="false">{{ CONFIGURED_SECRET_TAG }}</n-tag>
    </div>
    <n-input
      :value="modelValue"
      type="password"
      show-password-on="click"
      :placeholder="configured ? CONFIGURED_SECRET_PLACEHOLDER : placeholder"
      @update:value="emit('update:modelValue', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { NInput, NTag } from 'naive-ui'
import { CONFIGURED_SECRET_PLACEHOLDER, CONFIGURED_SECRET_TAG } from '../constants/secretField'

withDefaults(
  defineProps<{
    label: string
    modelValue: string
    configured?: boolean
    placeholder?: string
  }>(),
  {
    configured: false,
    placeholder: '',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<style scoped>
.configured-secret-label {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
