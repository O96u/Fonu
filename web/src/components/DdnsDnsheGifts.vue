<template>
  <FonuCard flush class="detail-section">
    <div class="records-head">
      <h4 class="detail-section__title">转赠记录（{{ gifts.length }}）</h4>
      <n-button size="small" quaternary :loading="loading" @click="loadGifts">
        <template #icon><n-icon :component="RefreshOutline" /></template>
        刷新
      </n-button>
    </div>
    <div v-if="loading" class="loading-wrap"><n-spin size="small" /></div>
    <div v-else-if="gifts.length === 0" class="detail-empty-wrap">
      <EmptyState title="暂无转赠记录" description="转赠或接收域名后会显示在这里。" />
    </div>
    <div v-else class="record-table-wrap">
      <table class="record-table">
        <thead>
          <tr>
            <th>转赠码</th>
            <th>域名</th>
            <th>状态</th>
            <th>创建时间</th>
            <th>有效期至</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in gifts" :key="row.id">
            <td>
              <button type="button" class="record-table__link" :title="row.code" @click="copyText(row.code)">
                {{ row.code }}
              </button>
            </td>
            <td class="record-table__mono">{{ row.full_domain }}</td>
            <td>
              <StatusBadge :kind="giftStatusKind(row.status)" :text="giftStatusLabel(row.status)" />
            </td>
            <td>{{ row.created_at || '-' }}</td>
            <td>{{ row.expires_at || '-' }}</td>
            <td>
              <div v-if="row.status === 'pending'" class="row-actions">
                <n-button
                  size="tiny"
                  quaternary
                  type="error"
                  :loading="cancellingId === row.id"
                  @click="cancelGift(row)"
                >
                  取消
                </n-button>
              </div>
              <span v-else class="record-table__muted">-</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </FonuCard>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { NButton, NIcon, NSpin, useMessage } from 'naive-ui'
import { RefreshOutline } from '@vicons/ionicons5'
import { api } from '../api/client'
import type { DNSHEAccount, DNSHEGift } from '../api/types'
import EmptyState from './EmptyState.vue'
import FonuCard from './FonuCard.vue'
import StatusBadge from './StatusBadge.vue'
import type { StatusKind } from '../utils/status'

const props = defineProps<{
  account: DNSHEAccount
}>()

const message = useMessage()
const gifts = ref<DNSHEGift[]>([])
const loading = ref(false)
const cancellingId = ref<number | null>(null)

function giftStatusKind(status: string): StatusKind {
  const s = (status || '').toLowerCase()
  if (s === 'accepted') return 'success'
  if (s === 'pending') return 'warning'
  if (s === 'expired') return 'error'
  return 'unknown'
}

function giftStatusLabel(status: string): string {
  const map: Record<string, string> = {
    pending: '待接收',
    accepted: '已接收',
    cancelled: '已取消',
    expired: '已过期',
  }
  return map[(status || '').toLowerCase()] ?? (status || '未知')
}

async function loadGifts() {
  loading.value = true
  try {
    gifts.value = await api.listDNSHEGifts(props.account.id)
  } catch (e) {
    message.error(e instanceof Error ? e.message : '加载转赠记录失败')
  } finally {
    loading.value = false
  }
}

async function cancelGift(row: DNSHEGift) {
  cancellingId.value = row.id
  try {
    await api.cancelDNSHEGift(props.account.id, row.id)
    message.success('转赠已取消')
    await loadGifts()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '取消失败')
  } finally {
    cancellingId.value = null
  }
}

async function copyText(text: string) {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const ta = document.createElement('textarea')
      ta.value = text
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    message.success('已复制')
  } catch {
    message.error('复制失败')
  }
}

watch(
  () => props.account.id,
  () => {
    void loadGifts()
  },
  { immediate: true },
)

defineExpose({ reload: loadGifts })
</script>

<style scoped>
.detail-section {
  min-width: 0;
}

.detail-section__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--fonu-text);
}

.records-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-4) var(--fonu-space-5) 0;
  flex-wrap: wrap;
}

.record-table-wrap {
  overflow-x: auto;
  padding: var(--fonu-space-3) var(--fonu-space-5) var(--fonu-space-4);
}

.record-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.record-table th {
  padding: 10px 12px;
  text-align: left;
  font-size: 12px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
  border-bottom: 1px solid var(--fonu-border);
  white-space: nowrap;
}

.record-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--fonu-border);
  vertical-align: middle;
}

.record-table__mono {
  font-family: var(--fonu-mono);
  font-size: 12px;
}

.record-table__muted {
  color: var(--fonu-text-muted);
}

.record-table__link {
  font-family: var(--fonu-mono);
  font-size: 12px;
  font-weight: 500;
  color: var(--fonu-text);
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
  vertical-align: middle;
}

.record-table__link:hover {
  color: var(--fonu-primary);
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.loading-wrap {
  display: flex;
  justify-content: center;
  padding: var(--fonu-space-5) 0;
}

.detail-empty-wrap {
  padding: 0 var(--fonu-space-5) var(--fonu-space-4);
}
</style>
