<template>
  <div class="proxy-toolbar">
    <n-input :value="search" @update:value="emit('update:search',$event)" clearable size="small" placeholder="搜索名称、域名、入口或目标地址" class="proxy-toolbar__search"><template #prefix><n-icon :component="SearchOutline" /></template></n-input>
    <n-select :value="statusFilter" @update:value="emit('update:statusFilter',typeof $event==='string'?$event:null)" size="small" :options="statusOptions" clearable placeholder="全部状态" class="proxy-toolbar__filter" />
    <n-select :value="httpsFilter" @update:value="emit('update:httpsFilter',typeof $event==='string'?$event:null)" size="small" :options="httpsOptions" clearable placeholder="全部协议" class="proxy-toolbar__filter" />
    <n-button v-if="search || statusFilter || httpsFilter" size="small" quaternary @click="emit('clear')">清除</n-button>
    <div class="proxy-toolbar__actions">
      <div class="view-switch" role="group" aria-label="规则视图">
        <button type="button" :aria-pressed="viewMode==='grouped'" @click="emit('update:viewMode','grouped')"><n-icon :component="LayersOutline"/>入口</button>
        <button type="button" :aria-pressed="viewMode==='flat'" @click="emit('update:viewMode','flat')"><n-icon :component="ListOutline"/>列表</button>
      </div>
      <n-button v-if="grouped" size="small" quaternary :disabled="!canExpand" @click="emit('toggleAll')">{{ allExpanded?'全部收起':'全部展开' }}</n-button>
      <n-button size="small" quaternary :loading="loading" title="刷新规则" aria-label="刷新规则" @click="emit('refresh')"><template #icon><n-icon :component="RefreshOutline"/></template></n-button>
    </div>
  </div>
</template>
<script setup lang="ts">
import {NButton,NIcon,NInput,NSelect,type SelectOption} from 'naive-ui'
import {RefreshOutline,SearchOutline,LayersOutline,ListOutline} from '@vicons/ionicons5'
defineProps<{search:string;statusFilter:string|null;httpsFilter:string|null;statusOptions:SelectOption[];httpsOptions:SelectOption[];loading:boolean;viewMode:'grouped'|'flat';grouped:boolean;canExpand:boolean;allExpanded:boolean}>()
const emit=defineEmits<{'update:search':[value:string];'update:statusFilter':[value:string|null];'update:httpsFilter':[value:string|null];'update:viewMode':[value:'grouped'|'flat'];refresh:[];toggleAll:[];clear:[]}>()
</script>
<style scoped>
.proxy-toolbar{display:flex;align-items:center;gap:8px;flex-wrap:wrap;padding:10px 12px;border-block:1px solid var(--fonu-border);background:var(--fonu-surface)}
.proxy-toolbar__search{flex:1 1 220px;min-width:0}
.proxy-toolbar__filter{flex:0 1 112px;min-width:98px}
.proxy-toolbar__actions{display:flex;align-items:center;gap:6px;flex-wrap:wrap;margin-left:auto}
.view-switch{display:flex;padding:2px;gap:2px;border:1px solid var(--fonu-border);border-radius:6px;background:var(--fonu-bg-muted)}
.view-switch button{display:flex;align-items:center;gap:5px;padding:4px 9px;border:0;border-radius:4px;background:transparent;color:var(--fonu-text-secondary);font:inherit;font-size:12px;cursor:pointer;white-space:nowrap}
.view-switch button[aria-pressed=true]{background:var(--fonu-surface);color:var(--fonu-text);box-shadow:0 1px 3px rgba(0,0,0,.08);font-weight:600}
.view-switch button:focus-visible{outline:2px solid var(--fonu-brand)}
@media(max-width:760px){.proxy-toolbar__search{flex-basis:100%}.proxy-toolbar__filter{flex:1 1 100px}.proxy-toolbar__actions{margin-left:0;flex:1 1 100%;justify-content:flex-start}}
</style>