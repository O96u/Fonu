<template>
  <PageHeader
    title="日志中心"
    description="集中查看运行记录、访问请求与异常"
  ><template #actions><n-button secondary @click="router.push({ name: 'settings' })">日志设置</n-button></template></PageHeader>

  <LoadError v-if="pageError" :message="pageError" @retry="loadAll" />

  <div v-else class="log-workspace">
    <div class="log-source-grid">
      <button v-for="item in sourceCards" :key="item.key" type="button" class="log-source" :class="{ 'is-active': tab === item.key }" :aria-pressed="tab === item.key" @click="tab = item.key">
        <span class="source-heading"><n-icon :component="item.icon" />{{ item.label }}<n-icon v-if="tab === item.key" :component="CheckmarkCircleOutline" class="source-check" /></span>
        <strong>{{ loaded[item.key] ? item.count : '—' }}</strong>
        <small>{{ errors[item.key] ? '读取失败' : loaded[item.key] ? '已读取记录 · 最近片段' : '点击查看日志' }}</small>
      </button>
    </div>
    <FonuCard flush class="logs-card">
      <div class="log-workspace-head">
        <div><strong>{{ sourceCards.find(item => item.key === tab)?.label }}</strong><small>当前展示最近读取的日志片段，并非全部历史</small></div>
        <div class="log-head-actions"><n-button size="small" :disabled="!activeRows.length" @click="exportLoaded">导出筛选结果</n-button><n-button size="small" :type="autoRefresh ? 'primary' : 'default'" secondary @click="autoRefresh = !autoRefresh">{{ autoRefresh ? '自动刷新已开启' : '自动刷新' }}</n-button></div>
      </div>
      <div v-if="errors[tab]" class="log-error"><span>{{ errors[tab] }}</span><n-button size="small" @click="loadCurrent">重试</n-button></div>
      <div v-if="activeLoading && !loaded[tab]" class="log-loading"><n-spin size="small" /> 正在读取日志…</div>
    <n-tabs v-model:value="tab" type="line" animated class="logs-tabs">
      <n-tab-pane name="system" tab="系统日志">
        <LogToolbar
          v-model:keyword="systemKeyword"
          v-model:level="systemLevel"
          :show-level="true"
          @refresh="loadSystemLogs"
        />
        <p class="tab-hint">Fonu 全部运行日志（API、DDNS、证书、通知、Nginx 重载等），不含 Nginx 访问与错误文件。</p>
        <n-data-table
          v-if="filteredSystemLogs.length > 0"
          class="log-table"
          :columns="systemColumns"
          :data="pagedSystemLogs"
          :loading="loadingSystem"
          :bordered="false"
          :row-class-name="systemRowClassName"
          :row-props="row => detailRowProps('system', row)"
        />
        <LogPagination
          v-if="filteredSystemLogs.length > 0"
          v-model:page="systemPage"
          :item-count="filteredSystemLogs.length"
        />
        <EmptyState
          v-if="!loadingSystem && loaded['system'] && !errors['system'] && filteredSystemLogs.length === 0"
          :title="systemLogs.length ? '没有匹配的日志' : '暂无系统日志'"
          description="应用启动、DDNS 更新、证书操作或 API 请求会记录在这里。"
        />
      </n-tab-pane>

      <n-tab-pane name="access" tab="访问日志">
        <LogToolbar
          v-model:keyword="accessKeyword"
          v-model:status="accessStatus"
          v-model:kind="accessKind"
          :show-domain="true"
          :show-status="true"
          :show-kind="true"
          @refresh="loadAccess"
        />
        <p v-if="accessFilterBanner" class="tab-hint access-filter-banner">
          {{ accessFilterBanner }}
          <router-link v-if="accessKind || accessClientIP" :to="{ name: 'logs', query: { tab: 'access' } }" class="access-filter-clear">
            清除筛选
          </router-link>
        </p>
        <p v-if="accessKind === 'intercept' && accessFilterBanner" class="tab-hint">
          策略拦截：403 / 429 / 401 等拒绝响应。「拦截原因」为启发式说明，并非 Nginx 规则名。
        </p>
        <n-data-table
          v-if="filteredAccessLogs.length > 0"
          class="log-table"
          :columns="accessColumns"
          :data="pagedAccessLogs"
          :loading="loadingAccess"
          :bordered="false"
          :scroll-x="accessTableScrollX"
          :row-class-name="accessRowClassName"
          :row-props="row => detailRowProps('access', row)"
        />
        <LogPagination
          v-if="filteredAccessLogs.length > 0"
          v-model:page="accessPage"
          :item-count="filteredAccessLogs.length"
        />
        <EmptyState
          v-if="!loadingAccess && loaded['access'] && !errors['access'] && filteredAccessLogs.length === 0"
          :title="accessLogs.length ? '没有匹配的日志' : accessEmptyTitle"
          :description="accessLogs.length ? '请调整或清除当前筛选条件。' : accessEmptyDescription"
        />
      </n-tab-pane>

      <n-tab-pane name="nginx" tab="Nginx 日志">
        <p class="tab-hint">Nginx 错误日志（error.log），记录 SSL 握手失败、上游连接异常、配置冲突等。</p>
        <LogToolbar v-model:keyword="errorKeyword" v-model:level="errorLevel" :show-level="true" @refresh="loadErrorLogs" />
        <n-data-table
          v-if="filteredErrorLogs.length > 0"
          class="log-table"
          :columns="errorColumns"
          :data="pagedErrorLogs"
          :loading="loadingError"
          :bordered="false"
          :row-class-name="errorRowClassName"
          :row-props="row => detailRowProps('nginx', row)"
        />
        <LogPagination
          v-if="filteredErrorLogs.length > 0"
          v-model:page="errorPage"
          :item-count="filteredErrorLogs.length"
        />
        <EmptyState
          v-if="!loadingError && loaded['nginx'] && !errors['nginx'] && filteredErrorLogs.length === 0"
          :title="errorLogs.length ? '没有匹配的日志' : '暂无 Nginx 错误日志'"
          description="Nginx 出现 SSL、上游或配置相关错误时会记录在这里。"
        />
      </n-tab-pane>

      <n-tab-pane name="frp" tab="FRP 日志">
        <p class="tab-hint">
          frpc 运行日志（frpc.log），记录连接 frps、代理启停与隧道状态变化。
        </p>
        <LogToolbar v-model:keyword="frpKeyword" v-model:level="frpLevel" :show-level="true" @refresh="loadFrpLogs" />
        <n-data-table
          v-if="filteredFrpLogs.length > 0"
          class="log-table"
          :columns="frpColumns"
          :data="pagedFrpLogs"
          :loading="loadingFrp"
          :bordered="false"
          :row-class-name="frpRowClassName"
          :row-props="row => detailRowProps('frp', row)"
        />
        <LogPagination
          v-if="filteredFrpLogs.length > 0"
          v-model:page="frpPage"
          :item-count="filteredFrpLogs.length"
        />
        <EmptyState
          v-if="!loadingFrp && loaded['frp'] && !errors['frp'] && filteredFrpLogs.length === 0"
          :title="frpLogs.length ? '没有匹配的日志' : '暂无 FRP 日志'"
          description="启用内网穿透并启动 frpc 后，连接与隧道相关输出会记录在这里。"
        />
      </n-tab-pane>
    </n-tabs>
    </FonuCard>
  </div>
  <n-drawer v-model:show="showDetail" :width="'min(520px, 94vw)'">
    <n-drawer-content title="日志详情" closable>
      <template v-if="detail">
        <div class="detail-fields"><div v-for="field in detail.fields" :key="field.label"><span>{{ field.label }}</span><strong>{{ field.value || '—' }}</strong></div></div>
        <h4>{{ detail.rawLabel }}</h4><pre class="fonu-log-console detail-console">{{ detail.raw }}</pre>
        <div class="detail-actions"><n-button size="small" @click="copyDetail">复制记录</n-button><n-button v-if="detail.ip" size="small" type="primary" secondary @click="filterDetailIP">筛选此 IP</n-button></div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter, type RouteLocationNormalizedLoaded } from 'vue-router'
