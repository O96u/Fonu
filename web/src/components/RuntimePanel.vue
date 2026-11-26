<script setup lang="ts">
import { computed, onMounted, onUnmounted, onActivated, onDeactivated, ref } from 'vue'
import { api } from '../api/client'
import FonuCard from './FonuCard.vue'
type Snapshot = Awaited<ReturnType<typeof api.getRuntime>>
const sample = ref<Snapshot|null>(null)
const cpu = ref<number|null>(null)
const failed = ref(false)
const processCPU=ref<Record<string,number|null>>({})
let controller:AbortController|undefined
let timer: ReturnType<typeof setTimeout>|undefined
let active=false
let generation=0
function stop(){active=false;generation++;clearTimeout(timer);controller?.abort();controller=undefined}
async function poll(){
 const g=generation
 if(!active||document.hidden)return
 try{
  const ctl=new AbortController();controller=ctl
  const next=await api.getRuntime(ctl.signal)
  if(!active||g!==generation)return
  const prev=sample.value
  if(prev?.host_cpu_total!=null&&prev.host_cpu_idle!=null&&next.host_cpu_total!=null&&next.host_cpu_idle!=null){
   const dt=next.host_cpu_total-prev.host_cpu_total,di=next.host_cpu_idle-prev.host_cpu_idle
   cpu.value=dt>0&&di>=0&&di<=dt?100*(dt-di)/dt:null
   const result:Record<string,number|null>={}
   for(const [name,group] of Object.entries(next.processes??{})){
    const prior=prev.processes?.[name]?.processes??[]
    const pairs=group.processes.map(p=>({p,old:prior.find(o=>o.pid===p.pid&&o.start===p.start)}))
    result[name]=dt>0&&group.running&&pairs.length===prior.length&&pairs.every(v=>v.old&&v.p.ticks>=v.old.ticks)?100*pairs.reduce((sum,v)=>sum+v.p.ticks-(v.old?.ticks??0),0)/dt:null
   }
   processCPU.value=result
  }
  sample.value=next;failed.value=false
 }catch{if(active&&g===generation)failed.value=true}
 finally{if(active&&g===generation)timer=setTimeout(poll,15000)}
}
function start(){if(active)return;active=true;void poll()}
let mountedActive=true
function visibility(){if(!mountedActive)return;stop();if(!document.hidden)start()}
onMounted(()=>{document.addEventListener('visibilitychange',visibility);start()})
onActivated(()=>{mountedActive=true;start()})
onDeactivated(()=>{mountedActive=false;stop()})
onUnmounted(()=>{stop();document.removeEventListener('visibilitychange',visibility)})
const hostMemory=computed(()=>{const s=sample.value;return s?.host_memory_total&&s.host_memory_available!=null?Math.max(0,Math.min(100,100*(1-s.host_memory_available/s.host_memory_total))):null})
function percent(n:number|null){return n==null?'—':n.toFixed(1)+'%'}
function mb(n?:number){return n==null?'—':(n/1048576).toFixed(1)+' MiB'}
const uptime=computed(()=>{const n=sample.value?.uptime_seconds;if(n==null)return '—';return Math.floor(n/86400)+' 天 '+Math.floor(n%86400/3600)+' 小时 '+Math.floor(n%3600/60)+' 分'})
</script>
<template>
<FonuCard title="运行状态">
 <div class="runtime-head"><span><i/>Fonu 进程</span><small>{{ failed ? '读取失败' : sample ? '运行中' : '读取中' }}</small></div>
 <div class="runtime-pair"><span>运行时间</span><strong>{{uptime}}</strong></div>
 <div class="runtime-pair"><span>进程内存 RSS</span><strong>{{mb(sample?.rss_bytes)}}</strong></div>
 <div class="runtime-pair"><span>Go 堆内存</span><strong>{{mb(sample?.heap_bytes)}}</strong></div>
 <div class="runtime-pair"><span>Fonu / Go 版本</span><strong>{{sample?.versions?.fonu || "—"}} / {{sample?.versions?.go || "—"}}</strong></div>
 <div class="runtime-pair"><span>Fonu CPU · 整机占比</span><strong>{{percent(processCPU.fonu??null)}}</strong></div>
 <div class="runtime-pair"><span>Goroutine / FD</span><strong>{{sample?.goroutines??"—"}} / {{sample?.processes?.fonu?.processes[0]?.fd_count??"—"}}</strong></div>
 <template v-for="name in ['nginx','frpc']" :key="name">
 <div class="runtime-host"><span>{{name==='nginx'?'Nginx':'FRP'}}</span><small>{{sample?.processes?.[name]?.running?'运行中':sample?'未检测到':'读取中'}}</small></div>
 <div class="runtime-pair"><span>版本</span><strong>{{sample?.versions?.[name]||"—"}}</strong></div>
 <div class="runtime-pair"><span>CPU / RSS 合计</span><strong>{{percent(processCPU[name]??null)}} / {{sample?.processes?.[name]?.running?mb(sample?.processes?.[name]?.rss_bytes):"—"}}</strong></div>
 <div class="runtime-pair"><span>进程数</span><strong>{{sample?.processes?.[name]?.processes.length??"—"}}</strong></div>
 </template>
 <details class="runtime-details"><summary>Nginx 发布与错误日志</summary><div class="runtime-pair"><span>配置状态</span><strong>{{sample?.nginx_apply?.state||"—"}}</strong></div><small>{{sample?.nginx_apply?.updated_at||"暂无发布记录"}}</small><p v-if="sample?.nginx_apply?.last_error" class="runtime-log">{{sample.nginx_apply.last_error}}</p><p v-for="(line,i) in sample?.nginx_errors??[]" :key="i" class="runtime-log">{{line}}</p><router-link :to="{name:'logs',query:{tab:'error'}}">查看完整日志 →</router-link></details>
 <div class="runtime-host"><span>NAS 系统资源</span><small>15 秒采样</small></div>
 <div class="runtime-pair"><span>CPU</span><strong>{{percent(cpu)}}</strong></div><div class="runtime-track"><i :style="{width:(cpu??0)+'%'}"/></div>
 <div class="runtime-pair"><span>内存</span><strong>{{percent(hostMemory)}}</strong></div><div class="runtime-track"><i :style="{width:(hostMemory??0)+'%'}"/></div>
 <small class="runtime-note">CPU 为整机占比；Nginx RSS 合计可能重复计入共享内存。</small>
