<template>
  <n-form label-placement="top" class="basic-editor">
    <section class="editor-section">
      
      <div v-if="copySource" class="source-note"><n-icon :component="CopyOutline"/><div><strong>复制来源：{{copySource.name||copySource.domain}}</strong><small>{{copySource.domain}} · {{copySource.upstream}}</small><small>请填写新域名。访问认证凭据及手动 Nginx 文本不会复制。其他安全策略可独立修改。</small></div></div>
      <div class="field-grid">
        <n-form-item label="服务名称"><ProxyDraftInput :value="form.name" :maxlength="100" label="服务名称" placeholder="例如：飞牛影视" @update:value="emit('patch',{name:$event})"/></n-form-item>
        <n-form-item label="业务分组"><n-select :value="businessGroup" :options="businessOptions" @update:value="emit('business',$event)"/></n-form-item>
      </div>
      <n-form-item label="前端域名" required>
        <div class="field-stack"><ProxyDraftInput :value="form.hostsSelected.join('\n')" type="textarea" :rows="3" label="前端域名" placeholder="s.example.com&#10;api.example.com&#10;example.com:6893" @update:value="emit('patch',{hostsSelected:$event.split(/\r?\n/)})"/><n-select :value="null" filterable clearable :loading="ddnsHostsLoading" :options="hostSelectOptions" :filter="filterDdnsHostOption" placeholder="从 DDNS 选择域名（可选）" @update:value="appendHost" @focus="emit('loadHosts')"/>
          <small>每行一个域名，可包含端口。无需回车确认，每个域名创建独立规则。</small>
          <small v-if="form.hostsSelected.filter(h=>h.trim()).length>1" class="domain-count">本次涉及 {{form.hostsSelected.filter(h=>h.trim()).length}} 条独立规则</small>
        </div>
      </n-form-item>
        <div class="field-grid"><n-form-item label="监听端口" required><n-input-number :value="form.listen_port" :min="1" :max="65535" @update:value="emit('patch',{listen_port:$event??0})"/></n-form-item><n-form-item label="IP 协议"><div class="ip-options"><n-checkbox :checked="form.listen_ipv4" @update:checked="emit('patch',{listen_ipv4:$event})">IPv4</n-checkbox><n-checkbox :checked="form.listen_ipv6" @update:checked="emit('patch',{listen_ipv6:$event})">IPv6</n-checkbox></div></n-form-item></div>
      <n-form-item label="目标地址" required><div class="field-stack"><ProxyDraftInput :value="form.upstream" label="目标地址" placeholder="http://192.168.1.100:8005" @update:value="emit('patch',{upstream:$event})"/><small>支持 HTTP / HTTPS 内网服务。无需连接测试即可保存。</small><div class="target-action"><n-button size="small" :disabled="!validTarget" @click="openTarget">在浏览器打开目标</n-button><small>仅测试当前浏览器能否访问，不代表 Fonu 后端连通。</small></div></div></n-form-item>
        <div class="option-row"><div><strong>启用 HTTPS</strong><small>使用匹配前端域名的有效证书。</small></div><n-switch :value="form.https_enabled" @update:value="emit('patch',{https_enabled:$event})"/></div>
        <div class="option-row"><div><strong>HTTP 跳转 HTTPS</strong><small>将 HTTP 请求跳转至 HTTPS。</small></div><n-switch :value="form.http_redirect" :disabled="!form.https_enabled" @update:value="emit('patch',{http_redirect:$event})"/></div>
      <div class="option-row"><div><strong>启用规则</strong><small>保存后应用该独立代理规则。</small></div><n-switch :value="form.enabled" @update:value="emit('patch',{enabled:$event})"/></div>
      <div v-if="form.https_enabled" class="section-tip"><n-icon :component="LockClosedOutline"/><div class="field-stack"><span>证书根据域名自动匹配。以下为证书列表预览，最终以服务器实际证书校验为准。</span><small v-if="certificatesLoading">正在读取证书…</small><small v-else-if="certificatesError">暂时无法读取证书列表，请稍后重试。</small><template v-else><small v-if="!certificatePreview.length">填写前端域名后查看匹配状态。</small><div v-for="item in certificatePreview" :key="item.host"><strong>{{item.host}}</strong><small v-if="item.cert">候选证书：{{item.cert.name || item.cert.domain}} · 到期 {{item.cert.expires_at?.slice(0,10)}}</small><small v-else>证书列表中没有未过期的匹配证书</small></div></template><router-link :to="{name:'certificates'}" target="_blank" rel="noopener noreferrer">打开证书管理（申请或导入）</router-link></div></div>
    </section>
  </n-form>