import {
  NButton,
  NDataTable,
  NDrawer,
  NDrawerContent,
  NIcon,
  NSpin,
  NInput,
  NPagination,
  NSelect,
  NTabPane,
  NTabs,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { api, asList } from '../api/client'
import type { AccessLogEntry, ProxyRule, SystemLogEntry } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import {
  accessServiceTooltip,
  buildProxyBindingIndex,
  formatAccessFallback,
  resolveAccessServiceLabel,
} from '../utils/accessService'
import { useAppTimezone } from '../composables/useAppTimezone'
import { formatLogTime, formatMs } from '../utils/format'
import { displayGeoLabel } from '../utils/geoDisplay'
import {
  frpLevelTagType,
  latencyClass,
  nginxLevelTagType,
  parseFrpLogLine,
  parseNginxErrorLine,
  type ParsedFrpLog,
  type ParsedNginxError,
  displaySystemLog,
  systemLevelTagType,
  systemModuleTagType,
} from '../utils/logDisplay'
import HttpStatusBadge from '../components/HttpStatusBadge.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { copyToClipboard } from '../utils/clipboard'
import { DesktopOutline, GlobeOutline, AlertCircleOutline, GitNetworkOutline, CheckmarkCircleOutline } from '@vicons/ionicons5'

const PAGE_SIZE = 20
const logTimeZone = useAppTimezone()

const LogPagination = defineComponent({
  name: 'LogPagination',
  props: {
    page: { type: Number, required: true },
    itemCount: { type: Number, required: true },
  },
  emits: ['update:page'],
  setup(props, { emit }) {
    return () =>
      h('div', { class: 'log-pagination' }, [
        h(NPagination, {
          page: props.page,
          pageSize: PAGE_SIZE,
          itemCount: props.itemCount,
          showSizePicker: false,
          'onUpdate:page': (p: number) => emit('update:page', p),
        }),
      ])
  },
})

const LogToolbar = defineComponent({
  name: 'LogToolbar',
  props: {
    keyword: { type: String, default: '' },
    status: { type: Number as () => number | null, default: null },
    kind: { type: String, default: '' },
    level: { type: String, default: '' },
    showDomain: Boolean,
    showStatus: Boolean,
    showKind: Boolean,
    showLevel: Boolean,
    autoRefresh: { type: Boolean, default: undefined },
  },
  emits: ['update:keyword', 'update:status', 'update:kind', 'update:level', 'refresh', 'toggle-auto'],
  setup(props, { emit }) {
    return () =>
      h('div', { class: 'toolbar' }, [
        h(NInput, {
          value: props.keyword,
          placeholder: '关键词搜索',
          clearable: true,
          style: 'max-width: 220px',
          'onUpdate:value': (v: string) => emit('update:keyword', v),
        }),
        props.showStatus
          ? h(NSelect, {
              value: props.status,
              placeholder: '状态码',
              clearable: true,
              style: 'width: 120px',
              options: [200, 201, 204, 301, 302, 304, 400, 401, 403, 404, 408, 429, 499, 500, 502, 503, 504].map((c) => ({ label: String(c), value: c })),
              'onUpdate:value': (v: number | null) => emit('update:status', v),
            })
          : null,
        props.showKind
          ? h(NSelect, {
              value: props.kind || null,
              placeholder: '访问类型',
              clearable: true,
              style: 'width: 160px',
              options: [
                { label: '策略拦截', value: 'intercept' },
                { label: '403 拒绝', value: '403' },
                { label: '429 限流', value: '429' },
                { label: '401 未授权', value: '401' },
                { label: '疑似扫描', value: 'scan' },
              ],
              'onUpdate:value': (v: string | null) => emit('update:kind', v ?? ''),
            })
          : null,
        props.showLevel
          ? h(NSelect, {
              value: props.level,
              placeholder: '日志级别',
              clearable: true,
              style: 'width: 120px',
              options: [
                { label: '信息', value: 'INFO' },
                { label: '警告', value: 'WARN' },
                { label: '错误', value: 'ERROR' },
              ],
              'onUpdate:value': (v: string) => emit('update:level', v),
            })
          : null,
        h(NButton, { onClick: () => emit('refresh') }, () => '刷新'),
        props.autoRefresh !== undefined
          ? h(NButton, { quaternary: true, onClick: () => emit('toggle-auto') }, () =>
              props.autoRefresh ? '关闭自动刷新' : '自动刷新',
            )
          : null,
      ])
  },
})

const message = useMessage()
const route = useRoute()
const router = useRouter()
const LOG_TABS = ['system', 'access', 'nginx', 'frp'] as const
type LogTab = (typeof LOG_TABS)[number]

function resolveTab(query: RouteLocationNormalizedLoaded['query']): LogTab {
  const qtab = query.tab
  if (qtab === 'error' || qtab === 'nginx') return 'nginx'
  if (qtab === 'stream') return 'access'
  if (qtab === 'frp') return 'frp'
  if (typeof qtab === 'string' && (LOG_TABS as readonly string[]).includes(qtab)) {
    return qtab as LogTab
  }
  if (typeof query.kind === 'string' && query.kind.trim()) return 'access'
  if (typeof query.ip === 'string' && query.ip.trim()) return 'access'
  return 'system'
}

function applyRouteQuery(from: RouteLocationNormalizedLoaded = route) {
  syncingRouteQuery = true
  accessKind.value = typeof from.query.kind === 'string' ? from.query.kind : ''
  accessClientIP.value = typeof from.query.ip === 'string' ? from.query.ip : ''
  if (typeof from.query.keyword === 'string') {
    accessKeyword.value = from.query.keyword
  }
  tab.value = resolveTab(from.query)
  syncingRouteQuery = false
}

const ACCESS_KIND_LABELS: Record<string, string> = {
  intercept: '策略拦截（403 / 429 / 401）',
  '403': 'HTTP 403',
  '429': 'HTTP 429',
  '401': 'HTTP 401',
  scan: '疑似扫描',
}

const accessFilterBanner = computed(() => {
  const parts: string[] = []
  if (accessKind.value) {
    parts.push(`筛选：${ACCESS_KIND_LABELS[accessKind.value] ?? accessKind.value}`)
  }
  if (accessClientIP.value) {
    parts.push(`来源 IP：${accessClientIP.value}`)
  }
  if (parts.length === 0) return ''
  return `${parts.join(' · ')} · 共 ${filteredAccessLogs.value.length} 条（最近日志片段内匹配）`
})

const tab = ref<LogTab>(resolveTab(route.query))
const pageError = ref('')
const autoRefresh = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const accessLogs = ref<AccessLogEntry[]>([])
const proxyRules = ref<ProxyRule[]>([])
const proxyBindingIndex = computed(() => buildProxyBindingIndex(proxyRules.value))
const errorLogs = ref<ParsedNginxError[]>([])
const frpLogs = ref<ParsedFrpLog[]>([])
const systemLogs = ref<SystemLogEntry[]>([])
const loadingAccess = ref(false)
const loadingError = ref(false)
const loadingFrp = ref(false)
const loadingSystem = ref(false)

const accessKeyword = ref(typeof route.query.keyword === 'string' ? route.query.keyword : '')
const accessStatus = ref<number | null>(null)
const accessKind = ref(typeof route.query.kind === 'string' ? route.query.kind : '')
const accessClientIP = ref(typeof route.query.ip === 'string' ? route.query.ip : '')
const errorKeyword = ref('')
const errorLevel = ref('')
const frpKeyword = ref('')
const frpLevel = ref('')
const systemKeyword = ref('')
const systemLevel = ref('')

const accessPage = ref(1)
const errorPage = ref(1)
const frpPage = ref(1)
const systemPage = ref(1)

/** Avoid writing URL while applying ?kind= / ?ip= from navigation (route.query can lag one tick). */
let syncingRouteQuery = false

const usesServerAccessFilter = computed(() => Boolean(accessKind.value.trim() || accessClientIP.value.trim()))

const accessEmptyTitle = computed(() => {
  if (accessKind.value === 'intercept') return '暂无策略拦截记录'
  if (accessClientIP.value) return '该 IP 暂无匹配访问'
  return '暂无访问日志'
})

const accessEmptyDescription = computed(() => {
  if (accessKind.value === 'intercept') {
    return '近期反代访问中未出现 403/429/401。可在反代规则中启用 IP 白名单、限流或认证后再观察。'
  }
  if (accessClientIP.value) {
    return `未在最近扫描的 access.log 片段中找到 ${accessClientIP.value} 的记录。`
  }
  return '此处记录经 Nginx 反向代理的访问（非 Fonu 管理界面本身）。请通过代理域名访问后刷新。'
})

const showAccessBlockReason = computed(
  () =>
    accessKind.value === 'intercept' ||
    accessKind.value === '403' ||
    accessKind.value === '429' ||
    accessKind.value === '401' ||
    accessKind.value === 'scan' ||
    accessLogs.value.some((row) => Boolean(row.block_reason)),
)

const accessTableScrollX = computed(() => (showAccessBlockReason.value ? 1120 : 920))

const filteredAccessLogs = computed(() => {
  if (usesServerAccessFilter.value) {
    return accessLogs.value
  }
  return accessLogs.value.filter((log) => {
    if (accessKeyword.value) {
      const kw = accessKeyword.value.toLowerCase()
      const service = resolveAccessServiceLabel(log, proxyBindingIndex.value)
      const fallback = formatAccessFallback(log)
      if (!`${service} ${fallback} ${log.domain} ${log.path} ${log.client_ip}`.toLowerCase().includes(kw)) return false
    }
    if (accessStatus.value && log.status !== accessStatus.value) return false
    return true
  })
})

const filteredErrorLogs = computed(() => {
  const kw = errorKeyword.value.toLowerCase()
  return errorLogs.value.filter(entry => entry.raw.toLowerCase().includes(kw) && levelMatches(nginxLevelTagType(entry.level), errorLevel.value))
})

const filteredFrpLogs = computed(() => frpLogs.value.filter(entry => entry.raw.toLowerCase().includes(frpKeyword.value.toLowerCase()) && levelMatches(frpLevelTagType(entry.level), frpLevel.value)))

const filteredSystemLogs = computed(() => {
  return systemLogs.value.filter((log) => {
    const display = displaySystemLog(log)
    if (!levelMatches(systemLevelTagType(display.displayLevel), systemLevel.value)) return false
    if (
      systemKeyword.value &&
      !`${display.displayModule} ${log.message}`.toLowerCase().includes(systemKeyword.value.toLowerCase())
    ) {
      return false
    }
    return true
  })
})

function levelMatches(type: string, level: string) { return !level || type === ({ INFO:'info', WARN:'warning', ERROR:'error' } as Record<string,string>)[level] }
function paginate<T>(items: T[], page: number) {
  const start = (page - 1) * PAGE_SIZE
  return items.slice(start, start + PAGE_SIZE)
}

const pagedAccessLogs = computed(() => paginate(filteredAccessLogs.value, accessPage.value))
const pagedErrorLogs = computed(() => paginate(filteredErrorLogs.value, errorPage.value))
const pagedFrpLogs = computed(() => paginate(filteredFrpLogs.value, frpPage.value))
const pagedSystemLogs = computed(() => paginate(filteredSystemLogs.value, systemPage.value))

const methodTagType = (method: string) => {
  const map: Record<string, 'success' | 'info' | 'warning' | 'error' | 'default'> = {
    GET: 'success',
    POST: 'info',
    PUT: 'warning',
    DELETE: 'error',
  }
  return map[method] ?? 'default'
}

function levelBadge(type: string, text: string) { return h(StatusBadge, { kind: type === 'error' ? 'error' : type === 'warning' ? 'warning' : 'unknown', text }) }
const logTimeCell = (time: string) =>
  h('span', { class: 'log-time' }, formatLogTime(time, { timeZone: logTimeZone.value }))

function accessRowClassName(row: AccessLogEntry) {
  if (row.status >= 500) return 'log-row log-row--error'
  if (row.status >= 400) return 'log-row log-row--warn'
  return 'log-row'
}

function errorRowClassName(row: ParsedNginxError) {
  const type = nginxLevelTagType(row.level)
  if (type === 'error') return 'log-row log-row--error'
  if (type === 'warning') return 'log-row log-row--warn'
  return 'log-row'
}

function frpRowClassName(row: ParsedFrpLog) {
  const type = frpLevelTagType(row.level)
  if (type === 'error') return 'log-row log-row--error'
  if (type === 'warning') return 'log-row log-row--warn'
  return 'log-row'
}

function systemRowClassName(row: SystemLogEntry) {
  const display = displaySystemLog(row)
  if (display.displayLevel === '错误') return 'log-row log-row--error'
  if (display.displayLevel === '警告') return 'log-row log-row--warn'
  return 'log-row'
}

const accessColumns = computed<DataTableColumns<AccessLogEntry>>(() => {
  const cols: DataTableColumns<AccessLogEntry> = [
  { title: '时间', key: 'time', width: 168, render: (row) => logTimeCell(row.time) },
  {
    title: '服务',
    key: 'service',
    width: 128,
    render: (row) => {
      const tooltip = accessServiceTooltip(row, proxyBindingIndex.value)
      const label = resolveAccessServiceLabel(row, proxyBindingIndex.value)
      return h('span', { class: 'log-service nowrap', title: tooltip }, label)
    },
  },
  {
    title: '方法',
    key: 'method',
    width: 68,
    render: (row) => h(NTag, { size: 'small', bordered: false, type: methodTagType(row.method) }, () => row.method),
  },
  {
    title: '路径',
    key: 'path',
    width: 220,
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'log-path mono nowrap', title: row.path }, row.path),
  },
  {
    title: '状态',
    key: 'status',
    width: 68,
    render: (row) => h(HttpStatusBadge, { code: row.status }),
  },
  ]

  if (showAccessBlockReason.value) {
    cols.push({
      title: '拦截原因',
      key: 'block_reason',
      width: 200,
      ellipsis: { tooltip: true },
      render: (row) =>
        h(
          'span',
          { class: 'log-block-reason', title: row.block_reason || '—' },
          row.block_reason?.trim() || '—',
        ),
    })
  }

  cols.push(
  {
    title: '耗时',
    key: 'response_time',
    width: 80,
    render: (row) => h('span', { class: latencyClass(row.response_time) }, formatMs(row.response_time)),
  },
  {
    title: '来源 IP',
    key: 'client_ip',
    width: 168,
    render: (row) => {
      const geoRaw = row.geo?.trim()
      const geo = geoRaw ? displayGeoLabel(geoRaw) : ''
      return h('div', { class: 'ip-geo-cell' }, [
        h('span', { class: 'mono ip-geo-cell__ip', title: row.client_ip }, row.client_ip),
        geo
          ? h('span', { class: 'ip-geo-cell__geo', title: geoRaw }, geo)
          : h('span', { class: 'ip-geo-cell__geo ip-geo-cell__geo--empty' }, '—'),
      ])
    },
  },
  )
  return cols
})

