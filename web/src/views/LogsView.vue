<template>
  <PageHeader
    title="日志中心"
    description="系统日志记录 Fonu 全部运行输出；访问日志与 Nginx 错误日志来自反向代理"
  />

  <LoadError v-if="pageError" :message="pageError" @retry="loadAll" />

  <FonuCard v-else flush class="logs-card">
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
        />
        <LogPagination
          v-if="filteredSystemLogs.length > 0"
          v-model:page="systemPage"
          :item-count="filteredSystemLogs.length"
        />
        <EmptyState
          v-if="!loadingSystem && systemLogs.length === 0"
          title="暂无系统日志"
          description="应用启动、DDNS 更新、证书操作或 API 请求会记录在这里。"
        />
      </n-tab-pane>

      <n-tab-pane name="access" tab="访问日志">
        <LogToolbar
          v-model:keyword="accessKeyword"
          v-model:status="accessStatus"
          :show-domain="true"
          :show-status="true"
          :auto-refresh="autoRefresh"
          @refresh="loadAccess"
          @toggle-auto="autoRefresh = !autoRefresh"
        />
        <n-data-table
          v-if="filteredAccessLogs.length > 0"
          class="log-table"
          :columns="accessColumns"
          :data="pagedAccessLogs"
          :loading="loadingAccess"
          :bordered="false"
          :scroll-x="920"
          :row-class-name="accessRowClassName"
        />
        <LogPagination
          v-if="filteredAccessLogs.length > 0"
          v-model:page="accessPage"
          :item-count="filteredAccessLogs.length"
        />
        <EmptyState
          v-if="!loadingAccess && accessLogs.length === 0"
          title="暂无访问日志"
          description="此处记录经 Nginx 反向代理的访问（非 Fonu 管理界面本身）。请通过代理域名访问后刷新。"
        />
      </n-tab-pane>

      <n-tab-pane name="nginx" tab="Nginx 日志">
        <p class="tab-hint">Nginx 错误日志（error.log），记录 SSL 握手失败、上游连接异常、配置冲突等。</p>
        <LogToolbar v-model:keyword="errorKeyword" @refresh="loadErrorLogs" />
        <n-data-table
          v-if="filteredErrorLogs.length > 0"
          class="log-table"
          :columns="errorColumns"
          :data="pagedErrorLogs"
          :loading="loadingError"
          :bordered="false"
          :row-class-name="errorRowClassName"
        />
        <LogPagination
          v-if="filteredErrorLogs.length > 0"
          v-model:page="errorPage"
          :item-count="filteredErrorLogs.length"
        />
        <EmptyState
          v-if="!loadingError && errorLogs.length === 0"
          title="暂无 Nginx 错误日志"
          description="Nginx 出现 SSL、上游或配置相关错误时会记录在这里。"
        />
      </n-tab-pane>
    </n-tabs>
  </FonuCard>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  NButton,
  NDataTable,
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
import { formatLogTime, formatMs } from '../utils/format'
import {
  latencyClass,
  nginxLevelTagType,
  parseNginxErrorLine,
  type ParsedNginxError,
  displaySystemLog,
  systemLevelTagType,
  systemModuleTagType,
} from '../utils/logDisplay'
import { httpStatusKind } from '../utils/status'

const PAGE_SIZE = 20

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
    level: { type: String, default: '' },
    showDomain: Boolean,
    showStatus: Boolean,
    showLevel: Boolean,
    autoRefresh: Boolean,
  },
  emits: ['update:keyword', 'update:status', 'update:level', 'refresh', 'toggle-auto'],
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
              options: [200, 301, 400, 403, 404, 500].map((c) => ({ label: String(c), value: c })),
              'onUpdate:value': (v: number | null) => emit('update:status', v),
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
const LOG_TABS = ['system', 'access', 'nginx'] as const
type LogTab = (typeof LOG_TABS)[number]

function resolveTab(queryTab: unknown): LogTab {
  if (queryTab === 'error' || queryTab === 'nginx') return 'nginx'
  if (queryTab === 'stream') return 'access'
  if (typeof queryTab === 'string' && (LOG_TABS as readonly string[]).includes(queryTab)) {
    return queryTab as LogTab
  }
  return 'system'
}

const tab = ref(resolveTab(route.query.tab))
const pageError = ref('')
const autoRefresh = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const accessLogs = ref<AccessLogEntry[]>([])
const proxyRules = ref<ProxyRule[]>([])
const proxyBindingIndex = computed(() => buildProxyBindingIndex(proxyRules.value))
const errorLogs = ref<ParsedNginxError[]>([])
const systemLogs = ref<SystemLogEntry[]>([])
const loadingAccess = ref(false)
const loadingError = ref(false)
const loadingSystem = ref(false)

const accessKeyword = ref(typeof route.query.keyword === 'string' ? route.query.keyword : '')
const accessStatus = ref<number | null>(null)
const errorKeyword = ref('')
const systemKeyword = ref('')
const systemLevel = ref('')

