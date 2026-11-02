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

      <section class="awareness-section">
        <div class="awareness-header">
          <div>
            <h2 class="awareness-title">动态感知</h2>
            <p class="awareness-subtitle">基于反代访问日志，实时识别 IP、服务访问与策略拦截（启发式分析，非专业 WAF）</p>
          </div>
          <div class="awareness-range" role="tablist" aria-label="时间范围">
            <button
              v-for="opt in awarenessRangeOptions"
              :key="opt.key"
              type="button"
              class="awareness-range__btn"
              :class="{ 'awareness-range__btn--active': awarenessRange === opt.key }"
              :aria-selected="awarenessRange === opt.key"
              @click="setAwarenessRange(opt.key)"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>
        <n-alert v-if="awarenessNginxWarning" type="warning" :bordered="false" class="awareness-alert">
          <div class="awareness-alert__content">
            <span>{{ awarenessNginxWarning }}</span>
            <router-link v-if="status?.nginx_available !== false" :to="{ name: 'proxies' }" class="awareness-alert__link">
              前往反向代理
            </router-link>
          </div>
        </n-alert>
        <n-alert v-else type="success" :bordered="false" class="awareness-alert">
          数据更新至 {{ awarenessUpdatedLabel }} · 仅统计 Fonu 反代流量
        </n-alert>

        <div class="awareness-kpi-row">
          <div class="awareness-kpi">
            <span class="awareness-kpi__label">独立 IP</span>
            <span class="awareness-kpi__value">{{ (awareness?.today.unique_ips ?? 0).toLocaleString() }}</span>
          </div>
          <div class="awareness-kpi">
            <span class="awareness-kpi__label">{{ awarenessVisitLabel }}</span>
            <span class="awareness-kpi__value">{{ (awareness?.today.total_requests ?? 0).toLocaleString() }}</span>
          </div>
          <div class="awareness-kpi awareness-kpi--danger">
            <span class="awareness-kpi__label">拦截次数</span>
            <router-link
              :to="{ name: 'logs', query: { tab: 'access', kind: 'intercept' } }"
              class="awareness-kpi__value awareness-kpi__link"
            >
              {{ (awareness?.today.blocked_total ?? 0).toLocaleString() }}
            </router-link>
            <span v-if="awareness" class="awareness-kpi__hint">
              403 {{ awareness.today.blocked_403 }} · 429 {{ awareness.today.rate_limited_429 }} · 401 {{ awareness.today.auth_failed_401 }}
            </span>
          </div>
        </div>

        <div class="awareness-funnel">
          <div class="awareness-funnel__step">
            <span class="awareness-funnel__label">外部总访问</span>
            <span class="awareness-funnel__value">{{ (awareness?.today.total_requests ?? 0).toLocaleString() }} 次</span>
          </div>
          <div class="awareness-funnel__arrow" aria-hidden="true">→</div>
          <div class="awareness-funnel__split">
            <div class="awareness-funnel__box awareness-funnel__box--safe">
              <span class="awareness-funnel__box-label">安全访问</span>
              <span class="awareness-funnel__box-value">{{ (awareness?.today.safe ?? 0).toLocaleString() }}</span>
            </div>
            <div class="awareness-funnel__box awareness-funnel__box--block">
              <span class="awareness-funnel__box-label">策略拦截</span>
              <router-link
                :to="{ name: 'logs', query: { tab: 'access', kind: 'intercept' } }"
                class="awareness-funnel__box-value awareness-funnel__link"
              >
                {{ (awareness?.today.blocked_total ?? 0).toLocaleString() }}
              </router-link>
            </div>
            <div class="awareness-funnel__box awareness-funnel__box--warn">
              <span class="awareness-funnel__box-label">疑似扫描/异常</span>
              <span class="awareness-funnel__box-value">{{ awarenessScanTotal }}</span>
            </div>
          </div>
        </div>

        <FonuCard class="panel awareness-charts-panel">
          <div class="awareness-charts-panel__block">
            <div class="panel__head">
              <div class="panel-title">
                <n-icon :component="FlashOutline" class="panel-title__icon panel-title__icon--amber" />
                <span>{{ awarenessTrendTitle }}</span>
              </div>
              <n-tag size="small" :bordered="false" class="range-tag">{{ awarenessTrendTag }}</n-tag>
            </div>
            <div class="panel__body awareness-charts-panel__trend">
              <AwarenessTrendChart
                v-if="awareness"
                :labels="awareness.hourly_labels"
                :requests="awareness.hourly_requests"
                :uniqueIPs="awareness.hourly_unique_ips"
                :current-bucket="awareness.current_hour_bucket"
              />
              <EmptyState v-else title="加载中" description="正在连接感知引擎…" />
            </div>
          </div>
          <div class="awareness-charts-panel__divider" aria-hidden="true" />
          <div class="awareness-charts-panel__block awareness-charts-panel__block--traffic">
            <div class="awareness-traffic-panel__head">
              <div class="panel-title">
                <n-icon :component="SwapVerticalOutline" class="panel-title__icon panel-title__icon--teal" />
                <span>实时流量</span>
                <n-tag size="small" :bordered="false" class="range-tag">最近 5 分钟</n-tag>
              </div>
              <div class="traffic-rates traffic-rates--compact">
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
            </div>
            <div class="awareness-traffic-panel__chart">
              <MiniTrafficChart compact :labels="trafficChart.labels" :upload="trafficChart.upload" :download="trafficChart.download" />
            </div>
          </div>
        </FonuCard>

        <div class="awareness-lists">
          <FonuCard flush class="awareness-list-card">
            <template #title>
              <n-icon :component="EyeOutline" class="card-title-icon" />
              <span>访问记录 · 最近 30 条</span>
            </template>
            <template #header>
              <div class="awareness-list-card__header-actions">
                <n-tag size="small" :bordered="false">{{ awarenessRecentScopeLabel }}</n-tag>
                <router-link
                  :to="{ name: 'logs', query: { tab: 'access', kind: 'intercept' } }"
                  class="card-link"
                >
                  拦截记录
                  <n-icon :component="ChevronForwardOutline" />
                </router-link>
                <router-link :to="{ name: 'logs', query: { tab: 'access' } }" class="card-link">
                  查看全部
                  <n-icon :component="ChevronForwardOutline" />
                </router-link>
              </div>
            </template>
            <div class="awareness-list-card__body">
              <n-data-table
                v-if="awarenessEvents.length"
                class="awareness-table"
                :columns="awarenessEventColumns"
                :data="awarenessEvents"
                :bordered="false"
                :max-height="awarenessTableScrollHeight"
                size="small"
              />
              <EmptyState v-else title="暂无访问" description="access.log 中尚无反代访问记录（已排除 favicon 等噪声）。" />
            </div>
          </FonuCard>

          <FonuCard flush class="awareness-list-card">
            <template #title>
              <n-icon :component="GitNetworkOutline" class="card-title-icon" />
              <span>按服务汇总</span>
            </template>
            <template #header>
              <div class="awareness-list-card__header-actions">
                <n-tag size="small" :bordered="false" class="range-tag">{{ awarenessPeriodLabel }}</n-tag>
                <span v-if="awarenessServiceListHint" class="awareness-list-card__meta">{{ awarenessServiceListHint }}</span>
              </div>
            </template>
            <div class="awareness-list-card__body">
              <n-data-table
                v-if="awarenessServices.length"
                class="awareness-table"
                :columns="awarenessServiceColumns"
                :data="awarenessServices"
                :bordered="false"
                :max-height="awarenessTableScrollHeight"
                size="small"
              />
              <EmptyState v-else title="暂无数据" description="访问各反代服务后将出现汇总。" />
            </div>
          </FonuCard>
        </div>
      </section>
    </n-spin>
  </template>
