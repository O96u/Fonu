<template>
<div class="stats-row">
      <div class="stat-card">
        <n-icon :component="LayersOutline" class="stat-card__icon stat-card__icon--blue" />
        <div class="stat-card__value">{{ totalCount }}</div>
        <div class="stat-card__label">服务总数</div>
        <div class="stat-card__sub">{{ enabledCount }} 运行 · {{ disabledCount }} 停止</div>
      </div>
      <div class="stat-card">
        <n-icon :component="CloudUploadOutline" class="stat-card__icon stat-card__icon--green" />
        <div class="stat-card__value stat-card__value--sm">{{ formatBytes(trafficTotals.upload) }}</div>
        <div class="stat-card__label">总上传</div>
        <div class="stat-card__sub">累计流量</div>
      </div>
      <div class="stat-card">
        <n-icon :component="CloudDownloadOutline" class="stat-card__icon stat-card__icon--blue" />
        <div class="stat-card__value stat-card__value--sm">{{ formatBytes(trafficTotals.download) }}</div>
        <div class="stat-card__label">总下载</div>
        <div class="stat-card__sub">累计流量</div>
      </div>
      <div class="stat-card">
        <n-icon :component="ArrowUpOutline" class="stat-card__icon stat-card__icon--amber" />
        <div class="stat-card__value stat-card__value--sm">{{ formatRate(trafficTotals.uploadRate) }}</div>
        <div class="stat-card__label">当前上传</div>
        <div class="stat-card__sub">实时速率</div>
      </div>
      <div class="stat-card">
        <n-icon :component="ArrowDownOutline" class="stat-card__icon stat-card__icon--green" />
        <div class="stat-card__value stat-card__value--sm">{{ formatRate(trafficTotals.downloadRate) }}</div>
        <div class="stat-card__label">当前下载</div>
        <div class="stat-card__sub">实时速率</div>
      </div>
      <div class="stat-card">
        <n-icon :component="PeopleOutline" class="stat-card__icon stat-card__icon--purple" />
        <div class="stat-card__value">{{ trafficTotals.connections }}</div>
        <div class="stat-card__label">当前连接</div>
        <div class="stat-card__sub">全部规则合计</div>
      </div>
    </div>
</template>

<script setup lang="ts">
import { NIcon } from 'naive-ui'
import { LayersOutline, CloudUploadOutline, CloudDownloadOutline, ArrowUpOutline, ArrowDownOutline, PeopleOutline } from '@vicons/ionicons5'
import { formatBytes, formatRate } from '../utils/format'
defineProps<{ totalCount: number; enabledCount: number; disabledCount: number; trafficTotals: { upload: number; download: number; uploadRate: number; downloadRate: number; connections: number } }>()
</script>

<style scoped>
.stats-row {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.stat-card {
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  padding: var(--fonu-space-4);
  min-height: 118px;
}

.stat-card__icon { font-size: 18px; margin-bottom: 4px; }
.stat-card__icon--blue { color: #3b82f6; }
.stat-card__icon--green { color: #10b981; }
.stat-card__icon--amber { color: #f59e0b; }
.stat-card__icon--purple { color: #8b5cf6; }

.stat-card__value {
  margin-top: 4px;
  font-size: 28px;
  font-weight: 700;
  line-height: 1.1;
  color: var(--fonu-text);
}

.stat-card__value--sm { font-size: 22px; }

.stat-card__label {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
}

.stat-card__sub {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}
@media (max-width: 1200px) { .stats-row { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@media (max-width: 640px) { .stats-row { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
