<template>
  <PageHeader title="日志" description="查看访问、错误与系统运行日志" />

  <LoadError v-if="pageError" :message="pageError" @retry="loadAll" />

  <FonuCard v-else flush class="logs-card">
    <n-tabs v-model:value="tab" type="line" animated class="logs-tabs">
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
          :columns="accessColumns"
          :data="pagedAccessLogs"
          :loading="loadingAccess"
          :bordered="false"
          :scroll-x="1000"
        />
        <LogPagination
          v-if="filteredAccessLogs.length > 0"
          v-model:page="accessPage"
          :item-count="filteredAccessLogs.length"
        />
        <EmptyState
          v-if="!loadingAccess && accessLogs.length === 0"
          title="暂无访问日志"
          description="产生访问请求后，日志会显示在这里。"
        />
      </n-tab-pane>

      <n-tab-pane name="error" tab="错误日志">
        <LogToolbar v-model:keyword="errorKeyword" @refresh="loadErrorLogs" />
        <n-data-table
          v-if="filteredErrorLogs.length > 0"
          :columns="errorColumns"
          :data="pagedErrorLogs"
          :loading="loadingError"
          :bordered="false"
        />
        <LogPagination
          v-if="filteredErrorLogs.length > 0"
          v-model:page="errorPage"
          :item-count="filteredErrorLogs.length"
        />
        <EmptyState
          v-if="!loadingError && errorLogs.length === 0"
          title="暂无错误日志"
          description="系统或 Nginx 出现错误时会记录在这里。"
        />
      </n-tab-pane>

      <n-tab-pane name="system" tab="系统日志">
        <LogToolbar
          v-model:keyword="systemKeyword"
          v-model:level="systemLevel"
          :show-level="true"
          @refresh="loadSystemLogs"
        />
        <n-data-table
          v-if="filteredSystemLogs.length > 0"
          :columns="systemColumns"
          :data="pagedSystemLogs"
          :loading="loadingSystem"
          :bordered="false"
        />
        <LogPagination
          v-if="filteredSystemLogs.length > 0"
          v-model:page="systemPage"
          :item-count="filteredSystemLogs.length"
        />
        <EmptyState
          v-if="!loadingSystem && systemLogs.length === 0"
          title="暂无系统日志"
          description="应用运行后会产生系统日志。"
        />
      </n-tab-pane>

      <n-tab-pane name="stream" tab="实时日志">
        <div class="toolbar">
          <n-select v-model:value="streamType" :options="streamOptions" style="width: 180px" />
          <n-button @click="toggleStream">{{ streaming ? '暂停' : '开始' }}</n-button>
          <n-button quaternary @click="clearStream">清空视图</n-button>
        </div>
        <div ref="streamBox" class="stream-box" :class="{ paused: !streaming }">
          <div v-for="(line, i) in streamLines" :key="i" class="stream-line" :class="lineClass(line)">
            {{ formatLogLine(line) }}
          </div>
          <div v-if="streamLines.length === 0" class="stream-empty text-muted">等待日志输出…</div>
        </div>
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
import { api } from '../api/client'
import type { AccessLogEntry, SystemLogEntry } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import { formatLogLine, formatLogTime, formatMs } from '../utils/format'
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
              options: ['INFO', 'WARN', 'ERROR'].map((l) => ({ label: l, value: l })),
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
const LOG_TABS = ['access', 'error', 'system', 'stream'] as const

function resolveTab(queryTab: unknown) {
  if (typeof queryTab === 'string' && (LOG_TABS as readonly string[]).includes(queryTab)) {
    return queryTab
  }
  return 'access'
}

const tab = ref(resolveTab(route.query.tab))
const pageError = ref('')
const autoRefresh = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const accessLogs = ref<AccessLogEntry[]>([])
const errorLogs = ref<{ line: string }[]>([])
const systemLogs = ref<SystemLogEntry[]>([])
const loadingAccess = ref(false)
const loadingError = ref(false)
const loadingSystem = ref(false)

const accessKeyword = ref('')
const accessStatus = ref<number | null>(null)
const errorKeyword = ref('')
const systemKeyword = ref('')
const systemLevel = ref('')

const accessPage = ref(1)
const errorPage = ref(1)
const systemPage = ref(1)

const streamType = ref('system')
const streamLines = ref<string[]>([])
const streaming = ref(false)
const streamBox = ref<HTMLElement | null>(null)
let eventSource: EventSource | null = null

const streamOptions = [
  { label: '系统日志', value: 'system' },
  { label: '访问日志', value: 'access' },
  { label: '错误日志', value: 'error' },
]

const filteredAccessLogs = computed(() => {
  return accessLogs.value.filter((log) => {
    if (accessKeyword.value) {
      const kw = accessKeyword.value.toLowerCase()
      if (!`${log.domain} ${log.path} ${log.client_ip}`.toLowerCase().includes(kw)) return false
    }
    if (accessStatus.value && log.status !== accessStatus.value) return false
    return true
  })
})

