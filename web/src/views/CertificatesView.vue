<template>
  <PageHeader title="HTTPS 证书" description="通过 ACME 自动申请和续签证书">
    <template #actions>
      <n-button type="primary" :loading="applying" @click="apply">申请证书</n-button>
      <n-button :loading="renewing" @click="renew">立即续签</n-button>
    </template>
  </PageHeader>

  <LoadError v-if="loadError" :message="loadError" @retry="load" />

  <template v-else>
    <FonuCard v-if="primaryCert" class="cert-overview">
      <div class="overview-grid">
        <div class="overview-stat">
          <span class="overview-label">状态</span>
          <StatusBadge :value="primaryCert.status" text="正常" />
        </div>
        <div class="overview-stat">
          <span class="overview-label">有效证书</span>
          <span class="overview-value">{{ records.length }} 张</span>
        </div>
        <div class="overview-stat">
          <span class="overview-label">剩余天数</span>
          <span class="overview-value">{{ primaryCert.days_left }} 天</span>
        </div>
        <div class="overview-stat">
          <span class="overview-label">到期时间</span>
          <span class="overview-value">{{ formatDate(primaryCert.expires_at) }}</span>
        </div>
      </div>
    </FonuCard>

    <FonuCard title="证书列表" flush class="section-gap">
      <n-data-table
        v-if="records.length > 0"
        :columns="columns"
        :data="records"
        :loading="loading"
        :bordered="false"
        :scroll-x="800"
      />
      <EmptyState v-if="!loading && records.length === 0" title="还没有证书" description="完成 DNS 配置后，可通过 ACME 自动申请 HTTPS 证书。">
        <template #action>
          <n-button type="primary" :loading="applying" @click="apply">申请证书</n-button>
        </template>
      </EmptyState>
    </FonuCard>

    <div class="info-tip">
      <span>证书通过 ACME DNS-01 自动申请，需在 DDNS 页面配置有效的 DNS API 凭证（Cloudflare / DNSPod / 阿里云）。</span>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { NButton, NDataTable, useMessage, type DataTableColumns } from 'naive-ui'
import { api } from '../api/client'
import type { CertificateRecord } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatDate } from '../utils/format'
import { statusLabel } from '../utils/status'

const message = useMessage()
const records = ref<CertificateRecord[]>([])
const loading = ref(false)
const loadError = ref('')
const applying = ref(false)
const renewing = ref(false)

const primaryCert = computed(() => records.value[0] ?? null)

const columns: DataTableColumns<CertificateRecord> = [
  { title: '域名', key: 'domain' },
  { title: '类型', key: 'wildcard', render: (row) => (row.wildcard ? '通配符' : '主域名') },
  { title: '状态', key: 'status', render: (row) => h(StatusBadge, { value: row.status, text: statusLabel(row.status) }) },
  { title: '剩余', key: 'days_left', render: (row) => `${row.days_left} 天` },
  { title: '到期', key: 'expires_at', render: (row) => formatDate(row.expires_at) },
  { title: '上次续签', key: 'last_renew_at', render: (row) => formatDate(row.last_renew_at) || '-' },
]

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    records.value = await api.listCertificates()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

async function apply() {
  applying.value = true
  try {
    records.value = await api.applyCertificate()
    message.success('证书申请成功')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '申请失败')
  } finally {
    applying.value = false
  }
}

async function renew() {
  renewing.value = true
  try {
    const record = await api.renewCertificate()
    records.value = [record, ...records.value.filter((r) => r.id !== record.id)]
    message.success('证书续签成功')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '续签失败')
  } finally {
    renewing.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.cert-overview { margin-bottom: var(--fonu-space-5); }

.overview-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--fonu-space-4);
}

.overview-stat {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-2);
}

.overview-label {
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.overview-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--fonu-text);
}

.section-gap { margin-top: 0; }

.info-tip {
  margin-top: var(--fonu-space-5);
  padding: var(--fonu-space-4);
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius-sm);
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

@media (max-width: 767px) {
  .overview-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
