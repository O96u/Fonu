<script setup lang="ts">
import {onMounted,onUnmounted,ref} from 'vue'
import {NButton,NTag} from 'naive-ui'
import FonuCard from './FonuCard.vue'
import {api} from '../api/client'
const rows=ref<Awaited<ReturnType<typeof api.getNotifyHistory>>>([])
const busy=ref(false),error=ref('')
let controller:AbortController|undefined
let timer:ReturnType<typeof setTimeout>|undefined
let active=true
const labels:Record<string,string>={ddns_ip_change:'DDNS IP 变更',ddns_failure:'DDNS 更新失败',cert_expiry:'证书到期',cert_renew_success:'证书续签成功',cert_renew_failure:'证书续签失败',ip_frequent_access:'IP 频繁访问',suspected_scan:'疑似扫描',suspected_cc:'疑似 CC',login_failure:'登录异常',nginx_reload_failure:'Nginx 重载失败',test:'测试通知'}
async function refresh(){
 if(!active||busy.value)return
 clearTimeout(timer);busy.value=true
 const ctl=new AbortController();controller=ctl
 try{const next=await api.getNotifyHistory(ctl.signal);if(active&&!ctl.signal.aborted){rows.value=next.slice(0,10);error.value=''}}
 catch{if(active&&!ctl.signal.aborted)error.value='读取发送记录失败'}
 finally{busy.value=false;if(active)timer=setTimeout(refresh,15000)}
}
onMounted(refresh)
onUnmounted(()=>{active=false;controller?.abort();clearTimeout(timer)})
</script>
<template><FonuCard title="通知发送记录" subtitle="最近 10 条 · 已保存配置触发的事件与测试通知">
<template #header><n-button size="small" :loading="busy" @click="refresh">刷新</n-button></template>
<p v-if="error" class="history-muted">{{error}}</p><p v-else-if="!rows.length" class="history-muted">暂无触发记录。勾选并保存事件后，发生对应事件才会发送通知。</p>
<div v-for="(row,i) in rows" :key="row.at+'-'+i" class="history-row"><div><strong>{{labels[row.event]||row.event}}</strong><small>{{row.detail}} · {{row.channel||'未选择通道'}}</small></div><div class="history-right"><n-tag size="small" :type="row.state==='sent'?'success':row.state==='failed'?'error':'default'">{{row.state==='sent'?'已发送':row.state==='failed'?'失败':'跳过'}}</n-tag><small>{{new Date(row.at).toLocaleString()}}</small></div></div>
<p class="history-muted">当前进程保留最近 100 条，重启后可在系统日志查询历史结果。通道接收不代表终端用户已读。</p>
</FonuCard></template>
<style scoped>.history-row{display:flex;justify-content:space-between;gap:8px;padding:8px 0;border-bottom:1px solid var(--fonu-border);font-size:12px}.history-row div{min-width:0}.history-row small{display:block;color:var(--fonu-text-secondary);overflow-wrap:anywhere;margin-top:3px}.history-right{text-align:right;flex-shrink:0}.history-muted{font-size:12px;color:var(--fonu-text-secondary)}</style>
