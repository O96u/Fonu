<template>
  <ProxyEditorFrame :show="showDiscoveryModal" title="发现内网服务" subtitle="扫描常见 Web 端口，批量创建独立代理规则" :icon="SearchOutline" :busy="discoveryLoading||discoveryAdding" wide @close="emit('update:showDiscoveryModal',false)">
    <div class="discovery-workspace">
      <section class="scan-card">
        <div class="scan-title"><n-icon :component="SearchOutline"/><strong>扫描目标</strong></div>
        <div class="scan-input"><ProxyDraftInput :value="discoveryHost" :disabled="discoveryLoading||discoveryAdding" label="扫描地址" placeholder="127.0.0.1 或内网 IP" @update:value="emit('update:discoveryHost',$event)" @enter="emit('scan')"/><n-button type="primary" :loading="discoveryLoading" :disabled="discoveryAdding" @click="emit('scan')">{{scanned?'重新扫描':'开始发现'}}</n-button></div>
        <small>扫描由 Fonu 后端执行，仅识别支持的常见 HTTP / HTTPS 服务。重新扫描同一地址时保留已填写内容。</small>
        <div v-if="discoveryLoading" class="scan-status"><n-spin size="small"/>正在扫描 {{discoveryScanHost}} …</div>
      </section>
      <div class="discovery-options"><label>监听端口<n-input-number :value="listenPort" :min="1" :max="65535" :disabled="discoveryAdding" @update:value="emit('update:listenPort',$event??0)"/></label><label>协议<n-select :value="httpsEnabled?'https':'http'" :options="[{label:'HTTPS',value:'https'},{label:'HTTP',value:'http'}]" :disabled="discoveryAdding" @update:value="emit('update:httpsEnabled',$event==='https')"/></label><label>业务分组<n-select :value="businessGroup" :options="businessOptions" :disabled="discoveryAdding" @update:value="emit('update:businessGroup',$event)"/></label></div>
      <n-alert v-if="error" type="error" :bordered="false">{{error}}</n-alert>
      <section class="scan-results">
        <div class="result-heading"><div><strong>发现结果</strong><small>{{discoverySummary|| (scanned?'扫描完成':'填写目标后开始扫描')}}</small></div><n-checkbox v-if="discoveryRows.length" :checked="discoverySelectAllChecked" :indeterminate="discoverySelectAllIndeterminate" :disabled="discoveryLoading||discoveryAdding" @update:checked="emit('selectAll',$event)">全选</n-checkbox></div>
        <n-empty v-if="!discoveryRows.length&&!discoveryLoading" :description="scanned?'未发现可添加的服务':'尚未扫描'"/>
        <article v-for="row in discoveryRows" :key="row.id" class="scan-result" :class="{'scan-result--selected':row.selected}">
          <div class="scan-result-head"><n-checkbox :checked="row.selected" :disabled="discoveryLoading||discoveryAdding" :aria-label="'选择 '+row.service.name" @update:checked="emit('patchRow',row.id,{selected:$event})"/><div class="scan-result-icon"><n-icon :component="ServerOutline"/></div><div class="scan-result-title"><strong>{{row.service.name}}</strong><small>{{row.service.upstream}}</small></div><n-tag size="small" :bordered="false" type="success">已发现</n-tag></div>
          <div v-if="row.service.suggestion" class="scan-hint">{{row.service.suggestion}}</div>
          <div class="scan-fields"><label>规则名称<ProxyDraftInput :value="row.ruleName" small :maxlength="100" label="规则名称" placeholder="服务名称" :disabled="!row.selected||discoveryLoading||discoveryAdding" @update:value="emit('patchRow',row.id,{ruleName:$event})"/></label><label>前端域名<ProxyDraftInput :value="row.hostname" small label="前端域名" placeholder="nas.example.com" :disabled="!row.selected||discoveryLoading||discoveryAdding" @update:value="emit('patchRow',row.id,{hostname:$event})"/></label></div>
        </article>
      </section>
    </div>
    <template #footer><span class="scan-count">已选 {{selectedCount}} 个 · {{missingCount?'尚有 '+missingCount+' 项缺少域名':'每个服务创建独立规则'}}</span><n-button :disabled="discoveryLoading||discoveryAdding" @click="emit('update:showDiscoveryModal',false)">取消</n-button><n-button type="primary" :loading="discoveryAdding" :disabled="!hasSelectedDiscoveryRows||discoveryLoading" @click="emit('submit')">创建 {{selectedCount}} 条规则</n-button></template>
  </ProxyEditorFrame>
</template>
<script setup lang="ts">
import {computed} from 'vue'
import {NButton,NIcon,NCheckbox,NSpin,NEmpty,NSelect,NAlert,NTag,NInputNumber} from 'naive-ui'
import {SearchOutline,ServerOutline} from '@vicons/ionicons5'
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
.discovery-workspace{display:grid;gap:14px}.scan-card{border:1px solid var(--fonu-border);border-radius:9px;padding:12px;background:var(--fonu-bg-muted);display:grid;gap:9px}.scan-title,.scan-input,.scan-status,.result-heading,.scan-result-head{display:flex;align-items:center;gap:9px}.scan-title :deep(.n-icon){color:var(--fonu-brand-text);font-size:18px}.scan-input :deep(.draft-field){flex:1;min-width:0}.scan-card small,.result-heading small,.scan-result-title small{display:block;color:var(--fonu-text-secondary);font-size:12px;overflow-wrap:anywhere}.scan-status{color:var(--fonu-brand-text);font-size:12px}.discovery-options,.scan-fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px}.discovery-options label,.scan-fields label{display:grid;gap:5px;font-size:12px;color:var(--fonu-text-secondary);min-width:0}.scan-results{display:grid;gap:9px}.result-heading{justify-content:space-between;flex-wrap:wrap}.scan-result{border:1px solid var(--fonu-border);border-radius:8px;padding:11px;display:grid;gap:9px}.scan-result--selected{border-color:var(--fonu-brand);background:color-mix(in srgb,var(--fonu-brand-soft) 22%,var(--fonu-surface))}.scan-result-icon{display:grid;place-items:center;width:33px;height:33px;border-radius:7px;background:var(--fonu-bg-muted);color:var(--fonu-brand-text);flex-shrink:0;font-size:18px}.scan-result-title{flex:1;min-width:0;overflow-wrap:anywhere}.scan-hint{font-size:11px;color:var(--fonu-text-secondary)}.scan-count{flex:1 1 130px;font-size:12px;color:var(--fonu-text-secondary)}
@container proxy-editor(max-width:520px){.discovery-options,.scan-fields{grid-template-columns:1fr}.scan-input{flex-wrap:wrap}.scan-input :deep(.draft-field){flex-basis:100%}}
</style>