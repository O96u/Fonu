<template>
  <section class="diagnostics-panel">
    <div class="diag-section"><div class="diag-heading"><div><h3>上游连接诊断</h3><small>{{rule.upstream}}</small></div><n-button type="primary" size="small" :loading="running" :disabled="loading" @click="run">立即检测</n-button></div>
      <div class="diag-auto"><div><strong>当前面板自动检测</strong><small>默认关闭。开启后每 60 秒检测一次，关闭详情或离开页面即停止。</small></div><n-switch :value="auto" :disabled="loading" @update:value="setAuto"/></div>
      <n-alert type="info" :bordered="false">由 Fonu 后端向已保存的上游发出 HEAD 请求，最多 5 秒，不跟随重定向、不携带认证信息。使用系统 CA 验证 TLS，与 Nginx 手动配置或跳过验证策略可能不同。检测结果不影响规则启停或保存。</n-alert>
      <n-alert v-if="error" type="error" :bordered="false">{{error}}</n-alert>
      <div v-if="latest" class="diag-latest"><div class="diag-heading"><n-tag :type="kind(latest.state)" :bordered="false">{{label(latest.state)}}</n-tag><small>{{formatTime(latest.checked_at)}}</small></div><strong>{{latest.http_status?'HTTP '+latest.http_status+' · ':''}}{{latest.duration_ms}} ms</strong><p>{{latest.detail}}</p></div>
      <div v-else class="diag-empty">{{loading?'正在读取诊断记录…':'尚未检测当前目标地址'}}</div>
    </div>
    <div class="diag-section"><div class="diag-heading"><h3>最近检测记录</h3><small>最多 5 条</small></div><div v-for="row in history" :key="row.id" class="diag-record"><div><n-tag size="small" :type="kind(row.state)" :bordered="false">{{label(row.state)}}</n-tag><strong>{{row.http_status?'HTTP '+row.http_status+' · ':''}}{{row.duration_ms}} ms</strong><small>{{formatTime(row.checked_at)}}</small></div><small>{{row.upstream}}{{row.upstream!==rule.upstream?'（旧目标）':''}}</small><p>{{row.detail}}</p></div><div v-if="!history.length&&!loading" class="diag-empty">暂无检测记录</div></div>
    <div class="diag-section"><div class="diag-heading"><h3>Nginx 全局配置发布</h3><n-button size="small" quaternary :loading="publicationLoading" @click="loadPublication">刷新</n-button></div><n-alert v-if="publicationError" type="warning">{{publicationError}}</n-alert><template v-if="publication"><n-tag :type="publication.state==='confirmed'?'success':publication.state==='failed'?'error':'default'">{{publicationLabel(publication.state)}}</n-tag><small v-if="publication.updated_at">{{formatTime(publication.updated_at)}}</small><p v-if="publication.last_error">{{publication.last_error}}</p><div v-for="(row,i) in (publication.history??[]).slice(0,5)" :key="i" class="diag-record"><div><strong>{{publicationLabel(row.state)}}</strong><small>{{formatTime(row.created_at)}}</small></div></div></template><small>全局发布记录不是单条规则的可用性证明。</small></div>
  </section>
