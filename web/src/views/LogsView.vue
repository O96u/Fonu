<template>
  <PageHeader title="日志中心" description="运行记录、访问请求与异常排查">
    <template #actions><n-button size="small" :loading="summaryLoading || loading" @click="refreshAll(true)">刷新全部</n-button></template>
  </PageHeader>
  <div class="log-workspace">
    <div class="log-source-grid">
      <button v-for="item in sourceCards" :key="item.key" type="button" class="log-source" :data-testid="'log-source-'+item.key" :class="{ 'is-active': tab === item.key }" :aria-pressed="tab === item.key" @click="selectSource(item.key)">
        <span class="source-heading"><n-icon :component="item.icon" />{{ item.label }}<n-icon v-if="tab === item.key" :component="CheckmarkCircleOutline" class="source-check" /></span>
        <strong v-if="summary[item.key] && !summary[item.key].error">{{ summary[item.key].count.toLocaleString() }}</strong>
        <strong v-else>—</strong>
        <small>{{ summary[item.key]?.error ? '统计失败 · 可重试' : summaryLoading && !summary[item.key] ? '统计中…' : '留存记录' }}</small>
      </button>
    </div>
    <div class="summary-caption"><span>{{ summaryError || (summaryAt ? '统计更新于 '+formatLogTime(summaryAt,{timeZone:logTimeZone}) : '正在统计四类日志') }}</span><span>统计范围：当前文件及可识别的轮转归档</span></div>
    <FonuCard flush class="logs-card">
      <div class="log-workspace-head">
        <div><strong>{{ sourceCards.find(item => item.key === tab)?.label }}</strong><small>{{ sourceDescriptions[tab] }}</small></div>
        <div class="log-head-actions">
          <n-button size="small" :loading="exporting" :disabled="loading || !activeRows.length" @click="exportHistory">导出筛选结果</n-button>
          <n-button size="small" :type="autoRefresh ? 'primary' : 'default'" secondary @click="autoRefresh = !autoRefresh">{{ autoRefresh ? '自动刷新已开启' : '自动刷新' }}</n-button>
        </div>
      </div>
      <div class="history-filters">
        <div class="filter-row compact-filters" :class="{ 'compact-filters--access': tab==='access' }" data-testid="log-filters">
          <n-input v-model:value="current.keyword" class="keyword" placeholder="关键词搜索" clearable />
          <n-input v-model:value="current.service" class="service-filter" :placeholder="tab==='access'?'服务或域名':tab==='system'?'模块名称':tab==='frp'?'隧道或来源':'域名或来源'" clearable />
          <n-input v-if="tab==='access'" v-model:value="current.ip" class="ip-filter" placeholder="客户端 IP" clearable />
          <n-select v-model:value="current.range" :options="rangeOptions" class="range-select" data-testid="log-range-select" />
          <n-select v-if="tab==='access'" v-model:value="current.status" :options="statusOptions" placeholder="全部状态" clearable class="status-select"/>
          <n-select v-else v-model:value="current.level" :options="levelOptions" placeholder="全部级别" clearable class="status-select"/>
          <n-select v-if="tab==='access'" v-model:value="current.kind" :options="kindOptions" placeholder="访问类型" clearable class="kind-filter" />
          <div class="filter-actions"><n-button size="small" :loading="loading" @click="loadHistory">刷新</n-button><n-button text size="small" @click="resetFilters">清除筛选</n-button></div>
        </div>
        <div v-if="current.range==='custom'" class="custom-range-row">
          <span>时间范围</span>
          <n-date-picker v-model:value="current.custom" type="datetimerange" start-placeholder="开始日期和时间" end-placeholder="结束日期和时间" clearable data-testid="log-custom-range"/>
        </div>
        <p v-if="tab==='access' && current.kind" class="filter-note">拦截及扫描类型由日志字段推断，不代表已确认命中的具体规则。</p>
        <p v-if="current.range==='custom'" class="filter-note">自定义时间按当前浏览器时区输入，列表按 Fonu 设置的时区显示。</p>
      </div>
      <div v-if="error" class="log-error"><span>{{ error }}</span><n-button size="small" @click="loadHistory">重试</n-button></div>
      <div v-if="loading && !activeRows.length" class="log-loading"><n-spin size="small" /> 正在读取日志…</div>
      <template v-if="activeRows.length">
        <n-data-table v-if="tab==='system'" class="log-table" :columns="systemColumns" :data="systemLogs" :loading="loading" :bordered="false" :scroll-x="650" :row-class-name="systemRowClassName" :row-props="row => detailRowProps('system',row)"/>
        <n-data-table v-else-if="tab==='access'" class="log-table" :columns="accessColumns" :data="accessLogs" :loading="loading" :bordered="false" :scroll-x="showAccessBlockReason?1280:1080" :row-class-name="accessRowClassName" :row-props="row => detailRowProps('access',row)"/>
        <n-data-table v-else-if="tab==='nginx'" class="log-table" :columns="errorColumns" :data="errorLogs" :loading="loading" :bordered="false" :scroll-x="600" :row-class-name="errorRowClassName" :row-props="row => detailRowProps('nginx',row)"/>
        <n-data-table v-else class="log-table" :columns="frpColumns" :data="frpLogs" :loading="loading" :bordered="false" :scroll-x="720" :row-class-name="frpRowClassName" :row-props="row => detailRowProps('frp',row)"/>
      </template>
      <EmptyState v-else-if="!loading && !error" :title="awaitingCustomRange ? '请选择时间范围' : hasFilters ? '没有匹配的日志' : '暂无日志'" :description="awaitingCustomRange ? '选择开始与结束时间并确认后自动查询。' : hasFilters ? '请调整时间范围或筛选条件。' : '当前日志文件及归档中暂无记录。'"/>
      <div class="history-footer">
        <span>{{ activeRows.length ? '第 '+(offset+1)+'–'+(offset+activeRows.length)+' 条匹配记录' : '暂无匹配记录' }}<small v-if="autoRefresh && offset>0">历史页暂停自动刷新</small></span>
        <div class="history-pagination"><n-select v-model:value="pageSize" :options="pageSizeOptions" size="small" style="width:106px"/><n-button size="small" :disabled="loading||offset===0" @click="previousPage">上一页</n-button><span>第 {{ Math.floor(offset/pageSize)+1 }} 页</span><n-button size="small" :disabled="loading||!hasMore" @click="nextPage">下一页</n-button></div>
      </div>
    </FonuCard>
  </div>
  <n-drawer v-model:show="showDetail" :width="'min(520px, 94vw)'">
    <n-drawer-content title="日志详情" closable>
      <template v-if="detail">
        <div class="detail-fields" data-testid="log-detail-fields"><div v-for="field in detail.fields" :key="field.label"><span>{{ field.label }}</span><strong>{{ field.value || '—' }}</strong></div></div>
        <h4>{{ detail.rawLabel }}</h4><pre class="fonu-log-console detail-console">{{ detail.raw }}</pre>
        <div class="detail-actions"><n-button size="small" @click="copyDetail">复制记录</n-button><n-button v-if="detail.ip" size="small" type="primary" secondary @click="filterDetailIP">筛选此 IP</n-button><n-button v-if="detail.access" size="small" type="warning" secondary :loading="blacklistBusy" :disabled="!detail.access.client_ip" @click="manageBlacklist(detail.access)">{{blacklistLabel(detail.access, true)}}</n-button></div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NButton,
  NDataTable,
  NDrawer,
  NDrawerContent,
  NIcon,
  NSpin,
  NInput,
  NDatePicker,
  NSelect,
  NTag,
  useMessage,
  useDialog,
  type DataTableColumns,
} from 'naive-ui'
import { api } from '../api/client'
import type { AccessLogEntry, SystemLogEntry } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import PageHeader from '../components/PageHeader.vue'
import {
  accessServiceTooltip,
  buildProxyBindingIndex,
  resolveAccessServiceLabel,
} from '../utils/accessService'
import { useAppTimezone } from '../composables/useAppTimezone'
import { formatLogTime, formatMs } from '../utils/format'
import { displayGeoLabel } from '../utils/geoDisplay'
import {
  frpLevelTagType,
  latencyClass,
  nginxLevelTagType,
  parseFrpLogLine,
  parseNginxErrorLine,
  type ParsedFrpLog,
  type ParsedNginxError,
  displaySystemLog,
  systemLevelTagType,
  systemModuleTagType,
} from '../utils/logDisplay'
import HttpStatusBadge from '../components/HttpStatusBadge.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { copyToClipboard } from '../utils/clipboard'
import { DesktopOutline, GlobeOutline, AlertCircleOutline, GitNetworkOutline, CheckmarkCircleOutline } from '@vicons/ionicons5'