const filteredErrorLogs = computed(() => {
  if (!errorKeyword.value) return errorLogs.value
  const kw = errorKeyword.value.toLowerCase()
  return errorLogs.value.filter((l) => l.line.toLowerCase().includes(kw))
})

const filteredSystemLogs = computed(() => {
  return systemLogs.value.filter((log) => {
    if (systemLevel.value && log.level.toUpperCase() !== systemLevel.value) return false
    if (systemKeyword.value && !log.message.toLowerCase().includes(systemKeyword.value.toLowerCase())) return false
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
  h('span', { style: 'white-space: nowrap' }, formatLogTime(time))

const accessColumns: DataTableColumns<AccessLogEntry> = [
  { title: '时间', key: 'time', width: 170, render: (row) => logTimeCell(row.time) },
  { title: '域名', key: 'domain' },
  {
    title: '方法',
    key: 'method',
    width: 80,
    render: (row) => h(NTag, { size: 'small', bordered: false, type: methodTagType(row.method) }, () => row.method),
  },
  { title: '路径', key: 'path', ellipsis: { tooltip: true } },
  {
    title: '状态',
    key: 'status',
    width: 80,
    render: (row) => h(NTag, { size: 'small', bordered: false, type: statusTagType(row.status) }, () => String(row.status)),
  },
  { title: '耗时', key: 'response_time', width: 90, render: (row) => formatMs(row.response_time) },
  { title: '来源 IP', key: 'client_ip', width: 130 },
]

const errorColumns: DataTableColumns<{ line: string }> = [
  { title: '内容', key: 'line', ellipsis: { tooltip: true }, render: (row) => formatLogLine(row.line) },
]

const systemColumns: DataTableColumns<SystemLogEntry> = [
  { title: '时间', key: 'time', width: 170, render: (row) => logTimeCell(row.time) },
  { title: '级别', key: 'level', width: 80 },
  { title: '模块', key: 'module', width: 100 },
  { title: '消息', key: 'message', ellipsis: { tooltip: true } },
]

async function loadAccess() {
  loadingAccess.value = true
  try {
    accessLogs.value = await api.getAccessLogs({ limit: 100 })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载访问日志失败')
  } finally {
    loadingAccess.value = false
  }
}

async function loadErrorLogs() {
  loadingError.value = true
  try {
    const errors = await api.getErrorLogs({ limit: 100 })
    errorLogs.value = errors.map((line) => ({ line }))
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载错误日志失败')
  } finally {
    loadingError.value = false
  }
}

async function loadSystemLogs() {
  loadingSystem.value = true
  try {
    systemLogs.value = await api.getSystemLogs({ limit: 100 })
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

function toggleStream() {
  if (streaming.value) {
    eventSource?.close()
    eventSource = null
    streaming.value = false
    return
  }
  eventSource = new EventSource(`/api/logs/stream?type=${streamType.value}`)
  eventSource.addEventListener('log', (event) => {
    streamLines.value.push(event.data)
    if (streamLines.value.length > 500) streamLines.value = streamLines.value.slice(-400)
    if (streaming.value) {
      requestAnimationFrame(() => {
        streamBox.value?.scrollTo({ top: streamBox.value.scrollHeight })
      })
    }
  })
  streaming.value = true
}

function clearStream() {
  streamLines.value = []
}

function lineClass(line: string) {
  if (line.includes('ERROR')) return 'level-error'
  if (line.includes('WARN')) return 'level-warn'
  if (line.includes('INFO')) return 'level-info'
  return ''
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

watch(streamType, () => {
  if (streaming.value) {
    eventSource?.close()
    streaming.value = false
  }
})

watch(autoRefresh, (on) => {
  if (refreshTimer) clearInterval(refreshTimer)
  if (on) refreshTimer = setInterval(() => loadAccess(), 10000)
})

watch(
  () => route.query.tab,
  (queryTab) => {
    tab.value = resolveTab(queryTab)
  },
)

onMounted(() => {
  tab.value = resolveTab(route.query.tab)
  loadAll()
})
onUnmounted(() => {
  eventSource?.close()
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

.log-pagination {
  display: flex;
  justify-content: flex-end;
  padding: var(--fonu-space-4) var(--fonu-space-5);
  border-top: 1px solid var(--fonu-border);
}

.stream-box {
  margin-top: var(--fonu-space-4);
  background: #0f172a;
  color: #e2e8f0;
  border-radius: var(--fonu-radius);
  border: 1px solid var(--fonu-border);
  padding: var(--fonu-space-4);
  min-height: 360px;
  max-height: 520px;
  overflow: auto;
  font-family: var(--fonu-mono);
  font-size: 12px;
  line-height: 1.7;
}

html[data-theme='dark'] .stream-box,
html.dark .stream-box {
  background: #020617;
}

.stream-box.paused {
  opacity: 0.85;
}

.stream-line {
  white-space: pre-wrap;
  word-break: break-all;
}

.stream-line.level-info { color: #7dd3fc; }
.stream-line.level-warn { color: #fbbf24; }
.stream-line.level-error { color: #f87171; }

.stream-empty {
  padding: var(--fonu-space-6);
  text-align: center;
}
</style>