</template>
<script setup lang="ts">
import {computed} from 'vue'
import ProxyDraftInput from './ProxyDraftInput.vue'
import {NForm,NFormItem,NInputNumber,NSelect,NCheckbox,NSwitch,NButton,NIcon,type SelectGroupOption} from 'naive-ui'
import {CopyOutline,LockClosedOutline} from '@vicons/ionicons5'
import type {CertificateRecord,ProxyRule} from '../../api/types'
import type {DdnsHostSelectOption} from '../../utils/ddnsHosts'
import {previewProxyCertificates} from '../../utils/proxyCertificates'
interface BasicDraft {name:string;hostsSelected:string[];listen_port:number;listen_ipv4:boolean;listen_ipv6:boolean;https_enabled:boolean;http_redirect:boolean;tls_min_13_only:boolean;upstream:string;enabled:boolean}
const props=defineProps<{certificates:CertificateRecord[];certificatesLoading:boolean;certificatesError:boolean;form:BasicDraft;businessGroup:number;businessOptions:{label:string;value:number}[];copySource:ProxyRule|null;ddnsHostsLoading:boolean;hostSelectOptions:SelectGroupOption[];filterDdnsHostOption:(pattern:string,option:DdnsHostSelectOption)=>boolean}>()
const emit=defineEmits<{patch:[value:Partial<BasicDraft>];loadHosts:[];business:[value:number]}>()
function appendHost(value:string|null){if(!value)return;const hosts=props.form.hostsSelected.filter(h=>h.trim());if(!hosts.some(h=>h.trim().toLowerCase()===value.toLowerCase()))emit('patch',{hostsSelected:[...hosts,value]})}
const certificatePreview=computed(()=>previewProxyCertificates(props.form.hostsSelected,props.certificates))
const validTarget=computed(()=>{try{const u=new URL(props.form.upstream);return ['http:','https:'].includes(u.protocol)&&!u.username&&!u.password}catch{return false}})
function openTarget(){if(validTarget.value)window.open(props.form.upstream,'_blank','noopener,noreferrer')}
</script>
<style scoped>
.basic-editor{width:100%;min-width:0}.editor-section{display:grid;gap:12px;text-align:left;min-width:0}.section-heading h3{margin:0;font-size:15px}.section-heading p{margin:5px 0 0;font-size:12px;color:var(--fonu-text-secondary)}.field-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.field-stack{display:grid;gap:6px;width:100%;min-width:0}.field-stack small,.option-row small,.source-note small{display:block;font-size:12px;color:var(--fonu-text-secondary);line-height:1.5;overflow-wrap:anywhere}
.basic-editor :deep(.n-form-item){margin:0;min-width:0}.basic-editor :deep(.n-input-number),.basic-editor :deep(.n-select){width:100%;min-width:0}.domain-count{color:var(--fonu-brand-text)!important}.source-note,.section-tip{display:flex;align-items:flex-start;gap:8px;border:1px solid var(--fonu-border);border-radius:8px;background:var(--fonu-bg-muted);padding:12px;font-size:12px;line-height:1.6;overflow-wrap:anywhere}.source-note strong{display:block;font-size:13px}.source-note :deep(.n-icon),.section-tip :deep(.n-icon){font-size:17px;color:var(--fonu-brand-text);flex-shrink:0;margin-top:2px}
.option-row{display:flex;align-items:center;justify-content:space-between;gap:12px;border:1px solid var(--fonu-border);border-radius:8px;padding:12px;background:var(--fonu-bg)}.option-row strong{font-size:13px;font-weight:600}.option-row :deep(.n-switch){flex-shrink:0}.ip-options,.target-action{display:flex;gap:10px;align-items:center;flex-wrap:wrap}.target-action small{flex:1 1 180px}
@container proxy-editor(max-width:520px){.field-grid{grid-template-columns:1fr;gap:2px}.editor-section{gap:10px}}
</style>