type LogTab = 'system'|'access'|'nginx'|'frp'
type LogFilter = {keyword:string;service:string;ip:string;kind:string|null;status:number|null;level:string|null;range:string;custom:[number,number]|null}
const makeFilter=():LogFilter=>({keyword:'',service:'',ip:'',kind:null,status:null,level:null,range:'all',custom:null})
const filters=reactive<Record<LogTab,LogFilter>>({system:makeFilter(),access:makeFilter(),nginx:makeFilter(),frp:makeFilter()})
const route=useRoute(),router=useRouter(),message=useMessage(),dialog=useDialog(),logTimeZone=useAppTimezone()
const blacklistBusy=ref(false)
function resolveTab(value:unknown):LogTab{
 if(value==='error')return 'nginx'
 if(value==='stream')return 'access'
 if(value==='access'||value==='nginx'||value==='frp'||value==='system')return value
 return route.query.ip||route.query.kind?'access':'system'
}
const tab=ref<LogTab>(resolveTab(route.query.tab))
const current=computed(()=>filters[tab.value])
const awaitingCustomRange=computed(()=>current.value.range==='custom'&&!current.value.custom)
const sourceCards=[{key:'system' as const,label:'系统日志',icon:DesktopOutline},{key:'access' as const,label:'访问日志',icon:GlobeOutline},{key:'nginx' as const,label:'Nginx 日志',icon:AlertCircleOutline},{key:'frp' as const,label:'FRP 日志',icon:GitNetworkOutline}]
const sourceDescriptions={system:'应用运行、DDNS、证书与维护记录（常规 HTTP INFO 噪声已排除）',access:'经 Nginx 的反代访问请求',nginx:'Nginx 错误、上游与 TLS 异常',frp:'frpc 连接、代理与隧道记录'}
const rangeOptions=[{label:'全部留存时间',value:'all'},{label:'最近 24 小时',value:'24h'},{label:'最近 7 天',value:'7d'},{label:'最近 30 天',value:'30d'},{label:'自定义时间',value:'custom'}]
const statusOptions=[200,201,204,301,302,304,400,401,403,404,408,429,499,500,502,503,504].map(value=>({value,label:String(value)}))
const levelOptions=[{label:'信息',value:'INFO'},{label:'警告',value:'WARN'},{label:'错误',value:'ERROR'},{label:'调试',value:'DEBUG'},{label:'未知',value:'UNKNOWN'}]
const kindOptions=[{label:'正常访问',value:'access'},{label:'策略拦截',value:'intercept'},{label:'403 拒绝',value:'403'},{label:'429 限流',value:'429'},{label:'401 未授权',value:'401'},{label:'疑似扫描',value:'scan'}]
const pageSizeOptions=[20,50,100].map(value=>({value,label:value+' 条/页'}))
const summary=ref<Record<string,{count:number;error?:string}>>({}),summaryAt=ref(''),summaryError=ref(''),summaryLoading=ref(false)
const loading=ref(false),error=ref(''),exporting=ref(false),autoRefresh=ref(true)
const offset=ref(0),pageSize=ref(20),hasMore=ref(false)
const accessLogs=ref<AccessLogEntry[]>([]),systemLogs=ref<SystemLogEntry[]>([]),errorLogs=ref<ParsedNginxError[]>([]),frpLogs=ref<ParsedFrpLog[]>([])
type LogBlacklistStatus = Awaited<ReturnType<typeof api.logBlacklistStatus>>[number]
const blacklistStatuses=ref<Record<string,LogBlacklistStatus>>({})
let blacklistStatusController:AbortController|undefined,blacklistStatusGeneration=0
function blacklistInput(row:AccessLogEntry){return {host:row.domain||'',port:row.server_port||0,ip:row.client_ip||''}}
function blacklistKey(row:AccessLogEntry){return JSON.stringify(blacklistInput(row))}
function blacklistStatus(row:AccessLogEntry){return blacklistStatuses.value[blacklistKey(row)]}
function blacklistLabel(row:AccessLogEntry,detail=false){
 const s=blacklistStatus(row)
 if(!s||s.error)return detail?'管理 IP 黑名单':'黑名单'
 if(s.exact)return s.scope==='global'?'全局已加入':'规则已加入'
 if(s.covered)return '网段已覆盖'
 return detail?'加入黑名单':'加入黑名单'
}
function blacklistButtonType(row:AccessLogEntry){const s=blacklistStatus(row);return s?.exact||s?.covered?'error':'warning'}
async function loadBlacklistStatuses(){
 const id=++blacklistStatusGeneration
 blacklistStatusController?.abort()
 if(disposed||tab.value!=='access'){blacklistStatuses.value={};return}
 const unique=new Map(accessLogs.value.map(row=>[blacklistKey(row),blacklistInput(row)]))
 const keys=[...unique.keys()]
 if(!keys.length){blacklistStatuses.value={};return}
 const ctl=new AbortController();blacklistStatusController=ctl
 try{
  const rows=await api.logBlacklistStatus([...unique.values()],ctl.signal)
  if(disposed||id!==blacklistStatusGeneration)return
  if(!Array.isArray(rows)||rows.length!==keys.length)throw Error('invalid blacklist status response')
  blacklistStatuses.value=Object.fromEntries(keys.map((key,i)=>[key,rows[i]]))
 }catch{if(!disposed&&id===blacklistStatusGeneration)blacklistStatuses.value={}}
}
function updateBlacklistStatuses(row:AccessLogEntry,result:Awaited<ReturnType<typeof api.logBlacklist>>){
 const next={...blacklistStatuses.value}
 const clicked=blacklistKey(row)
 for(const [key,s] of Object.entries(next)){
  if(s.ip!==result.ip)continue
  if(result.scope==='global')next[key]={...s,global_exact:result.exact,global_covered:result.covered,...(s.scope==='global'?{exact:result.exact,covered:result.covered}:{})}
  else if(s.scope==='rule'&&s.rule_id===result.rule_id)next[key]={...s,exact:result.exact,covered:result.covered}
 }
 const previous=next[clicked]
 next[clicked]={...previous,scope:result.scope,rule_id:result.rule_id,ip:result.ip,exact:result.exact,covered:result.covered,global_exact:result.scope==='global'?result.exact:previous?.global_exact??false,global_covered:result.scope==='global'?result.covered:previous?.global_covered??false,error:undefined}
 blacklistStatuses.value=next
}
const activeRows=computed(()=>({access:accessLogs.value,system:systemLogs.value,nginx:errorLogs.value,frp:frpLogs.value})[tab.value])
const proxyBindingIndex=ref(buildProxyBindingIndex([]))
const showAccessBlockReason=computed(()=>Boolean(current.value.kind)||accessLogs.value.some(row=>Boolean(row.block_reason)))
const hasFilters=computed(()=>{const c=current.value;return Boolean(c.keyword||c.service||c.ip||c.kind||c.status||c.level||c.range!=='all')})
function queryParams(){
 const c=current.value
 let from:string|undefined,to:string|undefined
 if(c.range==='custom'&&c.custom){from=new Date(c.custom[0]).toISOString();to=new Date(c.custom[1]).toISOString()}
 else if(c.range!=='all'&&c.range!=='custom'){
  const hours=c.range==='24h'?24:c.range==='7d'?168:720
  from=new Date(Date.now()-hours*3600000).toISOString()
 }
 return {source:tab.value,keyword:c.keyword.trim(),service:c.service.trim(),ip:tab.value==='access'?c.ip.trim():undefined,kind:tab.value==='access'?c.kind||undefined:undefined,status:tab.value==='access'?c.status??undefined:undefined,level:tab.value!=='access'?c.level||undefined:undefined,from,to}
}
let disposed=false,generation=0,historyController:AbortController|undefined,summaryController:AbortController|undefined,exportController:AbortController|undefined
let debounce:ReturnType<typeof setTimeout>|undefined,timer:ReturnType<typeof setInterval>|undefined
let nextSummaryAt=0
let pageSnapshot=''
function clearRows(){blacklistStatusGeneration++;blacklistStatusController?.abort();blacklistStatuses.value={};accessLogs.value=[];systemLogs.value=[];errorLogs.value=[];frpLogs.value=[];hasMore.value=false}
async function loadSummary(force=false){
 if(disposed||summaryLoading.value)return
 summaryLoading.value=true;summaryError.value=''
 summaryController=new AbortController()
 try{const r=await api.getLogSummary(summaryController.signal,force);if(!disposed){summary.value=r.sources;summaryAt.value=r.updated_at;nextSummaryAt=Date.now()+30000}}
 catch(e){if(!disposed)summaryError.value=e instanceof Error?e.message:'统计失败'}
 finally{summaryLoading.value=false}
}
async function loadHistory(){
 if(disposed)return
 clearTimeout(debounce)
 if(awaitingCustomRange.value){loading.value=false;error.value='';return}
 historyController?.abort()
 const ctl=new AbortController();historyController=ctl
 const id=++generation,source=tab.value
 if(offset.value===0)pageSnapshot=new Date().toISOString()
 loading.value=true;error.value=''
 try{
  const r=await api.getLogHistory({...queryParams(),offset:offset.value,limit:pageSize.value,snapshot:pageSnapshot},ctl.signal)
  if(disposed||id!==generation)return
  if(source==='access'){accessLogs.value=r.rows as AccessLogEntry[];void loadBlacklistStatuses()}
  else if(source==='system')systemLogs.value=r.rows as SystemLogEntry[]
  else if(source==='nginx')errorLogs.value=(r.rows as string[]).map(parseNginxErrorLine)
  else frpLogs.value=(r.rows as string[]).map(parseFrpLogLine)
  hasMore.value=r.has_more
 }catch(e){if(!disposed&&id===generation)error.value=e instanceof Error?e.message:'读取历史日志失败'}
 finally{if(id===generation)loading.value=false}
}
function refreshAll(force=false){void loadSummary(force);void loadHistory()}
function selectSource(key:LogTab){
 if(key===tab.value)return
 const c=filters[key]
 void router.replace({name:'logs',query:{tab:key,...(key==='access'?{ip:c.ip||undefined,kind:c.kind||undefined,keyword:c.keyword||undefined}:{})}})
}
function resetFilters(){
 Object.assign(current.value,makeFilter())
 if(tab.value==='access')void router.replace({name:'logs',query:{tab:'access'}})
}
function previousPage(){offset.value=Math.max(0,offset.value-pageSize.value);clearRows();void loadHistory()}
function nextPage(){offset.value+=pageSize.value;clearRows();void loadHistory()}
function applyRoute(){
 tab.value=resolveTab(route.query.tab)
 if(tab.value==='access'){
  filters.access.ip=typeof route.query.ip==='string'?route.query.ip:''
  filters.access.kind=typeof route.query.kind==='string'?route.query.kind:null
  filters.access.keyword=typeof route.query.keyword==='string'?route.query.keyword:''
 }
}
async function exportHistory(){
 if(exporting.value||loading.value)return
 exporting.value=true
 const ctl=new AbortController();exportController=ctl
 const source=tab.value
 try{
  const r=await api.getLogHistory({...queryParams(),offset:0,limit:10000,export:1},ctl.signal)
  if(disposed)return
  if(r.has_more){message.warning('匹配记录超过 10000 条，请缩小时间范围后导出');return}
  const blob=new Blob([JSON.stringify(r.rows,null,2)],{type:'application/json;charset=utf-8'})
  const url=URL.createObjectURL(blob),a=document.createElement('a')
  a.href=url;a.download='fonu-'+source+'-logs.json';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)
  message.success('已导出 '+r.rows.length+' 条匹配记录')
 }catch(e){if(!disposed)message.error(e instanceof Error?e.message:'导出失败')}
 finally{exporting.value=false}
}
applyRoute()
watch(()=>route.fullPath,()=>{if(route.name==='logs')applyRoute()})
watch(()=>JSON.stringify([tab.value,current.value,pageSize.value]),()=>{
 historyController?.abort();generation++;offset.value=0;clearRows();error.value='';loading.value=true
 clearTimeout(debounce);debounce=setTimeout(()=>void loadHistory(),250)
})
onMounted(()=>{
 refreshAll()
 timer=setInterval(()=>{
  if(disposed||document.hidden||route.name!=='logs')return
  if(Date.now()>=nextSummaryAt)void loadSummary()
  if(autoRefresh.value&&offset.value===0&&!loading.value&&!awaitingCustomRange.value)void loadHistory()
 },10000)
})
onUnmounted(()=>{disposed=true;generation++;blacklistStatusController?.abort();historyController?.abort();summaryController?.abort();exportController?.abort();clearTimeout(debounce);if(timer)clearInterval(timer)})
const methodTagType = (method: string) => {
  const map: Record<string, 'success' | 'info' | 'warning' | 'error' | 'default'> = {
    GET: 'success',
    POST: 'info',
    PUT: 'warning',
    DELETE: 'error',
  }
  return map[method] ?? 'default'
}