</FonuCard>
</template>
<style scoped>
.runtime-head,.runtime-pair,.runtime-host{display:flex;justify-content:space-between;gap:8px;align-items:center;font-size:12px;margin-bottom:7px}.runtime-head span{display:flex;align-items:center;gap:6px}.runtime-head i{width:7px;height:7px;border-radius:50%;background:var(--fonu-brand)}.runtime-head small,.runtime-host small,.runtime-note{color:var(--fonu-text-secondary);font-size:11px}.runtime-pair span{color:var(--fonu-text-secondary)}.runtime-pair strong{font-weight:600;font-variant-numeric:tabular-nums}.runtime-host{border-top:1px solid var(--fonu-border);padding-top:9px;margin-top:10px;font-weight:600}.runtime-track{height:5px;background:var(--fonu-bg-muted);border-radius:5px;overflow:hidden;margin-bottom:8px}.runtime-track i{display:block;height:100%;background:var(--fonu-brand);border-radius:5px}
.runtime-pair strong{min-width:0;overflow-wrap:anywhere;text-align:right}.runtime-details{font-size:12px;border-top:1px solid var(--fonu-border);padding-top:8px}.runtime-details summary{cursor:pointer;margin-bottom:8px}.runtime-log{white-space:pre-wrap;overflow-wrap:anywhere;font-size:11px;max-height:90px;overflow:auto;color:var(--fonu-text-secondary)}
</style>