const errorColumns: DataTableColumns<ParsedNginxError> = [
  { title: '时间', key: 'time', width: 170, render: (row) => logTimeCell(row.time || '-') },
  {
    title: '级别',
    key: 'level',
    width: 88,
    render: (row) =>
      levelBadge(nginxLevelTagType(row.level), row.level === 'unknown' ? '未知' : row.level.toUpperCase()),
  },
  {
    title: '消息',
    key: 'message',
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'log-message mono' }, row.message),
  },
]

const frpColumns: DataTableColumns<ParsedFrpLog> = [
  { title: '时间', key: 'time', width: 170, render: (row) => logTimeCell(row.time || '-') },
  {
    title: '级别',
    key: 'level',
    width: 88,
    render: (row) =>
      levelBadge(frpLevelTagType(row.level), row.level === 'unknown' ? '未知' : row.level),
  },
  {
    title: '来源',
    key: 'source',
    width: 180,
    ellipsis: { tooltip: true },
    render: (row) =>
      h('span', { class: 'mono nowrap', title: row.source }, row.source || '—'),
  },
  {
    title: '消息',
    key: 'message',
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'log-message mono' }, row.message),
  },
]

const systemColumns: DataTableColumns<SystemLogEntry> = [
  { title: '时间', key: 'time', width: 168, render: (row) => logTimeCell(row.time) },
  {
    title: '级别',
    key: 'level',
    width: 80,
    render: (row) => {
      const display = displaySystemLog(row)
      return levelBadge(systemLevelTagType(display.displayLevel), display.displayLevel)
    },
  },
  {
    title: '模块',
    key: 'module',
    width: 108,
    render: (row) => {
      const display = displaySystemLog(row)
      return h(
        NTag,
        {
          size: 'small',
          bordered: false,
          type: systemModuleTagType(display.displayModule),
          class: 'log-module-tag',
        },
        () => display.displayModule,
      )
    },
  },
  {
    title: '消息',
    key: 'message',
    ellipsis: { tooltip: true },
    render: (row) => {
      const display = displaySystemLog(row)
      return h('span', { class: 'log-message' }, display.message)
    },
  },
]