const accessPage = ref(1)
const errorPage = ref(1)
const systemPage = ref(1)

const filteredAccessLogs = computed(() => {
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
  if (!errorKeyword.value) return errorLogs.value
  const kw = errorKeyword.value.toLowerCase()
  return errorLogs.value.filter((entry) => entry.raw.toLowerCase().includes(kw))
})

const filteredSystemLogs = computed(() => {
  return systemLogs.value.filter((log) => {
    const display = displaySystemLog(log)
    if (systemLevel.value && log.level.toUpperCase() !== systemLevel.value) return false
    if (
      systemKeyword.value &&
      !`${display.displayModule} ${log.message}`.toLowerCase().includes(systemKeyword.value.toLowerCase())
    ) {
      return false
    }
    return true
  })
})

function paginate<T>(items: T[], page: number) {
  const start = (page - 1) * PAGE_SIZE
  return items.slice(start, start + PAGE_SIZE)
}

const pagedAccessLogs = computed(() => paginate(filteredAccessLogs.value, accessPage.value))
const pagedErrorLogs = computed(() => paginate(filteredErrorLogs.value, errorPage.value))
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

const statusTagType = (code: number) => {
  const kind = httpStatusKind(code)
  if (kind === 'success') return 'success'
  if (kind === 'warning') return 'warning'
  if (kind === 'error') return 'error'
  return 'info'
}

const logTimeCell = (time: string) =>
  h('span', { class: 'log-time' }, formatLogTime(time))

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

function systemRowClassName(row: SystemLogEntry) {
  const display = displaySystemLog(row)
  if (display.displayLevel === '错误') return 'log-row log-row--error'
  if (display.displayLevel === '警告') return 'log-row log-row--warn'
  return 'log-row'
}

const accessColumns = computed<DataTableColumns<AccessLogEntry>>(() => [
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
    render: (row) => h(NTag, { size: 'small', bordered: false, type: statusTagType(row.status) }, () => String(row.status)),
  },
  {
    title: '耗时',
    key: 'response_time',
    width: 80,
    render: (row) => h('span', { class: latencyClass(row.response_time) }, formatMs(row.response_time)),
  },
  {
    title: '来源 IP',
    key: 'client_ip',
    width: 140,
    render: (row) => h('span', { class: 'mono nowrap', title: row.client_ip }, row.client_ip),
  },
])

const errorColumns: DataTableColumns<ParsedNginxError> = [
  { title: '时间', key: 'time', width: 170, render: (row) => logTimeCell(row.time || '-') },
  {
    title: '级别',
    key: 'level',
    width: 88,
    render: (row) =>
      h(
        NTag,
        { size: 'small', bordered: false, type: nginxLevelTagType(row.level) },
        () => (row.level === 'unknown' ? '未知' : row.level.toUpperCase()),
      ),
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
      return h(
        NTag,
        { size: 'small', bordered: false, type: systemLevelTagType(display.displayLevel) },
        () => display.displayLevel,
      )
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

async function loadProxyRules() {
  try {
    proxyRules.value = asList(await api.listProxies())
  } catch {
    proxyRules.value = []
  }
}

async function loadAccess() {
  loadingAccess.value = true
  try {
    await loadProxyRules()
    accessLogs.value = asList(await api.getAccessLogs({ limit: 100 }))
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载访问日志失败')
  } finally {
    loadingAccess.value = false
  }
}

async function loadErrorLogs() {
  loadingError.value = true
  try {
    const errors = asList(await api.getErrorLogs({ limit: 100 }))
    errorLogs.value = errors.map((line) => parseNginxErrorLine(line))
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载错误日志失败')
  } finally {
    loadingError.value = false
  }
}

async function loadSystemLogs() {
  loadingSystem.value = true
  try {
    systemLogs.value = asList(await api.getSystemLogs({ limit: 100 }))
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载系统日志失败')
  } finally {
    loadingSystem.value = false
  }
}

async function loadAll() {
  pageError.value = ''
  try {
    await Promise.all([loadAccess(), loadErrorLogs(), loadSystemLogs()])
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  }
}

watch([accessKeyword, accessStatus], () => {
  accessPage.value = 1
})

watch(errorKeyword, () => {
  errorPage.value = 1
})

watch([systemKeyword, systemLevel], () => {
  systemPage.value = 1
})

watch(autoRefresh, (on) => {
  if (refreshTimer) clearInterval(refreshTimer)
  if (on) refreshTimer = setInterval(() => loadAccess(), 10000)
})

watch(
  () => route.query.keyword,
  (keyword) => {
    if (typeof keyword === 'string') accessKeyword.value = keyword
  },
)

watch(
  () => route.query.tab,
  (queryTab) => {
    tab.value = resolveTab(queryTab)
  },
)

watch(tab, (name) => {
  if (name === 'access') loadAccess()
  else if (name === 'nginx') loadErrorLogs()
  else if (name === 'system') loadSystemLogs()
})

onMounted(() => {
  tab.value = resolveTab(route.query.tab)
  loadAll()
})
onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
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
</style>