</template>

<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, ref } from 'vue'
import { NAlert, NDataTable, NIcon, NSpin, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import {
  CheckmarkCircleOutline,
  ChevronForwardOutline,
  EyeOutline,
  FlashOutline,
  GitNetworkOutline,
  GlobeOutline,
  ShieldCheckmarkOutline,
  SwapVerticalOutline,
  WifiOutline,
} from '@vicons/ionicons5'
import { RouterLink } from 'vue-router'
import bannerImg from '../assets/brand/banner.png'
import { api, asList } from '../api/client'
import type {
  AwarenessEvent,
  AwarenessEventKind,
  AwarenessLiveSnapshot,
  AwarenessRange,
  AwarenessServiceRow,
  CertificateRecord,
  DashboardStatus,
  DDNSConfig,
  ProxyTraffic,
} from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import AwarenessTrendChart from '../components/AwarenessTrendChart.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import MiniTrafficChart from '../components/MiniTrafficChart.vue'
import StatCard from '../components/StatCard.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useAppTimezone } from '../composables/useAppTimezone'
import { formatDate, formatLogTime, formatRate, formatRelativeTime } from '../utils/format'
import { httpStatusKind } from '../utils/status'

const message = useMessage()
const logTimeZone = useAppTimezone()
const status = ref<DashboardStatus | null>(null)
const awareness = ref<AwarenessLiveSnapshot | null>(null)
const awarenessRange = ref<AwarenessRange>('today')

const awarenessRangeOptions: { key: AwarenessRange; label: string }[] = [
  { key: 'today', label: '今天' },
  { key: '7d', label: '最近 7 天' },
  { key: '30d', label: '最近 30 天' },
  { key: 'all', label: '全部' },
]
const ddnsConfigs = ref<DDNSConfig[]>([])
const certificates = ref<CertificateRecord[]>([])
const loading = ref(false)
const loadError = ref('')
const trafficByRule = ref<Record<number, ProxyTraffic>>({})
type RatePoint = { at: number; upload: number; download: number }
const rateHistory = ref<RatePoint[]>([])
const RATE_HISTORY_MS = 5 * 60 * 1000
let trafficTimer: ReturnType<typeof setInterval> | null = null
let awarenessTimer: ReturnType<typeof setInterval> | null = null
const AWARENESS_POLL_MS = 5000
const awarenessTableScrollHeight = 360

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

const coreServiceCountLabel = computed(() => '4 个')

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

const awarenessUpdatedLabel = computed(() => {
  const raw = awareness.value?.updated_at
  if (!raw) return '—'
  return formatRelativeTime(raw) || formatLogTime(raw, { timeZone: logTimeZone.value })
})

const nginxNotRunningForAwareness = computed(
  () => status.value?.nginx_status != null && status.value.nginx_status !== 'running',
)

const awarenessNginxWarning = computed((): string | null => {
  if (!nginxNotRunningForAwareness.value) return null
  const bin = status.value?.nginx_bin?.trim() || 'nginx'
  if (status.value?.nginx_available === false) {
    return `未找到 Nginx 可执行文件（${bin}）。本地开发请安装 nginx 或设置 FONU_NGINX_BIN；配置仍可保存，但不会产生 access.log，动态感知与「最近访问」将为空。`
  }
  return 'Nginx 尚未运行。请在「反向代理」保存规则以启动进程，再通过反代端口（默认 18080 / 9443）产生访问后，动态感知才会有数据。'
})

const awarenessScanTotal = computed(() => {
  const t = awareness.value?.today
  if (!t) return '0'
  return (t.scan_suspected + t.abnormal_burst).toLocaleString()
})

const awarenessPeriodLabel = computed(() => {
  switch (awarenessRange.value) {
    case '7d':
      return '近 7 天'
    case '30d':
      return '近 30 天'
    case 'all':
      return '全部'
    default:
      return '当日'
  }
})

const awarenessVisitLabel = computed(() => {
  if (awarenessRange.value === 'today') return '当日访问'
  return `${awarenessPeriodLabel.value}访问`
})

const awarenessTrendTitle = computed(() => {
  if (awarenessRange.value === 'today') return '今日流量趋势'
  return `${awarenessPeriodLabel.value}流量趋势`
})

const awarenessTrendTag = computed(() => {
  const g = awareness.value?.trend_granularity
  if (g === 'daily') return '按日统计'
  return '2 小时粒度 · 已过去时段'
})

const awarenessRecentScopeLabel = '全部访问'

function setAwarenessRange(r: AwarenessRange) {
  if (awarenessRange.value === r) return
  awarenessRange.value = r
  void refreshAwareness()
}

const awarenessEvents = computed(() => awareness.value?.recent_events ?? [])
const awarenessServices = computed(() => awareness.value?.by_service ?? [])

/** Backend returns at most 20 services sorted by request volume. */
const awarenessServiceListHint = computed(() => {
  const n = awarenessServices.value.length
  if (n === 0) return ''
  if (n >= 20) return '按请求量 Top 20，非全部服务'
  return `共 ${n} 个服务`
})

function renderAccessIPCell(row: { client_ip: string; geo?: string }) {
  const geo = row.geo?.trim()
  return h('div', { class: 'ip-geo-cell' }, [
    h(
      RouterLink,
      {
        to: { name: 'logs', query: { tab: 'access', ip: row.client_ip } },
        class: 'mono ip-geo-cell__ip',
        title: row.client_ip,
      },
      () => row.client_ip,
    ),
    geo
      ? h('span', { class: 'ip-geo-cell__geo', title: geo }, geo)
      : h('span', { class: 'ip-geo-cell__geo ip-geo-cell__geo--empty' }, '—'),
  ])
}

function awarenessKindTag(kind: AwarenessEventKind): {
  type: 'success' | 'warning' | 'error' | 'default'
  label: string
} {
  switch (kind) {
    case 'blocked':
    case 'rate_limited':
    case 'auth_failed':
      return { type: 'error', label: '拦截' }
    case 'scan_suspected':
      return { type: 'warning', label: '扫描' }
    case 'abnormal_burst':
      return { type: 'warning', label: '异常' }
    default:
      return { type: 'success', label: '正常' }
  }
}

const awarenessEventColumns: DataTableColumns<AwarenessEvent> = [
  {
    title: '时间',
    key: 'at',
    width: 100,
    render: (r) => formatRelativeTime(r.at) || formatLogTime(r.at, { timeZone: logTimeZone.value }),
  },
  { title: 'IP / 归属', key: 'client_ip', width: 132, render: (r) => renderAccessIPCell(r) },
  { title: '服务', key: 'service', width: 108, ellipsis: { tooltip: true } },
  {
    title: '路径',
    key: 'path',
    minWidth: 140,
    ellipsis: { tooltip: true },
    render: (r) => h('span', { class: 'mono', title: r.path }, r.path),
  },
  {
    title: '状态',
    key: 'status',
    width: 56,
    render: (r) => {
      const kind = httpStatusKind(r.status)
      const type = kind === 'success' ? 'success' : kind === 'warning' ? 'warning' : 'error'
      return h(NTag, { size: 'tiny', bordered: false, type }, () => String(r.status))
    },
  },
  {
    title: '类型',
    key: 'kind',
    width: 64,
    render: (r) => {
      if (r.kind === 'access') {
        return h('span', { class: 'awareness-muted' }, '—')
      }
      const t = awarenessKindTag(r.kind)
      return h(NTag, { size: 'small', bordered: false, type: t.type }, () => t.label)
    },
  },
]

const awarenessServiceColumns: DataTableColumns<AwarenessServiceRow> = [
  { title: '服务', key: 'service', ellipsis: { tooltip: true } },
  {
    title: '请求',
    key: 'requests',
    width: 72,
    align: 'right',
    titleAlign: 'right',
    render: (r) => h('span', { class: 'tabular-nums' }, String(r.requests)),
  },
  {
    title: '独立 IP',
    key: 'unique_ips',
    width: 80,
    align: 'right',
    titleAlign: 'right',
    render: (r) => h('span', { class: 'tabular-nums' }, String(r.unique_ips)),
  },
  {
    title: '拦截',
    key: 'blocked',
    width: 64,
    align: 'right',
    titleAlign: 'right',
    render: (r) => h('span', { class: 'tabular-nums' }, String(r.blocked)),
  },
  {
    title: '疑似',
    key: 'scan_suspected',
    width: 64,
    align: 'right',
    titleAlign: 'right',
    render: (r) => h('span', { class: 'tabular-nums' }, String(r.scan_suspected)),
  },
]

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

async function refreshAwareness() {
  try {
    awareness.value = await api.getAwarenessLive(awarenessRange.value)
    if (status.value && awareness.value && awarenessRange.value === 'today') {
      status.value = {
        ...status.value,
        request_today: awareness.value.today.total_requests,
        requests_hourly: awareness.value.hourly_requests,
        requests_hourly_labels: awareness.value.hourly_labels,
        error_today: awareness.value.today.blocked_total,
      }
    }
  } catch {
    // ignore poll errors
  }
}

function startAwarenessPoll() {
  stopAwarenessPoll()
  awarenessTimer = setInterval(refreshAwareness, AWARENESS_POLL_MS)
}

function stopAwarenessPoll() {
  if (awarenessTimer) {
    clearInterval(awarenessTimer)
    awarenessTimer = null
  }
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
    status.value = await api.getStatus()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
    message.error('加载仪表盘失败')
    return
  } finally {
    loading.value = false
  }
  void loadDashboardExtras()
}

