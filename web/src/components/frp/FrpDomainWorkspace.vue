<template>
  <section class="domain-workspace">
    <div class="domain-tools"><strong>穿透域名 · {{ domains.length }}</strong><n-input v-model:value="query" clearable placeholder="搜索域名" /></div>
    <div v-if="!filtered.length" class="empty">{{ domains.length ? '没有匹配的域名' : '暂无域名，点击新增或同步反代' }}</div>
    <article v-for="domain in filtered" :key="domain" class="domain-row">
      <div class="domain-icon"><n-icon :component="GlobeOutline" /></div>
      <div class="domain-content"><strong :title="domain">{{ domain }}</strong><small>HTTP / HTTPS · Fonu Nginx</small></div>
      <StatusBadge :kind="httpState === 'registered' && httpsState === 'registered' ? 'success' : httpState === 'failed' || httpsState === 'failed' ? 'error' : 'unknown'" :text="httpState === 'registered' && httpsState === 'registered' ? '网关已注册' : httpState === 'failed' || httpsState === 'failed' ? '网关异常' : '待确认'" />
      <div class="domain-actions">
        <n-button size="tiny" quaternary @click="emit('copy', domain)">复制</n-button>
        <n-button v-if="!domain.startsWith('*.')" size="tiny" quaternary tag="a" :href="'https://' + domain" target="_blank" rel="noopener noreferrer">打开</n-button>
        <n-button size="tiny" quaternary :disabled="busy" @click="emit('edit', domain)">编辑</n-button>
        <n-button size="tiny" quaternary type="error" :disabled="busy" @click="emit('remove', domain)">删除</n-button>
      </div>
    </article>
    <small>网关注册不代表域名 DNS、证书和上游服务均可访问。</small>
  </section>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import { NButton, NIcon, NInput } from 'naive-ui'
import { GlobeOutline } from '@vicons/ionicons5'
import StatusBadge from '../StatusBadge.vue'
const props = defineProps<{ domains: string[]; busy: boolean; httpState?: string; httpsState?: string }>()
const emit = defineEmits<{ edit: [domain:string]; remove:[domain:string]; copy:[domain:string] }>()
const query = ref('')
const filtered = computed(() => props.domains.filter(d => d.toLowerCase().includes(query.value.trim().toLowerCase())))
</script>
<style scoped>
.domain-workspace{display:grid;gap:8px;padding:0 16px 16px;min-width:0}
.domain-tools{display:flex;align-items:center;justify-content:space-between;gap:8px;flex-wrap:wrap}.domain-tools .n-input{width:min(100%,280px)}
.domain-row{display:grid;grid-template-columns:36px minmax(0,1fr) auto auto;gap:10px;align-items:center;padding:12px;border:1px solid var(--fonu-border);border-radius:var(--fonu-radius);min-width:0}
.domain-icon{width:36px;height:36px;display:grid;place-items:center;background:var(--fonu-bg-muted);border-radius:var(--fonu-radius);font-size:20px;color:var(--fonu-text-secondary)}
.domain-content{min-width:0;display:grid;gap:3px}.domain-content strong{overflow-wrap:anywhere;color:var(--fonu-text)}small,.empty{color:var(--fonu-text-secondary);font-size:12px}
.domain-actions{display:flex;flex-wrap:wrap;gap:2px;justify-content:flex-end}.empty{padding:16px;text-align:center;border:1px dashed var(--fonu-border);border-radius:var(--fonu-radius)}
@media(max-width:760px){.domain-workspace{padding:0 8px 12px}.domain-row{grid-template-columns:36px minmax(0,1fr);gap:8px}.domain-row :deep(.status-badge){grid-column:2;justify-self:start}.domain-actions{grid-column:1/-1;justify-content:flex-end;border-top:1px solid var(--fonu-border);padding-top:6px}}
</style>
