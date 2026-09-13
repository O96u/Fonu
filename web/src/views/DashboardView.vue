<template>
  <LoadError v-if="loadError" :message="loadError" @retry="load" />

  <template v-else>
    <section class="hero-banner">
      <div class="hero-content">
        <h1 class="hero-banner__title">{{ greetingText }}</h1>
        <p class="hero-banner__desc">Fonu 正在为你的 NAS 提供稳定、安全的访问服务。</p>
        <div class="hero-banner__badges">
          <span class="hero-banner__badge"><n-icon :component="CheckmarkCircleOutline" /> 简单易用</span>
          <span class="hero-banner__badge"><n-icon :component="FlashOutline" /> 安全稳定</span>
          <span class="hero-banner__badge"><n-icon :component="GlobeOutline" /> 随时随地访问</span>
        </div>
      </div>
      <img class="hero-art" :src="bannerImg" alt="" aria-hidden="true" />
    </section>

    <n-spin :show="loading && !status">
      <div class="stats-row">
        <StatCard label="公网 IP" tone="blue">
          <template #icon><n-icon :component="GlobeOutline" /></template>
          <template #extra><StatusBadge v-if="publicIPv4Label !== '-'" value="ok" text="正常" /></template>
          <template #value>
            <span class="stat-primary mono">{{ publicIPv4Label }}</span>
          </template>
          <p class="stat-desc mono">{{ publicIPv6Label }}</p>
          <div class="stat-foot">
            <div class="stat-foot__line">{{ publicIPSourceLabel }}</div>
            <div class="stat-foot__muted">上次更新 {{ formatRelativeTime(status?.ddns_last_updated) || '刚刚' }}</div>
          </div>
        </StatCard>

        <StatCard label="域名" tone="green">
          <template #icon><n-icon :component="WifiOutline" /></template>
          <template #extra><StatusBadge :value="status?.ddns_status" text="正常" /></template>
          <template #value>{{ domainCountLabel }}</template>
          <p class="stat-desc">{{ domainSubLabel }}</p>
          <div class="stat-foot">
            <div class="stat-foot__line">{{ domainExamplesLabel }}</div>
            <div class="stat-foot__muted">上次检查 {{ formatRelativeTime(status?.ddns_last_updated) || '刚刚' }}</div>
          </div>
        </StatCard>

        <StatCard label="HTTPS 证书" tone="blue">
          <template #icon><n-icon :component="ShieldCheckmarkOutline" /></template>
          <template #extra><StatusBadge :value="status?.certificate_status" text="正常" /></template>
          <template #value>{{ certCountLabel }}</template>
          <p class="stat-desc">{{ certSubLabel }}</p>
          <div v-if="primaryCert" class="stat-foot">
            <div class="stat-foot__line mono">{{ certDisplayName }}</div>
            <div class="stat-foot__muted">
              到期 {{ formatDate(primaryCert.expires_at) }} · 剩余 {{ primaryCert.days_left }} 天
            </div>
          </div>
        </StatCard>

        <StatCard label="服务" tone="teal">
          <template #icon><n-icon :component="GitNetworkOutline" /></template>
          <template #extra><StatusBadge value="ok" text="正常" /></template>
          <template #value>{{ coreServiceCountLabel }}</template>
          <p class="stat-desc">核心服务运行正常</p>
          <div class="stat-foot">
            <div class="stat-foot__line">Fonu / Nginx / DDNS / 证书</div>
            <div class="stat-foot__muted">上次检查 {{ formatRelativeTime(status?.started_at) || '刚刚' }}</div>
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
              <n-tag v-if="requestTrend !== null" size="small" :bordered="false" :type="requestTrend >= 0 ? 'success' : 'warning'">
                {{ requestTrend >= 0 ? '↑' : '↓' }} {{ requestTrend >= 0 ? '+' : '' }}{{ requestTrend }}%
              </n-tag>
            </div>
            <div class="panel-chart">
              <MiniBarChart :values="hourlyBars.values" :labels="hourlyBars.labels" />
            </div>
          </div>
        </FonuCard>

        <FonuCard class="panel traffic-panel">
          <div class="panel__head">
            <div class="panel-title">
              <n-icon :component="SwapVerticalOutline" class="panel-title__icon panel-title__icon--teal" />
              <span>实时流量</span>
            </div>
            <n-tag size="small" :bordered="false" class="range-tag">最近 5 分钟</n-tag>
          </div>
          <div class="panel__body traffic-panel__body">
            <div class="traffic-rates">
              <div class="traffic-rate traffic-rate--up">
                <span class="traffic-rate__dot" />
                <span class="traffic-rate__label">上传</span>
                <span class="traffic-rate__value">{{ formatRate(trafficTotals.uploadRate) }}</span>
              </div>
              <div class="traffic-rate traffic-rate--down">
                <span class="traffic-rate__dot" />
                <span class="traffic-rate__label">下载</span>
                <span class="traffic-rate__value">{{ formatRate(trafficTotals.downloadRate) }}</span>
              </div>
            </div>
            <div class="panel-chart">
              <MiniTrafficChart
                :labels="trafficChart.labels"
                :upload="trafficChart.upload"
                :download="trafficChart.download"
              />
            </div>
          </div>
        </FonuCard>

        <FonuCard class="panel health-panel">
          <div class="panel__head">
            <div class="panel-title">
              <n-icon :component="PulseOutline" class="panel-title__icon panel-title__icon--teal" />
              <span>运行状态</span>
            </div>
            <router-link :to="{ name: 'settings' }" class="card-link">
              查看详情
              <n-icon :component="ChevronForwardOutline" />
            </router-link>
          </div>
          <div class="panel__body health-list">
            <div v-for="item in healthItems" :key="item.name" class="health-item">
              <div class="health-item__icon" :class="`health-item__icon--${item.tone}`">
                <n-icon :component="item.icon" />
              </div>
              <div class="health-item__main">
                <span class="health-item__name">{{ item.name }}</span>
                <span class="health-item__meta">{{ item.desc }}</span>
                <span class="health-item__detail">{{ item.meta }}</span>
              </div>
              <StatusBadge :value="item.status" :text="item.label" />
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
              查看全部
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
              :scroll-x="940"
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
              查看全部
              <n-icon :component="ChevronForwardOutline" />
            </router-link>
          </template>
          <div class="bottom-card__body">
            <n-data-table
              v-if="systemLogs.length > 0"
              :columns="systemColumns"
              :data="systemLogs"
              :bordered="false"
              size="small"
              :scroll-x="720"
            />
            <EmptyState v-else title="暂无系统日志" description="应用运行后会产生日志。" />
          </div>
        </FonuCard>
      </div>
    </n-spin>
  </template>