function levelBadge(type: string, text: string) { return h(StatusBadge, { kind: type === 'error' ? 'error' : type === 'warning' ? 'warning' : 'unknown', text }) }
const logTimeCell = (time: string) =>
  h('span', { class: 'log-time' }, formatLogTime(time, { timeZone: logTimeZone.value }))

function accessRowClassName(row: AccessLogEntry) {
  if (row.status >= 500) return 'log-row log-row--error'
  if (row.status >= 400) return 'log-row log-row--warn'
  return 'log-row'
}

function errorRowClassName(row: ParsedNginxError) {
  const type = nginxLevelTagType(row.level)
  if (type === 'error') return 'log-row log-row--error'
  if (type === 'warning') return 'log-row log-row--warn'
  return 'log-row'
}

function frpRowClassName(row: ParsedFrpLog) {
  const type = frpLevelTagType(row.level)
  if (type === 'error') return 'log-row log-row--error'
  if (type === 'warning') return 'log-row log-row--warn'
  return 'log-row'
}

function systemRowClassName(row: SystemLogEntry) {
  const display = displaySystemLog(row)
  if (display.displayLevel === '错误') return 'log-row log-row--error'
  if (display.displayLevel === '警告') return 'log-row log-row--warn'
  return 'log-row'
}

const accessColumns = computed<DataTableColumns<AccessLogEntry>>(() => {
  const cols: DataTableColumns<AccessLogEntry> = [
  { title: '时间', key: 'time', width: 168, render: (row) => logTimeCell(row.time) },
  {
    title: '服务',
    key: 'service',
    width: 128,
    render: (row) => {
      const tooltip = accessServiceTooltip(row, proxyBindingIndex.value)
      const label = resolveAccessServiceLabel(row, proxyBindingIndex.value)
      return h('span', { class: 'log-service nowrap', title: tooltip }, label)
    },
  },
  {
    title: '方法',
    key: 'method',
    width: 68,
    render: (row) => h(NTag, { size: 'small', bordered: false, type: methodTagType(row.method) }, () => row.method),
  },
  {
    title: '路径',
    key: 'path',
    width: 220,
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'log-path mono nowrap', title: row.path }, row.path),
  },
  {
    title: '状态',
    key: 'status',
    width: 68,
    render: (row) => h(HttpStatusBadge, { code: row.status }),
  },
  ]

  if (showAccessBlockReason.value) {
    cols.push({
      title: '拦截原因',
      key: 'block_reason',
      width: 200,
      ellipsis: { tooltip: true },
      render: (row) =>
        h(
          'span',
          { class: 'log-block-reason', title: row.block_reason || '—' },
          row.block_reason?.trim() || '—',
        ),
    })
  }

  cols.push(
  {
    title: '耗时',
    key: 'response_time',
    width: 80,
    render: (row) => h('span', { class: latencyClass(row.response_time) }, formatMs(row.response_time)),
  },
  {
    title: '来源 IP',
    key: 'client_ip',
    width: 168,
    render: (row) => {
      const geoRaw = row.geo?.trim()
      const geo = geoRaw ? displayGeoLabel(geoRaw) : ''
      return h('div', { class: 'ip-geo-cell' }, [
        h('span', { class: 'mono ip-geo-cell__ip', title: row.client_ip }, row.client_ip),
        geo
          ? h('span', { class: 'ip-geo-cell__geo', title: geoRaw }, geo)
          : h('span', { class: 'ip-geo-cell__geo ip-geo-cell__geo--empty' }, '—'),
      ])
    },
  },
  )
  cols.push({title:'操作',key:'access_actions',width:160,render:row=>{
    const s=blacklistStatus(row)
    return h('div',{class:'log-blacklist-action'},[
      h(NButton,{size:'tiny',quaternary:!(s?.exact||s?.covered),secondary:Boolean(s?.exact||s?.covered),type:blacklistButtonType(row),disabled:!row.client_ip||row.client_ip==='-'||blacklistBusy.value,title:s?.error||'当前黑名单配置状态，点击管理；不代表历史请求被拦截',onClick:(e:MouseEvent)=>{e.stopPropagation();void manageBlacklist(row)},onKeydown:(e:KeyboardEvent)=>e.stopPropagation()},()=>blacklistLabel(row)),
      s?.scope==='rule'&&(s.global_exact||s.global_covered)?h('small',{class:'log-blacklist-global',title:'该 IP 同时受全局黑名单限制'},'全局已限制'):null
    ])
  }})
  return cols
})

