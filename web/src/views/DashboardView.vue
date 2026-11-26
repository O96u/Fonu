<template>
<div class="dashboard-v6">
  <nav class="dashboard-tabs" aria-label="首页分区"><button v-for="tab in tabs" :key="tab.key" type="button" :aria-pressed="activeTab === tab.key" @click="selectTab(tab.key)">{{ tab.label }}</button></nav>
  <LoadError v-if="loadError" :message="loadError" @retry="load" />
  <n-alert v-if="awarenessNginxWarning && activeTab !== 'security'" type="warning" :bordered="false">{{ awarenessNginxWarning }}</n-alert>
  <template v-if="activeTab === 'overview'">
    <FonuCard class="status-strip">    <n-spin v-if="activeTab === 'overview'" :show="loading && !status">
      <div class="stats-row">
        <StatCard label="公网 IP" tone="blue">
          <template #icon><n-icon :component="GlobeOutline" /></template>
          <template #extra><StatusBadge v-if="status?.public_ipv4 || status?.public_ipv6" value="ok" text="已获取" /></template>
          <template #value>
            <span class="stat-primary mono">{{ publicIPv4Label }}</span>
          </template>
          <p class="stat-desc" :title="publicIPv6Label + ' · ' + publicIPSourceLabel">{{ publicIPv6Label }} · {{ publicIPSourceLabel }}</p>
        </StatCard>

        <StatCard label="域名" tone="green">
          <template #icon><n-icon :component="WifiOutline" /></template>
          <template #extra><StatusBadge :value="status?.ddns_status" /></template>
          <template #value>{{ domainCountLabel }}</template>
          <p class="stat-desc" :title="domainExamplesLabel">{{ domainExamplesLabel }}</p>
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
    </n-spin></FonuCard>
    <div class="workspace-grid"><div class="workspace-main"><FonuCard class="traffic-panel"><AnalyticsInsightSection v-model:range="awarenessRange" compact :show-kpis="true" :show-map="false" section-title="访问趋势" :data="analyticsOverview" :loading="awarenessLoading" :error="awarenessError" @retry="refreshAwareness(true)" /><div class="chart-caption"><span>{{ awarenessTrendTitle }}</span><span>{{ awarenessTrendTag }} · 更新于 {{ awarenessUpdatedLabel }}</span></div><AwarenessTrendChart v-if="analyticsOverview" :labels="analyticsOverview.series.labels" :requests="analyticsOverview.series.requests" :uniqueIPs="analyticsOverview.series.unique_ips" :current-bucket="analyticsOverview.series.current_bucket" /><EmptyState v-else title="正在读取访问趋势" description="统计数据加载后将显示访问趋势。"/></FonuCard>
    <div class="rank-grid">
      <FonuCard title="来源排行" class="rank-card" data-testid="dashboard-source-rank">
        <template #header><button class="text-link" @click="selectTab('access')">查看分析 →</button></template>
        <div v-for="r in sourceRanking" :key="r.country" class="rank-row"><span :title="r.country">{{ r.country }}</span><div class="rank-track"><i :style="{width:rankPercent(r.requests,sourceRanking[0]?.requests)}"/></div><strong>{{ r.requests.toLocaleString() }}</strong></div>
        <div v-for="n in Math.max(0, HOME_PREVIEW_LIMIT-sourceRanking.length)" :key="'source-empty-'+n" class="rank-row rank-placeholder" aria-hidden="true"><span>—</span><div class="rank-track"/><span>—</span></div>
        <p v-if="!sourceRanking.length" class="rank-empty-label">{{ awarenessLoading ? '正在统计…' : '暂无来源数据' }}</p>
      </FonuCard>
      <FonuCard title="服务排行" class="rank-card" data-testid="dashboard-service-rank">
        <template #header><button class="text-link" @click="selectTab('access')">查看全部 →</button></template>
        <div v-for="r in servicePreview" :key="r.service" class="rank-row service-rank"><span class="service-name"><ServiceFavicon :rule-id="serviceIconRules.get(r.service)" /><span :title="r.service">{{ r.service }}</span></span><div class="rank-track"><i :style="{width:rankPercent(r.requests,servicePreview[0]?.requests)}"/></div><strong>{{ r.requests.toLocaleString() }}</strong></div>
        <div v-for="n in Math.max(0, HOME_PREVIEW_LIMIT-servicePreview.length)" :key="'service-empty-'+n" class="rank-row rank-placeholder" aria-hidden="true"><span>—</span><div class="rank-track"/><span>—</span></div>
        <p v-if="!servicePreview.length" class="rank-empty-label">{{ awarenessLoading ? '正在统计…' : '暂无已识别服务' }}</p>
      </FonuCard>
    </div>
        <FonuCard title="最近访问" class="recent-full" data-testid="dashboard-recent" subtitle="最近 5 条 · 当前统计范围">
          <template #header><router-link class="text-link" :to="{name:'logs',query:{tab:'access'}}">查看日志 →</router-link></template>
          <div v-for="(event,index) in recentPreview" :key="event.at+'-'+index" class="recent-row">
            <div class="recent-service"><strong :title="event.service || event.domain">{{event.service || event.domain || '未识别服务'}}</strong><small :title="event.path">{{event.path || '/'}}</small></div>
            <div class="recent-client"><span :title="event.client_ip">{{event.client_ip || '—'}}</span><small>{{event.geo ? displayGeoLabel(event.geo) : '—'}}</small></div>
            <HttpStatusBadge :code="event.status" class="recent-status" />
            <small class="recent-time">{{formatRelativeTime(event.at) || formatLogTime(event.at,{timeZone:logTimeZone})}}</small>
          </div>
          <div v-for="n in Math.max(0, HOME_PREVIEW_LIMIT - recentPreview.length)" :key="'recent-empty-'+n" class="recent-row recent-placeholder" :aria-hidden="n > 1 ? 'true' : undefined"><span>{{ n === 1 ? (awarenessLoading ? '正在读取访问记录…' : recentPreview.length ? '暂无更多访问记录' : '当前统计范围暂无访问记录') : '—' }}</span></div>
        </FonuCard>
      </div>
      <aside class="workspace-side">
        <RuntimePanel/>
        <FonuCard title="快捷操作"><div class="quick-links"><router-link v-for="item in quickEntries" :key="item.route" :to="{name:item.route}"><n-icon :component="item.icon"/><span>{{ item.label }}</span></router-link></div></FonuCard>
        <FonuCard title="待处理事项"><div v-for="notice in notices.slice(0, HOME_PREVIEW_LIMIT)" :key="notice.title" class="notice-row"><n-icon :component="ShieldCheckmarkOutline"/><router-link :to="{name:notice.route}"><strong>{{ notice.title }}</strong><small>{{ notice.detail }}</small></router-link></div><p v-if="!notices.length" class="muted">{{ status ? '暂无待处理事项' : '正在获取状态' }}</p><p v-if="notices.length > HOME_PREVIEW_LIMIT" class="muted">另有 {{ notices.length - HOME_PREVIEW_LIMIT }} 项，请前往证书或反代管理查看。</p></FonuCard>
      </aside>
    </div>


  </template>
  <template v-else-if="activeTab === 'access'">
    <FonuCard class="traffic-panel"><AnalyticsInsightSection v-model:range="awarenessRange" compact :show-kpis="true" :show-map="false" section-title="访问趋势" :data="analyticsOverview" :loading="awarenessLoading" :error="awarenessError" @retry="refreshAwareness(true)" /><div class="chart-caption"><span>{{ awarenessTrendTitle }}</span><span>{{ awarenessTrendTag }} · 更新于 {{ awarenessUpdatedLabel }}</span></div><AwarenessTrendChart v-if="analyticsOverview" :labels="analyticsOverview.series.labels" :requests="analyticsOverview.series.requests" :uniqueIPs="analyticsOverview.series.unique_ips" :current-bucket="analyticsOverview.series.current_bucket" /><EmptyState v-else title="正在读取访问趋势" description="统计数据加载后将显示访问趋势。"/></FonuCard>
    <AnalyticsInsightSection v-model:range="awarenessRange" map-only show-map :data="analyticsOverview" :loading="awarenessLoading" />
    <FonuCard title="按服务汇总" :subtitle="awarenessServiceListHint"><n-data-table v-if="awarenessServices.length" :columns="awarenessServiceColumns" :data="awarenessServices" :bordered="false" :scroll-x="620" :pagination="{pageSize:8}" size="small"/><EmptyState v-else title="暂无已识别服务" description="当前统计范围内没有可展示的已识别服务。"/></FonuCard>
    <FonuCard title="最近访问 · 30 条"><template #header><router-link class="text-link" :to="{name:'logs',query:{tab:'access'}}">查看全部 →</router-link></template><n-data-table v-if="awarenessEvents.length" :columns="awarenessEventColumns" :data="awarenessEvents" :bordered="false" :max-height="awarenessTableScrollHeight" size="small"/><EmptyState v-else title="暂无访问记录" description="当前统计范围内没有访问记录。"/></FonuCard>
  </template>
  <template v-else>
    <AnalyticsAnomalySection v-if="analyticsOverview?.anomaly" :data="analyticsOverview"/>
    <EmptyState v-else title="正在读取安全分析" description="安全统计加载完成后将显示异常分析。"/>
  </template>