</template>

<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, ref } from 'vue'
import { NDataTable, NIcon, NSpin, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import {
  AppsOutline,
  CheckmarkCircleOutline,
  ChevronForwardOutline,
  CloudOutline,
  DocumentTextOutline,
  FlashOutline,
  GitNetworkOutline,
  GlobeOutline,
  ListOutline,
  PulseOutline,
  ServerOutline,
  ShieldCheckmarkOutline,
  SwapVerticalOutline,
  WifiOutline,
} from '@vicons/ionicons5'
import bannerImg from '../assets/brand/banner.png'
import { api, asList } from '../api/client'
import type {
  AccessLogEntry,
  CertificateRecord,
  DashboardStatus,
  DDNSConfig,
  ProxyTraffic,
  SystemLogEntry,
} from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import MiniBarChart from '../components/MiniBarChart.vue'
import MiniTrafficChart from '../components/MiniTrafficChart.vue'
import StatCard from '../components/StatCard.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatDate, formatLogTime, formatRate, formatRelativeTime, formatUptime } from '../utils/format'
import { httpStatusKind } from '../utils/status'

const message = useMessage()
const status = ref<DashboardStatus | null>(null)
const ddnsConfigs = ref<DDNSConfig[]>([])
const certificates = ref<CertificateRecord[]>([])
const accessLogs = ref<AccessLogEntry[]>([])
const systemLogs = ref<SystemLogEntry[]>([])
const loading = ref(false)
const loadError = ref('')
const trafficByRule = ref<Record<number, ProxyTraffic>>({})
type RatePoint = { at: number; upload: number; download: number }
const rateHistory = ref<RatePoint[]>([])
const RATE_HISTORY_MS = 5 * 60 * 1000
let trafficTimer: ReturnType<typeof setInterval> | null = null

function timeGreeting(hour: number): string {
  if (hour >= 5 && hour < 12) return '上午好'
  if (hour >= 12 && hour < 18) return '下午好'
  return '晚上好'
}

const greetingText = computed(() => `${timeGreeting(new Date().getHours())}，管理员`)

const primaryCert = computed(() => {
  if (certificates.value.length === 0) return null
  return [...certificates.value].sort((a, b) => a.days_left - b.days_left)[0]
})
const certWildcard = computed(() => {
  const domains = primaryCert.value?.domains ?? []
  return domains.find((d) => d.startsWith('*.')) ?? (primaryCert.value?.wildcard ? `*.${primaryCert.value.domain}` : '')
})
const certDisplayName = computed(() => certWildcard.value || primaryCert.value?.domain || '')

const publicIPv4Label = computed(() => status.value?.public_ipv4 || '-')
const publicIPv6Label = computed(() => {
  if (status.value?.public_ip_source === 'ddns' && !status.value?.public_ipv6) return '未获取'
  return status.value?.public_ipv6 || '-'
})
const publicIPSourceLabel = computed(() => {
  switch (status.value?.public_ip_source) {
    case 'ddns':
      return status.value?.public_ipv4 || status.value?.public_ipv6
        ? '来源：DDNS 已同步记录'
        : '来源：DDNS 已配置，等待同步'
    case 'detect':
      return '来源：出口 IP 探测（未配置 DDNS）'
    default:
      return '来源：暂无'
  }
})

const domainCountLabel = computed(() => {
  const count = status.value?.ddns_count ?? ddnsConfigs.value.length
  if (count === 0) return '未配置'
  return `${count} 个`
})

const domainSubLabel = computed(() => {
  const count = status.value?.ddns_count ?? ddnsConfigs.value.length
  if (count === 0) return '尚未配置域名解析'
  return '已配置并正常解析的域名'
})

const domainExamplesLabel = computed(() => {
  const examples: string[] = []
  for (const cfg of ddnsConfigs.value.filter((c) => c.enabled)) {
    const root = cfg.root_domain
    const names = cfg.record_names?.length ? cfg.record_names : [cfg.record_name || '@']
    for (const name of names) {
      if (!name || name === '@') examples.push(root)
      else if (name === '*') examples.push(`*.${root}`)
      else if (name.includes('.')) examples.push(name)
      else examples.push(`${name}.${root}`)
    }
  }
  return examples.length > 0 ? examples.join(', ') : '暂无域名记录'
})

const certCountLabel = computed(() => {
  const count = status.value?.certificate_count ?? certificates.value.length
  if (count === 0) return '未申请'
  return `${count} 个`
})

const certSubLabel = computed(() => {
  if (!primaryCert.value) return '尚未申请 HTTPS 证书'
  if (status.value?.certificate_status === 'ok') return '证书有效，自动续期中'
  return '证书需要关注'
})

const coreServiceCountLabel = computed(() => `${healthItems.value.length} 个`)

const trafficTotals = computed(() => {
  let uploadRate = 0
  let downloadRate = 0
  for (const stats of Object.values(trafficByRule.value)) {
    uploadRate += stats.upload_rate
    downloadRate += stats.download_rate
  }
  return { uploadRate, downloadRate }
})

const trafficChart = computed(() => {
  const now = Date.now()
  const labels: string[] = []
  const upload: number[] = []
  const download: number[] = []
  for (let i = 5; i >= 0; i--) {
    const bucketEnd = now - i * 60 * 1000
    const bucketStart = bucketEnd - 60 * 1000
    const points = rateHistory.value.filter((p) => p.at > bucketStart && p.at <= bucketEnd)
    const avg = (key: 'upload' | 'download') =>
      points.length ? points.reduce((sum, p) => sum + p[key], 0) / points.length : 0
    upload.push(avg('upload'))
    download.push(avg('download'))
    const d = new Date(bucketEnd)
    labels.push(
      `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`,
    )
  }
  return { labels, upload, download }
})

const hourlyBars = computed(() => {
  const labels = Array.from({ length: 12 }, (_, i) => String(i * 2).padStart(2, '0'))
  const values = Array(12).fill(0)
  for (const log of accessLogs.value) {
    const hour = new Date(log.time).getHours()
    values[Math.floor(hour / 2)]++
  }
  const hasData = values.some((v) => v > 0)
  if (!hasData && (status.value?.request_today ?? 0) > 0) {
    const total = status.value?.request_today ?? 0
    const base = total / 12
    return {
      labels,
      values: [base * 0.5, base * 0.7, base * 0.9, base * 1.1, base * 1.3, base * 1.2, base * 1.0, base * 0.8, base * 0.7, base * 0.6, base * 0.5, base * 0.4].map(Math.round),
    }
  }
  return { labels, values }
})

const requestTrend = computed(() => {
  const values = hourlyBars.value.values
  if (values.every((v) => v === 0)) return null
  const first = values.slice(0, 6).reduce((a, b) => a + b, 0)
  const second = values.slice(6).reduce((a, b) => a + b, 0)
  if (first === 0) return second > 0 ? 100 : 0
  return Math.round(((second - first) / first) * 100)
})

const healthItems = computed(() => [
  {
    name: 'Fonu',
    desc: '应用核心服务',
    meta: `运行 ${formatUptime(status.value?.uptime_seconds ?? 0)}`,
    status: 'running',
    label: '运行中',
    icon: AppsOutline,
    tone: 'brand',
  },
  {
    name: 'Nginx',
    desc: '反向代理服务',
    meta: status.value?.nginx_status === 'running' ? `运行 ${formatUptime(status.value?.uptime_seconds ?? 0)}` : '已停止',
    status: status.value?.nginx_status,
    label: status.value?.nginx_status === 'running' ? '运行中' : '已停止',
    icon: ServerOutline,
    tone: 'blue',
  },
  {
    name: 'DDNS',
    desc: '域名动态解析',
    meta: formatRelativeTime(status.value?.ddns_last_updated) || '未更新',
    status: status.value?.ddns_status,
    label: status.value?.ddns_status === 'ok' ? '运行中' : '异常',
    icon: CloudOutline,
    tone: 'teal',
  },
  {
    name: 'HTTPS 证书',
    desc: '自动续期管理',
    meta: primaryCert.value ? `${primaryCert.value.days_left} 天后到期` : '未配置',
    status: status.value?.certificate_status,
    label: status.value?.certificate_status === 'ok' ? '运行中' : '需关注',
    icon: ShieldCheckmarkOutline,
    tone: 'green',
  },
])

const accessColumns: DataTableColumns<AccessLogEntry> = [
  { title: '时间', key: 'time', width: 168, render: (r) => formatLogTime(r.time) },
  { title: '域名', key: 'domain', width: 132, ellipsis: { tooltip: true } },
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
  { title: '延迟(ms)', key: 'response_time', width: 80, render: (r) => Math.round(r.response_time * 1000) },
  {
    title: '来源 IP',
    key: 'client_ip',
    width: 210,
    render: (r) => h('span', { class: 'mono ip-cell' }, r.client_ip),
  },
]

const systemColumns: DataTableColumns<SystemLogEntry> = [
  { title: '时间', key: 'time', width: 168, render: (r) => formatLogTime(r.time) },
  {
    title: '级别',
    key: 'level',
    width: 72,
    render: (r) => h(NTag, { size: 'tiny', bordered: false, type: levelTag(r.level) }, () => r.level),
  },
  { title: '模块', key: 'module', width: 80 },
  { title: '内容', key: 'message', ellipsis: { tooltip: true } },
]

function levelTag(level: string) {
  if (level === 'ERROR') return 'error'
  if (level === 'WARN') return 'warning'
  return 'info'
}

function recordRateSample() {
  const now = Date.now()
  rateHistory.value.push({
    at: now,
    upload: trafficTotals.value.uploadRate,
    download: trafficTotals.value.downloadRate,
  })
  const cutoff = now - RATE_HISTORY_MS
  rateHistory.value = rateHistory.value.filter((p) => p.at >= cutoff)
}

async function refreshTraffic() {
  try {
    const rows = asList(await api.getProxyTraffic())
    const next: Record<number, ProxyTraffic> = {}
    for (const row of rows) {
      next[row.rule_id] = row
    }
    trafficByRule.value = next
    recordRateSample()
  } catch {
    // ignore polling errors
  }
}

function startTrafficPoll() {
  stopTrafficPoll()
  trafficTimer = setInterval(refreshTraffic, 2000)
}

function stopTrafficPoll() {
  if (trafficTimer) {
    clearInterval(trafficTimer)
    trafficTimer = null
  }
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const [s, access, system, certs] = await Promise.all([
      api.getStatus(),
      api.getAccessLogs({ limit: 10 }),
      api.getSystemLogs({ limit: 10 }),
      api.listCertificates(),
    ])
    status.value = s
    accessLogs.value = asList(access)
    systemLogs.value = asList(system)
    certificates.value = asList(certs)
    api.listDDNSLite()
      .then((ddns) => {
        ddnsConfigs.value = asList(ddns)
      })
      .catch(() => {})
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
    message.error('加载仪表盘失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  refreshTraffic()
  startTrafficPoll()
})

onUnmounted(stopTrafficPoll)
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
}