const loaded = ref<Record<LogTab, boolean>>({ system:false, access:false, nginx:false, frp:false })
const errors = ref<Record<LogTab, string>>({ system:'', access:'', nginx:'', frp:'' })
const sourceCards = computed(() => [
  { key:'system' as const, label:'系统日志', icon:DesktopOutline, count:systemLogs.value.length },
  { key:'access' as const, label:'访问日志', icon:GlobeOutline, count:accessLogs.value.length },
  { key:'nginx' as const, label:'Nginx 日志', icon:AlertCircleOutline, count:errorLogs.value.length },
  { key:'frp' as const, label:'FRP 日志', icon:GitNetworkOutline, count:frpLogs.value.length },
])
const activeRows = computed(() => ({ system:filteredSystemLogs.value, access:filteredAccessLogs.value, nginx:filteredErrorLogs.value, frp:filteredFrpLogs.value })[tab.value])
const activeLoading = computed(() => ({ system:loadingSystem.value, access:loadingAccess.value, nginx:loadingError.value, frp:loadingFrp.value })[tab.value])
const showDetail = ref(false)
const detail = ref<{ fields: {label:string;value:string}[]; raw:string; rawLabel:string; ip?:string } | null>(null)
function detailRowProps(kind: LogTab, row: unknown) {
  const open = () => openDetail(kind, row)
  return { tabindex:0, style:'cursor:pointer', onClick:open, onKeydown:(event:KeyboardEvent) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault();open() } } }
}
function openDetail(kind: LogTab, row: unknown) {
  const fields: { label:string;value:string }[] = []
  const add = (label:string, value:unknown) => fields.push({label,value:String(value ?? '')})
  let raw = JSON.stringify(row,null,2), rawLabel = '记录字段（JSON）', ip:string|undefined
  if (kind === 'access') {
    const r = row as AccessLogEntry
    add('访问时间', formatLogTime(r.time,{timeZone:logTimeZone.value}));add('服务',resolveAccessServiceLabel(r,proxyBindingIndex.value));add('域名',r.domain);add('端口',r.server_port);add('请求方法',r.method);add('路径',r.path);add('客户端 IP',r.client_ip);add('归属地',r.geo ? displayGeoLabel(r.geo) : '');add('HTTP 状态',r.status);add('响应耗时',formatMs(r.response_time));add('上游',r.upstream);add('拦截原因（推测）',r.block_reason);ip=r.client_ip
  } else if (kind === 'system') {
    const r = displaySystemLog(row as SystemLogEntry)
    add('时间',formatLogTime(r.time,{timeZone:logTimeZone.value}));add('级别',r.displayLevel);add('模块',r.displayModule);add('消息',r.message)
  } else {
    const r = row as ParsedNginxError | ParsedFrpLog
    add('时间',r.time ? formatLogTime(r.time,{timeZone:logTimeZone.value}) : '');add('级别',r.level);if ('source' in r) add('来源',r.source);add('消息',r.message);raw=r.raw;rawLabel='原始日志'
  }
  detail.value={fields,raw,rawLabel,ip};showDetail.value=true
}
async function copyDetail() {
  if (!detail.value) return
  if (await copyToClipboard(detail.value.raw)) message.success('已复制记录')
  else message.error('复制失败')
}
function filterDetailIP() {
  if (!detail.value?.ip) return
  showDetail.value=false
  void router.replace({name:'logs',query:{tab:'access',ip:detail.value.ip}})
}
function exportLoaded() {
  const blob=new Blob([JSON.stringify(activeRows.value.map(row => 'raw' in row ? row.raw : row),null,2)],{type:'application/json;charset=utf-8'})
  const url=URL.createObjectURL(blob), a=document.createElement('a')
  a.href=url;a.download='fonu-'+tab.value+'-logs.json';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)
}
async function loadProxyRules() {
  try {
    proxyRules.value = asList(await api.listProxies())
  } catch {
    proxyRules.value = []
  }
}

