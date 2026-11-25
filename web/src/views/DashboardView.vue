<template>
  <div class="dashboard">
  <nav class="dashboard-tabs" aria-label="首页分区"><button v-for="tab in tabs" :key="tab.key" type="button" :aria-pressed="activeTab === tab.key" @click="selectTab(tab.key)">{{ tab.label }}</button></nav>
  <LoadError v-if="loadError" :message="loadError" @retry="load" />

    <n-spin v-if="activeTab === 'overview' || activeTab === 'services'" :show="loading && !status">
      <div class="stats-row">
        <StatCard label="公网 IP" tone="blue">
          <template #icon><n-icon :component="GlobeOutline" /></template>
          <template #extra><StatusBadge v-if="publicIPv4Label !== '-'" value="ok" text="已获取" /></template>
          <template #value>
            <span class="stat-primary mono">{{ publicIPv4Label }}</span>
          </template>
          <p class="stat-desc">{{ publicIPv6Label }} · {{ publicIPSourceLabel }}</p>
        </StatCard>

        <StatCard label="域名" tone="green">
          <template #icon><n-icon :component="WifiOutline" /></template>
          <template #extra><StatusBadge :value="status?.ddns_status" /></template>
          <template #value>{{ domainCountLabel }}</template>
          <p class="stat-desc">{{ domainExamplesLabel }}</p>
        </StatCard>

        <StatCard label="证书" tone="blue">
          <template #icon><n-icon :component="ShieldCheckmarkOutline" /></template>
          <template #extra><StatusBadge :value="status?.certificate_status" /></template>
          <template #value>{{ certCountLabel }}</template>
          <p v-if="primaryCert" class="stat-desc">
            {{ certDisplayName }} · 剩余 {{ primaryCert.days_left }} 天
          </p>
          <p v-else class="stat-desc">还没有 HTTPS 证书</p>
        </StatCard>

        <StatCard label="Nginx" tone="teal">
          <template #icon><n-icon :component="GitNetworkOutline" /></template>
          <template #extra><StatusBadge :value="status?.nginx_status" /></template>
          <template #value>{{ nginxRunningLabel }}</template>
          <p class="stat-desc">反代进程</p>
        </StatCard>
      </div>
    </n-spin>


    <FonuCard v-if="activeTab === 'notifications'" title="状态提醒" subtitle="当前需要关注的证书与服务状态">
      <div v-if="!status">尚未获取服务状态</div>
      <div v-for="notice in notices" :key="notice.title" class="notice-row"><span class="notice-icon"><n-icon :component="ShieldCheckmarkOutline"/></span><div class="notice-detail"><strong>{{ notice.title }}</strong><p>{{ notice.detail }}</p></div><router-link :to="{name:notice.route}">查看</router-link></div>
      <EmptyState v-if="status && !notices.length" title="暂无状态提醒" description="当前已获取的状态中未发现需要处理的事项。" />
      <router-link :to="{name:'settings',query:{tab:'notify'}}">通知渠道设置</router-link>
      <router-link class="notification-version" :to="{name:'settings',query:{tab:'version'}}">版本与更新日志</router-link>
    </FonuCard>
    <AnalyticsInsightSection v-if="activeTab === 'security' || activeTab === 'services'" v-model:range="awarenessRange" :show-map="false" :show-kpis="false" :section-title="activeTab === 'security' ? '安全统计范围' : '服务统计范围'" :data="analyticsOverview" :loading="awarenessLoading" :error="awarenessError" @retry="refreshAwareness(true)" />

    

      <section v-if="activeTab === 'overview' || activeTab === 'access' || activeTab === 'services'" class="awareness-section">
        <div v-if="activeTab === 'services'" class="awareness-header">
          <div>
            <h2 class="awareness-title">趋势与记录</h2>
            <p class="awareness-subtitle">当前展示：{{ awarenessPeriodLabel }}，与上方已加载统计一致。</p>
          </div>
          <span v-if="analyticsOverview" class="awareness-updated">更新于 {{ awarenessUpdatedLabel }}</span>
        </div>
        <n-alert v-if="awarenessNginxWarning" type="warning" :bordered="false" class="awareness-alert">
          <div class="awareness-alert__content">
            <span>{{ awarenessNginxWarning }}</span>
            <router-link v-if="status?.nginx_available !== false" :to="{ name: 'proxies' }" class="awareness-alert__link">
              前往反向代理
            </router-link>
          </div>
        </n-alert>

        <div v-if="activeTab !== 'services'" class="trend-workspace" :class="{ 'trend-workspace--overview': activeTab === 'overview' }"><FonuCard class="panel awareness-charts-panel">
          <AnalyticsInsightSection v-model:range="awarenessRange" compact :show-map="false" section-title="访问趋势" :data="analyticsOverview" :loading="awarenessLoading" :error="awarenessError" @retry="refreshAwareness(true)" />
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
                v-if="analyticsOverview"
                :labels="analyticsOverview.series.labels"
                :requests="analyticsOverview.series.requests"
                :uniqueIPs="analyticsOverview.series.unique_ips"
                :current-bucket="analyticsOverview.series.current_bucket"
              />
              <EmptyState v-else title="正在读取" description="访问趋势马上出来。" />
            </div>
          </div>
        </FonuCard>

        <div v-if="activeTab === 'overview'" class="overview-side"><FonuCard title="快捷操作" class="quick-card"><div class="quick-links"><router-link v-for="item in quickEntries" :key="item.route" :to="{name:item.route}"><span class="quick-icon"><n-icon :component="item.icon"/></span>{{ item.label }}<n-icon :component="ChevronForwardOutline" class="quick-arrow"/></router-link></div></FonuCard><FonuCard title="状态提醒"><template #header><router-link class="card-link" :to="{name:'dashboard',query:{tab:'notifications'}}">查看全部 <n-icon :component="ChevronForwardOutline"/></router-link></template><div v-for="notice in notices" :key="notice.title" class="notice-row"><span class="notice-icon"><n-icon :component="ShieldCheckmarkOutline"/></span><div class="notice-detail"><strong>{{ notice.title }}</strong><p>{{ notice.detail }}</p></div></div><p v-if="!notices.length" class="stat-desc">{{ status ? '暂无待处理提醒' : '正在获取服务状态' }}</p></FonuCard></div></div>
        <AnalyticsInsightSection v-if="activeTab === 'overview' || activeTab === 'access'" v-model:range="awarenessRange" map-only show-map :data="analyticsOverview" :loading="awarenessLoading" />
        <div class="awareness-lists">
          <FonuCard v-if="activeTab === 'access' || activeTab === 'overview'" flush class="awareness-list-card">
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
                :data="activeTab === 'overview' ? awarenessEvents.slice(0,5) : awarenessEvents"
                :bordered="false"
                :max-height="awarenessTableScrollHeight"
                size="small"
              />
              <EmptyState v-else title="暂无访问" description="access.log 中尚无反代访问记录（已排除 favicon 等噪声）。" />
            </div>
          </FonuCard>

          <FonuCard v-if="activeTab === 'services'" flush class="awareness-list-card">
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
      <AnalyticsAnomalySection v-if="activeTab === 'security' && analyticsOverview?.anomaly" :data="analyticsOverview" />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, h, onActivated, onDeactivated, onUnmounted, ref, shallowRef, watch } from 'vue'