const errorColumns: DataTableColumns<ParsedNginxError> = [
  { title: '时间', key: 'time', width: 170, render: (row) => logTimeCell(row.time || '-') },
  {
    title: '级别',
    key: 'level',
    width: 88,
    render: (row) =>
      levelBadge(nginxLevelTagType(row.level), row.level === 'unknown' ? '未知' : row.level.toUpperCase()),
  },
  {
    title: '消息',
    key: 'message',
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'log-message mono' }, row.message),
  },
]

const frpColumns: DataTableColumns<ParsedFrpLog> = [
  { title: '时间', key: 'time', width: 170, render: (row) => logTimeCell(row.time || '-') },
  {
    title: '级别',
    key: 'level',
    width: 88,
    render: (row) =>
      levelBadge(frpLevelTagType(row.level), row.level === 'unknown' ? '未知' : row.level),
  },
  {
    title: '来源',
    key: 'source',
    width: 180,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('span', { class: 'mono nowrap', title: row.source }, row.source || '—'),
  },
  {
    title: '消息',
    key: 'message',
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'log-message mono' }, row.message),
  },
]

const systemColumns: DataTableColumns<SystemLogEntry> = [
  { title: '时间', key: 'time', width: 168, render: (row) => logTimeCell(row.time) },
  {
    title: '级别',
    key: 'level',
    width: 80,
    render: (row) => {
      const display = displaySystemLog(row)
      return levelBadge(systemLevelTagType(display.displayLevel), display.displayLevel)
    },
  },
  {
    title: '模块',
    key: 'module',
    width: 108,
    render: (row) => {
      const display = displaySystemLog(row)
      return h(
        NTag,
        {
          size: 'small',
          bordered: false,
          type: systemModuleTagType(display.displayModule),
          class: 'log-module-tag',
        },
        () => display.displayModule,
      )
    },
  },
  {
    title: '消息',
    key: 'message',
    ellipsis: { tooltip: true },
    render: (row) => {
      const display = displaySystemLog(row)
      return h('span', { class: 'log-message' }, display.message)
    },
  },
]


