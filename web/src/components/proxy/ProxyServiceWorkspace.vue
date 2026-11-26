<template>
  <section class="workspace">
    <div class="workspace-heading"><div><h2>代理服务</h2><small>{{ blocks.length }} 个服务 · {{ rules.length }} 条独立规则</small></div><div class="view-switch" role="group" aria-label="展示方式"><button :aria-pressed="mode==='list'" @click="mode='list'"><n-icon :component="ListOutline"/> 列表</button><button :aria-pressed="mode==='card'" @click="mode='card'"><n-icon :component="GridOutline"/> 卡片</button></div></div>
    <div class="workspace-tools">
      <div class="group-tabs" role="group" aria-label="业务分组">
        <button :aria-pressed="groupId===-1" @click="selectGroup(-1)">全部 <span>{{rules.length}}</span></button>
        <button v-for="g in business.groups" :key="g.id" :aria-pressed="groupId===g.id" @click="selectGroup(g.id)">{{g.name}} <span>{{counts.get(g.id)??0}}</span></button>
        <button :aria-pressed="groupId===0" @click="selectGroup(0)">未分组 <span>{{counts.get(0)??0}}</span></button>
        <n-button size="small" quaternary title="管理业务分组" aria-label="管理业务分组" @click="showGroups=true"><template #icon><n-icon :component="AddOutline"/></template></n-button>
      </div>
      <div class="filters">
        <n-input v-model:value="search" clearable placeholder="搜索服务名称、域名或目标地址…" size="small"><template #prefix><n-icon :component="SearchOutline"/></template></n-input>
        <n-select v-model:value="status" :options="statusOptions" size="small" class="status-filter"/>
        <n-popover trigger="click" placement="bottom-end"><template #trigger><n-button size="small"><template #icon><n-icon :component="FilterOutline"/></template>筛选</n-button></template><div class="extra-filters"><small>监听入口</small><n-select v-model:value="entry" :options="entryOptions" size="small"/><small>协议</small><n-select v-model:value="protocol" :options="protocolOptions" size="small"/><n-button size="small" @click="resetFilters">清除筛选</n-button></div></n-popover>
        <n-button size="small" :loading="loading" title="刷新规则" aria-label="刷新规则" @click="emit('refresh')"><template #icon><n-icon :component="RefreshOutline"/></template></n-button>
      </div>
    </div>
    <div v-if="loading && !rules.length" class="workspace-empty"><n-spin/></div>
    <div v-else-if="!filtered.length" class="workspace-empty"><n-empty :description="rules.length?'没有匹配的代理服务':'暂无代理规则'"><template #extra><n-button size="small" @click="rules.length?resetFilters():emit('create')">{{rules.length?'清除筛选':'新增代理'}}</n-button></template></n-empty></div>
    <div v-else class="service-blocks" :class="{'service-blocks--cards':mode==='card'}">
      <article v-for="block in paged" :key="block.key" class="service-block" :class="{'is-open':isOpen(block.key),'is-drag-target':dragTarget===block.key}" @dragover.prevent="dragTarget=block.key" @dragleave="dragTarget=null" @drop.prevent="dropBlock(block.rules.map(r=>r.id))">
        <div v-if="block.rules.length>1" class="service-summary" :class="{'service-summary--card':mode==='card'}">
          <button type="button" class="summary-identity" :aria-expanded="isOpen(block.key)" @click="toggle(block.key)">
            <n-icon :component="isOpen(block.key)?ChevronDownOutline:ChevronForwardOutline"/>
            <ServiceFavicon :rule-id="block.rules[0].id"/>
            <span><strong>{{block.name}}</strong><small>{{block.rules.length}} 条独立规则 · {{domainCount(block.rules)}} 个域名</small></span>
          </button>
          <div class="summary-metrics">
            <span title="各规则近期访问 IP 数相加，非全局去重"><n-icon :component="PeopleOutline"/> {{sum(block.rules).connections}}</span>
            <span>↑ {{formatRate(sum(block.rules).upload_rate)}}<br/>↓ {{formatRate(sum(block.rules).download_rate)}}</span>
            <span title="累计上传与下载">↑ {{formatBytes(sum(block.rules).upload_total)}}<br/>↓ {{formatBytes(sum(block.rules).download_total)}}</span>
            <span class="summary-state">{{block.rules.filter(r=>r.enabled).length}} / {{block.rules.length}} 启用</span>
          </div>
          <div class="summary-actions">
            <n-dropdown trigger="click" :options="groupMenu(block.groupId)" @select="key=>assignBlock(block.rules.map(r=>r.id),key)"><n-button size="small" quaternary title="移动服务显示分组" aria-label="移动服务显示分组"><template #icon><n-icon :component="EllipsisHorizontalOutline"/></template></n-button></n-dropdown>
            <span v-if="canDrag" draggable="true" class="drag-handle" title="拖动服务排序" @dragstart.stop="startDrag($event,block.rules.map(r=>r.id))" @dragend="endDrag"><n-icon :component="ReorderThreeOutline"/></span>
          </div>
        </div>
        <template v-if="block.rules.length===1 || isOpen(block.key)">
          <div v-if="block.rules.length>1" class="child-header"><span>域名（独立规则）</span><span>目标地址</span><span>访问 IP</span><span>实时流量</span><span>累计流量</span><span>安全</span><span>状态</span><span>操作</span></div>
          <div v-for="rule in block.rules" :key="rule.id" class="rule-line" @dragover.stop.prevent @drop.stop.prevent="dropBlock([rule.id])">
            <span v-if="canDrag" draggable="true" class="drag-handle rule-drag" title="拖动独立规则排序" @dragstart.stop="startDrag($event,[rule.id])" @dragend="endDrag"><n-icon :component="ReorderThreeOutline"/></span>
            <ProxyServiceRow :rule="rule" :name="ruleName(rule)" :hosts="ruleHosts(rule)" :tags="securityTags(rule)" :traffic="traffic[rule.id]" :groups="business.groups" :group-id="block.groupId" :access-url="accessUrl" :child="block.rules.length>1" :card="mode==='card'" :busy="togglingId===rule.id" :toggle-locked="togglingId!==null" @detail="emit('detail',rule)" @edit="emit('edit',rule)" @duplicate="emit('duplicate',rule)" @delete="emit('delete',rule)" @toggle="emit('toggle',rule,$event)" @assign="changeBusiness({action:'assign',id:$event,rule_ids:[rule.id]})" @copy-link="emit('copy-link',$event)"/>
          </div>
        </template>
      </article>
    </div>
    <div class="workspace-footer"><small>共 {{filtered.length}} 个服务 · {{filteredRuleCount}} 条规则</small><n-select v-model:value="pageSize" :options="sizeOptions" size="small" class="page-size"/><n-pagination v-model:page="page" :page-count="pageCount" :page-slot="5" size="small"/></div>
    <n-modal v-model:show="showGroups"><div class="group-modal"><div class="group-modal-head"><h3>业务分组管理</h3><n-button quaternary @click="showGroups=false">关闭</n-button></div><small>分组仅用于界面管理，删除分组不会删除任何代理规则。</small><div class="group-create"><n-input v-model:value="newName" maxlength="60" placeholder="新分组名称" @keydown.enter.prevent="createGroup"/><n-button type="primary" :loading="groupBusy" :disabled="!newName.trim()" @click="createGroup">新增</n-button></div><div v-for="(g,i) in business.groups" :key="g.id" class="group-manage-row"><span>{{g.name}} <small>({{counts.get(g.id)??0}})</small></span><n-button size="tiny" :disabled="groupBusy||i===0" @click="moveGroup(i,-1)">↑</n-button><n-button size="tiny" :disabled="groupBusy||i===business.groups.length-1" @click="moveGroup(i,1)">↓</n-button><n-button size="tiny" quaternary :disabled="groupBusy" @click="rename(g.id,g.name)">重命名</n-button><n-button size="tiny" quaternary type="error" :disabled="groupBusy" @click="remove(g.id,g.name)">删除</n-button></div></div></n-modal>
  </section>