import { NAlert, NDataTable, NIcon, NSpin, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import {
  ChevronForwardOutline,
  EyeOutline,
  FlashOutline,
  GitNetworkOutline,
  GlobeOutline,
  ShieldCheckmarkOutline,
  WifiOutline,
} from '@vicons/ionicons5'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { api, asList } from '../api/client'
import type {
  AnalyticsOverview,
  AwarenessEvent,
  AwarenessEventKind,
  AwarenessRange,
  AwarenessServiceRow,
  CertificateRecord,
  DashboardStatus,
  DDNSConfig,
} from '../api/types'
import EmptyState from '../components/EmptyState.vue'
const AnalyticsInsightSection = defineAsyncComponent(() => import('../components/AnalyticsInsightSection.vue'))
const AnalyticsAnomalySection = defineAsyncComponent(() => import('../components/AnalyticsAnomalySection.vue'))
const AwarenessTrendChart = defineAsyncComponent(() => import('../components/AwarenessTrendChart.vue'))
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import StatCard from '../components/StatCard.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useAppTimezone } from '../composables/useAppTimezone'
import { formatLogTime, formatRelativeTime } from '../utils/format'
import { displayGeoLabel } from '../utils/geoDisplay'
import { httpStatusKind } from '../utils/status'

defineOptions({ name: 'DashboardView' })

const route = useRoute()
const router = useRouter()
const tabs = [{key:'overview',label:'概览'},{key:'notifications',label:'通知'},{key:'access',label:'访问'},{key:'security',label:'安全'},{key:'services',label:'服务'}] as const
const activeTab = computed(() => tabs.find(t => t.key === route.query.tab)?.key ?? 'overview')
function selectTab(key: string) { void router.replace({name:'dashboard',query:key === 'overview' ? {} : {tab:key}}) }
const quickEntries = [{route:'ddns',label:'DDNS',icon:GlobeOutline},{route:'certificates',label:'证书管理',icon:ShieldCheckmarkOutline},{route:'proxies',label:'反向代理',icon:GitNetworkOutline},{route:'frp',label:'内网穿透',icon:WifiOutline},{route:'logs',label:'日志中心',icon:EyeOutline},{route:'settings',label:'系统设置',icon:GitNetworkOutline}]
const message = useMessage()
const logTimeZone = useAppTimezone()
const status = ref<DashboardStatus | null>(null)
const analyticsOverview = shallowRef<AnalyticsOverview | null>(null)
const displayedRange = computed(() => analyticsOverview.value?.range ?? awarenessRange.value)
const awarenessLoading = ref(false)
const awarenessError = ref('')
const awarenessRange = ref<AwarenessRange>('today')
const ddnsConfigs = ref<DDNSConfig[]>([])
const certificates = ref<CertificateRecord[]>([])
const loading = ref(false)
const loadError = ref('')
let awarenessTimer: ReturnType<typeof setInterval> | null = null
const pollInterval = () => awarenessRange.value === 'today' ? 15000 : awarenessRange.value === '7d' ? 30000 : 60000
const awarenessTableScrollHeight = 520
const notices = computed(() => {
  const rows: {title:string;detail:string;route:string}[] = []
  for (const cert of certificates.value.filter(c => c.days_left <= 30)) rows.push({title:cert.domain,detail:cert.days_left < 0 ? '证书已过期' : '证书剩余 '+cert.days_left+' 天',route:'certificates'})
  if (status.value?.nginx_status === 'stopped') rows.push({title:'Nginx 已停止',detail:'检查反向代理配置与运行状态',route:'proxies'})
  return rows
})

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
      return status.value?.public_ipv4 || status.value?.public_ipv6 ? 'DDNS' : 'DDNS 等待同步'
    case 'detect':
      return '出口探测'
    default:
      return '暂无来源'
  }
})

