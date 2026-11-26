<template>
  <section class="diagnostics"><header><div><strong>运行诊断</strong><p>检查已保存的连接配置，不会修改隧道。</p></div><n-button size="small" :loading="loading" @click="run">开始检测</n-button></header>
  <div v-if="error" class="error">{{ error }}</div>
  <div v-for="item in checks" :key="item.name" class="check"><span>{{ item.name }}</span><span class="detail">{{ item.detail }}</span><StatusBadge :kind="item.state === 'success' ? 'success' : item.state === 'error' ? 'error' : 'unknown'" :text="item.state === 'success' ? '正常' : item.state === 'error' ? '异常' : '待确认'" /></div>
  <p v-if="!checks.length && !loading && !error">点击检测，查看客户端、DNS、控制端口、登录和网关状态。</p></section>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { NButton } from 'naive-ui'
import StatusBadge from '../StatusBadge.vue'
import { api } from '../../api/client'
const checks=ref<Awaited<ReturnType<typeof api.diagnoseFRP>>>([])
const loading=ref(false), error=ref('')
async function run(){if(loading.value)return;loading.value=true;error.value='';try{checks.value=await api.diagnoseFRP()}catch(e){error.value=e instanceof Error?e.message:'检测失败'}finally{loading.value=false}}
</script>
<style scoped>
.diagnostics{border:1px solid var(--fonu-border);border-radius:var(--fonu-radius);padding:12px;display:grid;gap:8px;margin:12px 0}.diagnostics header{display:flex;align-items:center;justify-content:space-between;gap:12px}.diagnostics p{margin:4px 0;color:var(--fonu-text-secondary);font-size:12px}.check{display:grid;grid-template-columns:minmax(90px,1fr) minmax(0,2fr) auto;gap:8px;align-items:center;border-top:1px solid var(--fonu-border);padding:8px 0}.detail{font-size:12px;color:var(--fonu-text-secondary);overflow-wrap:anywhere}.error{color:var(--fonu-error)}@media(max-width:550px){.check{grid-template-columns:minmax(0,1fr) auto}.detail{grid-row:2;grid-column:1/-1}}
</style>
