<template>
  <div class="proxy-stats">
    <div class="metric"><span class="metric-label">代理规则</span><strong>{{ totalCount }}</strong><small><span class="status-dot"/>{{ enabledCount }} 已启用 · {{ disabledCount }} 已停用</small></div>
    <div class="metric"><span class="metric-label">实时流量</span><strong>↓ {{ formatRate(trafficTotals.downloadRate) }}</strong><small>↑ {{ formatRate(trafficTotals.uploadRate) }}</small></div>
    <div class="metric"><span class="metric-label">累计流量</span><strong>↓ {{ formatBytes(trafficTotals.download) }}</strong><small>↑ {{ formatBytes(trafficTotals.upload) }}</small></div>
    <div class="metric"><span class="metric-label">近期访问 IP</span><strong>{{ trafficTotals.connections }}</strong><small>各规则 IP 数相加，非去重</small></div>
  </div>
</template>
<script setup lang="ts">
import {formatBytes,formatRate} from '../utils/format'
defineProps<{totalCount:number;enabledCount:number;disabledCount:number;trafficTotals:{upload:number;download:number;uploadRate:number;downloadRate:number;connections:number}}>()
</script>
<style scoped>
.proxy-stats{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:1px;background:var(--fonu-border);border:1px solid var(--fonu-border);border-radius:var(--fonu-radius);overflow:hidden;margin-bottom:12px}
.metric{display:flex;flex-direction:column;gap:5px;min-width:0;padding:12px 14px;background:var(--fonu-surface)}
.metric-label{font-size:12px;color:var(--fonu-text-secondary)}
.metric strong{font-size:22px;line-height:1.25;font-weight:650;color:var(--fonu-text);font-variant-numeric:tabular-nums;overflow-wrap:anywhere}
.metric small{color:var(--fonu-text-secondary);font-size:11px;overflow-wrap:anywhere}
.status-dot{display:inline-block;width:6px;height:6px;border-radius:50%;background:var(--fonu-success);margin-right:5px;vertical-align:middle}
@media(max-width:760px){.proxy-stats{grid-template-columns:repeat(2,minmax(0,1fr))}.metric{padding:10px 12px}.metric strong{font-size:18px}}
</style>