<template>
  <n-alert v-if="status && status.state !== 'unknown'" :type="kind" :title="title" style="margin-bottom: 16px">
    <div v-if="status.last_error">{{ status.last_error }}</div>
    <div v-if="status.desired_revision">目标版本：{{ status.desired_revision.slice(0, 12) }}</div>
    <div v-if="status.confirmed_revision">已确认 worker 版本：{{ status.confirmed_revision.slice(0, 12) }}</div>
    <div v-else-if="status.last_requested_revision">最近提交重载版本：{{ status.last_requested_revision.slice(0, 12) }}</div>
    <details v-if="status.history?.length" class="apply-history">
      <summary>最近发布记录（{{ status.history.length }}）</summary>
      <div v-for="(item,index) in status.history" :key="item.created_at+'-'+index" class="apply-history__row">
        <span>{{ historyLabel(item.state) }}</span>
        <span>{{ item.revision?.slice(0,12) || '—' }}</span>
        <span>{{ formatDate(item.created_at) }}</span>
      </div>
    </details>
  </n-alert>
</template>

<script setup lang="ts">
import { computed, onActivated, onDeactivated, onMounted, onUnmounted, ref } from 'vue'
import { NAlert } from 'naive-ui'
import { api } from '../api/client'
import type { NginxApplyStatus } from '../api/types'

const status = ref<NginxApplyStatus | null>(null)
const title = computed(() => ({
  unknown: '配置状态未知',
  applying: '正在应用 Nginx 配置',
  failed: 'Nginx 配置应用失败',
  saved: '配置已保存，未重载',
  reload_requested: '已请求 Nginx 重载',
  confirmed: 'Nginx 新 worker 已就绪',
}[status.value?.state ?? 'unknown']))
const kind = computed(() => status.value?.state === 'failed' ? 'error' : status.value?.state === 'saved' ? 'warning' : status.value?.state === 'confirmed' ? 'success' : 'info')
function historyLabel(state:string) {
  return ({confirmed:'Worker 已就绪',failed:'发布失败',saved:'仅保存',reload_requested:'已请求重载'} as Record<string,string>)[state] || state
}
function formatDate(value:string) {
  const d=new Date(value)
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString()
}
let timer: ReturnType<typeof setInterval> | undefined
let pending = false
async function refresh() {
  if (pending || document.hidden) return
  pending = true
  try { status.value = await api.getNginxApplyStatus() } catch { status.value = null } finally { pending = false }
}
function start() {
  if (timer !== undefined) return
  void refresh()
  timer = setInterval(refresh, 5000)
}
function stop() {
  if (timer !== undefined) clearInterval(timer)
  timer = undefined
}
onMounted(start)
onActivated(start)
onDeactivated(stop)
onUnmounted(stop)
</script>

<style scoped>
.apply-history{margin-top:6px}
.apply-history summary{cursor:pointer}
.apply-history__row{display:flex;gap:12px;flex-wrap:wrap;font-size:12px;margin:6px 0}
</style>
