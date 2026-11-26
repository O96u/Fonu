<script setup lang="ts">
import {computed,onMounted,onUnmounted,ref} from 'vue'
import {NButton,NTag,NSpace,useDialog,useMessage} from 'naive-ui'
import FonuCard from './FonuCard.vue'
import {api} from '../api/client'
import {formatDate} from '../utils/format'
const props=defineProps<{disabled?:boolean}>()
const emit=defineEmits<{restore:[name:string]}>()
const rows=ref<Awaited<ReturnType<typeof api.listLocalBackups>>>([])
const busy=ref(false),error=ref(''),page=ref(1)
const visible=computed(()=>rows.value.slice((page.value-1)*5,page.value*5))
let alive=true,ctl:AbortController|undefined
const dialog=useDialog(),message=useMessage()
function size(n:number){return n>=1048576?(n/1048576).toFixed(1)+' MiB':(n/1024).toFixed(1)+' KiB'}
async function refresh(){
 ctl?.abort();const c=new AbortController();ctl=c
 try{const data=await api.listLocalBackups(c.signal);if(alive&&!c.signal.aborted){rows.value=data;page.value=Math.min(page.value,Math.max(1,Math.ceil(data.length/5)));error.value=''}}
 catch(e){if(alive&&!c.signal.aborted)error.value=e instanceof Error?e.message:'读取备份失败'}
}
async function create(){if(busy.value||props.disabled)return;busy.value=true;try{await api.createLocalBackup();page.value=1;await refresh();message.success('本地备份已创建')}catch(e){message.error(e instanceof Error?e.message:'创建失败')}finally{busy.value=false}}
function remove(name:string){if(busy.value||props.disabled)return;dialog.warning({title:'删除本地备份',content:'删除 '+name+' 后无法找回，是否继续？',positiveText:'删除',negativeText:'取消',onPositiveClick:async()=>{busy.value=true;try{await api.deleteLocalBackup(name);await refresh();message.success('备份已删除')}catch(e){message.error(e instanceof Error?e.message:'删除失败')}finally{busy.value=false}}})}
function restore(name:string){if(busy.value||props.disabled)return;dialog.warning({title:'恢复本地备份',content:'将校验并在重启时覆盖当前数据库、证书、密钥与 Nginx 配置，完成后需要重新登录。建议先创建当前备份。',positiveText:'确认恢复',negativeText:'取消',onPositiveClick:()=>emit('restore',name)})}
onMounted(refresh)
onUnmounted(()=>{alive=false;ctl?.abort()})
</script>
<template><FonuCard title="本地备份" subtitle="保存在 Fonu 数据目录 · 自动备份在常规设置中启用">
<template #header><n-space :size="6"><n-button size="small" :disabled="busy||disabled" @click="refresh">刷新</n-button><n-button type="primary" size="small" :loading="busy" :disabled="disabled" @click="create">创建备份</n-button></n-space></template>
<p v-if="error" class="backup-muted">{{error}}</p><p v-else-if="!rows.length" class="backup-muted">暂无本地备份，可手动创建或启用自动备份。</p>
<div v-for="row in visible" :key="row.name" class="local-row"><div class="local-info"><strong>{{formatDate(row.created_at)}}</strong><n-tag size="small" :bordered="false" :type="row.kind==='auto'?'success':'default'">{{row.kind==='auto'?'自动':'手动'}}</n-tag><small>{{row.name}} · {{size(row.size)}}</small></div><n-space :size="4"><n-button size="tiny" tag="a" :href="'/api/backup/local/'+encodeURIComponent(row.name)" download :disabled="busy||disabled">下载</n-button><n-button size="tiny" :disabled="busy||disabled" @click="restore(row.name)">恢复</n-button><n-button size="tiny" type="error" quaternary :disabled="busy||disabled" @click="remove(row.name)">删除</n-button></n-space></div>
<div v-if="rows.length>5" class="local-pages"><small>共 {{rows.length}} 份</small><n-button size="tiny" :disabled="page===1" @click="page--">上一页</n-button><small>{{page}} / {{Math.ceil(rows.length/5)}}</small><n-button size="tiny" :disabled="page*5>=rows.length" @click="page++">下一页</n-button></div>
<p class="backup-muted">手动备份最多 30 份；自动保留策略仅清理自动备份。本地备份不能替代异地备份。</p>
</FonuCard></template>
<style scoped>.local-row{display:flex;flex-wrap:wrap;justify-content:space-between;align-items:center;gap:8px;padding:10px 0;border-bottom:1px solid var(--fonu-border)}.local-info{display:flex;align-items:center;flex-wrap:wrap;gap:6px;min-width:0;font-size:13px}.local-info small{width:100%;overflow-wrap:anywhere;color:var(--fonu-text-secondary);font-size:11px}.local-pages{display:flex;align-items:center;justify-content:flex-end;gap:8px;margin-top:10px}.backup-muted{color:var(--fonu-text-secondary);font-size:12px}</style>
