<template>
  <LoadError v-if="loadError" :message="loadError" @retry="load" />

  <template v-else>
    <n-spin :show="loading && !status">
      <section class="hero">
        <div class="hero__mountains" aria-hidden="true" />
        <div class="hero__inner">
          <div class="hero__content">
            <h1 class="hero__title">你好，管理员 👋</h1>
            <p class="hero__desc">一切运行正常，Fonu 正在为你的 NAS 提供安全的访问服务。</p>
          </div>
          <div class="hero__visual" aria-hidden="true">
            <p class="hero__slogan">让 NAS 访问更简单</p>
            <div class="hero__nas" />
          </div>
        </div>
      </section>

      <div class="stats-row">
        <StatCard label="公网 IP" tone="blue">
          <template #icon><n-icon :component="GlobeOutline" /></template>
          <template #extra><StatusBadge v-if="status?.public_ipv4" value="ok" text="正常" /></template>
          <template #value>
            <div class="kv-list">
              <div class="kv-row">
                <span class="kv-row__k">IPv4</span>
                <span class="kv-row__v mono">{{ status?.public_ipv4 || '-' }}</span>
              </div>
              <div class="kv-row">
                <span class="kv-row__k">IPv6</span>
                <span class="kv-row__v mono kv-row__v--sub">{{ status?.public_ipv6 || '-' }}</span>
              </div>
            </div>
          </template>
        </StatCard>

        <StatCard label="DDNS" tone="green">
          <template #icon><n-icon :component="WifiOutline" /></template>
          <template #extra><StatusBadge :value="status?.ddns_status" text="正常" /></template>
          <template v-if="primaryDdns" #value>
            <div class="ddns-stack">
              <span class="ddns-provider">{{ ddnsProviderName }}</span>
              <span class="ddns-domain">{{ primaryDdns.root_domain }}</span>
            </div>
          </template>
          <template v-else #value>{{ ddnsLabel }}</template>
          <div v-if="primaryDdns" class="stat-foot">
            <div class="stat-foot__line">{{ ddnsRecordLabel }}</div>
            <div class="stat-foot__muted">上次更新 {{ formatRelativeTime(status?.ddns_last_updated) }}</div>
          </div>
        </StatCard>

        <StatCard label="证书" tone="blue">
          <template #icon><n-icon :component="ShieldCheckmarkOutline" /></template>
          <template #extra><StatusBadge :value="status?.certificate_status" text="正常" /></template>
          <template v-if="primaryCert" #value>
            <div class="cert-stack">
              <span class="cert-domain">{{ primaryCert.domain }}</span>
              <span v-if="certWildcard" class="cert-wildcard mono">{{ certWildcard }}</span>
            </div>
          </template>
          <template v-else #value>未申请</template>
          <div v-if="primaryCert" class="stat-foot">
            <div class="stat-foot__line">
              <span class="stat-foot__muted">剩余</span>
              <span class="stat-highlight">{{ primaryCert.days_left }} 天</span>
            </div>
            <div class="stat-foot__muted">到期 {{ formatDate(primaryCert.expires_at) }}</div>
          </div>
        </StatCard>

        <StatCard label="反向代理" :value="status?.proxy_count ?? 0" tone="teal">
          <template #icon><n-icon :component="GitNetworkOutline" /></template>
          <template #extra><StatusBadge value="ok" text="正常" /></template>
          <div class="proxy-metrics">
            <div class="proxy-metric">
              <span class="proxy-metric__val proxy-metric__val--ok">{{ proxyEnabled }}</span>
              <span class="proxy-metric__lbl">已启用</span>
            </div>
            <div class="proxy-metric">
              <span class="proxy-metric__val">{{ proxyDisabled }}</span>
              <span class="proxy-metric__lbl">已禁用</span>
            </div>
          </div>
        </StatCard>
      </div>

      <div class="metrics-row">
        <FonuCard class="panel requests-panel">
          <div class="panel__head">
            <div class="panel-title">
              <n-icon :component="FlashOutline" class="panel-title__icon panel-title__icon--amber" />
              <span>今日请求</span>
            </div>
            <n-tag size="small" :bordered="false" class="range-tag">最近 24 小时</n-tag>
          </div>
          <div class="panel__body requests-panel__body">
            <div class="metric-hero">
              <span class="metric-hero__value">{{ (status?.request_today ?? 0).toLocaleString() }}</span>
              <n-tag v-if="(status?.request_today ?? 0) > 0" size="small" :bordered="false" type="success">
                成功率 {{ successRate }}%
              </n-tag>
            </div>
            <MiniBarChart :values="hourlyBars" />
          </div>
        </FonuCard>

        <FonuCard class="panel perf-panel">
          <div class="perf-block">
            <div class="perf-block__label">错误请求</div>
            <div class="perf-block__value perf-block__value--error">{{ status?.error_today ?? 0 }}</div>
            <div class="perf-block__meta perf-block__meta--error">占比 {{ errorRate }}%</div>
          </div>
          <div class="perf-divider" />
          <div class="perf-block">
            <div class="perf-block__label">平均响应时间</div>
            <div class="perf-block__value">{{ Math.round(status?.avg_response_ms ?? 0) }}<small>ms</small></div>
            <div class="perf-block__meta">今日访问均值</div>
          </div>
        </FonuCard>

        <FonuCard class="panel health-panel">
          <div class="panel__head">
            <div class="panel-title">
              <n-icon :component="PulseOutline" class="panel-title__icon panel-title__icon--teal" />
              <span>运行状态</span>
            </div>
          </div>
          <div class="panel__body health-list">
            <div v-for="item in healthItems" :key="item.name" class="health-item">
              <StatusBadge :value="item.status" :text="item.label" />
              <span class="health-item__name">{{ item.name }}</span>
              <span class="health-item__meta">{{ item.meta }}</span>
            </div>
          </div>
        </FonuCard>
      </div>

      <div class="bottom-row">
        <FonuCard flush class="bottom-card">
          <template #title>
            <n-icon :component="ListOutline" class="card-title-icon" />
            <span>最近访问</span>
          </template>
          <template #header>
            <router-link :to="{ name: 'logs', query: { tab: 'access' } }" class="card-link">
              查看更多
              <n-icon :component="ChevronForwardOutline" />
            </router-link>
          </template>
          <div class="bottom-card__body">
            <n-data-table
              v-if="accessLogs.length > 0"
              :columns="accessColumns"
              :data="accessLogs"
              :bordered="false"
              size="small"
              :scroll-x="860"
            />
            <EmptyState v-else title="暂无访问记录" description="产生访问后这里会显示最近请求。" />
          </div>
        </FonuCard>

        <FonuCard flush class="bottom-card">
          <template #title>
            <n-icon :component="DocumentTextOutline" class="card-title-icon" />
            <span>最新日志</span>
          </template>
          <template #header>
            <router-link :to="{ name: 'logs', query: { tab: 'system' } }" class="card-link">
              查看更多
              <n-icon :component="ChevronForwardOutline" />
            </router-link>
          </template>
          <div class="bottom-card__body">
            <div v-if="systemLogs.length > 0" class="log-list">
              <div v-for="(log, i) in systemLogs" :key="i" class="log-item">
                <span class="log-time">{{ formatLogTime(log.time) }}</span>
                <n-tag size="tiny" :bordered="false" :type="levelTag(log.level)">{{ log.level }}</n-tag>
                <span class="log-module">{{ log.module }}</span>
                <span class="log-msg">{{ log.message }}</span>
              </div>
            </div>
            <EmptyState v-else title="暂无系统日志" description="应用运行后会产生日志。" />
          </div>
        </FonuCard>
      </div>
    </n-spin>
  </template>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { NDataTable, NIcon, NSpin, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import {
  ChevronForwardOutline,
  DocumentTextOutline,
  FlashOutline,
  GitNetworkOutline,
  GlobeOutline,
  ListOutline,
  PulseOutline,
  ShieldCheckmarkOutline,
  WifiOutline,
} from '@vicons/ionicons5'
import { api } from '../api/client'
import type {
  AccessLogEntry,
  CertificateRecord,
  DashboardStatus,
  DDNSConfig,
  ProxyRule,
  SystemLogEntry,
} from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import MiniBarChart from '../components/MiniBarChart.vue'
import StatCard from '../components/StatCard.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatDate, formatLogTime, formatRelativeTime, formatUptime } from '../utils/format'
import { httpStatusKind } from '../utils/status'

const message = useMessage()
const status = ref<DashboardStatus | null>(null)
const proxies = ref<ProxyRule[]>([])
const ddnsConfigs = ref<DDNSConfig[]>([])
const certificates = ref<CertificateRecord[]>([])
const accessLogs = ref<AccessLogEntry[]>([])
const systemLogs = ref<SystemLogEntry[]>([])
const loading = ref(false)
const loadError = ref('')

const providerLabels: Record<string, string> = {
  cloudflare: 'Cloudflare',
  dnspod: 'DNSPod',
  alidns: '阿里云',
}

const primaryDdns = computed(() => ddnsConfigs.value[0] ?? null)
const primaryCert = computed(() => {
  if (certificates.value.length === 0) return null
  return [...certificates.value].sort((a, b) => a.days_left - b.days_left)[0]
})
const certWildcard = computed(() => {
  const domains = primaryCert.value?.domains ?? []
  return domains.find((d) => d.startsWith('*.')) ?? (primaryCert.value?.wildcard ? `*.${primaryCert.value.domain}` : '')
})

const ddnsLabel = computed(() => {
  const count = status.value?.ddns_count ?? 0
  if (count === 0) return '未配置'
  return `${count} 个域名`
})
const ddnsProviderName = computed(() => {
  const p = primaryDdns.value?.provider
  return p ? (providerLabels[p] ?? p) : ''
})
const ddnsRecordLabel = computed(() => {
  if (!primaryDdns.value) return ''
  const name = primaryDdns.value.record_name || '@'
  return name === '@' ? primaryDdns.value.root_domain : `${name}.${primaryDdns.value.root_domain}`
})

const proxyEnabled = computed(() => proxies.value.filter((p) => p.enabled).length)
const proxyDisabled = computed(() => proxies.value.filter((p) => !p.enabled).length)

const successRate = computed(() => {
  const total = status.value?.request_today ?? 0
  if (total === 0) return 100
  const errors = status.value?.error_today ?? 0
  return Math.round(((total - errors) / total) * 100)
})

const errorRate = computed(() => {
  const total = status.value?.request_today ?? 0
  if (total === 0) return 0
  return Math.round(((status.value?.error_today ?? 0) / total) * 100)
})

const hourlyBars = computed(() => {
  const total = status.value?.request_today ?? 0
  if (total === 0) return [20, 35, 55, 80, 65, 45]
  const base = total / 6
  return [base * 0.4, base * 0.6, base * 0.9, base * 1.2, base * 0.8, base * 0.5].map(Math.round)
})

const healthItems = computed(() => [
  {
    name: 'Fonu 应用',
    meta: `运行 ${formatUptime(status.value?.uptime_seconds ?? 0)}`,
    status: 'running',
    label: '运行中',
  },
  {
    name: 'Nginx',
    meta: status.value?.nginx_status === 'running' ? `运行 ${formatUptime(status.value?.uptime_seconds ?? 0)}` : '已停止',
    status: status.value?.nginx_status,
    label: status.value?.nginx_status === 'running' ? '运行中' : '已停止',
  },
  {
    name: 'DDNS',
    meta: formatRelativeTime(status.value?.ddns_last_updated) || '未更新',
    status: status.value?.ddns_status,
    label: status.value?.ddns_status === 'ok' ? '运行中' : '异常',
  },
  {
    name: '证书',
    meta: primaryCert.value ? `${primaryCert.value.days_left} 天后到期` : '未配置',
    status: status.value?.certificate_status,
    label: status.value?.certificate_status === 'ok' ? '正常' : '需关注',
  },
])

const accessColumns: DataTableColumns<AccessLogEntry> = [
  { title: '时间', key: 'time', width: 168, render: (r) => formatLogTime(r.time) },
  { title: '域名', key: 'domain', ellipsis: { tooltip: true } },
  {
    title: '方法',
    key: 'method',
    width: 68,
    render: (r) =>
      h(
        NTag,
        { size: 'tiny', bordered: false, type: r.method === 'GET' ? 'info' : r.method === 'POST' ? 'success' : 'warning' },
        () => r.method,
      ),
  },
  { title: '路径', key: 'path', ellipsis: { tooltip: true } },
  {
    title: '状态',
    key: 'status',
    width: 68,
    render: (r) => {
      const kind = httpStatusKind(r.status)
      const type = kind === 'success' ? 'success' : kind === 'warning' ? 'warning' : 'error'
      return h(NTag, { size: 'tiny', bordered: false, type }, () => String(r.status))
    },
  },
  { title: '耗时(ms)', key: 'response_time', width: 80, render: (r) => Math.round(r.response_time * 1000) },
  { title: '来源 IP', key: 'client_ip', width: 118 },
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
    const [s, access, system, proxyList, ddns, certs] = await Promise.all([
      api.getStatus(),
      api.getAccessLogs({ limit: 10 }),
      api.getSystemLogs({ limit: 10 }),
      api.listProxies(),
      api.listDDNS(),
      api.listCertificates(),
    ])
    status.value = s
    accessLogs.value = access
    systemLogs.value = system
    proxies.value = proxyList
    ddnsConfigs.value = ddns
    certificates.value = certs
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
.dashboard {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
}

.hero {
  position: relative;
  border-radius: var(--fonu-radius);
  overflow: hidden;
  margin-bottom: var(--fonu-space-4);
  min-height: 156px;
  background: linear-gradient(90deg, #f8fbff 0%, #eef5ff 55%, #dbeafe 100%);
}

html[data-theme='dark'] .hero,
html.dark .hero {
  background: linear-gradient(90deg, #0f172a 0%, #1e293b 55%, #1e3a5f 100%);
}

.hero__mountains {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 42%;
  background:
    linear-gradient(90deg, rgba(248, 251, 255, 1) 0%, transparent 28%),
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 480 156'%3E%3Cdefs%3E%3ClinearGradient id='sky' x1='0' y1='0' x2='0' y2='1'%3E%3Cstop offset='0%25' stop-color='%23dbeafe'/%3E%3Cstop offset='1%25' stop-color='%23eff6ff'/%3E%3C/linearGradient%3E%3C/defs%3E%3Crect width='480' height='156' fill='url(%23sky)'/%3E%3Cpath fill='%2394a3b8' opacity='0.35' d='M120 95 L200 55 L280 75 L360 40 L440 60 L480 45 L480 156 L120 156Z'/%3E%3Cpath fill='%2364748b' opacity='0.25' d='M180 110 L260 80 L340 95 L420 70 L480 85 L480 156 L180 156Z'/%3E%3Cpath fill='%23fff' opacity='0.5' d='M200 58 L230 42 L260 58 L245 58 L245 75 L215 75 L215 58Z'/%3E%3C/svg%3E")
    right center / cover no-repeat;
  pointer-events: none;
}

html[data-theme='dark'] .hero__mountains,
html.dark .hero__mountains {
  background:
    linear-gradient(90deg, #0f172a 0%, transparent 30%),
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 480 156'%3E%3Cpath fill='%23334155' opacity='0.5' d='M120 95 L200 55 L280 75 L360 40 L440 60 L480 45 L480 156 L120 156Z'/%3E%3C/svg%3E")
    right center / cover no-repeat;
}

.hero__inner {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-5);
  padding: 28px var(--fonu-space-5);
  min-height: 156px;
}

.hero__title {
  margin: 0;
  font-size: 30px;
  font-weight: 700;
  color: var(--fonu-text);
  letter-spacing: -0.03em;
}

.hero__desc {
  margin: 10px 0 0;
  font-size: 14px;
  color: var(--fonu-text-secondary);
  max-width: 460px;
  line-height: 1.65;
}

.hero__visual {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  margin-right: 8%;
}

.hero__slogan {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: #2563eb;
  font-style: italic;
}

.hero__nas {
  width: 132px;
  height: 78px;
  background: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 132 78'%3E%3Crect x='10' y='18' width='112' height='52' rx='8' fill='%23f1f5f9' stroke='%23cbd5e1' stroke-width='1.5'/%3E%3Crect x='18' y='28' width='20' height='32' rx='3' fill='%23e2e8f0'/%3E%3Crect x='42' y='28' width='20' height='32' rx='3' fill='%23e2e8f0'/%3E%3Crect x='66' y='28' width='20' height='32' rx='3' fill='%23e2e8f0'/%3E%3Crect x='90' y='28' width='20' height='32' rx='3' fill='%23e2e8f0'/%3E%3Ccircle cx='28' cy='38' r='3' fill='%2310b981'/%3E%3Ccircle cx='52' cy='38' r='3' fill='%2310b981'/%3E%3Ccircle cx='76' cy='38' r='3' fill='%23f59e0b'/%3E%3Ccircle cx='100' cy='38' r='3' fill='%2310b981'/%3E%3Crect x='24' y='10' width='84' height='10' rx='4' fill='%2310b981'/%3E%3C/svg%3E")
    center / contain no-repeat;
  filter: drop-shadow(0 8px 16px rgba(15, 23, 42, 0.12));
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.kv-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.kv-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.kv-row__k {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.kv-row__v {
  font-size: 17px;
  font-weight: 700;
  color: var(--fonu-text);
  line-height: 1.3;
}

.kv-row__v--sub {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
}

.ddns-stack,
.cert-stack {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ddns-provider {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.ddns-domain,
.cert-domain {
  font-size: 17px;
  font-weight: 700;
  color: var(--fonu-text);
  line-height: 1.3;
}

.cert-wildcard {
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.stat-foot {
  margin-top: var(--fonu-space-3);
  padding-top: var(--fonu-space-3);
  border-top: 1px solid var(--fonu-border);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-foot__line {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--fonu-text);
}

.stat-foot__muted {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.stat-highlight {
  color: var(--fonu-brand);
  font-weight: 600;
}

.proxy-metrics {
  display: flex;
  gap: var(--fonu-space-5);
  margin-top: var(--fonu-space-3);
  padding-top: var(--fonu-space-3);
  border-top: 1px solid var(--fonu-border);
}

.proxy-metric {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.proxy-metric__val {
  font-size: 18px;
  font-weight: 700;
  color: var(--fonu-text);
}

.proxy-metric__val--ok {
  color: var(--fonu-brand);
}

.proxy-metric__lbl {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.metrics-row {
  display: grid;
  grid-template-columns: minmax(0, 2.1fr) minmax(0, 0.72fr) minmax(0, 1.05fr);
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
  align-items: stretch;
}

.panel :deep(.fonu-card__body) {
  padding-top: 0;
}

.panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--fonu-space-5) var(--fonu-space-5) 0;
}

.panel__body {
  padding: var(--fonu-space-3) var(--fonu-space-5) var(--fonu-space-5);
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--fonu-text);
}

.panel-title__icon {
  font-size: 18px;
  color: var(--fonu-brand);
}

.panel-title__icon--amber {
  color: #f59e0b;
}

.panel-title__icon--teal {
  color: #14b8a6;
}

.range-tag {
  background: var(--fonu-bg) !important;
  color: var(--fonu-text-secondary) !important;
}

.metric-hero {
  display: flex;
  align-items: baseline;
  gap: var(--fonu-space-3);
  margin-bottom: 4px;
}

.metric-hero__value {
  font-size: 34px;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: var(--fonu-text);
  line-height: 1.1;
}

.requests-panel,
.health-panel {
  display: flex;
  flex-direction: column;
}

.requests-panel :deep(.fonu-card__body),
.health-panel :deep(.fonu-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.perf-panel :deep(.fonu-card__body) {
  display: flex;
  flex-direction: column;
  justify-content: stretch;
  height: 100%;
  padding: var(--fonu-space-5);
}

.perf-panel {
  display: flex;
  flex-direction: column;
}

.perf-block {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: var(--fonu-space-3) 0;
}

.perf-block__label {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.perf-block__value {
  margin-top: 6px;
  font-size: 30px;
  font-weight: 700;
  color: var(--fonu-text);
  line-height: 1.1;
}

.perf-block__value--error {
  color: var(--fonu-error);
}

.perf-block__value small {
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text-secondary);
  margin-left: 2px;
}

.perf-block__meta {
  margin-top: 4px;
  font-size: 11px;
  color: var(--fonu-text-muted);
}

.perf-block__meta--error {
  color: var(--fonu-error);
  opacity: 0.85;
}

.perf-divider {
  height: 1px;
  background: var(--fonu-border);
  margin: var(--fonu-space-2) 0;
}

.health-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 1;
}

.health-item {
  display: grid;
  grid-template-columns: 72px 1fr auto;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--fonu-radius-sm);
  background: var(--fonu-bg);
}

.health-item :deep(.status-badge) {
  justify-self: start;
  white-space: nowrap;
}

.health-item__name {
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.health-item__meta {
  font-size: 12px;
  color: var(--fonu-text-muted);
  white-space: nowrap;
}

.bottom-row {
  display: grid;
  grid-template-columns: minmax(0, 1.72fr) minmax(0, 1fr);
  gap: var(--fonu-space-4);
  align-items: stretch;
}

.bottom-card {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.bottom-card :deep(.fonu-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.bottom-card :deep(.fonu-card__header) {
  padding-bottom: var(--fonu-space-3);
}

.bottom-card__body {
  flex: 1;
  height: 392px;
  padding: 0 var(--fonu-space-5) var(--fonu-space-5);
  overflow: auto;
}

.card-title-icon {
  font-size: 18px;
  color: var(--fonu-brand);
}

.card-link {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  font-size: 13px;
  color: var(--fonu-text-secondary);
  text-decoration: none;
  font-weight: 500;
}

.card-link:hover {
  color: var(--fonu-brand);
}

.log-list {
  display: flex;
  flex-direction: column;
}

.log-item {
  display: grid;
  grid-template-columns: 148px 50px 68px 1fr;
  gap: 8px;
  align-items: center;
  font-size: 13px;
  min-height: 38px;
  padding: 7px 0;
  border-bottom: 1px solid var(--fonu-border);
}

.log-item:last-child {
  border-bottom: none;
}

.log-time {
  color: var(--fonu-text-muted);
  font-family: var(--fonu-mono);
  font-size: 12px;
  white-space: nowrap;
}

.log-module {
  color: var(--fonu-text-secondary);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.log-msg {
  color: var(--fonu-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mono {
  font-family: var(--fonu-mono);
}

@media (max-width: 1199px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .metrics-row { grid-template-columns: 1fr; }
  .bottom-row { grid-template-columns: 1fr; }
  .hero__visual { display: none; }
}

@media (max-width: 767px) {
  .stats-row { grid-template-columns: 1fr; }
  .hero__title { font-size: 24px; }
  .log-item { grid-template-columns: 1fr; gap: 4px; }
}
</style>