const showDetail = ref(false)
const detail = ref<{ fields: {label:string;value:string}[]; raw:string; rawLabel:string; ip?:string; access?:AccessLogEntry } | null>(null)
function detailRowProps(kind: LogTab, row: unknown) {
  const open = () => openDetail(kind, row)
  return { tabindex:0, style:'cursor:pointer', onClick:open, onKeydown:(event:KeyboardEvent) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault();open() } } }
}
function openDetail(kind: LogTab, row: unknown) {
  const fields: { label:string;value:string }[] = []
  const add = (label:string, value:unknown) => fields.push({label,value:String(value ?? '')})
  let raw = JSON.stringify(row,null,2) ?? '', rawLabel = '记录字段（JSON）', ip:string|undefined
  if (kind === 'access') {
    const r = row as AccessLogEntry
    add('访问时间', formatLogTime(r.time,{timeZone:logTimeZone.value}));add('服务',resolveAccessServiceLabel(r,proxyBindingIndex.value));add('域名',r.domain);add('端口',r.server_port);add('请求方法',r.method);add('路径',r.path);add('客户端 IP',r.client_ip);add('归属地',r.geo ? displayGeoLabel(r.geo) : '');add('HTTP 状态',r.status);add('响应耗时',formatMs(r.response_time));add('上游',r.upstream);add('User-Agent',r.user_agent);add('请求字节',r.request_length);add('响应字节',r.bytes_sent);add('拦截原因（推测）',r.block_reason);ip=r.client_ip
  } else if (kind === 'system') {
    const r = displaySystemLog(row as SystemLogEntry)
    add('时间',formatLogTime(r.time,{timeZone:logTimeZone.value}));add('级别',r.displayLevel);add('模块',r.displayModule);add('消息',r.message)
  } else {
    const r = row as ParsedNginxError | ParsedFrpLog
    add('时间',r.time ? formatLogTime(r.time,{timeZone:logTimeZone.value}) : '');add('级别',r.level);if ('source' in r) add('来源',r.source);add('消息',r.message);raw=r.raw;rawLabel='原始日志'
  }
  detail.value={fields,raw,rawLabel,ip,access:kind==='access'?row as AccessLogEntry:undefined};showDetail.value=true
}
async function copyDetail() {
  if (!detail.value) return
  if (await copyToClipboard(detail.value.raw)) message.success('已复制记录')
  else message.error('复制失败')
}
function filterDetailIP() {
  if (!detail.value?.ip) return
  showDetail.value=false
  filters.access.ip=detail.value.ip
  void router.replace({name:'logs',query:{tab:'access',ip:detail.value.ip}})
}

