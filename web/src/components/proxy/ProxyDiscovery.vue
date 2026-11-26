<template>
  <ProxyEditorFrame :show="showDiscoveryModal" title="发现内网服务" subtitle="扫描常见 Web 端口，批量创建独立代理规则" :icon="SearchOutline" :busy="discoveryLoading||discoveryAdding" wide @close="emit('update:showDiscoveryModal',false)">
    <div class="discovery-workspace">
      <section class="scan-card">
        
        <div class="scan-input"><ProxyDraftInput :value="discoveryHost" :disabled="discoveryLoading||discoveryAdding" label="扫描地址" placeholder="127.0.0.1 或内网 IP" @update:value="emit('update:discoveryHost',$event)" @enter="emit('scan')"/><n-button type="primary" :loading="discoveryLoading" :disabled="discoveryAdding" @click="emit('scan')">{{scanned?'重新扫描':'开始发现'}}</n-button></div>
        <small>{{discoverySummary || "扫描常见 HTTP / HTTPS 端口，勾选服务后填写实际域名。"}}</small>
        <div v-if="discoveryLoading" class="scan-status"><n-spin size="small"/>正在扫描 {{discoveryScanHost}} …</div>
      </section>
      <div class="discovery-options"><label>监听端口<n-input-number :value="listenPort" :min="1" :max="65535" :disabled="discoveryAdding" @update:value="emit('update:listenPort',$event??0)"/></label><label>协议<n-select :value="httpsEnabled?'https':'http'" :options="[{label:'HTTPS',value:'https'},{label:'HTTP',value:'http'}]" :disabled="discoveryAdding" @update:value="emit('update:httpsEnabled',$event==='https')"/></label><label>业务分组<n-select :value="businessGroup" :options="businessOptions" :disabled="discoveryAdding" @update:value="emit('update:businessGroup',$event)"/></label></div>
      <n-alert v-if="error" type="error" :bordered="false">{{error}}</n-alert>
      <section class="scan-results">
        <div class="scan-table-head"><n-checkbox :checked="discoverySelectAllChecked" :indeterminate="discoverySelectAllIndeterminate" :disabled="discoveryLoading||discoveryAdding||!discoveryRows.length" aria-label="全选" @update:checked="emit('selectAll',$event)"/><strong>服务</strong><strong>规则名称</strong><strong>前端域名</strong></div>
        <n-empty v-if="!discoveryRows.length&&!discoveryLoading" :description="scanned?'未发现可添加的服务':'尚未扫描'"/>
        <article v-for="row in discoveryRows" :key="row.id" class="scan-result">
          <n-checkbox :checked="row.selected" :disabled="discoveryLoading||discoveryAdding" :aria-label="'选择 '+row.service.name" @update:checked="emit('patchRow',row.id,{selected:$event})"/>
          <div class="scan-result-title"><small>{{row.service.upstream}}</small><strong>{{row.service.name}}</strong><small v-if="row.service.suggestion">{{row.service.suggestion}}</small></div>
          <label><span class="scan-mobile-label">规则名称</span><ProxyDraftInput :value="row.ruleName" small :maxlength="100" label="规则名称" placeholder="服务名称" :disabled="!row.selected||discoveryLoading||discoveryAdding" @update:value="emit('patchRow',row.id,{ruleName:$event})"/></label>
          <label><span class="scan-mobile-label">前端域名</span><ProxyDraftInput :value="row.hostname" small label="前端域名" placeholder="nas.example.com" :disabled="!row.selected||discoveryLoading||discoveryAdding" @update:value="emit('patchRow',row.id,{hostname:$event})"/></label>
        </article>
      </section>
    </div>
    <template #footer><span class="scan-count">已选 {{selectedCount}} 个 · {{missingCount?'尚有 '+missingCount+' 项缺少域名':'每个服务创建独立规则'}}</span><n-button :disabled="discoveryLoading||discoveryAdding" @click="emit('update:showDiscoveryModal',false)">取消</n-button><n-button type="primary" :loading="discoveryAdding" :disabled="!hasSelectedDiscoveryRows||discoveryLoading" @click="emit('submit')">创建 {{selectedCount}} 条规则</n-button></template>
  </ProxyEditorFrame>