let accessGeneration = 0
let accessRequestKey = ''
let accessDebounce: ReturnType<typeof setTimeout> | undefined
let disposed = false
async function loadAccess() {
  const params = {limit:usesServerAccessFilter.value ? 200 : 100, keyword:accessKeyword.value.trim() || undefined, status:accessStatus.value || undefined, kind:accessKind.value.trim() || undefined, ip:accessClientIP.value.trim() || undefined}
  const key=JSON.stringify(params)
  if (loadingAccess.value && key === accessRequestKey) return
  const generation=++accessGeneration
  accessRequestKey=key
  loadingAccess.value=true;errors.value.access=''
  try {
    await loadProxyRules()
    if (disposed || generation !== accessGeneration) return
    const rows=asList(await api.getAccessLogs(params))
    if (disposed || generation !== accessGeneration) return
    accessLogs.value=rows;loaded.value.access=true
  } catch (error) {
    if (!disposed && generation === accessGeneration) errors.value.access=error instanceof Error ? error.message : '加载访问日志失败'
  } finally {
    if (generation === accessGeneration) loadingAccess.value=false
  }
}
async function loadErrorLogs() {
  if (loadingError.value) return
  loadingError.value=true;errors.value.nginx=''
  try { const rows=asList(await api.getErrorLogs({limit:100}));if (!disposed) {errorLogs.value=rows.map(parseNginxErrorLine);loaded.value.nginx=true} }
  catch(error) {if (!disposed) errors.value.nginx=error instanceof Error ? error.message : '加载 Nginx 日志失败'}
  finally {loadingError.value=false}
}
async function loadFrpLogs() {
  if (loadingFrp.value) return
  loadingFrp.value=true;errors.value.frp=''
  try {const rows=asList(await api.getFRPLogs(200));if (!disposed) {frpLogs.value=rows.map(parseFrpLogLine);loaded.value.frp=true}}
  catch(error) {if (!disposed) errors.value.frp=error instanceof Error ? error.message : '加载 FRP 日志失败'}
  finally {loadingFrp.value=false}
}
async function loadSystemLogs() {
  if (loadingSystem.value) return
  loadingSystem.value=true;errors.value.system=''
  try {const rows=asList(await api.getSystemLogs({limit:100}));if (!disposed) {systemLogs.value=rows;loaded.value.system=true}}
  catch(error) {if (!disposed) errors.value.system=error instanceof Error ? error.message : '加载系统日志失败'}
  finally {loadingSystem.value=false}
}
function loadCurrent() {
  if (disposed) return
  if(tab.value==='access') void loadAccess()
  else if(tab.value==='nginx') void loadErrorLogs()
  else if(tab.value==='frp') void loadFrpLogs()
  else void loadSystemLogs()
}
function loadAll() {pageError.value='';loadCurrent()}
function scheduleAccess() {
  clearTimeout(accessDebounce)
  accessDebounce=setTimeout(()=>{if(tab.value==='access' && !disposed) void loadAccess()},250)
}
watch([accessKeyword,accessStatus],()=>{accessPage.value=1;if(tab.value==='access' && usesServerAccessFilter.value) scheduleAccess()})
watch(accessKind,kind=>{
  accessPage.value=1
  if(syncingRouteQuery)return
  const query={...route.query,tab:'access',kind:kind || undefined}
  if((route.query.kind || '')!==kind) void router.replace({name:'logs',query})
},{flush:'sync'})
watch([errorKeyword,errorLevel],()=>{errorPage.value=1})
watch([frpKeyword,frpLevel],()=>{frpPage.value=1})
watch([systemKeyword,systemLevel],()=>{systemPage.value=1})
watch(filteredAccessLogs,rows=>{accessPage.value=Math.min(accessPage.value,Math.max(1,Math.ceil(rows.length/PAGE_SIZE)))})
watch(filteredErrorLogs,rows=>{errorPage.value=Math.min(errorPage.value,Math.max(1,Math.ceil(rows.length/PAGE_SIZE)))})
watch(filteredFrpLogs,rows=>{frpPage.value=Math.min(frpPage.value,Math.max(1,Math.ceil(rows.length/PAGE_SIZE)))})
watch(filteredSystemLogs,rows=>{systemPage.value=Math.min(systemPage.value,Math.max(1,Math.ceil(rows.length/PAGE_SIZE)))})
watch(autoRefresh,on=>{
  if(refreshTimer)clearInterval(refreshTimer)
  refreshTimer=on ? setInterval(()=>{if(!document.hidden && route.name==='logs' && !activeLoading.value)loadCurrent()},10000):null
})
watch(()=>route.fullPath,()=>{
  if(route.name!=='logs')return
  applyRouteQuery()
  if(tab.value==='access') {accessPage.value=1;scheduleAccess()}
  else loadCurrent()
})
watch(tab,name=>{
  if(route.name!=='logs')return
  if(resolveTab(route.query)!==name)void router.replace({name:'logs',query:{...route.query,tab:name}})
  else if(name==='access')scheduleAccess()
  else loadCurrent()
})
onMounted(()=>{applyRouteQuery();loadCurrent()})
onUnmounted(()=>{disposed=true;accessGeneration++;clearTimeout(accessDebounce);if(refreshTimer)clearInterval(refreshTimer)})
</script>