async function manageBlacklist(row:AccessLogEntry){
  if(blacklistBusy.value||!row.client_ip)return
  const input={host:row.domain||'',port:row.server_port||0,ip:row.client_ip}
  blacklistBusy.value=true
  try{
    const current=await api.logBlacklist({...input,action:'inspect'})
    if(disposed)return
    const global=current.scope==='global'
    const scopeLabel=global?'全局黑名单':'当前规则黑名单'
    if(current.covered&&!current.exact){
      dialog.info({title:'该 IP 已被'+scopeLabel+'中的网段覆盖',content:'IP：'+current.ip+'。请在'+(global?'系统设置 → 反代安全':'当前反代规则的安全设置')+'中管理对应 CIDR，快捷操作不会删除整个网段。',positiveText:'知道了'})
      return
    }
    const removing=current.exact
    dialog.warning({
      title:(removing?'确认解除':'确认加入')+scopeLabel,
      content:()=>h('div',{class:'blacklist-confirm'},[
        h('p','来源 IP：'+current.ip),
        h('p','日志入口：'+(row.domain||'未识别入口')+(row.server_port?'（端口 '+row.server_port+'）':'')),
        h('p',global?'未匹配当前启用的反代规则。作用范围：全局反代 IP 策略。':'当前规则：'+(current.rule_name||('#'+current.rule_id))+'。作用范围：该规则下的所有域名，不影响其他独立规则。'),
        h('p',removing?'仅移除该范围内的单地址条目；其他黑名单、网段限制或安全策略仍可能拒绝访问。':global?'加入后可能无法访问所有自动生成配置的反代服务，直到手动解除。':'加入后该 IP 将受到当前规则的黑名单限制，直到手动解除。'),
        h('p','请确认此 IP 是真实客户端地址，而非路由器、CDN、转发代理或共享出口。'),
        current.custom_config?h('p','存在手动 Nginx 配置，可能覆盖全局访问策略，需另行核验。'):null,
        current.whitelist_mode?h('p','当前规则启用了白名单；黑名单优先于白名单。'):null
      ]),
      positiveText:removing?'确认解除':'确认加入',negativeText:'取消',
      onPositiveClick:async()=>{
        if(blacklistBusy.value||disposed)return false
        blacklistBusy.value=true
        try{
          const result=await api.logBlacklist({...input,ip:current.ip,scope:current.scope,rule_id:current.rule_id,action:removing?'unblock':'block',confirmed:true})
          if(!disposed){updateBlacklistStatuses(row,result);void loadBlacklistStatuses();message.success(result.changed?(removing?(result.covered?'已移除单地址，仍受网段限制':'已移除'+scopeLabel+'中的单地址'):'已加入'+scopeLabel):'黑名单状态已更新，无需重复修改')}
          return true
        }catch(e){if(!disposed)message.error(e instanceof Error?e.message:'黑名单更新失败');return false}
        finally{blacklistBusy.value=false}
      }
    })
  }catch(e){if(!disposed)message.warning(e instanceof Error?e.message:'无法确认黑名单作用范围')}
  finally{blacklistBusy.value=false}
}

