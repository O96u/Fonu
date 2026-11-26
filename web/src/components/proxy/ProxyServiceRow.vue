<template>
  <div class="service-row" :class="{'service-row--child':child,'service-row--card':card}">
    <div class="service-identity">
      <ServiceFavicon v-if="!child" :rule-id="rule.id"/>
      <n-icon v-else :component="GlobeOutline" class="domain-icon"/>
      <div class="service-identity__body">
        <button v-if="!child" class="service-name" :title="name" type="button" @click="emit('detail')">{{ name }}</button>
        <div class="service-hosts">
          <a class="primary-host" :href="accessUrl(rule,hosts[0]||rule.domain)" target="_blank" rel="noopener noreferrer" :title="accessUrl(rule,hosts[0]||rule.domain)">{{ accessUrl(rule,hosts[0]||rule.domain).replace(/^https?:\/\//, '') }}</a>
          <a class="open-link" :href="accessUrl(rule,hosts[0]||rule.domain)" target="_blank" rel="noopener noreferrer" title="在新窗口打开（含监听端口）" aria-label="在新窗口打开"><n-icon :component="OpenOutline"/></a>
          <n-button size="tiny" quaternary aria-label="复制链接" title="复制链接" @click="emit('copy-link',accessUrl(rule,hosts[0]||rule.domain))"><template #icon><n-icon :component="CopyOutline"/></template></n-button>
          <n-popover v-if="hosts.length>1" trigger="click" placement="bottom-start">
            <template #trigger><n-button size="tiny" quaternary>+{{ hosts.length-1 }} 域名</n-button></template>
            <div class="host-popover" v-for="host in hosts" :key="host"><a :href="accessUrl(rule,host)" target="_blank" rel="noopener noreferrer">{{host}}</a><n-button size="tiny" quaternary @click="emit('copy-link',accessUrl(rule,host))">复制</n-button></div>
          </n-popover>
        </div>
      </div>
    </div>
    <div class="service-target"><span :title="rule.upstream">{{ rule.upstream }}</span><small>{{ rule.listen_port }} · {{ rule.https_enabled?'HTTPS':'HTTP' }} · {{ rule.listen_ipv4?'IPv4':'' }}{{ rule.listen_ipv4&&rule.listen_ipv6?'/':'' }}{{ rule.listen_ipv6?'IPv6':'' }}</small></div>
    <button type="button" class="service-visitors" title="近期访问 IP 数（非实时 TCP 连接数）" @click="emit('detail')"><n-icon :component="PeopleOutline"/><span>{{ traffic?.connections??'—' }}</span></button>
    <div class="service-rate"><span>↑ {{ traffic?formatRate(traffic.upload_rate):'—' }}</span><span>↓ {{ traffic?formatRate(traffic.download_rate):'—' }}</span></div>
    <div class="service-total"><span>↑ {{ traffic?formatBytes(traffic.upload_total):'—' }}</span><span>↓ {{ traffic?formatBytes(traffic.download_total):'—' }}</span></div>
    <div class="service-security" :title="tags.map(tag=>securityLabel(tag)).join(' · ') || '未设置安全策略'">
      <span v-for="tag in visibleTags" :key="tag" class="security-chip" :class="'security-chip--'+tagTone(tag)">{{securityLabel(tag)}}</span>
      <span v-if="tags.length>visibleTags.length" class="security-chip security-chip--more">+{{tags.length-visibleTags.length}}</span>
      <span v-if="!tags.length" class="security-none">—</span>
    </div>
    <div class="service-status"><n-switch size="small" :value="rule.enabled" :loading="busy" :disabled="toggleLocked" :aria-label="rule.enabled?'停用规则':'启用规则'" @update:value="emit('toggle',$event)"/></div>
    <div class="service-actions">
      <n-button size="small" quaternary title="详情" aria-label="详情" @click="emit('detail')"><template #icon><n-icon :component="InformationCircleOutline"/></template></n-button>
      <n-button size="small" quaternary title="复制规则" aria-label="复制规则" @click="emit('duplicate')"><template #icon><n-icon :component="CopyOutline"/></template></n-button>
      <n-button size="small" quaternary title="编辑规则" aria-label="编辑规则" @click="emit('edit')"><template #icon><n-icon :component="CreateOutline"/></template></n-button>
      <n-dropdown trigger="click" :options="menu" @select="selectAction"><n-button size="small" quaternary title="更多操作" aria-label="更多操作"><template #icon><n-icon :component="EllipsisHorizontalOutline"/></template></n-button></n-dropdown>
    </div>
  </div>
</template>
<script setup lang="ts">
import {computed} from 'vue'
import {NButton,NDropdown,NIcon,NPopover,NSwitch,type DropdownOption} from 'naive-ui'
import {GlobeOutline,PeopleOutline,CopyOutline,CreateOutline,InformationCircleOutline,EllipsisHorizontalOutline,OpenOutline} from '@vicons/ionicons5'
import type {ProxyRule,ProxyTraffic,ProxyBusinessGroup} from '../../api/types'
import {formatBytes,formatRate} from '../../utils/format'
import ServiceFavicon from '../ServiceFavicon.vue'
const props=defineProps<{rule:ProxyRule;name:string;hosts:string[];tags:string[];traffic?:ProxyTraffic;child?:boolean;card?:boolean;busy:boolean;toggleLocked:boolean;groups:ProxyBusinessGroup[];groupId:number;accessUrl:(r:ProxyRule,h:string)=>string}>()
const emit=defineEmits<{detail:[];edit:[];duplicate:[];delete:[];toggle:[value:boolean];assign:[id:number];'copy-link':[url:string]}>()
const visibleTags=computed(()=>props.card?props.tags:props.tags.slice(0,2))
function securityLabel(tag:string){return ({Auth:"认证",CN:"大陆 IP",IP:"IP 控制",Limit:"限流"} as Record<string,string>)[tag]||tag}
function tagTone(tag:string){return tag==="Auth"?"auth":tag==="CN"||tag==="IP"?"access":tag==="Limit"?"traffic":"protect"}
const menu=computed<DropdownOption[]>(()=>[{label:'移至未分组',key:'group:0',disabled:props.groupId===0},...props.groups.map(g=>({label:'移至 '+g.name,key:'group:'+g.id,disabled:g.id===props.groupId})),{type:'divider',key:'divider'},{label:'删除规则',key:'delete'}])
function selectAction(key:string|number){if(key==='delete')emit('delete');else if(String(key).startsWith('group:'))emit('assign',Number(String(key).slice(6)))}
</script>
<style scoped>
.service-row{display:grid;grid-template-columns:minmax(150px,1.5fr) minmax(120px,1.35fr) 54px minmax(86px,.75fr) minmax(78px,.7fr) minmax(80px,.65fr) 48px 112px;align-items:center;gap:8px;padding:12px 10px;min-width:0}
 .service-identity{display:flex;align-items:center;gap:9px;min-width:0}.service-identity__body{min-width:0;flex:1}
.service-identity :deep(.service-favicon){width:34px;height:34px;border-radius:8px}.service-identity :deep(.service-favicon img){width:26px;height:26px}
.service-name{display:block;max-width:100%;text-align:left;border:0;background:none;padding:0;color:var(--fonu-text);font:inherit;font-weight:650;cursor:pointer;overflow-wrap:anywhere}
.service-hosts{display:flex;align-items:center;flex-wrap:wrap;gap:2px;min-width:0;font-size:12px}.service-hosts a,.host-popover a{color:var(--fonu-brand-text);text-decoration:none;overflow-wrap:anywhere;min-width:0}.service-hosts a:hover,.host-popover a:hover{text-decoration:underline}
.service-target{display:grid;gap:3px;min-width:0;font-size:12px;color:var(--fonu-text-secondary);overflow-wrap:anywhere}.service-target small{font-size:10px}
.service-visitors{display:flex;align-items:center;gap:5px;justify-content:flex-start;background:none;border:0;color:var(--fonu-text);cursor:pointer;font:inherit;font-size:12px;padding:0}
.service-rate,.service-total{display:grid;gap:3px;font-size:11px;line-height:1.4;font-variant-numeric:tabular-nums;overflow-wrap:anywhere}.service-rate span:first-child{color:var(--fonu-success)}.service-rate span:last-child{color:var(--fonu-text)}.service-total{color:var(--fonu-text-secondary)}
.service-security{display:flex;gap:3px;flex-wrap:wrap;font-size:11px;color:var(--fonu-text-secondary)}.security-chip{font-size:10px;line-height:1.4;white-space:nowrap;padding:3px 6px;border-radius:5px;background:var(--fonu-bg-muted);color:var(--fonu-text-secondary)}.security-chip--auth,.security-chip--access{background:var(--fonu-brand-soft);color:var(--fonu-brand-text)}.security-chip--traffic,.security-chip--protect{background:var(--fonu-info-soft);color:var(--fonu-text-secondary)}.security-chip--more{font-weight:600}.security-none{color:var(--fonu-text-muted)}.service-hosts .open-link{display:inline-flex;align-items:center;justify-content:center;width:22px;height:22px;flex:0 0 22px;color:var(--fonu-text-secondary);border-radius:4px}.service-hosts .open-link:hover{color:var(--fonu-brand-text);background:var(--fonu-brand-soft)}
.service-actions{display:flex;align-items:center;justify-content:flex-end;gap:1px;flex-wrap:wrap}.service-actions :deep(.n-button){width:26px;padding:0}
.domain-icon{color:var(--fonu-text-secondary);font-size:16px;flex-shrink:0}.host-popover{display:flex;align-items:center;gap:6px;overflow-wrap:anywhere;max-width:320px}
.service-row--child{background:var(--fonu-surface)}.service-row--child .service-identity{padding-left:12px}
.service-row--card{grid-template-columns:minmax(0,1fr) minmax(0,auto);grid-template-rows:auto auto 1fr auto auto;gap:8px;padding:12px;min-height:216px}.service-row--card .service-identity{grid-column:1;align-items:flex-start;min-height:48px;align-self:start}.service-row--card .service-identity :deep(.service-favicon){flex:0 0 34px;width:34px;height:34px;margin-top:1px}.service-row--card .service-name{white-space:nowrap;overflow:hidden;text-overflow:ellipsis;line-height:20px}.service-row--card .service-hosts{flex-wrap:nowrap;align-items:center;min-height:23px}.service-hosts .primary-host{display:block;flex:1 1 auto;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;overflow-wrap:normal}.service-hosts :deep(.n-button){flex-shrink:0}.service-row--card .service-hosts :deep(.n-button){width:22px;padding:0}.service-row--card .service-hosts .open-link{width:20px;flex-basis:20px}.service-row--card .service-status{align-self:start;margin-top:8px}.service-row--card .service-status{grid-column:2;grid-row:1}.service-row--card .service-target{grid-column:1/-1}.service-row--card .service-visitors{grid-column:1}.service-row--card .service-security{grid-column:2;justify-content:flex-end}.service-row--card .service-rate{grid-column:1}.service-row--card .service-total{grid-column:2;text-align:right}.service-row--card .service-actions{grid-column:1/-1;border-top:1px solid var(--fonu-border);padding-top:6px}
@container proxy-workspace (max-width:960px){
 .service-row:not(.service-row--card){grid-template-columns:minmax(0,1fr) minmax(0,1fr) 48px;gap:7px;padding:10px}
 .service-row:not(.service-row--card) .service-identity{grid-column:1/3}.service-row:not(.service-row--card) .service-status{grid-column:3;grid-row:1}
 .service-row:not(.service-row--card) .service-target{grid-column:1/-1}.service-row:not(.service-row--card) .service-visitors{grid-column:1}.service-row:not(.service-row--card) .service-security{grid-column:2/-1;justify-content:flex-end}.service-row:not(.service-row--card) .service-rate{grid-column:1}.service-row:not(.service-row--card) .service-total{grid-column:2/-1;text-align:right}.service-row:not(.service-row--card) .service-actions{grid-column:1/-1}
}
</style>
