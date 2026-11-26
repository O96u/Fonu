<template>
  <ProxyEditorFrame :show="showEntryModal" :title="entryModalTitle" subtitle="管理监听端口及 HTTPS 配置" :icon="LayersOutline" :busy="savingEntry" @close="emit('update:showEntryModal',false)">
    <n-form label-placement="top" class="entry-editor">
      <n-form-item label="入口名称"><ProxyDraftInput :value="entryForm.name" :maxlength="100" label="入口名称" placeholder="例如：默认 HTTPS 入口" @update:value="emit('patch',{name:$event})"/></n-form-item>
      <div class="entry-grid"><n-form-item label="监听端口" required><n-input-number :value="entryForm.listen_port" :min="1" :max="65535" @update:value="emit('patch',{listen_port:$event??0})"/></n-form-item><n-form-item label="IP 协议"><div class="entry-ip"><n-checkbox :checked="entryForm.listen_ipv4" @update:checked="emit('patch',{listen_ipv4:$event})">IPv4</n-checkbox><n-checkbox :checked="entryForm.listen_ipv6" @update:checked="emit('patch',{listen_ipv6:$event})">IPv6</n-checkbox></div></n-form-item></div>
      <div class="entry-option"><div><strong>启用 HTTPS</strong><small>前端域名需要匹配有效证书。</small></div><n-switch :value="entryForm.https_enabled" @update:value="emit('patch',{https_enabled:$event})"/></div>
      <div class="entry-option"><div><strong>HTTP 跳转 HTTPS</strong><small>将 HTTP 请求自动重定向。</small></div><n-switch :value="entryForm.http_redirect" :disabled="!entryForm.https_enabled" @update:value="emit('patch',{http_redirect:$event})"/></div>
      <n-alert v-if="cloningSourceId" type="warning" :bordered="false">将复制入口及其 {{cloneRuleCount}} 条独立规则到新端口，两份规则分别管理。</n-alert>
      <n-alert v-else-if="editingEntryId" type="info" :bordered="false">修改入口将影响关联规则的监听设置；规则的域名、目标地址和安全策略仍独立。</n-alert>
      <n-alert v-else type="info" :bordered="false">创建入口后可用于新增代理或发现服务。业务分组与监听入口独立。</n-alert>
    </n-form>
    <template #footer><n-button :disabled="savingEntry" @click="emit('update:showEntryModal',false)">取消</n-button><n-button type="primary" :loading="savingEntry" @click="emit('save')">{{cloningSourceId?'复制入口':'保存入口'}}</n-button></template>
  </ProxyEditorFrame>
</template>
<script setup lang="ts">
import {NButton,NForm,NFormItem,NInputNumber,NCheckbox,NSwitch,NAlert} from 'naive-ui'
import {LayersOutline} from '@vicons/ionicons5'
import ProxyEditorFrame from './ProxyEditorFrame.vue'
import ProxyDraftInput from './ProxyDraftInput.vue'
interface EntryDraft {
  name: string
  listen_port: number
  listen_ipv4: boolean
  listen_ipv6: boolean
  https_enabled: boolean
  http_redirect: boolean
}
defineProps<{
  showEntryModal: boolean
  entryModalTitle: string
  entryForm: EntryDraft
  cloningSourceId: number | null
  editingEntryId: number | null
  cloneRuleCount: number
  savingEntry: boolean
}>()
const emit = defineEmits<{
  'update:showEntryModal': [value: boolean]
  patch: [value: Partial<EntryDraft>]
  save: []
}>()
</script>
<style scoped>
.entry-editor{display:grid;gap:12px}.entry-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.entry-ip{display:flex;gap:14px;flex-wrap:wrap}.entry-editor :deep(.n-input-number){width:100%}.entry-option{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px;border:1px solid var(--fonu-border);border-radius:8px}.entry-option strong,.entry-option small{display:block}.entry-option small{font-size:12px;color:var(--fonu-text-secondary);margin-top:4px}.entry-option :deep(.n-switch){flex-shrink:0}@container proxy-editor(max-width:520px){.entry-grid{grid-template-columns:1fr;gap:0}}
</style>