async function loadDashboardExtras() {
  try {
    certificates.value = asList(await api.listCertificates())
  } catch {
    // 次要区块失败不阻断首页
  }
  api.listDDNSLite()
    .then((ddns) => {
      ddnsConfigs.value = asList(ddns)
    })
    .catch(() => {})
}

onMounted(() => {
  load()
  refreshAwareness()
  refreshTraffic()
  startTrafficPoll()
  startAwarenessPoll()
})

onUnmounted(() => {
  stopTrafficPoll()
  stopAwarenessPoll()
})
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

.awareness-section {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.awareness-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  flex-wrap: wrap;
}

.awareness-title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--fonu-text);
}

.awareness-subtitle {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--fonu-text-secondary);
  max-width: 640px;
  line-height: 1.55;
}

.awareness-range {
  display: inline-flex;
  flex-shrink: 0;
  padding: 3px;
  border-radius: 10px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  gap: 2px;
}

.awareness-range__btn {
  border: none;
  background: transparent;
  padding: 6px 12px;
  font-size: 13px;
  border-radius: 8px;
  color: var(--fonu-text-secondary);
  cursor: pointer;
  white-space: nowrap;
  line-height: 1.2;
}

.awareness-range__btn--active {
  background: var(--fonu-brand);
  color: #fff;
  font-weight: 600;
}

