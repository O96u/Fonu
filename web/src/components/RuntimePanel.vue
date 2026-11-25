<script setup lang="ts">
import { computed, onMounted, onUnmounted, onActivated, onDeactivated, ref } from 'vue'
import { api } from '../api/client'
import FonuCard from './FonuCard.vue'
type Snapshot = Awaited<ReturnType<typeof api.getRuntime>>
const sample = ref<Snapshot|null>(null)
const cpu = ref<number|null>(null)
const failed = ref(false)
let timer: ReturnType<typeof setTimeout>|undefined
let active=false
let generation=0
function stop(){active=false;generation++;clearTimeout(timer)}
async function poll(){
 const g=generation
 if(!active||document.hidden)return
 try{
  const next=await api.getRuntime()
  if(!active||g!==generation)return
  const prev=sample.value
  if(prev?.host_cpu_total!=null&&prev.host_cpu_idle!=null&&next.host_cpu_total!=null&&next.host_cpu_idle!=null){
   const dt=next.host_cpu_total-prev.host_cpu_total,di=next.host_cpu_idle-prev.host_cpu_idle
   cpu.value=dt>0&&di>=0&&di<=dt?100*(dt-di)/dt:null
  }
  sample.value=next;failed.value=false
 }catch{if(active&&g===generation)failed.value=true}
 finally{if(active&&g===generation)timer=setTimeout(poll,15000)}
}
function start(){if(active)return;active=true;void poll()}
function visibility(){stop();if(!document.hidden)start()}
onMounted(()=>{document.addEventListener('visibilitychange',visibility);start()})
onActivated(start)
onDeactivated(stop)
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
 <div class="runtime-host"><span>NAS 系统资源</span><small>15 秒采样</small></div>
 <div class="runtime-pair"><span>CPU</span><strong>{{percent(cpu)}}</strong></div><div class="runtime-track"><i :style="{width:(cpu??0)+'%'}"/></div>
 <div class="runtime-pair"><span>内存</span><strong>{{percent(hostMemory)}}</strong></div><div class="runtime-track"><i :style="{width:(hostMemory??0)+'%'}"/></div>
 <small class="runtime-note">系统资源不等于 Fonu 进程占用</small>
</FonuCard>
</template>
<style scoped>
.runtime-head,.runtime-pair,.runtime-host{display:flex;justify-content:space-between;gap:8px;align-items:center;font-size:12px;margin-bottom:7px}.runtime-head span{display:flex;align-items:center;gap:6px}.runtime-head i{width:7px;height:7px;border-radius:50%;background:var(--fonu-brand)}.runtime-head small,.runtime-host small,.runtime-note{color:var(--fonu-text-secondary);font-size:11px}.runtime-pair span{color:var(--fonu-text-secondary)}.runtime-pair strong{font-weight:600;font-variant-numeric:tabular-nums}.runtime-host{border-top:1px solid var(--fonu-border);padding-top:9px;margin-top:10px;font-weight:600}.runtime-track{height:5px;background:var(--fonu-bg-muted);border-radius:5px;overflow:hidden;margin-bottom:8px}.runtime-track i{display:block;height:100%;background:var(--fonu-brand);border-radius:5px}
</style>