.hero-banner {
  position: relative;
  overflow: hidden;
  margin-bottom: var(--fonu-space-4);
  min-height: 208px;
  height: 220px;
  border-radius: 16px;
  box-shadow: var(--fonu-shadow);
  border: 1px solid rgba(16, 185, 129, 0.08);
  background: linear-gradient(100deg, #f4fffc 0%, #f1fbff 55%, #eef9ff 100%);
}

html[data-theme='dark'] .hero-banner,
html.dark .hero-banner {
  background: linear-gradient(100deg, #0f172a 0%, #134e4a 100%);
  border-color: rgba(16, 185, 129, 0.15);
}

.hero-content {
  position: relative;
  z-index: 2;
  display: flex;
  flex-direction: column;
  justify-content: center;
  width: 46%;
  height: 100%;
  padding: 28px 32px;
  box-sizing: border-box;
}

.hero-art {
  position: absolute;
  z-index: 1;
  right: 0;
  bottom: 0;
  height: 100%;
  width: auto;
  max-width: none;
  object-fit: contain;
  object-position: right bottom;
  pointer-events: none;
  user-select: none;
}

html[data-theme='dark'] .hero-art,
html.dark .hero-art {
  opacity: 0.9;
}

.hero-banner__title {
  margin: 0;
  font-size: 30px;
  font-weight: 700;
  color: var(--fonu-text);
  letter-spacing: -0.03em;
}

.hero-banner__desc {
  margin: 10px 0 0;
  font-size: 14px;
  color: var(--fonu-text-secondary);
  max-width: 520px;
  line-height: 1.65;
}

.hero-banner__badges {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 16px;
}

.hero-banner__badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 999px;
  font-size: 13px;
  color: var(--fonu-text-secondary);
  background: var(--fonu-surface);
  border: 1px solid rgba(16, 185, 129, 0.14);
}

.hero-banner__badge .n-icon {
  font-size: 16px;
  color: var(--fonu-brand);
}

@media (max-width: 768px) {
  .hero-content {
    width: 100%;
    padding: 22px 20px;
  }

  .hero-banner {
    height: auto;
    min-height: 180px;
  }

  .hero-art {
    height: 100%;
    opacity: 0.28;
  }

  html[data-theme='dark'] .hero-art,
  html.dark .hero-art {
    opacity: 0.22;
  }

  .hero-banner__title {
    font-size: 24px;
  }

  .hero-banner__badges {
    gap: 8px;
  }
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.stat-primary {
  font-size: 22px;
  font-weight: 700;
  color: var(--fonu-text);
  line-height: 1.3;
}

.stat-desc {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--fonu-text-secondary);
  line-height: 1.5;
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

.traffic-panel {
  display: flex;
  flex-direction: column;
}

.traffic-panel :deep(.fonu-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.requests-panel__body,
.traffic-panel__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.panel-chart {
  flex: 1;
  min-height: 240px;
}

.requests-panel .panel-chart :deep(.chart),
.traffic-panel .panel-chart :deep(.chart) {
  height: 100%;
  min-height: 240px;
}

.traffic-rates {
  display: flex;
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-3);
}

.traffic-rate {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.traffic-rate__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.traffic-rate--up .traffic-rate__dot {
  background: #10b981;
}

.traffic-rate--down .traffic-rate__dot {
  background: #3b82f6;
}

.traffic-rate__label {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.traffic-rate__value {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
  font-variant-numeric: tabular-nums;
}

.metrics-row {
  display: grid;
  grid-template-columns: 3fr 3fr 2fr;
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
  align-items: stretch;
}

.metrics-row > .fonu-card {
  height: 100%;
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

.requests-panel {
  display: flex;
  flex-direction: column;
}

.requests-panel :deep(.fonu-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.health-panel {
  display: flex;
  flex-direction: column;
}

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

.health-panel .panel__head {
  padding: var(--fonu-space-3) var(--fonu-space-4) 0;
}

.health-panel .panel__body {
  flex: 1;
  padding: var(--fonu-space-2) var(--fonu-space-4) var(--fonu-space-3);
}

.health-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
  justify-content: space-between;
  min-height: 0;
}

.health-item {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) 76px;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-height: 64px;
  padding: 12px 14px;
  border-radius: var(--fonu-radius-sm);
  background: var(--fonu-bg);
  min-width: 0;
}

.health-item__icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 17px;
  flex-shrink: 0;
}

.health-item__icon--brand {
  background: var(--fonu-brand-soft);
  color: var(--fonu-brand);
}

.health-item__icon--blue {
  background: rgba(59, 130, 246, 0.12);
  color: #3b82f6;
}

.health-item__icon--teal {
  background: rgba(20, 184, 166, 0.12);
  color: #14b8a6;
}

.health-item__icon--green {
  background: rgba(16, 185, 129, 0.12);
  color: #10b981;
}

.health-item__main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.health-item :deep(.status-badge) {
  justify-self: end;
  width: 100%;
  justify-content: center;
  white-space: nowrap;
}

.health-item__name {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
  line-height: 1.2;
}

.health-item__meta {
  font-size: 12px;
  color: var(--fonu-text-secondary);
  line-height: 1.3;
}

.health-item__detail {
  font-size: 11px;
  color: var(--fonu-text-muted);
  line-height: 1.3;
}

.ip-cell {
  display: inline-block;
  white-space: nowrap;
  font-size: 12px;
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

.mono {
  font-family: var(--fonu-mono);
}

@media (max-width: 1199px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .metrics-row { grid-template-columns: 1fr 1fr; }
  .health-panel { grid-column: 1 / 3; }
  .bottom-row { grid-template-columns: 1fr; }
}

@media (max-width: 767px) {
  .stats-row { grid-template-columns: 1fr; }
  .metrics-row { grid-template-columns: 1fr; }
  .health-panel { grid-column: auto; }
}
</style>