.awareness-range__btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.awareness-range__btn:not(:disabled):not(.awareness-range__btn--active):hover {
  background: rgba(16, 185, 129, 0.08);
  color: var(--fonu-text);
}

.awareness-alert {
  border-radius: 12px;
  line-height: 1.55;
}

.awareness-alert__content {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
}

.awareness-alert__link {
  display: inline-flex;
  align-items: center;
  font-weight: 600;
  color: var(--fonu-brand);
  text-decoration: none;
  white-space: nowrap;
}

.awareness-alert__link:hover {
  text-decoration: underline;
}

.awareness-kpi-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--fonu-space-4);
}

.awareness-kpi {
  padding: var(--fonu-space-4) var(--fonu-space-5);
  border-radius: 14px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.awareness-kpi--danger {
  border-color: rgba(239, 68, 68, 0.22);
}

.awareness-kpi--danger .awareness-kpi__value {
  color: #ef4444;
}

.awareness-kpi__label {
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.awareness-kpi__value {
  font-size: 28px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--fonu-text);
}

.awareness-kpi__link,
.awareness-funnel__link {
  text-decoration: none;
  color: inherit;
}

.awareness-kpi__link:hover,
.awareness-funnel__link:hover {
  text-decoration: underline;
  color: var(--fonu-danger, #d03050);
}

.awareness-kpi__hint {
  font-size: 11px;
  color: var(--fonu-text-muted);
}

.awareness-funnel {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-4);
  flex-wrap: wrap;
  padding: var(--fonu-space-4) var(--fonu-space-5);
  border-radius: 14px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
}

.awareness-funnel__step {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 120px;
}

.awareness-funnel__label {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.awareness-funnel__value {
  font-size: 20px;
  font-weight: 700;
}

.awareness-funnel__arrow {
  font-size: 20px;
  color: var(--fonu-text-muted);
}

.awareness-funnel__split {
  display: flex;
  flex: 1;
  gap: var(--fonu-space-3);
  flex-wrap: wrap;
}

.awareness-funnel__box {
  flex: 1;
  min-width: 140px;
  padding: 12px 16px;
  border-radius: 12px;
}

.awareness-funnel__box--safe {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.25);
}

.awareness-funnel__box--block {
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.2);
}