</script>
<style scoped>
.log-blacklist-action{display:flex;align-items:flex-start;flex-direction:column;gap:2px}.log-blacklist-global{font-size:11px;color:var(--fonu-error)}
.logs-card :deep(.n-tabs-nav) {
  padding: var(--fonu-space-4) var(--fonu-space-5) 0;
}

.logs-card :deep(.n-tabs-tab) {
  font-size: 14px;
}

.logs-card :deep(.n-tab-pane) {
  padding-top: var(--fonu-space-2);
}

.tab-hint {
  margin: 0 var(--fonu-space-5) var(--fonu-space-3);
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.access-filter-banner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--fonu-space-2);
  padding: var(--fonu-space-2) var(--fonu-space-3);
  border-radius: 8px;
  background: var(--fonu-surface-muted, rgba(0, 0, 0, 0.04));
  border: 1px solid var(--fonu-border);
  color: var(--fonu-text);
}

.access-filter-clear {
  margin-left: auto;
  font-size: 13px;
  color: var(--fonu-primary);
  text-decoration: none;
}

.access-filter-clear:hover {
  text-decoration: underline;
}

.log-pagination {
  display: flex;
  justify-content: flex-end;
  padding: var(--fonu-space-4) var(--fonu-space-5);
  border-top: 1px solid var(--fonu-border);
}

.logs-card :deep(.log-table .n-data-table-th) {
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
}

.logs-card :deep(.log-table .n-data-table-base-table) {
  table-layout: fixed;
}

.logs-card :deep(.log-table .n-data-table-td) {
  font-size: 13px;
  padding-top: 10px;
  padding-bottom: 10px;
}

.nowrap {
  white-space: nowrap;
}

.logs-card :deep(.log-table .n-data-table-tr.log-row--error .n-data-table-td) {
  background: color-mix(in srgb, var(--fonu-error) 7%, transparent);
}

.logs-card :deep(.log-table .n-data-table-tr.log-row--warn .n-data-table-td) {
  background: color-mix(in srgb, var(--fonu-warning) 7%, transparent);
}

.logs-card :deep(.log-table .n-data-table-tr.log-row:hover .n-data-table-td) {
  background: color-mix(in srgb, var(--fonu-brand) 5%, var(--fonu-surface));
}

.log-time {
  white-space: nowrap;
  color: var(--fonu-text-secondary);
  font-variant-numeric: tabular-nums;
}

.log-service {
  font-weight: 500;
}

.log-path {
  color: var(--fonu-text-secondary);
}

.log-message {
  line-height: 1.5;
}

.mono {
  font-family: var(--fonu-mono);
  font-size: 12px;
}

.logs-card :deep(.ip-geo-cell) {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  width: 100%;
}

