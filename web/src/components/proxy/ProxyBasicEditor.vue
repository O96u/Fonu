<template>
  <n-form label-placement="top" class="basic-editor">
    <section v-show="section==='basic'" class="editor-section">
      <div class="section-heading"><h3>服务信息</h3><p>为服务命名，指定前端域名和目标地址。</p></div>
      <div v-if="copySource" class="source-note"><n-icon :component="CopyOutline"/><div><strong>复制来源：{{copySource.name||copySource.domain}}</strong><small>{{copySource.domain}} · {{copySource.upstream}}</small><small>请填写新域名。认证密码及手动 Nginx 文本不会复制。</small></div></div>
      <div class="field-grid">
        <n-form-item label="服务名称"><ProxyDraftInput :value="form.name" :maxlength="100" label="服务名称" placeholder="例如：飞牛影视" @update:value="emit('patch',{name:$event})"/></n-form-item>
        <n-form-item label="业务分组"><n-select :value="businessGroup" :options="businessOptions" @update:value="emit('business',$event)"/></n-form-item>
      </div>
      <n-form-item label="前端域名" required>
        <div class="field-stack"><n-select :value="form.hostsSelected" multiple filterable tag :loading="ddnsHostsLoading" :options="hostSelectOptions" :filter="filterDdnsHostOption" placeholder="输入域名并回车，或从 DDNS 选择" @update:value="updateHosts" @focus="emit('loadHosts')"/>
          <small>每个域名将创建独立规则。支持 domain:端口，不会合并规则。</small>
          <small v-if="form.hostsSelected.length>1" class="domain-count">本次涉及 {{form.hostsSelected.length}} 条独立规则</small>
        </div>
      </n-form-item>
      <n-form-item label="目标地址" required><div class="field-stack"><ProxyDraftInput :value="form.upstream" label="目标地址" placeholder="http://192.168.1.100:8005" @update:value="emit('patch',{upstream:$event})"/><small>支持 HTTP / HTTPS 内网服务。无需连接测试即可保存。</small><div class="target-action"><n-button size="small" :disabled="!validTarget" @click="openTarget">在浏览器打开目标</n-button><small>仅测试当前浏览器能否访问，不代表 Fonu 后端连通。</small></div></div></n-form-item>
      <div class="option-row"><div><strong>启用规则</strong><small>保存后应用该独立代理规则。</small></div><n-switch :value="form.enabled" @update:value="emit('patch',{enabled:$event})"/></div>
    </section>
    <section v-show="section==='https'" class="editor-section">
      <div class="section-heading"><h3>监听与 HTTPS</h3><p>直接配置本规则的监听端口和协议，系统自动匹配兼容的监听配置。</p></div>
        <div class="field-grid"><n-form-item label="监听端口" required><n-input-number :value="form.listen_port" :min="1" :max="65535" @update:value="emit('patch',{listen_port:$event??0})"/></n-form-item><n-form-item label="IP 协议"><div class="ip-options"><n-checkbox :checked="form.listen_ipv4" @update:checked="emit('patch',{listen_ipv4:$event})">IPv4</n-checkbox><n-checkbox :checked="form.listen_ipv6" @update:checked="emit('patch',{listen_ipv6:$event})">IPv6</n-checkbox></div></n-form-item></div>
        <div class="option-row"><div><strong>启用 HTTPS</strong><small>使用匹配前端域名的有效证书。</small></div><n-switch :value="form.https_enabled" @update:value="emit('patch',{https_enabled:$event})"/></div>
        <div class="option-row"><div><strong>HTTP 跳转 HTTPS</strong><small>将 HTTP 请求跳转至 HTTPS。</small></div><n-switch :value="form.http_redirect" :disabled="!form.https_enabled" @update:value="emit('patch',{http_redirect:$event})"/></div>
      <div v-if="form.https_enabled" class="option-row"><div><strong>仅 TLS 1.3</strong><small>关闭时兼容 TLS 1.2；旧客户端可能不支持 TLS 1.3。</small></div><n-switch :value="form.tls_min_13_only" @update:value="emit('patch',{tls_min_13_only:$event})"/></div>
      <div v-if="form.https_enabled" class="section-tip"><n-icon :component="LockClosedOutline"/><span>证书根据域名自动匹配。缺少有效证书时，后端拒绝发布无效配置；请先在证书管理中申请或导入。</span></div>
    </section>
  </n-form>
