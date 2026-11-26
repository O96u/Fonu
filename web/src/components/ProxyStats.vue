<template>
  <div class="proxy-stats">
    <div class="metric"><div class="metric-label"><n-icon :component="LayersOutline"/>代理规则</div><strong>{{ totalCount }}</strong><small><span class="status-dot"/> {{ enabledCount }} 已启用 · {{ disabledCount }} 已停用</small></div>
    <div class="metric"><div class="metric-label"><n-icon :component="PeopleOutline"/>近期访问 IP</div><strong>{{ trafficTotals.connections }}</strong><small title="各规则近期访问 IP 数相加，未跨规则去重">各规则合计</small></div>
    <div class="metric"><div class="metric-label"><n-icon :component="SwapVerticalOutline"/>实时流量</div><strong class="metric-traffic"><span>↑ {{ formatRate(trafficTotals.uploadRate) }}</span><span>↓ {{ formatRate(trafficTotals.downloadRate) }}</span></strong></div>
    <div class="metric"><div class="metric-label"><n-icon :component="ServerOutline"/>累计流量</div><strong>{{formatBytes(trafficTotals.upload+trafficTotals.download)}}</strong><small>↑ {{formatBytes(trafficTotals.upload)}} · ↓ {{formatBytes(trafficTotals.download)}}</small></div>
  </div>
</template>
<script setup lang="ts">
import {NIcon} from 'naive-ui'
import {LayersOutline,PeopleOutline,SwapVerticalOutline,ServerOutline} from '@vicons/ionicons5'
import {formatBytes,formatRate} from '../utils/format'
defineProps<{totalCount:number;enabledCount:number;disabledCount:number;trafficTotals:{upload:number;download:number;uploadRate:number;downloadRate:number;connections:number}}>()
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
</style>