.logs-card :deep(.ip-geo-cell__ip) {
  display: block;
  width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.logs-card :deep(.ip-geo-cell__geo) {
  display: block;
  width: 100%;
  font-size: 11px;
  color: var(--fonu-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.logs-card :deep(.ip-geo-cell__geo--empty) {
  opacity: 0.45;
}

.ip-geo-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  min-width: 0;
  max-width: 100%;
}

.ip-geo-cell__ip {
  display: block;
  white-space: nowrap;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ip-geo-cell__geo {
  display: block;
  font-size: 11px;
  line-height: 1.35;
  color: var(--fonu-text-muted);
  white-space: nowrap;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-duration {
  font-variant-numeric: tabular-nums;
}

.log-duration--slow {
  color: var(--fonu-warning);
  font-weight: 600;
}

.log-duration--critical {
  color: var(--fonu-error);
  font-weight: 600;
}

.logs-card :deep(.log-module-tag) {
  white-space: nowrap;
}
@media (max-width: 760px) {
  .logs-card { min-width: 0; }
}
.logs-card :deep(.fonu-card__header) { padding-bottom:10px; }
.logs-card :deep(.n-tabs-nav) { background:var(--fonu-bg-muted); padding:8px 12px 0; }
.logs-card :deep(.log-table .n-data-table-td) { padding-top:7px; padding-bottom:7px; font-size:12px; }
.logs-card :deep(.log-table .n-data-table-tr.log-row--error .n-data-table-td) { background:color-mix(in srgb,var(--fonu-error) 7%,transparent); }
.logs-card :deep(.log-table .n-data-table-tr.log-row--warn .n-data-table-td) { background:color-mix(in srgb,var(--fonu-warning) 7%,transparent); }

.log-workspace { display:grid; gap:12px; min-width:0; container-type:inline-size; }
.log-source-grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:10px; }
.log-source { min-width:0; display:grid; gap:6px; padding:14px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); background:var(--fonu-surface); text-align:left; color:var(--fonu-text); font:inherit; cursor:pointer; }
.log-source.is-active { border-color:var(--fonu-brand); background:color-mix(in srgb,var(--fonu-brand-soft) 45%,var(--fonu-surface)); }
.source-heading { display:flex; align-items:center; gap:7px; font-size:13px; color:var(--fonu-text-secondary); }
.source-heading .n-icon { font-size:18px; color:var(--fonu-brand-text); }
.source-check { margin-left:auto; }
.log-source strong { font-size:23px; line-height:1.2; font-variant-numeric:tabular-nums; }
.log-source small { font-size:11px; color:var(--fonu-text-secondary); }
.log-workspace-head { display:flex; align-items:center; justify-content:space-between; gap:12px; padding:16px; flex-wrap:wrap; border-bottom:1px solid var(--fonu-border); }
.log-workspace-head strong { font-size:16px; }
.log-workspace-head small { display:block; margin-top:4px; font-size:12px; color:var(--fonu-text-secondary); }
.log-head-actions { display:flex; gap:8px; flex-wrap:wrap; }
.log-error,.log-loading { display:flex; align-items:center; gap:10px; padding:12px 16px; color:var(--fonu-text-secondary); font-size:12px; }
.log-error { color:var(--fonu-error); }
.detail-fields { display:grid; gap:0; }
.detail-fields > div { display:grid; grid-template-columns:105px minmax(0,1fr); gap:12px; padding:10px 0; border-bottom:1px solid var(--fonu-border); font-size:13px; }
.detail-fields span { color:var(--fonu-text-secondary); }
.detail-fields strong { font-weight:500; overflow-wrap:anywhere; }
.detail-console { padding:12px; max-height:260px; margin:0; }
.detail-actions { display:flex; gap:8px; margin-top:12px; flex-wrap:wrap; }
@container(max-width:680px) { .log-source-grid {grid-template-columns:repeat(2,minmax(0,1fr))} .log-source{padding:10px} .logs-card :deep(.n-tabs-nav){padding-inline:12px} }

.log-workspace { gap:10px; }
.log-source-grid { gap:10px; }
.log-source { padding:12px; }
.summary-caption { display:flex; justify-content:space-between; flex-wrap:wrap; gap:8px; font-size:11px; color:var(--fonu-text-secondary); }
.history-filters { display:grid; gap:9px; padding:12px 16px; border-bottom:1px solid var(--fonu-border); }
.filter-row { display:flex; align-items:center; gap:8px; flex-wrap:wrap; }
.filter-row > .n-input { flex:1 1 180px; min-width:0; width:auto; }
.filter-row > .n-select { width:145px; }
.filter-row > .range-select { width:150px; }
.filter-row > .n-date-picker { flex:1 1 300px; min-width:0; }
.filter-note { margin:0; color:var(--fonu-text-secondary); font-size:11px; }
.history-footer { display:flex; align-items:center; justify-content:space-between; gap:12px; flex-wrap:wrap; padding:12px 16px; border-top:1px solid var(--fonu-border); font-size:12px; color:var(--fonu-text-secondary); }
.history-footer small { display:block; font-size:11px; }
.history-pagination { display:flex; align-items:center; gap:8px; flex-wrap:wrap; }
/* One compact row at normal FPK widths; wrap rather than overflow in narrow windows. */
.compact-filters { display:flex; flex-wrap:wrap; gap:8px; align-items:center; }
.compact-filters > .n-input { flex:1 1 110px; min-width:100px; width:auto; }
.compact-filters > .keyword { flex:2 1 150px; min-width:130px; }
.compact-filters > .range-select { flex:0 1 144px; width:144px; min-width:130px; }
.compact-filters > .status-select,.compact-filters > .kind-filter { flex:0 1 116px; width:116px; min-width:105px; }
.filter-actions { display:flex; align-items:center; gap:10px; flex:0 0 auto; white-space:nowrap; }
.custom-range-row { display:flex; align-items:center; gap:10px; flex-wrap:wrap; font-size:12px; color:var(--fonu-text-secondary); }
.custom-range-row > .n-date-picker { flex:0 1 420px; width:100%; min-width:0; }
@container(min-width:780px) { .compact-filters:not(.compact-filters--access) { flex-wrap:nowrap; } }
@container(min-width:1080px) { .compact-filters--access { flex-wrap:nowrap; } }
@container(max-width:550px) { .history-filters,.history-footer {padding:10px} .compact-filters > .n-select {flex:1 1 120px;width:auto} .compact-filters > .keyword {flex-basis:100%} .history-pagination{gap:5px} }
</style>