<style scoped>
.logs-card :deep(.n-tabs-nav) {
  padding: var(--fonu-space-4) var(--fonu-space-5) 0;
}

.logs-card :deep(.n-tabs-tab) {
  font-size: 14px;
}

.logs-card :deep(.n-tab-pane) {
  padding-top: var(--fonu-space-2);
}

.tab-hint {
  margin: 0 var(--fonu-space-5) var(--fonu-space-3);
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.access-filter-banner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--fonu-space-2);
  padding: var(--fonu-space-2) var(--fonu-space-3);
  border-radius: 8px;
  background: var(--fonu-surface-muted, rgba(0, 0, 0, 0.04));
  border: 1px solid var(--fonu-border);
  color: var(--fonu-text);
}

.access-filter-clear {
  margin-left: auto;
  font-size: 13px;
  color: var(--fonu-primary);
  text-decoration: none;
}

.access-filter-clear:hover {
  text-decoration: underline;
}

.log-pagination {
  display: flex;
  justify-content: flex-end;
  padding: var(--fonu-space-4) var(--fonu-space-5);
  border-top: 1px solid var(--fonu-border);
}

.logs-card :deep(.log-table .n-data-table-th) {
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
}

.logs-card :deep(.log-table .n-data-table-base-table) {
  table-layout: fixed;
}