const nginxRunningLabel = computed(() => {
  switch (status.value?.nginx_status) {
    case 'running':
      return '运行中'
    case 'stopped':
      return '已停止'
    default:
      return '—'
  }
})

const domainCountLabel = computed(() => {
  const count = status.value?.ddns_count ?? ddnsConfigs.value.length
  if (count === 0) return '未配置'
  return `${count} 个`
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
  return examples.length > 0 ? examples.join(', ') : '还没有域名'
})

const certCountLabel = computed(() => {
  const count = status.value?.certificate_count ?? certificates.value.length
  if (count === 0) return '未申请'
  return `${count} 个`
})

const awarenessUpdatedLabel = computed(() => {
  const raw = analyticsOverview.value?.updated_at
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

const awarenessPeriodLabel = computed(() => {
  switch (displayedRange.value) {
    case '7d':
      return '近 7 天'
    case '30d':
      return '近 30 天'
    case 'all':
      return '近一年'
    default:
      return '当日'
  }
})

const awarenessTrendTitle = computed(() => {
  if (displayedRange.value === 'today') return '今日流量趋势'
  return `${awarenessPeriodLabel.value}流量趋势`
})

const awarenessTrendTag = computed(() => {
  const g = analyticsOverview.value?.series?.granularity
  if (g === 'daily') return '按日统计'
  return '2 小时粒度 · 已过去时段'
})

const awarenessRecentScopeLabel = computed(() => awarenessPeriodLabel.value)

watch(awarenessRange, () => {
  cancelAwareness()
  awarenessLoading.value = true
  awarenessError.value = ''
  void refreshAwareness(true)
})

const awarenessEvents = computed(() => analyticsOverview.value?.recent_events ?? [])
const awarenessServices = computed(() => analyticsOverview.value?.by_service ?? [])

/** Backend returns at most 20 services sorted by request volume. */
const awarenessServiceListHint = computed(() => {
  const n = awarenessServices.value.length
  if (n === 0) return ''
  if (n >= 20) return '按请求量 Top 20，非全部服务'
  return `共 ${n} 个服务`
})

function renderAccessIPCell(row: { client_ip: string; geo?: string }) {
  const geoRaw = row.geo?.trim()
  const geo = geoRaw ? displayGeoLabel(geoRaw) : ''
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
      ? h('span', { class: 'ip-geo-cell__geo', title: geoRaw }, geo)
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

let awarenessController: AbortController | null = null
let dashboardActive = false
function cancelAwareness() {
  awarenessController?.abort()
  awarenessController = null
  awarenessLoading.value = false
  stopAwarenessPoll()
}
async function refreshAwareness(force = false) {
  if (!dashboardActive || document.hidden || awarenessController) return
  const range = awarenessRange.value
  const controller = new AbortController()
  awarenessController = controller
  if (force || !analyticsOverview.value) awarenessLoading.value = true
  awarenessError.value = ''
  try {
    const overview = await api.getAnalyticsOverview(range, controller.signal)
    if (!controller.signal.aborted && awarenessRange.value === range && dashboardActive) {
      analyticsOverview.value = overview
    }
  } catch (error) {
    if (!controller.signal.aborted) awarenessError.value = error instanceof Error ? error.message : '统计加载失败，请重试'
  } finally {
    if (awarenessController === controller) {
      awarenessController = null
      awarenessLoading.value = false
      startAwarenessPoll()
    }
  }
}
function startAwarenessPoll() {
  stopAwarenessPoll()
  if (dashboardActive && !document.hidden) awarenessTimer = setTimeout(() => { void refreshAwareness() }, pollInterval())
}
function stopAwarenessPoll() {
  if (awarenessTimer) {
    clearTimeout(awarenessTimer)
    awarenessTimer = null
  }
}
function onVisibilityChange() {
  if (!dashboardActive) return
  if (document.hidden) cancelAwareness()
  else void refreshAwareness(!analyticsOverview.value || displayedRange.value !== awarenessRange.value)
}

async function load() {
  const background = status.value != null
  if (!background) loading.value = true
  try {
    status.value = await api.getStatus()
    loadError.value = ''
  } catch (error) {
    if (!background) {
      loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
      message.error('加载仪表盘失败')
    }
    return
  } finally {
    loading.value = false
  }
  void loadDashboardExtras()
}

async function loadDashboardExtras() {
  try {
    await Promise.allSettled([
      api.listCertificates().then(rows => { certificates.value = asList(rows) }),
      api.listDDNSLite().then(rows => { ddnsConfigs.value = asList(rows) }),
    ])
  } catch {
    // 次要区块失败不阻断首页
  }
}

function resumeDashboard() {
  dashboardActive = true
  document.addEventListener('visibilitychange', onVisibilityChange)
  void load()
  void refreshAwareness(true)
}

onActivated(resumeDashboard)

function pauseDashboard() {
  dashboardActive = false
  document.removeEventListener('visibilitychange', onVisibilityChange)
  cancelAwareness()
  awarenessLoading.value = false
}
onDeactivated(pauseDashboard)
onUnmounted(pauseDashboard)
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
}

.stats-row :deep(.stat-card) {
  min-height: 0;
  padding: 14px 16px;
}

.stats-row :deep(.stat-card__header) {
  margin-bottom: 8px;
}

.stats-row :deep(.stat-card__value),
.stats-row :deep(.stat-card__value-slot) {
  font-size: 18px;
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
  font-size: 16px;
  font-weight: 700;
  color: var(--fonu-text);
}

.awareness-subtitle {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--fonu-text-secondary);
  max-width: 640px;
  line-height: 1.5;
}

.awareness-updated {
  font-size: 12px;
  color: var(--fonu-text-muted);
  padding-top: 2px;
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

.awareness-charts-panel__trend :deep(.chart) {
  height: 260px;
}

.tabular-nums {
  font-variant-numeric: tabular-nums;
}

.stat-primary {
  font-size: 18px;
  font-weight: 700;
  color: var(--fonu-text);
  line-height: 1.3;
}

.stat-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--fonu-text-secondary);
  line-height: 1.45;
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
  .awareness-lists {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 767px) {
  .stats-row { grid-template-columns: 1fr; }
}
.dashboard-heading { display:flex; justify-content:space-between; align-items:flex-start; margin: 0 0 4px; }
.dashboard-heading h1 { margin:4px 0; font-size:30px; font-weight:650; letter-spacing:-.035em; color:var(--fonu-text); }
.dashboard-heading p { margin:0; color:var(--fonu-text-secondary); font-size:13px; }
.dashboard-heading .dashboard-eyebrow { font-size:10px; letter-spacing:.10em; color:var(--fonu-brand-text); font-weight:600; }
.dashboard .stats-row { gap:0; border-block:1px solid var(--fonu-border); margin:0; }
.dashboard .stats-row :deep(.stat-card) { background:transparent; border:0; border-right:1px solid var(--fonu-border); border-radius:0; box-shadow:none; min-width:0; }
.dashboard .stats-row :deep(.stat-card:last-child) { border-right:0; }
.dashboard .stats-row :deep(.stat-card__icon) { background:transparent; color:var(--fonu-brand); width:22px; height:22px; }
.dashboard .awareness-section { gap:12px; }
@media(max-width:767px) { .dashboard-heading h1 { font-size:25px; } .dashboard .stats-row { grid-template-columns:repeat(2,minmax(0,1fr)); } .dashboard .stats-row :deep(.stat-card:nth-child(2)) { border-right:0; } .dashboard .stats-row :deep(.stat-card:nth-child(-n+2)) { border-bottom:1px solid var(--fonu-border); } }
.dashboard { gap:12px; }
.dashboard-tabs { display:flex; gap:8px; border-bottom:1px solid var(--fonu-border); flex-wrap:wrap; }
.dashboard-tabs button { font:inherit; font-weight:500; border:0; border-bottom:2px solid transparent; background:transparent; color:var(--fonu-text-secondary); padding:10px 14px; cursor:pointer; }
.dashboard-tabs button[aria-pressed=true] { color:var(--fonu-brand); border-color:var(--fonu-brand); font-weight:650; }
.quick-links { display:flex; gap:8px; flex-wrap:wrap; }
.quick-links a { padding:8px 12px; border:1px solid var(--fonu-border); border-radius:6px; color:var(--fonu-brand); text-decoration:none; }
.notice-row { display:flex; align-items:center; justify-content:space-between; gap:12px; padding:12px 0; border-bottom:1px solid var(--fonu-border); }
.notice-row p { margin:4px 0; color:var(--fonu-text-secondary); }
.notice-row a,.notification-version { color:var(--fonu-brand); }
.notification-version { margin-left:16px; }
.dashboard .awareness-lists { grid-template-columns:minmax(0,1fr); }
.dashboard .stats-row :deep(.stat-card) { padding:10px 12px; }
.dashboard .stats-row :deep(.stat-card__header) { margin-bottom:4px; }
.dashboard .awareness-section { margin-bottom:0; }
.dashboard-tabs { border:0; background:var(--fonu-surface); border-radius:12px; padding:5px; gap:4px; box-shadow:var(--fonu-shadow); }
.dashboard-tabs button { border:0; border-radius:8px; padding:9px 16px; }
.dashboard-tabs button[aria-pressed=true] { background:var(--fonu-brand-soft); color:var(--fonu-brand); }
.dashboard .stats-row { border:0; gap:12px; grid-template-columns:repeat(4,minmax(0,1fr)); }
.dashboard .stats-row :deep(.stat-card) { background:var(--fonu-surface); border:1px solid var(--fonu-border); border-radius:12px; box-shadow:var(--fonu-shadow); padding:14px; }
.dashboard .stats-row :deep(.stat-card__icon) { background:var(--fonu-brand-soft); color:var(--fonu-brand); width:30px; height:30px; border-radius:9px; }
.trend-workspace { display:grid; grid-template-columns:minmax(0,1fr); gap:12px; }
.trend-workspace--overview { grid-template-columns:minmax(0,2fr) minmax(220px,1fr); }
.quick-links { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:8px; }
.quick-links a { display:flex; align-items:center; gap:6px; padding:9px; background:var(--fonu-bg); color:var(--fonu-text); font-size:12px; min-width:0; }
.quick-links a:hover { background:var(--fonu-brand-soft); color:var(--fonu-brand); }
.quick-icon { display:grid; place-items:center; width:25px; height:25px; border-radius:7px; background:var(--fonu-brand-soft); color:var(--fonu-brand); flex-shrink:0; font-size:16px; }
.quick-arrow { margin-left:auto; color:var(--fonu-text-muted); }
.notice-icon { display:grid; place-items:center; width:36px; height:36px; border-radius:10px; color:var(--fonu-warning); background:color-mix(in srgb,var(--fonu-warning) 12%,transparent); }
.notice-detail { flex:1; }
@media(max-width:1200px) { .dashboard .stats-row { grid-template-columns:repeat(2,minmax(0,1fr)); } }
@media(max-width:850px) { .trend-workspace--overview { grid-template-columns:minmax(0,1fr); } }
@media(max-width:480px) { .dashboard .stats-row { grid-template-columns:minmax(0,1fr); } }
.dashboard { container-type:inline-size; gap:12px; }
.dashboard .stats-row { grid-template-columns:repeat(4,minmax(0,1fr)); gap:10px; }
.dashboard .stats-row :deep(.stat-card) { padding:10px 12px; min-height:90px; border:1px solid var(--fonu-border); }
.dashboard .stats-row :deep(.stat-card__header) { gap:4px; align-items:center; margin-bottom:5px; flex-wrap:wrap; }
.dashboard .stats-row :deep(.stat-card__title-row) { gap:7px; }
.dashboard .stats-row :deep(.stat-card__icon) { width:25px;height:25px;font-size:15px; }
.dashboard .stats-row :deep(.stat-card__value-slot),.dashboard .stats-row :deep(.stat-card__value) { font-size:18px; }
.dashboard .stats-row .stat-desc { margin:3px 0 0; overflow:hidden;text-overflow:ellipsis;white-space:nowrap; }
.dashboard-tabs { background:transparent;box-shadow:none;padding:3px 0; }
.dashboard-tabs button[aria-pressed=true] { color:var(--fonu-brand-text); }
.trend-workspace--overview { grid-template-columns:minmax(0,2fr) minmax(230px,1fr); }
.overview-side { display:flex;flex-direction:column;gap:12px;min-width:0; }
.overview-side .quick-links a { background:var(--fonu-surface); }
.overview-side .notice-row:last-child { border-bottom:0; }
@container (max-width:850px) { .dashboard .stats-row {grid-template-columns:repeat(2,minmax(0,1fr));} .trend-workspace--overview {grid-template-columns:minmax(0,1fr);} }
@container (max-width:420px) { .dashboard .stats-row {grid-template-columns:1fr;} }
</style>