.awareness-funnel__box--warn {
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.25);
}

.awareness-funnel__box-label {
  display: block;
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.awareness-funnel__box-value {
  font-size: 18px;
  font-weight: 700;
  margin-top: 4px;
  font-variant-numeric: tabular-nums;
}

.awareness-funnel__box--safe .awareness-funnel__box-value {
  color: #059669;
}

.awareness-funnel__box--block .awareness-funnel__box-value {
  color: #dc2626;
}

.awareness-funnel__box--warn .awareness-funnel__box-value {
  color: #d97706;
}

.awareness-lists {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--fonu-space-5);
  align-items: stretch;
}

.awareness-list-card :deep(.fonu-card__header) {
  padding-bottom: var(--fonu-space-3);
  gap: var(--fonu-space-3);
}

.awareness-list-card {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.awareness-list-card :deep(.fonu-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding-top: 0;
}

.awareness-list-card__body {
  flex: 1;
  padding: 0 var(--fonu-space-4) var(--fonu-space-4);
  min-height: 0;
}

.awareness-table :deep(.n-data-table-th) {
  background: var(--fonu-bg-muted) !important;
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
  padding: 10px 12px !important;
}

.awareness-table :deep(.n-data-table-td) {
  font-size: 13px;
  padding: 8px 12px !important;
  vertical-align: top;
}

.awareness-muted {
  color: var(--fonu-text-muted);
  font-size: 12px;
}

.awareness-table :deep(.n-data-table-th:last-child),
.awareness-table :deep(.n-data-table-td:last-child) {
  padding-right: 12px !important;
}

.awareness-table :deep(.n-data-table-th:first-child),
.awareness-table :deep(.n-data-table-td:first-child) {
  padding-left: 12px !important;
}

.awareness-table :deep(.n-scrollbar-rail),
.awareness-table :deep(.n-scrollbar-rail__scrollbar) {
  display: none !important;
}

.awareness-table :deep(.n-data-table-base-table-body) {
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.awareness-table :deep(.n-data-table-base-table-body)::-webkit-scrollbar {
  display: none;
}

.awareness-list-card__meta {
  font-size: 12px;
  color: var(--fonu-text-muted);
  white-space: nowrap;
}

.awareness-list-card__header-actions {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  flex-wrap: wrap;
}

.awareness-charts-panel :deep(.fonu-card__body) {
  padding: 0;
}

.awareness-charts-panel__divider {
  height: 1px;
  margin: 0 var(--fonu-space-5);
  background: var(--fonu-border);
}

.awareness-charts-panel__trend {
  min-height: 0;
  padding-bottom: var(--fonu-space-2);
}

.awareness-charts-panel__trend :deep(.chart) {
  height: 220px;
}

.awareness-charts-panel__block--traffic {
  padding-bottom: var(--fonu-space-2);
}

.awareness-traffic-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  flex-wrap: wrap;
  padding: var(--fonu-space-4) var(--fonu-space-5) var(--fonu-space-2);
}

.awareness-traffic-panel__head .panel-title {
  flex-wrap: wrap;
  gap: 8px;
}

.traffic-rates--compact {
  margin-bottom: 0;
  flex-shrink: 0;
}

.awareness-traffic-panel__chart {
  padding: 0 var(--fonu-space-5) var(--fonu-space-5);
}

.tabular-nums {
  font-variant-numeric: tabular-nums;
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
.traffic-panel:not(.awareness-traffic-panel) .panel-chart :deep(.chart) {
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

.awareness-table :deep(.ip-geo-cell) {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  width: 100%;
  line-height: 1.35;
}

.awareness-table :deep(.ip-geo-cell__ip) {
  display: block;
  width: 100%;
  font-size: 12px;
  font-weight: 500;
  color: var(--fonu-text);
  text-decoration: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.awareness-table :deep(a.ip-geo-cell__ip:hover) {
  color: var(--fonu-brand);
  text-decoration: underline;
}

.awareness-table :deep(.ip-geo-cell__geo) {
  display: block;
  width: 100%;
  font-size: 11px;
  color: var(--fonu-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.awareness-table :deep(.ip-geo-cell__geo--empty) {
  opacity: 0.45;
}

.ip-cell {
  display: inline-block;
  white-space: nowrap;
  font-size: 12px;
}

.ip-geo-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  min-width: 0;
  width: 100%;
}

.ip-geo-cell__ip {
  display: block;
  width: 100%;
  font-size: 12px;
  line-height: 1.35;
  font-weight: 500;
  color: var(--fonu-text);
  text-decoration: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

a.ip-geo-cell__ip:hover {
  color: var(--fonu-brand);
  text-decoration: underline;
}

.ip-geo-cell__geo {
  display: block;
  width: 100%;
  font-size: 11px;
  line-height: 1.35;
  color: var(--fonu-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ip-geo-cell__geo--empty {
  opacity: 0.45;
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
  .awareness-kpi-row,
  .awareness-lists {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 767px) {
  .stats-row { grid-template-columns: 1fr; }
  .awareness-traffic-panel__head {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
