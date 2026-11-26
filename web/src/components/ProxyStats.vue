<template>
  <div class="proxy-stats">
    <div class="metric"><div class="metric-label"><n-icon :component="LayersOutline"/>代理规则</div><strong>{{ totalCount }}</strong><small><span class="status-dot"/> {{ enabledCount }} 已启用 · {{ disabledCount }} 已停用</small></div>
    <div class="metric"><div class="metric-label"><n-icon :component="PeopleOutline"/>近期访问 IP</div><strong>{{ sampledAt?trafficTotals.connections:'—' }}</strong><small title="65 秒内已记录访问的 IP，跨规则去重">65 秒 · 全局去重</small></div>
    <div class="metric"><div class="metric-label"><n-icon :component="SwapVerticalOutline"/>实时流量</div><strong class="metric-traffic"><span>↑ {{ sampledAt?formatRate(trafficTotals.uploadRate):"—" }}</span><span>↓ {{ sampledAt?formatRate(trafficTotals.downloadRate):"—" }}</span></strong></div>
    <div class="metric"><div class="metric-label"><n-icon :component="ServerOutline"/>累计流量</div><strong>{{sampledAt?formatBytes(trafficTotals.upload+trafficTotals.download):"—"}}</strong><small>↑ {{sampledAt?formatBytes(trafficTotals.upload):"—"}} · ↓ {{sampledAt?formatBytes(trafficTotals.download):"—"}}</small></div>
  </div>
  <div class="traffic-freshness" role="status">{{error?'采样失败 · '+(sampledAt?'保留上次数据':'尚无数据')+' · '+error:sampledAt?'最近采样 '+new Date(sampledAt).toLocaleTimeString()+' · 5 秒日志统计速率':'正在获取流量数据…'}}</div>
</template>
<script setup lang="ts">
import {NIcon} from 'naive-ui'
import {LayersOutline,PeopleOutline,SwapVerticalOutline,ServerOutline} from '@vicons/ionicons5'
import {formatBytes,formatRate} from '../utils/format'
defineProps<{sampledAt:string;error:string;totalCount:number;enabledCount:number;disabledCount:number;trafficTotals:{upload:number;download:number;uploadRate:number;downloadRate:number;connections:number}}>()
</script>
<style scoped>
.proxy-stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,180px),1fr));gap:10px;margin-bottom:18px}
.metric{display:flex;flex-direction:column;gap:6px;min-width:0;padding:12px 14px;background:var(--fonu-surface);border:1px solid var(--fonu-border);border-radius:var(--fonu-radius)}
.metric-label{display:flex;align-items:center;gap:7px;font-size:12px;color:var(--fonu-text-secondary)}.metric-label :deep(.n-icon){font-size:17px;color:var(--fonu-brand-text)}
.metric strong{font-size:24px;line-height:1.2;font-weight:650;color:var(--fonu-text);font-variant-numeric:tabular-nums;overflow-wrap:anywhere}
.metric strong.metric-traffic{display:grid;gap:4px;font-size:16px}.metric-traffic span:first-child{color:var(--fonu-success)}
.metric small{color:var(--fonu-text-secondary);font-size:11px;overflow-wrap:anywhere}
.status-dot{display:inline-block;width:6px;height:6px;border-radius:50%;background:var(--fonu-success);vertical-align:middle}
@media(max-width:760px){.metric{padding:10px 12px}.metric strong{font-size:20px}}
.traffic-freshness{font-size:11px;color:var(--fonu-text-secondary);margin:-10px 0 12px;overflow-wrap:anywhere}
</style>