</template>
<script setup lang="ts">
import {computed,onMounted,onUnmounted,ref,watch} from 'vue'
import {NAlert,NButton,NSwitch,NTag} from 'naive-ui'
import {api} from '../../api/client'
import type {ProxyDiagnostic,ProxyRule,NginxApplyStatus} from '../../api/types'
const props=defineProps<{rule:ProxyRule}>()
const history=ref<ProxyDiagnostic[]>([]),loading=ref(false),running=ref(false),error=ref(''),auto=ref(false)
const latest=computed(()=>history.value.find(r=>r.upstream===props.rule.upstream))
const publication=ref<NginxApplyStatus|null>(null),publicationLoading=ref(false),publicationError=ref('')
let controller:AbortController|undefined, generation=0, timer:ReturnType<typeof setTimeout>|undefined,alive=true
function label(s:string){return ({reachable:'HTTP 可达',http_error:'上游 5xx',tls_error:'证书校验失败',dns_error:'DNS 失败',timeout:'超时',invalid:'无效目标',failed:'连接失败'} as Record<string,string>)[s]||s}
function kind(s:string):'success'|'warning'|'error'{return s==='reachable'?'success':s==='http_error'||s==='tls_error'?'warning':'error'}
function publicationLabel(s:string){return ({confirmed:'Worker 已就绪',failed:'发布失败',saved:'仅保存',reload_requested:'已请求重载',applying:'正在发布',unknown:'未知'} as Record<string,string>)[s]||s}
function formatTime(s:string){const d=new Date(s);return Number.isNaN(d.getTime())?s:d.toLocaleString()}
function stopTimer(){if(timer!==undefined)clearTimeout(timer);timer=undefined}
function schedule(){stopTimer();if(auto.value&&alive&&!document.hidden)timer=setTimeout(()=>void run(),60000)}
async function load(){controller?.abort();const c=new AbortController();controller=c;const g=++generation;loading.value=true;error.value='';try{const rows=await api.getProxyDiagnostics(props.rule.id,c.signal);if(g===generation)history.value=Array.isArray(rows)?rows:[]}catch(e){if(g===generation&&!c.signal.aborted)error.value=e instanceof Error?e.message:'读取失败'}finally{if(g===generation)loading.value=false}}
async function run(){if(!alive||loading.value||running.value||document.hidden)return;stopTimer();const c=new AbortController();controller=c;const g=++generation;running.value=true;error.value='';try{const row=await api.runProxyDiagnostic(props.rule.id,c.signal);if(g===generation)history.value=[row,...history.value.filter(x=>x.id!==row.id)].slice(0,5)}catch(e){if(g===generation&&!c.signal.aborted)error.value=e instanceof Error?e.message:'诊断失败'}finally{if(g===generation){running.value=false;schedule()}}}
function setAuto(v:boolean){auto.value=v;if(v)schedule();else stopTimer()}
function visibility(){if(document.hidden)stopTimer();else schedule()}
async function loadPublication(){if(publicationLoading.value)return;publicationLoading.value=true;publicationError.value='';try{const row=await api.getNginxApplyStatus();if(alive)publication.value=row}catch(e){if(alive)publicationError.value=e instanceof Error?e.message:'读取发布状态失败'}finally{publicationLoading.value=false}}
watch(()=>props.rule.upstream,()=>{setAuto(false);running.value=false;void load()})
onMounted(()=>{void load();void loadPublication();document.addEventListener('visibilitychange',visibility)})
onUnmounted(()=>{alive=false;++generation;controller?.abort();stopTimer();document.removeEventListener('visibilitychange',visibility)})
</script>
<style scoped>
.diagnostics-panel{display:grid;gap:12px;padding:14px;min-width:0}.diag-section{display:grid;gap:12px;padding:14px;border:1px solid var(--fonu-border);border-radius:10px;background:var(--fonu-surface);min-width:0}.diag-heading,.diag-auto,.diag-record>div{display:flex;align-items:center;gap:8px;justify-content:space-between;flex-wrap:wrap}.diag-heading>div,.diag-auto>div{flex:1;min-width:0}.diag-heading h3{font-size:15px;margin:0}.diag-section small{display:block;font-size:12px;color:var(--fonu-text-secondary);overflow-wrap:anywhere}.diag-auto{background:var(--fonu-bg-muted);border-radius:8px;padding:10px}.diag-auto strong{font-size:13px}.diag-auto :deep(.n-switch){flex-shrink:0}.diag-latest,.diag-record{display:grid;gap:6px;min-width:0}.diag-latest{padding:12px;border:1px solid var(--fonu-border);border-radius:8px}.diag-latest>strong{font-size:22px;font-variant-numeric:tabular-nums}.diag-record{border-top:1px solid var(--fonu-border);padding-top:10px}.diag-record strong{font-size:12px}.diag-section p{margin:0;font-size:12px;color:var(--fonu-text-secondary);overflow-wrap:anywhere}.diag-empty{font-size:13px;color:var(--fonu-text-muted);padding:12px 0}
</style>