.logs-card :deep(.log-table .n-data-table-td) {
  font-size: 13px;
  padding-top: 10px;
  padding-bottom: 10px;
}

.nowrap {
  white-space: nowrap;
}

.logs-card :deep(.log-table .n-data-table-tr.log-row--error .n-data-table-td) {
  background: color-mix(in srgb, #ef4444 7%, transparent);
}

.logs-card :deep(.log-table .n-data-table-tr.log-row--warn .n-data-table-td) {
  background: color-mix(in srgb, #f59e0b 7%, transparent);
}

.logs-card :deep(.log-table .n-data-table-tr.log-row:hover .n-data-table-td) {
  background: color-mix(in srgb, var(--fonu-brand) 5%, var(--fonu-surface));
}

.log-time {
  white-space: nowrap;
  color: var(--fonu-text-secondary);
  font-variant-numeric: tabular-nums;
}

.log-service {
  font-weight: 500;
}

.log-path {
  color: var(--fonu-text-secondary);
}

.log-message {
  line-height: 1.5;
}

.mono {
  font-family: var(--fonu-mono);
  font-size: 12px;
}

.logs-card :deep(.ip-geo-cell) {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  width: 100%;
}

.logs-card :deep(.ip-geo-cell__ip) {
  display: block;
  width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.logs-card :deep(.ip-geo-cell__geo) {
  display: block;
  width: 100%;
  font-size: 11px;
  color: var(--fonu-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.logs-card :deep(.ip-geo-cell__geo--empty) {
  opacity: 0.45;
}

.ip-geo-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  min-width: 0;
  max-width: 100%;
}

.ip-geo-cell__ip {
  display: block;
  white-space: nowrap;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ip-geo-cell__geo {
  display: block;
  font-size: 11px;
  line-height: 1.35;
  color: var(--fonu-text-muted);
  white-space: nowrap;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-duration {
  font-variant-numeric: tabular-nums;
}

.log-duration--slow {
  color: #d97706;
  font-weight: 600;
}

.log-duration--critical {
  color: #dc2626;
  font-weight: 600;
}

.logs-card :deep(.log-module-tag) {
  white-space: nowrap;
}
@media (max-width: 760px) {
  .logs-card { min-width: 0; }
}
.logs-card :deep(.fonu-card__header) { padding-bottom:10px; }
.logs-card :deep(.n-tabs-nav) { background:var(--fonu-bg-muted); padding:8px 12px 0; }
.logs-card :deep(.log-table .n-data-table-td) { padding-top:7px; padding-bottom:7px; font-size:12px; }
.logs-card :deep(.log-table .n-data-table-tr.log-row--error .n-data-table-td) { background:color-mix(in srgb,var(--fonu-error) 7%,transparent); }
.logs-card :deep(.log-table .n-data-table-tr.log-row--warn .n-data-table-td) { background:color-mix(in srgb,var(--fonu-warning) 7%,transparent); }

.log-workspace { display:grid; gap:12px; min-width:0; container-type:inline-size; }
.log-source-grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:10px; }
.log-source { min-width:0; display:grid; gap:6px; padding:14px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); background:var(--fonu-surface); text-align:left; color:var(--fonu-text); font:inherit; cursor:pointer; }
.log-source.is-active { border-color:var(--fonu-brand); background:color-mix(in srgb,var(--fonu-brand-soft) 45%,var(--fonu-surface)); }
.source-heading { display:flex; align-items:center; gap:7px; font-size:13px; color:var(--fonu-text-secondary); }
.source-heading .n-icon { font-size:18px; color:var(--fonu-brand-text); }
.source-check { margin-left:auto; }
.log-source strong { font-size:23px; line-height:1.2; font-variant-numeric:tabular-nums; }
.log-source small { font-size:11px; color:var(--fonu-text-secondary); }
.log-workspace-head { display:flex; align-items:center; justify-content:space-between; gap:12px; padding:16px; flex-wrap:wrap; border-bottom:1px solid var(--fonu-border); }
.log-workspace-head strong { font-size:16px; }
.log-workspace-head small { display:block; margin-top:4px; font-size:12px; color:var(--fonu-text-secondary); }
.log-head-actions { display:flex; gap:8px; flex-wrap:wrap; }
.log-error,.log-loading { display:flex; align-items:center; gap:10px; padding:12px 16px; color:var(--fonu-text-secondary); font-size:12px; }
.log-error { color:var(--fonu-error); }
.detail-fields { display:grid; gap:0; }
.detail-fields > div { display:grid; grid-template-columns:105px minmax(0,1fr); gap:12px; padding:10px 0; border-bottom:1px solid var(--fonu-border); font-size:13px; }
.detail-fields span { color:var(--fonu-text-secondary); }
.detail-fields strong { font-weight:500; overflow-wrap:anywhere; }
.detail-console { padding:12px; max-height:260px; margin:0; }
.detail-actions { display:flex; gap:8px; margin-top:12px; flex-wrap:wrap; }
@container(max-width:680px) { .log-source-grid {grid-template-columns:repeat(2,minmax(0,1fr))} .log-source{padding:10px} .logs-card :deep(.n-tabs-nav){padding-inline:12px} }
</style>
