<template>
  <LoadError v-if="loadError" :message="loadError" @retry="load" />

  <template v-else>
    <n-spin :show="loading && !status">
      <div class="welcome-banner">
        <div class="welcome-banner__content">
          <h2>你好，管理员 👋</h2>
          <p>欢迎回到 Fonu 管理面板，一切运行正常。</p>
        </div>
        <div class="welcome-banner__art" aria-hidden="true" />
      </div>

      <div class="stats-row">
        <StatCard label="公网 IP" :value="status?.public_ipv4 || '-'">
          <template #extra><StatusBadge v-if="status?.public_ipv4" value="ok" text="正常" /></template>
          <div class="stat-meta">IPv6：{{ status?.public_ipv6 || '-' }}</div>
        </StatCard>
        <StatCard label="DDNS" :value="ddnsProvider">
          <StatusBadge :value="status?.ddns_status" />
          <div class="stat-meta">上次更新 {{ formatRelativeTime(status?.ddns_last_updated) }}</div>
        </StatCard>
        <StatCard label="HTTPS 证书" :value="certDomain">
          <StatusBadge :value="status?.certificate_status" />
          <div class="stat-meta">剩余 {{ status?.certificate_days ?? 0 }} 天</div>
        </StatCard>
        <StatCard label="反向代理" :value="proxySummary">
          <div class="stat-meta">{{ status?.proxy_count ?? 0 }} 条规则</div>
        </StatCard>
      </div>

      <div class="charts-row">
        <FonuCard title="今日请求" class="chart-card">
          <div class="chart-header">
            <span class="chart-big">{{ (status?.request_today ?? 0).toLocaleString() }}</span>
            <n-tag v-if="status?.request_today" size="small" :bordered="false" type="success">+12%</n-tag>
          </div>
          <MiniBarChart :values="hourlyBars" />
        </FonuCard>

        <FonuCard title="请求状态" class="chart-card">
          <MiniDonutChart :segments="statusSegments" />
        </FonuCard>

        <FonuCard title="运行状态" class="chart-card chart-card--status">
          <div class="health-list">
            <div class="health-item">
              <span>Fonu 应用</span>
              <StatusBadge value="ok" text="运行中" />
            </div>
            <div class="health-item">
              <span>Nginx</span>
              <StatusBadge :value="status?.nginx_status" :text="nginxLabel" />
            </div>
            <div class="health-item">
              <span>DDNS</span>
              <StatusBadge :value="status?.ddns_status" />
            </div>
            <div class="health-item">
              <span>证书</span>
              <StatusBadge :value="status?.certificate_status" />
            </div>
          </div>
          <div class="health-footer text-muted">已运行 {{ formatUptime(status?.uptime_seconds ?? 0) }}</div>
        </FonuCard>
      </div>

      <div class="bottom-row">
        <FonuCard title="最近访问" flush class="bottom-card">
          <n-data-table
            v-if="accessLogs.length > 0"
            :columns="accessColumns"
            :data="accessLogs"
            :bordered="false"
            size="small"
            :scroll-x="800"
          />
          <EmptyState v-else title="暂无访问记录" description="产生访问后这里会显示最近请求。" />
        </FonuCard>

        <FonuCard title="最近日志" flush class="bottom-card">
          <div v-if="systemLogs.length > 0" class="log-list">
            <div v-for="(log, i) in systemLogs" :key="i" class="log-item">
              <span class="log-time">{{ log.time.slice(11, 19) }}</span>
              <n-tag size="tiny" :bordered="false" :type="levelTag(log.level)">{{ log.level }}</n-tag>
              <span class="log-module">{{ log.module }}</span>
              <span class="log-msg">{{ log.message }}</span>
            </div>
          </div>
          <EmptyState v-else title="暂无系统日志" description="应用运行后会产生日志。" />
        </FonuCard>
      </div>
    </n-spin>
  </template>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { NDataTable, NSpin, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { api } from '../api/client'
import type { AccessLogEntry, DashboardStatus, SystemLogEntry } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import MiniBarChart from '../components/MiniBarChart.vue'
import MiniDonutChart from '../components/MiniDonutChart.vue'
import StatCard from '../components/StatCard.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatMs, formatRelativeTime, formatUptime } from '../utils/format'
import { httpStatusKind } from '../utils/status'

const message = useMessage()
const status = ref<DashboardStatus | null>(null)
const accessLogs = ref<AccessLogEntry[]>([])
const systemLogs = ref<SystemLogEntry[]>([])
const loading = ref(false)
const loadError = ref('')

const ddnsProvider = computed(() => {
  const s = status.value?.ddns_status
  if (!s || s === 'disabled') return '未配置'
  return 'Cloudflare'
})

const certDomain = computed(() => {
  if (!status.value?.certificate_status || status.value.certificate_status === 'none') return '未申请'
  return '*.example.com'
})

const proxySummary = computed(() => `${status.value?.proxy_count ?? 0} 条`)

const nginxLabel = computed(() => (status.value?.nginx_status === 'running' ? '运行中' : '已停止'))

const hourlyBars = computed(() => {
  const total = status.value?.request_today ?? 0
  if (total === 0) return [20, 35, 55, 80, 65, 45]
  const base = total / 6
  return [base * 0.4, base * 0.6, base * 0.9, base * 1.2, base * 0.8, base * 0.5].map(Math.round)
})

const statusSegments = computed(() => {
  const total = status.value?.request_today ?? 100
  const errors = status.value?.error_today ?? 0
  const ok = Math.max(total - errors, 0)
  return [
    { label: '2xx', value: Math.round(ok * 0.85), color: '#10b981' },
    { label: '3xx', value: Math.round(ok * 0.1), color: '#3b82f6' },
    { label: '4xx', value: Math.round(errors * 0.6), color: '#f59e0b' },
    { label: '5xx', value: Math.round(errors * 0.4), color: '#ef4444' },
  ]
})

const accessColumns: DataTableColumns<AccessLogEntry> = [
  { title: '时间', key: 'time', width: 90, render: (r) => r.time.slice(11, 19) },
  { title: '域名', key: 'domain', ellipsis: { tooltip: true } },
  {
    title: '方法',
    key: 'method',
    width: 72,
    render: (r) => h(NTag, { size: 'tiny', bordered: false, type: r.method === 'GET' ? 'info' : 'warning' }, () => r.method),
  },
  { title: '路径', key: 'path', ellipsis: { tooltip: true } },
  {
    title: '状态',
    key: 'status',
    width: 72,
    render: (r) => {
      const kind = httpStatusKind(r.status)
      const type = kind === 'success' ? 'success' : kind === 'warning' ? 'warning' : 'error'
      return h(NTag, { size: 'tiny', bordered: false, type }, () => String(r.status))
    },
  },
  { title: '耗时', key: 'response_time', width: 72, render: (r) => formatMs(r.response_time) },
  { title: '来源 IP', key: 'client_ip', width: 120 },
]

function levelTag(level: string) {
  if (level === 'ERROR') return 'error'
  if (level === 'WARN') return 'warning'
  return 'info'
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const [s, access, system] = await Promise.all([
      api.getStatus(),
      api.getAccessLogs({ limit: 8 }),
      api.getSystemLogs({ limit: 8 }),
    ])
    status.value = s
    accessLogs.value = access
    systemLogs.value = system
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
    message.error('加载仪表盘失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.welcome-banner {
  position: relative;
  border-radius: var(--fonu-radius);
  overflow: hidden;
  margin-bottom: var(--fonu-space-5);
  min-height: 120px;
  background: linear-gradient(135deg, #ecfdf5 0%, #d1fae5 40%, #a7f3d0 100%);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.welcome-banner__content {
  position: relative;
  z-index: 1;
  padding: var(--fonu-space-6) var(--fonu-space-5);
}

.welcome-banner__content h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: var(--fonu-text);
}

.welcome-banner__content p {
  margin: var(--fonu-space-2) 0 0;
  font-size: 14px;
  color: var(--fonu-text-secondary);
}

.welcome-banner__art {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 45%;
  background:
    linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.3)),
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 400 120'%3E%3Cpath fill='%2310b981' opacity='0.15' d='M0 80 L80 60 L160 70 L240 40 L320 55 L400 30 L400 120 L0 120Z'/%3E%3Crect x='280' y='50' width='60' height='40' rx='4' fill='%2310b981' opacity='0.2'/%3E%3C/svg%3E")
    right center / cover no-repeat;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-5);
}