</div>
</template>
<script setup lang="ts">
import { computed, defineAsyncComponent, h, onActivated, onDeactivated, onUnmounted, ref, shallowRef, watch } from 'vue'
import { NAlert, NDataTable, NIcon, NSpin, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import {
  EyeOutline,
  SettingsOutline,
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
import ServiceFavicon from '../components/ServiceFavicon.vue'
import RuntimePanel from '../components/RuntimePanel.vue'
import type { ProxyRule } from '../api/types'
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
import HttpStatusBadge from '../components/HttpStatusBadge.vue'

defineOptions({ name: 'DashboardView' })

const route = useRoute()
const router = useRouter()
const tabs = [{key:'overview',label:'概览'},{key:'access',label:'访问分析'},{key:'security',label:'安全分析'}] as const
const activeTab = computed(() => route.query.tab === 'services' ? 'access' : tabs.find(t => t.key === route.query.tab)?.key ?? 'overview')
function selectTab(key: string) { void router.replace({name:'dashboard',query:key === 'overview' ? {} : {tab:key}}) }
const quickEntries = [{route:'ddns',label:'DDNS',icon:GlobeOutline},{route:'certificates',label:'证书管理',icon:ShieldCheckmarkOutline},{route:'proxies',label:'反向代理',icon:GitNetworkOutline},{route:'frp',label:'内网穿透',icon:WifiOutline},{route:'logs',label:'日志中心',icon:EyeOutline},{route:'settings',label:'系统设置',icon:SettingsOutline}]
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
const publicIPv6Label = computed(() => status.value?.public_ipv6 ? 'IPv6 '+status.value.public_ipv6 : status.value ? 'IPv6 未检测到' : 'IPv6 检测中')
const publicIPSourceLabel = computed(() => {
  switch (status.value?.public_ip_source) {
    case 'ddns':
      return status.value?.public_ipv4 || status.value?.public_ipv6 ? 'DDNS' : 'DDNS 等待同步'
    case 'mixed':
      return 'DDNS / 出口探测'
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


watch(awarenessRange, () => {
  cancelAwareness()
  awarenessLoading.value = true
  awarenessError.value = ''
  void refreshAwareness(true)
})

const awarenessEvents = computed(() => analyticsOverview.value?.recent_events ?? [])
const iconRules = shallowRef<ProxyRule[]>([])
let iconRulesLoaded = false
const serviceIconRules = computed(() => {
  const result = new Map<string, number>()
  const duplicates = new Set<string>()
  for (const rule of iconRules.value) {
    if (!rule.enabled) continue
    const label = rule.name.trim() || rule.domain.trim() || rule.upstream.trim()
    if (!label) continue
    if (result.has(label)) duplicates.add(label)
    else result.set(label, rule.id)
  }
  for (const label of duplicates) result.delete(label)
  return result
})
watch(activeTab, async tab => {
  if (tab === 'security' || iconRulesLoaded) return
  iconRulesLoaded = true
  try { iconRules.value = asList(await api.listProxies()) } catch { iconRulesLoaded = false }
}, { immediate: true })
const awarenessServices = computed(() => (analyticsOverview.value?.by_service ?? []).filter(r => { const name = r.service.trim(); return !!name && !/^(?:_|\*|):\d+$/.test(name) && name !== '_' && name !== '*' }))
const HOME_PREVIEW_LIMIT = 5
const servicePreview = computed(() => awarenessServices.value.slice(0, HOME_PREVIEW_LIMIT))
const recentPreview = computed(() => awarenessEvents.value.slice(0, HOME_PREVIEW_LIMIT))
const sourceRanking = computed(() => (analyticsOverview.value?.by_country ?? []).filter(r => r.country && r.country !== '未知' && r.requests > 0).slice().sort((a,b)=>b.requests-a.requests).slice(0,HOME_PREVIEW_LIMIT))
function rankPercent(value:number,max?:number) { return (max && max > 0 ? Math.min(100,value/max*100) : 0) + '%' }

/** Backend returns at most 20 services sorted by request volume. */
const awarenessServiceListHint = computed(() => {
  const n = awarenessServices.value.length
  if (n === 0) return ''
  return `已识别服务 ${n} 项 · 从请求量前 20 项中筛选，原始日志及总量不变`
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
      return h(HttpStatusBadge, { code: r.status })
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
  { title: '服务', key: 'service', width: 210, render: r => h('div',{class:'service-name',title:r.service},[h(ServiceFavicon,{ruleId:serviceIconRules.value.get(r.service)}),h('span',r.service)]) },
  {
    title: '请求',
    key: 'requests',
    width: 160,
    align: 'right',
    titleAlign: 'right',
    sorter: (a,b) => a.requests-b.requests,
    render: (r) => h('div',{class:'service-table-volume'},[h('span',{class:'service-table-bar'},[h('i',{style:{width:rankPercent(r.requests,awarenessServices.value[0]?.requests)}})]),h('span',r.requests.toLocaleString())]),
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
.dashboard-v6{display:flex;flex-direction:column;gap:12px;min-width:0;container-type:inline-size}
.dashboard-tabs{display:flex;gap:6px;align-items:center}
.dashboard-tabs button{border:0;background:transparent;color:var(--fonu-text-secondary);padding:8px 16px;border-radius:8px;cursor:pointer;font-size:14px}
.dashboard-tabs button[aria-pressed=true]{background:var(--fonu-brand-soft);color:var(--fonu-brand-text);font-weight:600}
.stats-row{display:grid;grid-template-columns:minmax(0,1.3fr) repeat(3,minmax(0,1fr));gap:0}
.status-strip :deep(.fonu-card__body){padding:12px 8px}
.stats-row :deep(.stat-card){min-width:0;min-height:0;padding:5px 12px;border:0;border-right:1px solid var(--fonu-border);border-radius:0;background:transparent;box-shadow:none}
.stats-row :deep(.stat-card:last-child){border-right:0}
.stats-row :deep(.stat-card__header){margin-bottom:4px;gap:4px}
.stats-row :deep(.stat-card__icon){width:25px;height:25px;border-radius:7px;background:var(--fonu-brand-soft);color:var(--fonu-brand-text)}
.stats-row :deep(.stat-card__value),.stats-row :deep(.stat-card__value-slot){font-size:19px;line-height:1.3;font-weight:650;letter-spacing:-.01em;overflow-wrap:anywhere;word-break:normal}
.stats-row :deep(.stat-card__title-row){gap:7px;min-width:0}
.stats-row :deep(.stat-card__header){align-items:center;flex-wrap:wrap;margin-bottom:8px}
.stats-row :deep(.stat-card__label){white-space:nowrap;font-size:12px}
.stat-primary{font-size:inherit; font-variant-numeric:tabular-nums}
.stats-row :deep(.stat-card){padding:6px 14px}
.stat-desc{font-size:11px;color:var(--fonu-text-secondary);margin:3px 0 0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.workspace-grid{display:grid;grid-template-columns:minmax(0,2fr) minmax(230px,1fr);gap:12px;align-items:start}
.workspace-main,.workspace-side{display:flex;flex-direction:column;gap:12px;min-width:0}.workspace-main{container-type:inline-size}
.recent-row{display:grid;grid-template-columns:minmax(0,1.3fr) minmax(0,1fr) auto minmax(44px,auto);align-items:center;gap:10px;padding:9px 0;border-bottom:1px solid var(--fonu-border);font-size:12px}.recent-row:last-child{border-bottom:0}.recent-service,.recent-client{min-width:0;display:grid;gap:2px}.recent-service strong,.recent-service small,.recent-client span,.recent-client small{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.recent-service small,.recent-client small,.recent-time{color:var(--fonu-text-secondary);font-size:11px}.recent-time{text-align:right}.recent-empty{padding:10px 0;margin:0}
@container(max-width:460px){.recent-row{grid-template-columns:minmax(0,1fr) auto auto;gap:7px}.recent-client{grid-row:2;grid-column:1/3}.recent-time{grid-column:3;grid-row:2}}
.traffic-panel :deep(.fonu-card__body){padding:16px}
.chart-caption{display:flex;justify-content:space-between;gap:8px;flex-wrap:wrap;color:var(--fonu-text-secondary);font-size:11px;margin:6px 0}
.quick-links{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:6px}
.quick-links a{display:flex;flex-direction:column;align-items:center;gap:5px;padding:9px 3px;border:1px solid var(--fonu-border);border-radius:8px;color:var(--fonu-text);text-decoration:none;font-size:11px}
.quick-links a:hover{background:var(--fonu-brand-soft);color:var(--fonu-brand-text)}
.quick-links :deep(.n-icon){font-size:19px;color:var(--fonu-brand-text)}
.notice-row{display:flex;gap:10px;align-items:center;padding:7px 0;border-bottom:1px solid var(--fonu-border)}
.notice-row:last-child{border:0}.notice-row>a{color:var(--fonu-text);text-decoration:none;min-width:0}.notice-row small{display:block;color:var(--fonu-text-secondary);margin-top:3px}.notice-row>.n-icon{color:var(--fonu-warning);font-size:22px}
.rank-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;container-type:inline-size}.rank-grid :deep(.fonu-card){min-width:0}.rank-row{display:grid;grid-template-columns:minmax(70px,1fr) minmax(45px,2fr) auto;gap:10px;align-items:center;padding:8px 0;font-size:12px}.rank-row>span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.rank-row strong{font-variant-numeric:tabular-nums;font-weight:600}.rank-track{height:7px;border-radius:5px;background:var(--fonu-bg-muted);overflow:hidden}.rank-track i{display:block;height:100%;background:var(--fonu-brand);border-radius:5px}.service-name{display:flex;align-items:center;gap:6px}.service-icon{display:grid;place-items:center;flex-shrink:0;width:25px;height:25px;background:var(--fonu-brand-soft);color:var(--fonu-brand-text);border-radius:7px}.muted{color:var(--fonu-text-muted);font-size:12px}.text-link{border:0;background:transparent;color:var(--fonu-brand-text);cursor:pointer;font-size:12px;text-decoration:none}
@container(max-width:820px){.stats-row{grid-template-columns:repeat(2,minmax(0,1fr))}.stats-row :deep(.stat-card:nth-child(2)){border-right:0}.workspace-grid,.rank-grid{grid-template-columns:1fr}}
@container(max-width:400px){.stats-row{grid-template-columns:1fr}.stats-row :deep(.stat-card){border-right:0}}
.service-table-volume{display:flex;align-items:center;justify-content:flex-end;gap:10px;font-variant-numeric:tabular-nums}.service-table-bar{width:70px;height:5px;background:var(--fonu-bg-muted);border-radius:5px;overflow:hidden}.service-table-bar i{display:block;height:100%;background:var(--fonu-brand);border-radius:5px}.service-name>span:last-child{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}

/* Fixed overview preview slots; the detail tabs keep their original limits. */
.rank-grid { align-items:stretch; }
.rank-card :deep(.fonu-card__body) { position:relative; }
.rank-card .rank-row { min-height:44px; box-sizing:border-box; padding:6px 0; grid-template-columns:minmax(75px,1fr) minmax(25px,1.45fr) auto; }
.rank-placeholder { color:var(--fonu-text-muted); opacity:.55; }
.rank-empty-label { position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); margin:0; padding:6px 10px; white-space:nowrap; background:var(--fonu-surface); color:var(--fonu-text-secondary); font-size:12px; }
.rank-card .service-name { min-width:0; }
.rank-card .service-name > span:last-child { display:block; min-width:0; overflow:hidden; text-overflow:ellipsis; }
.recent-full { width:100%; min-width:0; }
.recent-full .recent-row { min-height:44px; box-sizing:border-box; grid-template-columns:minmax(0,1.7fr) minmax(0,1fr) minmax(60px,auto) minmax(80px,auto); }
.recent-full .recent-placeholder { color:var(--fonu-text-muted); }
.recent-placeholder > span { grid-column:1/-1; }
@container(max-width:520px) { .recent-full .recent-row { min-height:64px; grid-template-columns:minmax(0,1fr) auto auto; } .recent-full .recent-client{grid-row:2;grid-column:1/3}.recent-full .recent-time{grid-column:3;grid-row:2} }
.recent-status { justify-self:center; }
</style>
