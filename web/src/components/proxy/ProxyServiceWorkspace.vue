<template>
  <section ref="workspaceRef" class="workspace" :class="{'is-sorting':dragging.length>0}">
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
        <n-popover trigger="click" placement="bottom-end"><template #trigger><n-button size="small"><template #icon><n-icon :component="FilterOutline"/></template>筛选</n-button></template><div class="extra-filters"><small>监听端口</small><n-select v-model:value="entry" :options="entryOptions" size="small"/><small>协议</small><n-select v-model:value="protocol" :options="protocolOptions" size="small"/><n-button size="small" @click="resetFilters">清除筛选</n-button></div></n-popover>
        <n-button size="small" :loading="loading" title="刷新规则" aria-label="刷新规则" @click="emit('refresh')"><template #icon><n-icon :component="RefreshOutline"/></template></n-button>
      </div>
    </div>
    <div v-if="loading && !rules.length" class="workspace-empty"><n-spin/></div>
    <div v-else-if="!filtered.length" class="workspace-empty"><n-empty :description="rules.length?'没有匹配的代理服务':'暂无代理规则'"><template #extra><n-button size="small" @click="rules.length?resetFilters():emit('create')">{{rules.length?'清除筛选':'新增代理'}}</n-button></template></n-empty></div>
    <div v-else class="service-blocks" :class="{'service-blocks--cards':mode==='card'}" :style="mode==='card'?{gridTemplateColumns:`repeat(${cardColumns},minmax(0,1fr))`}:undefined">
      <template v-for="(block,index) in filtered" :key="block.key">
      <article class="service-block" :class="{'is-open':block.rules.length>1&&isOpen(block.key),'is-drag-source':block.rules.some(r=>dragging.includes(r.id)),'is-drag-target':dragTarget===block.key,'drop-after':dragAfter}" @dragover="overTarget($event,block.key,block.rules.map(r=>r.id))" @dragleave="leaveTarget($event,block.key)" @drop.prevent="dropBlock(block.rules.map(r=>r.id))">
        <div v-if="block.rules.length>1" class="service-summary" :class="{'service-summary--card':mode==='card'}">
          <button type="button" class="summary-identity" :aria-expanded="isOpen(block.key)" @click="toggle(block.key)">
            <n-icon :component="isOpen(block.key)?ChevronDownOutline:ChevronForwardOutline"/>
            <ServiceFavicon :rule-id="block.rules[0].id"/>
            <span><strong>{{block.name}}</strong><small>{{block.rules.length}} 条独立规则 · {{domainCount(block.rules)}} 个域名</small></span>
          </button>
          <div v-if="mode==='card'" class="summary-listen"><small>{{[...new Set(block.rules.map(r=>r.listen_port+' · '+(r.https_enabled?'HTTPS':'HTTP')))].join(' / ')}}</small></div>
          <div class="summary-metrics">
            <span title="各规则近期访问 IP 数相加，非全局去重"><n-icon :component="PeopleOutline"/> {{block.rules.every(r=>traffic[r.id])?sum(block.rules).connections:'—'}}</span>
            <span>↑ {{block.rules.every(r=>traffic[r.id])?formatRate(sum(block.rules).upload_rate):'—'}}<br/>↓ {{block.rules.every(r=>traffic[r.id])?formatRate(sum(block.rules).download_rate):'—'}}</span>
            <span title="累计上传与下载">↑ {{block.rules.every(r=>traffic[r.id])?formatBytes(sum(block.rules).upload_total):'—'}}<br/>↓ {{block.rules.every(r=>traffic[r.id])?formatBytes(sum(block.rules).download_total):'—'}}</span>
            <span class="summary-state">{{block.rules.filter(r=>r.enabled).length}} / {{block.rules.length}} 启用</span>
          </div>
          <div class="summary-actions"><n-button v-if="mode==='card'" size="small" quaternary @click="toggle(block.key)">{{isOpen(block.key)?'收起规则':'展开规则'}}</n-button>
            <n-dropdown trigger="click" :options="groupMenu(block.groupId)" @select="key=>assignBlock(block.rules.map(r=>r.id),key)"><n-button size="small" quaternary title="移动服务显示分组" aria-label="移动服务显示分组"><template #icon><n-icon :component="EllipsisHorizontalOutline"/></template></n-button></n-dropdown>
            <span v-if="canDrag" draggable="true" class="drag-handle" title="拖动服务排序" @dragstart.stop="startDrag($event,block.rules.map(r=>r.id))" @dragend="endDrag"><n-icon :component="ReorderThreeOutline"/></span>
          </div>
        </div>
        <template v-if="block.rules.length===1 || (mode==='list'&&isOpen(block.key))">
          <div v-if="block.rules.length>1" class="child-header"><span>域名（独立规则）</span><span>目标地址</span><span>访问 IP</span><span>实时流量</span><span>累计流量</span><span>安全</span><span>状态</span><span>操作</span></div>
          <div v-for="rule in block.rules" :key="rule.id" class="rule-line" :class="{'is-drag-source':dragging.includes(rule.id),'is-drag-target':dragTarget==='rule:'+rule.id,'drop-after':dragAfter}" @dragover.stop="overTarget($event,'rule:'+rule.id,[rule.id])" @dragleave="leaveTarget($event,'rule:'+rule.id)" @drop.stop.prevent="dropBlock([rule.id])">
            <span v-if="canDrag" draggable="true" class="drag-handle rule-drag" title="拖动独立规则排序" @dragstart.stop="startDrag($event,[rule.id])" @dragend="endDrag"><n-icon :component="ReorderThreeOutline"/></span>
            <ProxyServiceRow :rule="rule" :name="ruleName(rule)" :hosts="ruleHosts(rule)" :tags="securityTags(rule)" :traffic="traffic[rule.id]" :groups="business.groups" :group-id="block.groupId" :access-url="accessUrl" :child="block.rules.length>1" :card="mode==='card'" :busy="togglingId===rule.id" :toggle-locked="togglingId!==null" @detail="emit('detail',rule)" @edit="emit('edit',rule)" @duplicate="emit('duplicate',rule)" @delete="emit('delete',rule)" @toggle="emit('toggle',rule,$event)" @assign="changeBusiness({action:'assign',id:$event,rule_ids:[rule.id]})" @copy-link="emit('copy-link',$event)"/>
          </div>
        </template>
      </article>
      <article v-for="expanded in expandedAfter(index)" :key="'expanded:'+expanded.key" class="expanded-panel">
        <div class="expanded-head"><div><strong>{{expanded.name}}</strong><small>{{expanded.rules.length}} 条独立规则 · 每条规则单独管理</small></div><n-button size="tiny" quaternary @click="toggle(expanded.key)">收起 <n-icon :component="ChevronDownOutline"/></n-button></div>
        <div class="child-header"><span>域名（独立规则）</span><span>目标地址</span><span>访问 IP</span><span>实时流量</span><span>累计流量</span><span>安全</span><span>状态</span><span>操作</span></div>
          <div v-for="rule in expanded.rules" :key="rule.id" class="rule-line" :class="{'is-drag-source':dragging.includes(rule.id),'is-drag-target':dragTarget==='rule:'+rule.id,'drop-after':dragAfter}" @dragover.stop="overTarget($event,'rule:'+rule.id,[rule.id])" @dragleave="leaveTarget($event,'rule:'+rule.id)" @drop.stop.prevent="dropBlock([rule.id])">
            <span v-if="canDrag" draggable="true" class="drag-handle rule-drag" title="拖动独立规则排序" @dragstart.stop="startDrag($event,[rule.id])" @dragend="endDrag"><n-icon :component="ReorderThreeOutline"/></span>
            <ProxyServiceRow :rule="rule" :name="ruleName(rule)" :hosts="ruleHosts(rule)" :tags="securityTags(rule)" :traffic="traffic[rule.id]" :groups="business.groups" :group-id="expanded.groupId" :access-url="accessUrl" :child="true" :card="false" :busy="togglingId===rule.id" :toggle-locked="togglingId!==null" @detail="emit('detail',rule)" @edit="emit('edit',rule)" @duplicate="emit('duplicate',rule)" @delete="emit('delete',rule)" @toggle="emit('toggle',rule,$event)" @assign="changeBusiness({action:'assign',id:$event,rule_ids:[rule.id]})" @copy-link="emit('copy-link',$event)"/>
          </div>
      </article>
      </template>
    </div>
    <div class="workspace-footer"><small>共 {{filtered.length}} 个服务 · {{filteredRuleCount}} 条规则</small><small v-if="reordering">正在保存排序…</small></div>
    <n-modal v-model:show="showGroups"><div class="group-modal"><div class="group-modal-head"><h3>业务分组管理</h3><n-button quaternary @click="showGroups=false">关闭</n-button></div><small>分组仅用于界面管理，删除分组不会删除任何代理规则。</small><div class="group-create"><n-input v-model:value="newName" maxlength="60" placeholder="新分组名称" @keydown.enter.prevent="createGroup"/><n-button type="primary" :loading="groupBusy" :disabled="!newName.trim()" @click="createGroup">新增</n-button></div><div v-for="(g,i) in business.groups" :key="g.id" class="group-manage-row"><span>{{g.name}} <small>({{counts.get(g.id)??0}})</small></span><n-button size="tiny" :disabled="groupBusy||i===0" @click="moveGroup(i,-1)">↑</n-button><n-button size="tiny" :disabled="groupBusy||i===business.groups.length-1" @click="moveGroup(i,1)">↓</n-button><n-button size="tiny" quaternary :disabled="groupBusy" @click="rename(g.id,g.name)">重命名</n-button><n-button size="tiny" quaternary type="error" :disabled="groupBusy" @click="remove(g.id,g.name)">删除</n-button></div></div></n-modal>
  </section>
