<template>
  <div class="tunnel-grid">
    <article v-for="row in rows" :key="row.id" class="tunnel-card">
      <header><div class="tunnel-icon"><n-icon :component="SwapHorizontalOutline" /></div><div class="name"><strong>{{ row.name }}</strong><small>{{ (row.protocol || 'tcp').toUpperCase() }} 隧道</small></div><StatusBadge :kind="badge(row).kind" :text="badge(row).text" /><n-switch size="small" :value="row.enabled" :disabled="busy" @update:value="value => emit('toggle', row.id, value)" /></header>
      <div class="mapping"><div><small>VPS 远程端口</small><strong>{{ row.remote_port || '自动分配' }}</strong></div><span>→</span><div><small>本地服务</small><code>{{ endpoint(row.local_ip, row.local_port) }}</code></div></div>
      <footer><code>{{ row.remote_port ? endpoint(serverHost || 'VPS', row.remote_port) : '保存后分配' }}</code><div class="actions"><n-button size="tiny" quaternary @click="emit('copy', row)">复制地址</n-button><n-button size="tiny" quaternary :disabled="busy" @click="emit('edit', row)">编辑</n-button><n-button size="tiny" quaternary :disabled="busy" @click="emit('duplicate', row)">克隆</n-button><n-button size="tiny" quaternary type="error" :disabled="busy" @click="emit('remove', row)">删除</n-button></div></footer>
    </article>
  </div>
</template>
<script setup lang="ts">
import { NButton, NIcon, NSwitch } from 'naive-ui'
import { SwapHorizontalOutline } from '@vicons/ionicons5'
import type { FRPTCPProxy, FRPStatus } from '../../api/types'
import StatusBadge from '../StatusBadge.vue'
const props=defineProps<{rows:FRPTCPProxy[];busy:boolean;status:FRPStatus;serverHost:string}>()
const emit=defineEmits<{edit:[row:FRPTCPProxy];duplicate:[row:FRPTCPProxy];remove:[row:FRPTCPProxy];copy:[row:FRPTCPProxy];toggle:[id:string,enabled:boolean]}>()
function badge(row:FRPTCPProxy){
  if(!row.enabled) return {kind:'disabled' as const,text:'已停用'}
  if(!props.status.enabled) return {kind:'disabled' as const,text:'服务未启用'}
  const state=props.status.tunnel_states?.[row.id]
  if(props.status.connected && state==='registered')return {kind:'success' as const,text:'已注册'}
  if(props.status.connected && state==='failed')return {kind:'error' as const,text:'注册失败'}
  return {kind:'warning' as const,text:props.status.connected?'待确认':'待连接'}
}
function endpoint(host:string,port:number){return (host.includes(':')&&!host.startsWith('[')?'['+host+']':host)+':'+port}
</script>
<style scoped>
.tunnel-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,340px),1fr));gap:10px}
.tunnel-card{min-width:0;border:1px solid var(--fonu-border);border-radius:var(--fonu-radius);padding:12px;display:grid;gap:12px;background:var(--fonu-surface)}
header{display:flex;align-items:center;gap:8px;flex-wrap:wrap}.tunnel-icon{display:grid;place-items:center;width:36px;height:36px;background:var(--fonu-bg-muted);border-radius:var(--fonu-radius);font-size:20px}.name{flex:1;min-width:70px;display:grid;gap:3px;overflow-wrap:anywhere}small{display:block;color:var(--fonu-text-secondary);font-size:12px}
.mapping{display:grid;grid-template-columns:minmax(0,1fr) auto minmax(0,1.5fr);align-items:center;gap:8px;background:var(--fonu-bg-muted);border-radius:var(--fonu-radius);padding:10px}.mapping code,footer code{overflow-wrap:anywhere;min-width:0;font-size:12px}
footer{display:flex;align-items:center;gap:6px;flex-wrap:wrap;justify-content:space-between}.actions{display:flex;flex-wrap:wrap;gap:2px}
</style>