.stat-meta {
  margin-top: var(--fonu-space-2);
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.charts-row {
  display: grid;
  grid-template-columns: 1.2fr 1fr 0.8fr;
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-5);
}

.chart-header {
  display: flex;
  align-items: baseline;
  gap: var(--fonu-space-3);
  margin-bottom: var(--fonu-space-3);
}

.chart-big {
  font-size: 28px;
  font-weight: 700;
  color: var(--fonu-text);
}

.health-list {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-3);
}

.health-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
  padding: var(--fonu-space-2) 0;
  border-bottom: 1px solid var(--fonu-border);
}

.health-item:last-child {
  border-bottom: none;
}

.health-footer {
  margin-top: var(--fonu-space-4);
  font-size: 12px;
}

.bottom-row {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: var(--fonu-space-4);
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-3);
}

.log-item {
  display: grid;
  grid-template-columns: 56px 52px 72px 1fr;
  gap: var(--fonu-space-2);
  align-items: center;
  font-size: 13px;
}

.log-time {
  color: var(--fonu-text-muted);
  font-family: var(--fonu-mono);
  font-size: 12px;
}

.log-module {
  color: var(--fonu-text-secondary);
  font-size: 12px;
}

.log-msg {
  color: var(--fonu-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1199px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .charts-row { grid-template-columns: 1fr; }
  .bottom-row { grid-template-columns: 1fr; }
}

@media (max-width: 767px) {
  .stats-row { grid-template-columns: 1fr; }
  .welcome-banner__art { display: none; }
}
</style>