</template>
<script setup lang="ts">
import {computed,ref,watch,h,onMounted,onBeforeUnmount} from 'vue'
import {NButton,NDropdown,NEmpty,NIcon,NInput,NModal,NPopover,NSelect,NSpin,useDialog,type DropdownOption} from 'naive-ui'
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
const groupId=ref(-1),search=ref(''),status=ref('all'),protocol=ref('all'),entry=ref(-1)
const opened=ref(new Set<string>()),showGroups=ref(false),newName=ref('')
const statusOptions=[{label:'全部状态',value:'all'},{label:'已启用',value:'enabled'},{label:'已停用',value:'disabled'}]
const protocolOptions=[{label:'全部协议',value:'all'},{label:'HTTPS',value:'https'},{label:'HTTP',value:'http'}]
const entryOptions=computed(()=>[{label:'全部端口',value:-1},...[...new Set(props.rules.map(r=>r.listen_port))].sort((a,b)=>a-b).map(port=>({label:String(port),value:port}))])
const memberships=computed(()=>new Map(props.business.members.map(m=>[m.rule_id,m.group_id])))
const counts=computed(()=>{const result=new Map<number,number>();for(const r of props.rules){const id=memberships.value.get(r.id)??0;result.set(id,(result.get(id)??0)+1)}return result})
const blocks=computed(()=>buildServiceBlocks(props.rules,props.business))
const filtered=computed(()=>{const q=search.value.trim().toLowerCase();return blocks.value.filter(b=>groupId.value===-1||b.groupId===groupId.value).map(b=>({...b,rules:b.rules.filter(r=>(!q||[props.ruleName(r),r.upstream,props.ruleHosts(r).join(' ')].join(' ').toLowerCase().includes(q))&&(status.value==='all'||r.enabled===(status.value==='enabled'))&&(protocol.value==='all'||r.https_enabled===(protocol.value==='https'))&&(entry.value===-1||r.listen_port===entry.value))})).filter(b=>b.rules.length)})
const filteredRuleCount=computed(()=>filtered.value.reduce((n,b)=>n+b.rules.length,0))
const workspaceRef=ref<HTMLElement|null>(null)
const cardColumns=ref(1)
let resizeObserver:ResizeObserver|undefined
onMounted(()=>{
  if(!workspaceRef.value)return
  const measure=()=>{const width=workspaceRef.value?.clientWidth??0;cardColumns.value=Math.max(1,Math.floor((width+9)/309))}
  measure()
  resizeObserver=new ResizeObserver(measure)
  resizeObserver.observe(workspaceRef.value)
})
onBeforeUnmount(()=>resizeObserver?.disconnect())
function expandedAfter(index:number){
  if(mode.value!=='card')return []
  const count=cardColumns.value
  if((index+1)%count!==0&&index!==filtered.value.length-1)return []
  const start=Math.floor(index/count)*count
  return filtered.value.slice(start,index+1).filter(b=>b.rules.length>1&&isOpen(b.key))
}
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
const dragging=ref<number[]>([]),dragTarget=ref<string|null>(null),dragAfter=ref(false)
function startDrag(event:DragEvent,ids:number[]){
  if(!canDrag.value){event.preventDefault();return}
  if(event.dataTransfer){
    event.dataTransfer.setData('text/plain',ids.join(','))
    event.dataTransfer.effectAllowed='move'
    const item=(event.currentTarget as HTMLElement).closest(ids.length>1?'.service-block':'.rule-line')
    if(item instanceof HTMLElement)event.dataTransfer.setDragImage(item,Math.min(50,item.offsetWidth/2),Math.min(20,item.offsetHeight/2))
  }
  dragging.value=ids
}
function overTarget(event:DragEvent,key:string,ids:number[]){
  if(!canDrag.value||!dragging.value.length||ids.some(id=>dragging.value.includes(id)))return
  event.preventDefault()
  if(event.dataTransfer)event.dataTransfer.dropEffect='move'
  dragAfter.value=props.rules.findIndex(r=>dragging.value.includes(r.id))<props.rules.findIndex(r=>ids.includes(r.id))
  dragTarget.value=key
}
function leaveTarget(event:DragEvent,key:string){
  if(dragTarget.value!==key)return
  const target=event.currentTarget as HTMLElement
  if(event.relatedTarget instanceof Node&&target.contains(event.relatedTarget))return
  dragTarget.value=null
}
function endDrag(){dragging.value=[];dragTarget.value=null;dragAfter.value=false}
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
.service-blocks{display:grid;gap:9px;min-width:0}.service-blocks--cards{grid-template-columns:repeat(auto-fit,minmax(min(100%,300px),1fr));align-items:stretch}
.service-block{background:var(--fonu-surface);border:1px solid var(--fonu-border);border-radius:var(--fonu-radius);min-width:0;overflow:hidden}.service-block.is-open{border-color:var(--fonu-brand)}.service-block.is-drag-target{outline:2px dashed var(--fonu-brand)}
.service-summary{display:flex;align-items:center;gap:12px;justify-content:space-between;padding:12px;background:var(--fonu-bg-muted);flex-wrap:wrap}.summary-identity{display:flex;align-items:center;gap:8px;min-width:0;border:0;padding:0;background:none;text-align:left;color:var(--fonu-text);cursor:pointer;flex:1 1 190px;font:inherit}.summary-identity strong,.summary-identity small{display:block;overflow-wrap:anywhere}.summary-identity small{font-size:11px;color:var(--fonu-text-secondary);margin-top:3px}.summary-identity :deep(.service-favicon){width:38px;height:38px}.summary-metrics{display:flex;gap:16px;align-items:center;flex-wrap:wrap;font-size:12px;color:var(--fonu-text-secondary);font-variant-numeric:tabular-nums}.summary-metrics span{display:inline-block}.summary-state{color:var(--fonu-text);font-weight:600}.summary-actions{display:flex;gap:3px;align-items:center}.service-summary--card .summary-metrics{width:100%;gap:10px}
.child-header{display:grid;grid-template-columns:minmax(150px,1.5fr) minmax(120px,1.35fr) 54px minmax(86px,.75fr) minmax(78px,.7fr) minmax(80px,.65fr) 48px 112px;gap:8px;padding:8px 10px 8px 30px;font-size:11px;color:var(--fonu-text-secondary);background:var(--fonu-bg-muted);border-top:1px solid var(--fonu-border)}
.rule-line{display:flex;align-items:stretch;min-width:0;border-top:1px solid var(--fonu-border)}.rule-line:first-child{border-top:0}.rule-line :deep(.service-row){flex:1;min-width:0}.drag-handle{display:flex;align-items:center;justify-content:center;color:var(--fonu-text-muted);cursor:grab;flex:0 0 20px}.drag-handle:active{cursor:grabbing}.rule-drag{padding-left:4px}
.workspace-footer{justify-content:flex-end;padding:4px 0}.workspace-footer small{margin-right:auto}.page-size{width:110px}.workspace-empty{display:grid;place-items:center;min-height:150px;border:1px solid var(--fonu-border);border-radius:var(--fonu-radius)}
.group-modal{width:min(540px,94vw);max-height:84vh;overflow:auto;background:var(--fonu-surface);border-radius:var(--fonu-radius);padding:16px;display:grid;gap:12px}.group-modal-head,.group-create,.group-manage-row{display:flex;align-items:center;gap:7px}.group-modal-head h3{flex:1;margin:0}.group-modal small{color:var(--fonu-text-secondary)}.group-manage-row{border-top:1px solid var(--fonu-border);padding:8px 0;flex-wrap:wrap}.group-manage-row>span{flex:1;overflow-wrap:anywhere}
.service-blocks--cards .child-header{display:none}
.service-blocks--cards .service-block{display:flex;flex-direction:column;min-height:240px}
.service-blocks--cards .service-block.is-open{border-color:var(--fonu-brand)}
.service-blocks--cards .rule-line{flex:1}
.service-blocks--cards .service-summary--card{flex:1;display:grid;grid-template-columns:minmax(0,1fr);grid-template-rows:auto auto 1fr auto;gap:8px;background:var(--fonu-surface)}
.summary-listen{display:grid;gap:3px;font-size:12px;color:var(--fonu-text-secondary);overflow-wrap:anywhere}
.summary-listen small{font-size:10px}
.service-summary--card .summary-metrics{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:8px}
.service-summary--card .summary-metrics span:nth-child(3){grid-column:2;grid-row:2;text-align:right}
.service-summary--card .summary-metrics span:nth-child(4){grid-column:2;grid-row:1;text-align:right}
.service-summary--card .summary-actions{border-top:1px solid var(--fonu-border);padding-top:6px;justify-content:flex-end}
.expanded-panel{grid-column:1/-1;border:1px solid var(--fonu-border);border-radius:var(--fonu-radius);background:var(--fonu-surface);overflow:hidden;min-width:0}.expanded-head{display:flex;align-items:center;justify-content:space-between;gap:8px;padding:10px 14px;background:var(--fonu-brand-soft)}.expanded-head strong,.expanded-head small{display:block}.expanded-head small{font-size:11px;color:var(--fonu-text-secondary);margin-top:2px}.service-blocks--cards .expanded-panel .child-header{display:grid}.service-blocks--cards .expanded-panel .rule-line{flex:none}.expanded-panel :deep(.service-row){padding-top:10px;padding-bottom:10px}.service-blocks--cards .service-block.is-open .service-summary{flex:1}
.service-block,.rule-line{position:relative;transition:background .15s,box-shadow .15s,opacity .15s}.drag-handle{border-radius:5px;transition:background .15s,color .15s;min-height:24px}.drag-handle:hover{color:var(--fonu-brand-text);background:var(--fonu-brand-soft)}.is-sorting .is-drag-source{opacity:.45;background:var(--fonu-bg-muted)}.is-sorting .is-drag-source .is-drag-source{opacity:1}.service-block.is-drag-target,.rule-line.is-drag-target{outline:none;background:var(--fonu-brand-soft);box-shadow:inset 0 0 0 2px var(--fonu-brand)}.is-drag-target::after{content:"";position:absolute;z-index:3;left:5px;right:5px;top:0;height:3px;background:var(--fonu-brand);border-radius:3px;pointer-events:none}.is-drag-target.drop-after::after{top:auto;bottom:0}.is-sorting .drag-handle{cursor:grabbing}
@container proxy-workspace (max-width:960px){.service-blocks--cards .expanded-panel .child-header{display:none}}
@container proxy-workspace (max-width:960px){.child-header{display:none}.workspace-tools{align-items:stretch}.group-tabs,.filters{flex:1 1 100%}.summary-metrics{gap:9px}.service-summary{padding:9px}}
@container proxy-workspace (max-width:480px){.filters{flex-wrap:wrap}.filters :deep(.n-input){flex:1 1 100%}.status-filter{flex:1}.workspace-footer{justify-content:flex-start}.summary-metrics{width:100%;font-size:11px}.group-tabs button{padding:5px 8px}.workspace-heading h2{font-size:17px}}
</style>