</template>
<script setup lang="ts">
import {computed} from 'vue'
import {NButton,NCheckbox,NSpin,NEmpty,NSelect,NAlert,NInputNumber} from 'naive-ui'
import {SearchOutline} from '@vicons/ionicons5'
import type {DiscoveredService} from '../../api/types'
import ProxyEditorFrame from './ProxyEditorFrame.vue'
import ProxyDraftInput from './ProxyDraftInput.vue'
interface DiscoveryRow{id:string;selected:boolean;service:DiscoveredService;ruleName:string;hostname:string}
const props=defineProps<{showDiscoveryModal:boolean;discoveryHost:string;listenPort:number;httpsEnabled:boolean;businessOptions:{label:string;value:number}[];businessGroup:number;discoveryLoading:boolean;discoveryScanHost:string;discoverySummary:string;discoveryRows:DiscoveryRow[];discoverySelectAllChecked:boolean;discoverySelectAllIndeterminate:boolean;discoveryAdding:boolean;hasSelectedDiscoveryRows:boolean;error:string;scanned:boolean}>()
const emit=defineEmits<{'update:showDiscoveryModal':[value:boolean];'update:discoveryHost':[value:string];'update:businessGroup':[value:number];'update:listenPort':[value:number];'update:httpsEnabled':[value:boolean];scan:[];selectAll:[checked:boolean];patchRow:[id:string,patch:Partial<DiscoveryRow>];submit:[]}>()
const selectedCount=computed(()=>props.discoveryRows.filter(row=>row.selected).length)
const missingCount=computed(()=>props.discoveryRows.filter(row=>row.selected&&!row.hostname.trim()).length)
</script>
<style scoped>
.discovery-workspace{display:grid;gap:12px;min-width:0}.scan-card{display:grid;gap:6px}.scan-input,.scan-status{display:flex;align-items:center;gap:8px}.scan-input :deep(.draft-field){flex:1;min-width:0}.scan-card small,.scan-result-title small{display:block;color:var(--fonu-text-secondary);font-size:12px;overflow-wrap:anywhere}.scan-status{color:var(--fonu-brand-text);font-size:12px}.discovery-options{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px}.discovery-options label{display:grid;gap:4px;font-size:12px;color:var(--fonu-text-secondary);min-width:0}.scan-results{border:1px solid var(--fonu-border);border-radius:8px;overflow:hidden;min-width:0}.scan-table-head,.scan-result{display:grid;grid-template-columns:24px minmax(0,1.2fr) minmax(0,1fr) minmax(0,1.1fr);align-items:center;gap:10px;padding:10px 12px}.scan-table-head{background:var(--fonu-bg-muted);font-size:12px;color:var(--fonu-text-secondary)}.scan-result{border-top:1px solid var(--fonu-border)}.scan-result-title,.scan-result label{min-width:0;overflow-wrap:anywhere}.scan-result-title strong{display:block;font-size:13px;margin:4px 0}.scan-mobile-label{display:none}.scan-count{flex:1 1 130px;font-size:12px;color:var(--fonu-text-secondary)}
@container proxy-editor(max-width:600px){.discovery-options{grid-template-columns:repeat(2,minmax(0,1fr))}.scan-table-head{display:none}.scan-result{grid-template-columns:24px minmax(0,1fr);gap:7px}.scan-result label{grid-column:2}.scan-mobile-label{display:block;font-size:12px;margin-bottom:4px;color:var(--fonu-text-secondary)}.scan-input{flex-wrap:wrap}.scan-input :deep(.draft-field){flex-basis:100%}}
</style>
