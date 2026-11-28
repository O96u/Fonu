<script setup lang="ts">
import {computed,onMounted,onUnmounted,ref,watch} from 'vue'
import {NButton,NTag,NPagination,NSelect} from 'naive-ui'
import FonuCard from './FonuCard.vue'
import {api} from '../api/client'
const rows=ref<Awaited<ReturnType<typeof api.getNotifyHistory>>>([])
const busy=ref(false),error=ref('')
const filter=ref('all'),page=ref(1),pageSize=ref(10)
const filterOptions=[{label:'全部状态',value:'all'},{label:'通道已接收',value:'sent'},{label:'发送失败',value:'failed'},{label:'未发送',value:'skipped'}]
const filtered=computed(()=>rows.value.filter(r=>filter.value==='all'||r.state===filter.value))
const pageCount=computed(()=>Math.max(1,Math.ceil(filtered.value.length/pageSize.value)))
const visible=computed(()=>filtered.value.slice((page.value-1)*pageSize.value,page.value*pageSize.value))
watch(filter,()=>{page.value=1})
watch(pageSize,()=>{page.value=1})
watch(pageCount,n=>{page.value=Math.min(page.value,n)})
let controller:AbortController|undefined
let timer:ReturnType<typeof setTimeout>|undefined
let active=true
const labels:Record<string,string>={ddns_ip_change:'DDNS · 公网 IP 已同步',ddns_failure:'DDNS · 更新失败',cert_expiry:'证书 · 到期提醒',cert_renew_success:'证书 · 续签成功',cert_renew_failure:'证书 · 续签失败',ip_frequent_access:'安全 · IP 频繁访问',suspected_scan:'安全 · 疑似恶意扫描',suspected_cc:'安全 · 疑似 CC / 高频访问',login_failure:'安全 · 登录异常',nginx_reload_failure:'系统 · Nginx 重载失败',test:'系统 · 测试通知'}
const channels:Record<string,string>={email:'邮件（SMTP）',telegram:'Telegram',bark:'Bark',ntfy:'ntfy',gotify:'Gotify',wecom:'企业微信群机器人',wecom_app:'企业微信应用',dingtalk:'钉钉',feishu:'飞书群机器人',feishu_app:'飞书应用',custom:'自定义 Webhook',webhook:'Webhook'}
async function refresh(){
 if(!active||busy.value)return
 clearTimeout(timer);busy.value=true
 const ctl=new AbortController();controller=ctl
 try{const next=await api.getNotifyHistory(ctl.signal);if(active&&!ctl.signal.aborted){rows.value=Array.isArray(next)?next:[];error.value=''}}
 catch{if(active&&!ctl.signal.aborted)error.value='无法读取最新通知历史，请稍后刷新。'}
 finally{busy.value=false;if(active)timer=setTimeout(refresh,15000)}
}
onMounted(refresh)
onUnmounted(()=>{active=false;controller?.abort();clearTimeout(timer)})
</script>
<template>
  <FonuCard title="通知发送记录" subtitle="记录事件通知与手动测试的发送结果 · 重启或覆盖升级后保留">
    <template #header><n-button size="small" :loading="busy" @click="refresh">刷新</n-button></template>
    <div class="history-tools"><small>已保留 {{rows.length}} 条记录</small><n-select v-model:value="filter" :options="filterOptions" size="small" aria-label="筛选通知状态"/></div>
    <p v-if="error" class="history-muted">{{error}}</p>
    <p v-else-if="!rows.length && !busy" class="history-muted">暂无发送记录。启用并保存通知事件后，或手动发送测试通知后，记录将在此显示。升级前仅存在内存中的记录不会自动恢复，可尝试在系统日志中查找。</p>
    <p v-else-if="!filtered.length && !busy" class="history-muted">没有符合当前状态的记录。</p>
    <div v-for="(row,i) in visible" :key="row.at+'-'+i" class="history-row">
      <div><strong>{{labels[row.event]||row.event}}</strong><small>{{channels[row.channel]||row.channel||'未选择通道'}} · {{row.detail}}</small></div>
      <div class="history-right"><n-tag size="small" :type="row.state==='sent'?'success':row.state==='failed'?'error':'default'">{{row.state==='sent'?'通道已接收':row.state==='failed'?'发送失败':'未发送'}}</n-tag><small>{{new Date(row.at).toLocaleString()}}</small></div>
    </div>
    <div v-if="filtered.length" class="history-pagination"><n-pagination v-model:page="page" v-model:page-size="pageSize" :page-count="pageCount" :page-sizes="[10,20,50]" show-size-picker simple/></div>
    <p class="history-muted">数据库保留最近 1,000 条发送结果，不保存消息正文或通道密钥。通道确认接收不代表终端用户已读。</p>
  </FonuCard>
</template>
<style scoped>
.history-tools,.history-pagination{display:flex;align-items:center;justify-content:space-between;gap:8px;flex-wrap:wrap}.history-tools small{font-size:12px;color:var(--fonu-text-secondary)}.history-tools :deep(.n-select){width:140px}.history-pagination{justify-content:flex-end;padding-top:10px}.history-row{display:flex;justify-content:space-between;gap:8px;padding:10px 0;border-bottom:1px solid var(--fonu-border);font-size:12px}.history-row div{min-width:0}.history-row small{display:block;color:var(--fonu-text-secondary);overflow-wrap:anywhere;margin-top:4px}.history-right{text-align:right;flex-shrink:0}.history-muted{font-size:12px;color:var(--fonu-text-secondary);line-height:1.6}
@media(max-width:480px){.history-row{flex-wrap:wrap}.history-right{text-align:left;display:flex;align-items:center;gap:8px}}
</style>
