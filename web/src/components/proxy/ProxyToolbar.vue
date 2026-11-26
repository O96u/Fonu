<template>
  <div class="proxy-toolbar">
    <n-input
      :value="search" @update:value="emit('update:search', $event)"
      clearable
      size="small"
      placeholder="搜索名称、域名或目标地址..."
      class="proxy-toolbar__search"
    >
      <template #prefix><n-icon :component="SearchOutline" /></template>
    </n-input>
    <n-select
      :value="statusFilter" @update:value="emit('update:statusFilter', typeof $event === 'string' ? $event : null)"
      size="small"
      :options="statusOptions"
      clearable
      placeholder="全部状态"
      class="proxy-toolbar__filter"
    />
    <n-select
      :value="httpsFilter" @update:value="emit('update:httpsFilter', typeof $event === 'string' ? $event : null)"
      size="small"
      :options="httpsOptions"
      clearable
      placeholder="全部协议"
      class="proxy-toolbar__filter"
    />
    <div class="proxy-toolbar__spacer" />
    <n-button size="small" quaternary :loading="loading" @click="emit('refresh')">
      <template #icon><n-icon :component="RefreshOutline" /></template>
    </n-button>
  </div>
</template>

<script setup lang="ts">
import { NButton, NIcon, NInput, NSelect, type SelectOption } from 'naive-ui'
import { RefreshOutline, SearchOutline } from '@vicons/ionicons5'

defineProps<{
  search: string
  statusFilter: string | null
  httpsFilter: string | null
  statusOptions: SelectOption[]
  httpsOptions: SelectOption[]
  loading: boolean
}>()
const emit = defineEmits<{
  'update:search': [value: string]
  'update:statusFilter': [value: string | null]
  'update:httpsFilter': [value: string | null]
  refresh: []
}>()
</script>

<style scoped>
.proxy-toolbar {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-4);
  border-bottom: 1px solid var(--fonu-border);
  flex-wrap: wrap;
}

.proxy-toolbar__search {
  width: 240px;
  max-width: 100%;
  flex-shrink: 0;
}

.proxy-toolbar__filter {
  width: 132px;
  flex-shrink: 0;
}

.proxy-toolbar__spacer {
  flex: 1;
  min-width: 0;
}


.proxy-toolbar { background:var(--fonu-bg-muted); padding:10px 12px; gap:8px; }
.proxy-toolbar__search { flex:1 1 220px; width:auto; min-width:0; }
.proxy-toolbar__filter { flex:0 1 132px; min-width:100px; }
.proxy-toolbar__spacer { display:none; }
@media(max-width:760px) {
  .proxy-toolbar__search { flex-basis:100%; }
  .proxy-toolbar__filter { flex:1 1 100px; }
}
</style>