</template>
<script setup lang="ts">
import {computed,ref,watch,h} from 'vue'
import {NButton,NDropdown,NEmpty,NIcon,NInput,NModal,NPagination,NPopover,NSelect,NSpin,useDialog,type DropdownOption} from 'naive-ui'
import {AddOutline,ChevronDownOutline,ChevronForwardOutline,EllipsisHorizontalOutline,FilterOutline,GridOutline,ListOutline,PeopleOutline,RefreshOutline,ReorderThreeOutline,SearchOutline} from '@vicons/ionicons5'
import type {ProxyRule,ProxyTraffic,ProxyEntry,ProxyBusinessState,ProxyBusinessChange} from '../../api/types'
import {buildServiceBlocks,sumServiceTraffic,moveServiceRules} from '../../utils/proxyServices'
import {formatBytes,formatRate} from '../../utils/format'
import ServiceFavicon from '../ServiceFavicon.vue'
import ProxyServiceRow from './ProxyServiceRow.vue'
const props=defineProps<{rules:ProxyRule[];entries:ProxyEntry[];traffic:Record<number,ProxyTraffic>;business:ProxyBusinessState;loading:boolean;groupBusy:boolean;reordering:boolean;togglingId:number|null;ruleName:(r:ProxyRule)=>string;ruleHosts:(r:ProxyRule)=>string[];securityTags:(r:ProxyRule)=>string[];accessUrl:(r:ProxyRule,h:string)=>string;changeBusiness:(change:ProxyBusinessChange)=>Promise<boolean>}>()
const emit=defineEmits<{refresh:[];create:[];detail:[r:ProxyRule];edit:[r:ProxyRule];duplicate:[r:ProxyRule];delete:[r:ProxyRule];toggle:[r:ProxyRule,v:boolean]; 'select-group':[id:number];'copy-link':[url:string];reorder:[rules:ProxyRule[]]}>()
const dialog=useDialog()
function readMode(){try{return localStorage.getItem('fonu.proxy.services.view')==='card'?'card':'list'}catch{return 'list'}}
const mode=ref<'list'|'card'>(readMode());watch(mode,v=>{try{localStorage.setItem('fonu.proxy.services.view',v)}catch{/* unavailable */}})
const groupId=ref(-1),search=ref(''),status=ref('all'),protocol=ref('all'),entry=ref(-1),page=ref(1),pageSize=ref(10)
const opened=ref(new Set<string>()),showGroups=ref(false),newName=ref('')
const statusOptions=[{label:'全部状态',value:'all'},{label:'已启用',value:'enabled'},{label:'已停用',value:'disabled'}]
const protocolOptions=[{label:'全部协议',value:'all'},{label:'HTTPS',value:'https'},{label:'HTTP',value:'http'}]
const entryOptions=computed(()=>[{label:'全部入口',value:-1},{label:'未绑定入口',value:0},...props.entries.map(e=>({label:e.name||e.listen_port+' · '+(e.https_enabled?'HTTPS':'HTTP'),value:e.id}))])
const sizeOptions=[10,20,50].map(value=>({label:'每页 '+value+' 个',value}))
const memberships=computed(()=>new Map(props.business.members.map(m=>[m.rule_id,m.group_id])))
const counts=computed(()=>{const result=new Map<number,number>();for(const r of props.rules){const id=memberships.value.get(r.id)??0;result.set(id,(result.get(id)??0)+1)}return result})
const blocks=computed(()=>buildServiceBlocks(props.rules,props.business))
const filtered=computed(()=>{const q=search.value.trim().toLowerCase();return blocks.value.filter(b=>groupId.value===-1||b.groupId===groupId.value).map(b=>({...b,rules:b.rules.filter(r=>(!q||[props.ruleName(r),r.upstream,props.ruleHosts(r).join(' ')].join(' ').toLowerCase().includes(q))&&(status.value==='all'||r.enabled===(status.value==='enabled'))&&(protocol.value==='all'||r.https_enabled===(protocol.value==='https'))&&(entry.value===-1||(r.entry_id??0)===entry.value))})).filter(b=>b.rules.length)})
const filteredRuleCount=computed(()=>filtered.value.reduce((n,b)=>n+b.rules.length,0))
const pageCount=computed(()=>Math.max(1,Math.ceil(filtered.value.length/pageSize.value)))
const paged=computed(()=>filtered.value.slice((page.value-1)*pageSize.value,page.value*pageSize.value))
watch([groupId,search,status,protocol,entry,pageSize],()=>page.value=1)
watch(pageCount,n=>page.value=Math.min(page.value,n))
watch(()=>props.business.groups,groups=>{if(groupId.value>0&&!groups.some(g=>g.id===groupId.value))selectGroup(-1)})
function selectGroup(id:number){groupId.value=id;emit('select-group',id)}
function resetFilters(){search.value='';status.value='all';protocol.value='all';entry.value=-1;selectGroup(-1)}
function isOpen(key:string){return opened.value.has(key)}
function toggle(key:string){const next=new Set(opened.value);if(next.has(key))next.delete(key);else next.add(key);opened.value=next}
function domainCount(rules:ProxyRule[]){return rules.reduce((n,r)=>n+props.ruleHosts(r).length,0)}
function sum(rules:ProxyRule[]){return sumServiceTraffic(rules,props.traffic)}
function groupMenu(id:number):DropdownOption[]{return [{label:'移至未分组',key:'0',disabled:id===0},...props.business.groups.map(g=>({label:'移至 '+g.name,key:String(g.id),disabled:g.id===id}))]}
function assignBlock(ids:number[],key:string|number){void props.changeBusiness({action:'assign',id:Number(key),rule_ids:ids})}
async function createGroup(){if(!newName.value.trim()||props.groupBusy)return;if(await props.changeBusiness({action:'create',name:newName.value.trim()}))newName.value=''}
function rename(id:number,name:string){const draft=ref(name);dialog.create({title:'重命名分组',content:()=>h(NInput,{value:draft.value,maxlength:60,'onUpdate:value':(v:string)=>draft.value=v}),positiveText:'保存',negativeText:'取消',onPositiveClick:async()=>!!draft.value.trim()&&await props.changeBusiness({action:'rename',id,name:draft.value.trim()})})}
function remove(id:number,name:string){dialog.warning({title:'删除分组「'+name+'」？',content:'规则将移至未分组，不会删除规则或重载 Nginx。',positiveText:'删除分组',negativeText:'取消',onPositiveClick:()=>props.changeBusiness({action:'delete',id})})}
function moveGroup(i:number,d:number){const ids=props.business.groups.map(g=>g.id);[ids[i],ids[i+d]]=[ids[i+d],ids[i]];void props.changeBusiness({action:'reorder',ids})}
const canDrag=computed(()=>!props.reordering&&!search.value.trim()&&status.value==='all'&&protocol.value==='all'&&entry.value===-1)
const dragging=ref<number[]>([]),dragTarget=ref<string|null>(null)
function startDrag(event:DragEvent,ids:number[]){if(!canDrag.value)return;dragging.value=ids;event.dataTransfer?.setData('text/plain',ids.join(','));if(event.dataTransfer)event.dataTransfer.effectAllowed='move'}
function endDrag(){dragging.value=[];dragTarget.value=null}
function dropBlock(ids:number[]){if(canDrag.value&&dragging.value.length){const next=moveServiceRules(props.rules,dragging.value,ids);if(next!==props.rules)emit('reorder',next)}endDrag()}
</script>
<style scoped>
.workspace{container:proxy-workspace / inline-size;min-width:0;width:100%;display:grid;gap:12px}
.workspace-heading,.workspace-tools,.workspace-footer{display:flex;align-items:center;gap:10px;flex-wrap:wrap;justify-content:space-between}
.workspace-heading h2{font-size:19px;margin:0;color:var(--fonu-text)}.workspace-heading small,.workspace-footer small{font-size:12px;color:var(--fonu-text-secondary)}
.view-switch,.group-tabs{display:flex;align-items:center;flex-wrap:wrap;gap:5px}.view-switch{background:var(--fonu-bg-muted);border:1px solid var(--fonu-border);padding:3px;border-radius:8px}
.view-switch button,.group-tabs button{border:0;border-radius:5px;cursor:pointer;font:inherit;font-size:12px;padding:6px 10px;background:transparent;color:var(--fonu-text-secondary);display:flex;align-items:center;gap:5px;max-width:100%;overflow-wrap:anywhere}
.view-switch button[aria-pressed=true],.group-tabs button[aria-pressed=true]{background:var(--fonu-brand-soft);color:var(--fonu-brand-text);font-weight:600}.group-tabs button{background:var(--fonu-bg-muted)}.group-tabs button span{font-variant-numeric:tabular-nums;font-size:11px}
.filters{display:flex;align-items:center;gap:7px;flex:1 1 330px;min-width:0;justify-content:flex-end}.filters :deep(.n-input){flex:1;min-width:100px}.status-filter{width:106px;flex-shrink:0}.extra-filters{display:grid;gap:7px;min-width:190px}
.service-blocks{display:grid;gap:9px;min-width:0}.service-blocks--cards{grid-template-columns:repeat(auto-fit,minmax(min(100%,300px),1fr));align-items:start}
.service-block{background:var(--fonu-surface);border:1px solid var(--fonu-border);border-radius:var(--fonu-radius);min-width:0;overflow:hidden}.service-block.is-open{border-color:var(--fonu-brand)}.service-block.is-drag-target{outline:2px dashed var(--fonu-brand)}
.service-summary{display:flex;align-items:center;gap:12px;justify-content:space-between;padding:12px;background:var(--fonu-bg-muted);flex-wrap:wrap}.summary-identity{display:flex;align-items:center;gap:8px;min-width:0;border:0;padding:0;background:none;text-align:left;color:var(--fonu-text);cursor:pointer;flex:1 1 190px;font:inherit}.summary-identity strong,.summary-identity small{display:block;overflow-wrap:anywhere}.summary-identity small{font-size:11px;color:var(--fonu-text-secondary);margin-top:3px}.summary-identity :deep(.service-favicon){width:38px;height:38px}.summary-metrics{display:flex;gap:16px;align-items:center;flex-wrap:wrap;font-size:12px;color:var(--fonu-text-secondary);font-variant-numeric:tabular-nums}.summary-metrics span{display:inline-block}.summary-state{color:var(--fonu-text);font-weight:600}.summary-actions{display:flex;gap:3px;align-items:center}.service-summary--card .summary-metrics{width:100%;gap:10px}
.child-header{display:grid;grid-template-columns:minmax(150px,1.5fr) minmax(120px,1.35fr) 54px minmax(86px,.75fr) minmax(78px,.7fr) minmax(45px,.5fr) 48px 112px;gap:8px;padding:8px 10px 8px 30px;font-size:11px;color:var(--fonu-text-secondary);background:var(--fonu-bg-muted);border-top:1px solid var(--fonu-border)}
.rule-line{display:flex;align-items:stretch;min-width:0;border-top:1px solid var(--fonu-border)}.rule-line:first-child{border-top:0}.rule-line :deep(.service-row){flex:1;min-width:0}.drag-handle{display:flex;align-items:center;justify-content:center;color:var(--fonu-text-muted);cursor:grab;flex:0 0 20px}.drag-handle:active{cursor:grabbing}.rule-drag{padding-left:4px}
.workspace-footer{justify-content:flex-end;padding:4px 0}.workspace-footer small{margin-right:auto}.page-size{width:110px}.workspace-empty{display:grid;place-items:center;min-height:150px;border:1px solid var(--fonu-border);border-radius:var(--fonu-radius)}
.group-modal{width:min(540px,94vw);max-height:84vh;overflow:auto;background:var(--fonu-surface);border-radius:var(--fonu-radius);padding:16px;display:grid;gap:12px}.group-modal-head,.group-create,.group-manage-row{display:flex;align-items:center;gap:7px}.group-modal-head h3{flex:1;margin:0}.group-modal small{color:var(--fonu-text-secondary)}.group-manage-row{border-top:1px solid var(--fonu-border);padding:8px 0;flex-wrap:wrap}.group-manage-row>span{flex:1;overflow-wrap:anywhere}
.service-blocks--cards .child-header{display:none}
@container proxy-workspace (max-width:960px){.child-header{display:none}.workspace-tools{align-items:stretch}.group-tabs,.filters{flex:1 1 100%}.summary-metrics{gap:9px}.service-summary{padding:9px}}
@container proxy-workspace (max-width:480px){.filters{flex-wrap:wrap}.filters :deep(.n-input){flex:1 1 100%}.status-filter{flex:1}.workspace-footer{justify-content:flex-start}.summary-metrics{width:100%;font-size:11px}.group-tabs button{padding:5px 8px}.workspace-heading h2{font-size:17px}}
</style>