</template>
<script setup lang="ts">
import {computed} from 'vue'
import ProxyDraftInput from './ProxyDraftInput.vue'
import {NForm,NFormItem,NInputNumber,NSelect,NCheckbox,NSwitch,NButton,NIcon,type SelectGroupOption} from 'naive-ui'
import {CopyOutline,LockClosedOutline} from '@vicons/ionicons5'
import type {ProxyRule} from '../../api/types'
import type {DdnsHostSelectOption} from '../../utils/ddnsHosts'
interface BasicDraft {name:string;hostsSelected:string[];listen_port:number;listen_ipv4:boolean;listen_ipv6:boolean;https_enabled:boolean;http_redirect:boolean;tls_min_13_only:boolean;upstream:string;enabled:boolean}
const props=defineProps<{section:'basic'|'https';form:BasicDraft;businessGroup:number;businessOptions:{label:string;value:number}[];copySource:ProxyRule|null;ddnsHostsLoading:boolean;hostSelectOptions:SelectGroupOption[];filterDdnsHostOption:(pattern:string,option:DdnsHostSelectOption)=>boolean}>()
const emit=defineEmits<{patch:[value:Partial<BasicDraft>];loadHosts:[];business:[value:number]}>()
function updateHosts(value:unknown){emit('patch',{hostsSelected:Array.isArray(value)?value.map(String):[]})}
const validTarget=computed(()=>{try{const u=new URL(props.form.upstream);return ['http:','https:'].includes(u.protocol)&&!u.username&&!u.password}catch{return false}})
function openTarget(){if(validTarget.value)window.open(props.form.upstream,'_blank','noopener,noreferrer')}
</script>
<style scoped>
.basic-editor{width:100%;min-width:0}.editor-section{display:grid;gap:12px;text-align:left;min-width:0}.section-heading h3{margin:0;font-size:15px}.section-heading p{margin:5px 0 0;font-size:12px;color:var(--fonu-text-secondary)}.field-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.field-stack{display:grid;gap:6px;width:100%;min-width:0}.field-stack small,.option-row small,.source-note small{display:block;font-size:12px;color:var(--fonu-text-secondary);line-height:1.5;overflow-wrap:anywhere}
.basic-editor :deep(.n-form-item){margin:0;min-width:0}.basic-editor :deep(.n-input-number),.basic-editor :deep(.n-select){width:100%;min-width:0}.domain-count{color:var(--fonu-brand-text)!important}.source-note,.section-tip{display:flex;align-items:flex-start;gap:8px;border:1px solid var(--fonu-border);border-radius:8px;background:var(--fonu-bg-muted);padding:12px;font-size:12px;line-height:1.6;overflow-wrap:anywhere}.source-note strong{display:block;font-size:13px}.source-note :deep(.n-icon),.section-tip :deep(.n-icon){font-size:17px;color:var(--fonu-brand-text);flex-shrink:0;margin-top:2px}
.option-row{display:flex;align-items:center;justify-content:space-between;gap:12px;border:1px solid var(--fonu-border);border-radius:8px;padding:12px}.option-row strong{font-size:13px;font-weight:600}.option-row :deep(.n-switch){flex-shrink:0}.ip-options,.target-action{display:flex;gap:10px;align-items:center;flex-wrap:wrap}.target-action small{flex:1 1 180px}
@container proxy-editor(max-width:520px){.field-grid{grid-template-columns:1fr;gap:2px}.editor-section{gap:10px}}
</style>
