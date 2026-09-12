<template>
  <PageHeader title="反代" description="管理通过域名访问的 NAS 服务">
    <template #actions>
      <n-button :loading="scanning" @click="scanServices">扫描服务</n-button>
      <n-button type="primary" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        新增规则
      </n-button>
    </template>
  </PageHeader>

  <LoadError v-if="loadError" :message="loadError" @retry="load" />

  <FonuCard v-else flush>
    <div class="toolbar">
      <n-input v-model:value="search" clearable placeholder="搜索域名或 IP" style="max-width: 260px">
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select v-model:value="statusFilter" :options="statusOptions" clearable placeholder="状态" style="width: 120px" />
      <n-select v-model:value="httpsFilter" :options="httpsOptions" clearable placeholder="HTTPS" style="width: 120px" />
    </div>

    <n-data-table
      v-if="filteredRules.length > 0"
      :columns="columns"
      :data="filteredRules"
      :loading="loading"
      :bordered="false"
      :scroll-x="1280"
      :row-key="(r: ProxyRule) => r.id"
    />

    <EmptyState
      v-if="!loading && rules.length === 0"
      title="还没有反向代理规则"
      description="创建第一条规则，让域名访问你的 NAS 服务。"
    >
      <template #action>
        <n-button type="primary" @click="openCreate">创建规则</n-button>
      </template>
    </EmptyState>
  </FonuCard>

  <n-modal v-model:show="showModal" :mask-closable="false" transform-origin="center">
    <div class="proxy-modal">
      <div class="proxy-modal__form">
        <h3 class="modal-title">{{ editing ? '编辑规则' : '新增规则' }}</h3>
        <n-form label-placement="top">
          <n-form-item label="监听类型">
            <div class="listen-types">
              <n-checkbox v-model:checked="form.listen_ipv4">IPv4</n-checkbox>
              <n-checkbox v-model:checked="form.listen_ipv6">IPv6</n-checkbox>
            </div>
          </n-form-item>
          <n-form-item label="监听端口">
            <n-input-number v-model:value="form.listen_port" :min="1" :max="65535" style="width: 160px" />
          </n-form-item>
          <n-form-item label="前端域名">
            <n-input
              v-model:value="form.hostsText"
              type="textarea"
              :rows="5"
              placeholder="每行一个域名，例如：&#10;1.example.com&#10;2.example.com&#10;example.org&#10;example.com:6893"
            />
          </n-form-item>
          <n-form-item label="目标地址">
            <n-input v-model:value="form.upstream" placeholder="http://192.168.1.10:5666" />
          </n-form-item>
          <n-form-item label="启用 HTTPS">
            <n-switch v-model:value="form.https_enabled" />
          </n-form-item>
          <n-form-item label="HTTP 跳转 HTTPS">
            <n-switch v-model:value="form.http_redirect" :disabled="!form.https_enabled" />
          </n-form-item>
          <n-form-item label="启用规则">
            <n-switch v-model:value="form.enabled" />
          </n-form-item>
        </n-form>
        <div class="modal-footer">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="save">{{ editing ? '保存' : '创建' }}</n-button>
        </div>
      </div>
      <div class="proxy-modal__help">
        <h4>配置说明</h4>
        <ol>
          <li>监听端口为整组规则共用，如 <code>8011</code> 表示通过 <code>域名:8011</code> 访问</li>
          <li>可添加多个前端域名，公网分别通过各自域名加同一端口访问</li>
          <li>如需单独端口，可写 <code>example.com:6893</code></li>
          <li>目标地址填写 NAS 内网服务，如 <code>http://192.168.8.3:6893</code></li>
        </ol>
        <p class="help-note">自定义端口需在 Docker / 路由器中映射对应端口到 Fonu Nginx。</p>
      </div>
    </div>
  </n-modal>

  <n-modal
    v-model:show="showLogModal"
    preset="card"
    :style="{ width: 'min(960px, 96vw)' }"
    :segmented="{ content: true }"
    :content-style="{ padding: 0 }"
    @after-leave="stopLogStream"
  >
    <template #header>
      <div class="log-modal-header">
        <div>
          <div class="log-modal-title">访问日志</div>
          <div class="log-modal-sub">{{ logRuleTitle }}</div>
        </div>
        <n-button size="small" quaternary @click="clearLogLines">清空</n-button>
      </div>
    </template>
    <div ref="logBox" class="proxy-log-box">
      <div v-for="(line, i) in logLines" :key="i" class="proxy-log-line" :class="logLineClass(line)">
        {{ formatAccessLine(line) }}
      </div>
      <div v-if="logLines.length === 0" class="proxy-log-empty">
        暂无记录。打开时会加载该域名最近 100 条访问日志，之后实时追加。请通过反代域名访问（非 Fonu 管理页）；完整历史见「日志 → 访问日志」。
      </div>
    </div>
  </n-modal>

  <n-modal
    v-model:show="showClientsModal"
    preset="card"
    :style="{ width: 'min(520px, 96vw)' }"
    :segmented="{ content: true }"
    @after-leave="stopClientsPoll"
  >
    <template #header>
      <div>
        <div class="log-modal-title">当前连接</div>
        <div class="log-modal-sub">{{ clientsRuleTitle }}</div>
      </div>
    </template>
    <n-spin :show="clientsLoading">
      <div v-if="clientRows.length > 0" class="clients-list">
        <div v-for="row in clientRows" :key="row.ip" class="clients-row">
          <span class="mono">{{ row.ip }}</span>
          <span class="text-muted">{{ formatRelativeTime(row.last_seen) }}</span>
        </div>
      </div>
      <div v-else class="clients-empty">最近 65 秒内暂无访问客户端</div>
    </n-spin>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import {
  NButton,
  NCheckbox,
  NDataTable,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpin,
  NSwitch,
  NTag,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { AddOutline, SearchOutline } from '@vicons/ionicons5'
import { api, asList } from '../api/client'
import type { ProxyClientConn, ProxyRule, ProxySavePayload, ProxyTraffic } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatBytes, formatRate, formatRelativeTime } from '../utils/format'
import { renderTableRowActions } from '../utils/tableActions'

const message = useMessage()
const dialog = useDialog()
const rules = ref<ProxyRule[]>([])
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const showModal = ref(false)
const editing = ref<ProxyRule | null>(null)
const search = ref('')
const statusFilter = ref<string | null>(null)
const httpsFilter = ref<string | null>(null)
const scanning = ref(false)
const showLogModal = ref(false)
const logRule = ref<ProxyRule | null>(null)
const logLines = ref<string[]>([])
const logBox = ref<HTMLElement | null>(null)
let logEventSource: EventSource | null = null
const trafficByRule = ref<Record<number, ProxyTraffic>>({})
const showClientsModal = ref(false)
const clientsRule = ref<ProxyRule | null>(null)
const clientRows = ref<ProxyClientConn[]>([])
const clientsLoading = ref(false)
let trafficTimer: ReturnType<typeof setInterval> | null = null
let clientsTimer: ReturnType<typeof setInterval> | null = null

const logRuleTitle = computed(() => (logRule.value ? ruleTitle(logRule.value) : ''))
const clientsRuleTitle = computed(() => (clientsRule.value ? ruleTitle(clientsRule.value) : ''))

const statusOptions = [
  { label: '运行中', value: 'enabled' },
  { label: '已停用', value: 'disabled' },
]
const httpsOptions = [
  { label: 'HTTPS', value: 'on' },
  { label: 'HTTP', value: 'off' },
]

const form = reactive({
  listen_port: 80,
  listen_ipv4: true,
  listen_ipv6: false,
  hostsText: '',
  upstream: '',
  https_enabled: true,
  http_redirect: true,
  enabled: true,
})

function ruleHosts(rule: ProxyRule): string[] {
  const hosts = asList(rule.hosts)
  if (hosts.length > 0) {
    return hosts.map((host) => {
      if (host.listen_port && host.listen_port !== rule.listen_port) {
        return `${host.hostname}:${host.listen_port}`
      }
      return host.hostname
    })
  }
  return rule.domain ? [rule.domain] : []
}

function hostsToText(rule: ProxyRule): string {
  return ruleHosts(rule).join('\n')
}

function ruleTitle(rule: ProxyRule): string {
  const hosts = ruleHosts(rule)
  if (hosts.length === 0) return `规则 #${rule.id}`
  if (hosts.length === 1) return hosts[0]
  return `${hosts[0]} 等 ${hosts.length} 个域名`
}

function listenLabel(rule: ProxyRule): string {
  const stacks = []
  if (rule.listen_ipv4) stacks.push('IPv4')
  if (rule.listen_ipv6) stacks.push('IPv6')
  const stack = stacks.length > 0 ? stacks.join('/') : '-'
  return `${rule.listen_port} (${stack})`
}

function parseHostsText(text: string): string[] {
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

const filteredRules = computed(() =>
  rules.value.filter((rule) => {
    const q = search.value.toLowerCase()
    const hostText = ruleHosts(rule).join(' ').toLowerCase()
    if (q && !hostText.includes(q) && !rule.upstream.toLowerCase().includes(q)) return false
    if (statusFilter.value === 'enabled' && !rule.enabled) return false
    if (statusFilter.value === 'disabled' && rule.enabled) return false
    if (httpsFilter.value === 'on' && !rule.https_enabled) return false
    if (httpsFilter.value === 'off' && rule.https_enabled) return false
    return true
  }),
)

const columns: DataTableColumns<ProxyRule> = [
  { type: 'selection' },
  {
    title: '前端域名',
    key: 'hosts',
    minWidth: 220,
    render: (row) =>
      h(
        'div',
        { class: 'host-tags' },
        ruleHosts(row).map((host) =>
          h(NTag, { size: 'small', bordered: false, round: true }, () => host),
        ),
      ),
  },
  {
    title: '监听',
    key: 'listen_port',
    width: 130,
    render: (row) => h('span', { class: 'mono text-secondary' }, listenLabel(row)),
  },
  { title: '目标地址', key: 'upstream', render: (row) => h('span', { class: 'mono text-secondary' }, row.upstream) },
  {
    title: 'HTTPS',
    key: 'https_enabled',
    width: 90,
    render: (row) =>
      row.https_enabled
        ? h(NTag, { size: 'small', type: 'info', bordered: false, round: true }, () => 'HTTPS')
        : h('span', { class: 'text-muted' }, '-'),
  },
  {
    title: '状态',
    key: 'enabled',
    width: 100,
    render: (row) => h(StatusBadge, { value: row.enabled ? 'ok' : 'disabled', text: row.enabled ? '运行中' : '已停用' }),
  },
  {
    title: '流量',
    key: 'traffic',
    width: 200,
    render: (row) => {
      const stats = trafficByRule.value[row.id]
      if (!stats) {
        return h('span', { class: 'text-muted' }, '-')
      }
      return h('div', { class: 'traffic-cell' }, [
        h('div', { class: 'traffic-line' }, [
          h('span', { class: 'traffic-dir' }, '↓'),
          h('span', { class: 'mono' }, formatBytes(stats.download_total)),
          h('span', { class: 'traffic-rate' }, formatRate(stats.download_rate)),
        ]),
        h('div', { class: 'traffic-line' }, [
          h('span', { class: 'traffic-dir' }, '↑'),
          h('span', { class: 'mono' }, formatBytes(stats.upload_total)),
          h('span', { class: 'traffic-rate' }, formatRate(stats.upload_rate)),
        ]),
        h(
          NButton,
          {
            size: 'tiny',
            quaternary: true,
            type: stats.connections > 0 ? 'info' : 'default',
            class: 'traffic-conn-btn',
            onClick: () => openClients(row),
          },
          () => `${stats.connections} 连接`,
        ),
      ])
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 196,
    render: (row) =>
      renderTableRowActions([
        { label: '日志', onClick: () => openLogs(row) },
        { label: '编辑', type: 'primary', onClick: () => openEdit(row) },
        { label: '删除', type: 'error', onClick: () => confirmDelete(row) },
      ]),
  },
]

const accessLineRe =
  /^(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\d{3})\s+([\d.]+)\s+(\S+)\s+(\S+)$/

function formatAccessLine(line: string): string {
  const m = line.match(accessLineRe)
  if (!m) return line
  const [, time, , method, path, status, rt, client, upstream] = m
  const shortTime = time.replace('T', ' ').replace(/([+-]\d{2}:\d{2}|Z)$/, '')
  const ms = (parseFloat(rt) * 1000).toFixed(1)
  return `${shortTime}  ${method} ${path}  ${status}  ${ms}ms  ${client}  → ${upstream}`
}

function logLineClass(line: string): string {
  const m = line.match(accessLineRe)
  if (!m) return ''
  const status = Number(m[5])
  if (status >= 500) return 'is-error'
  if (status >= 400) return 'is-warn'
  return ''
}

function openLogs(row: ProxyRule) {
  logRule.value = row
  logLines.value = []
  showLogModal.value = true
}

function stopLogStream() {
  logEventSource?.close()
  logEventSource = null
}

function startLogStream() {
  if (!logRule.value || logEventSource) return
  logEventSource = new EventSource(`/api/proxies/${logRule.value.id}/logs/stream?tail=100`, {
    withCredentials: true,
  })
  logEventSource.addEventListener('log', (event) => {
    logLines.value.push(event.data)
    if (logLines.value.length > 500) logLines.value = logLines.value.slice(-400)
    requestAnimationFrame(() => {
      logBox.value?.scrollTo({ top: logBox.value.scrollHeight })
    })
  })
  logEventSource.onerror = () => {
    message.warning('日志连接中断，请关闭后重新打开')
    stopLogStream()
  }
}

function clearLogLines() {
  logLines.value = []
}

watch(showLogModal, (open) => {
  if (open) startLogStream()
  else stopLogStream()
})

async function refreshTraffic() {
  try {
    const rows = asList(await api.getProxyTraffic())
    const next: Record<number, ProxyTraffic> = {}
    for (const row of rows) {
      next[row.rule_id] = row
    }
    trafficByRule.value = next
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

async function loadClients() {
  if (!clientsRule.value) return
  clientsLoading.value = true
  try {
    clientRows.value = asList(await api.getProxyClients(clientsRule.value.id))
  } catch (error) {
    message.error(error instanceof Error ? error.message : '读取连接失败')
  } finally {
    clientsLoading.value = false
  }
}

function openClients(row: ProxyRule) {
  clientsRule.value = row
  clientRows.value = []
  showClientsModal.value = true
}

function startClientsPoll() {
  stopClientsPoll()
  void loadClients()
  clientsTimer = setInterval(loadClients, 2000)
}

function stopClientsPoll() {
  if (clientsTimer) {
    clearInterval(clientsTimer)
    clientsTimer = null
  }
}

watch(showClientsModal, (open) => {
  if (open) startClientsPoll()
  else stopClientsPoll()
})

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    rules.value = asList(await api.listProxies())
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.listen_port = 80
  form.listen_ipv4 = true
  form.listen_ipv6 = false
  form.hostsText = ''
  form.upstream = ''
  form.https_enabled = true
  form.http_redirect = true
  form.enabled = true
}

function buildPayload(): ProxySavePayload {
  const hosts = parseHostsText(form.hostsText)
  if (hosts.length === 0) {
    throw new Error('至少需要一个前端域名')
  }
  if (!form.listen_ipv4 && !form.listen_ipv6) {
    throw new Error('至少需要启用 IPv4 或 IPv6 监听')
  }
  return {
    upstream: form.upstream,
    listen_port: form.listen_port,
    listen_ipv4: form.listen_ipv4,
    listen_ipv6: form.listen_ipv6,
    hosts,
    https_enabled: form.https_enabled,
    http_redirect: form.http_redirect,
    enabled: form.enabled,
  }
}

async function scanServices() {
  scanning.value = true
  try {
    const items = await api.scanDiscovery('127.0.0.1')
    const found = items.find((i) => i.detected)
    if (found) {
      resetForm()
      form.upstream = found.upstream
      showModal.value = true
      message.success(`检测到 ${found.name}，请填写前端域名`)
    } else {
      message.info('未检测到常见服务')
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '扫描失败')
  } finally {
    scanning.value = false
  }
}

function openCreate() {
  editing.value = null
  resetForm()
  showModal.value = true
}

function openEdit(rule: ProxyRule) {
  editing.value = rule
  Object.assign(form, {
    listen_port: rule.listen_port || 80,
    listen_ipv4: rule.listen_ipv4 ?? true,
    listen_ipv6: rule.listen_ipv6 ?? false,
    hostsText: hostsToText(rule),
    upstream: rule.upstream,
    https_enabled: rule.https_enabled,
    http_redirect: rule.http_redirect,
    enabled: rule.enabled,
  })
  showModal.value = true
}

async function save() {
  saving.value = true
  try {
    const payload = buildPayload()
    if (editing.value) {
      await api.updateProxy(editing.value.id, payload)
      message.success('规则已保存')
    } else {
      await api.createProxy(payload)
      message.success('规则已创建')
    }
    showModal.value = false
    await load()
  } catch (error) {
    const msg = error instanceof Error ? error.message : '保存失败'
    if (msg.startsWith('规则已保存')) {
      message.warning(msg)
      showModal.value = false
      await load()
    } else {
      message.error(msg)
    }
  } finally {
    saving.value = false
  }
}

function confirmDelete(rule: ProxyRule) {
  dialog.warning({
    title: `确定删除 ${ruleTitle(rule)}？`,
    content: '删除后相关域名将停止反向代理。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: () =>
      api
        .deleteProxy(rule.id)
        .then(async () => {
          rules.value = rules.value.filter((r) => r.id !== rule.id)
          message.success('规则已删除')
          await load()
        })
        .catch(async (error: unknown) => {
          const msg = error instanceof Error ? error.message : '删除失败'
          if (msg.startsWith('规则已删除')) {
            message.warning(msg)
            await load()
          } else {
            message.error(msg)
            return false
          }
        }),
  })
}

onMounted(async () => {
  await load()
  await refreshTraffic()
  startTrafficPoll()
})
onUnmounted(() => {
  stopLogStream()
  stopTrafficPoll()
  stopClientsPoll()
})
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  flex-wrap: wrap;
  margin-bottom: var(--fonu-space-4);
  padding: 0 var(--fonu-space-5);
  padding-top: var(--fonu-space-4);
}

.host-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.listen-types {
  display: flex;
  gap: var(--fonu-space-4);
}

.proxy-modal {
  display: flex;
  width: 760px;
  max-width: 95vw;
  background: var(--fonu-surface);
  border-radius: var(--fonu-radius);
  overflow: hidden;
  box-shadow: var(--fonu-shadow-md);
}

.proxy-modal__form {
  flex: 1;
  padding: var(--fonu-space-5);
}

.proxy-modal__help {
  width: 280px;
  padding: var(--fonu-space-5);
  background: var(--fonu-bg);
  border-left: 1px solid var(--fonu-border);
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.proxy-modal__help h4 {
  margin: 0 0 var(--fonu-space-3);
  font-size: 14px;
  color: var(--fonu-text);
}

.proxy-modal__help ol {
  margin: 0;
  padding-left: 18px;
  line-height: 1.7;
}

.proxy-modal__help code {
  font-size: 12px;
  background: var(--fonu-surface);
  padding: 1px 4px;
  border-radius: 4px;
}

.help-note {
  margin-top: var(--fonu-space-4);
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.modal-title {
  margin: 0 0 var(--fonu-space-4);
  font-size: 18px;
  font-weight: 600;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--fonu-space-3);
  margin-top: var(--fonu-space-4);
}

@media (max-width: 640px) {
  .proxy-modal {
    flex-direction: column;
  }
  .proxy-modal__help {
    width: 100%;
    border-left: none;
    border-top: 1px solid var(--fonu-border);
  }
}

.log-modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  width: 100%;
}

.log-modal-title {
  font-size: 16px;
  font-weight: 600;
}

.log-modal-sub {
  margin-top: 4px;
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.proxy-log-box {
  background: #0f172a;
  color: #e2e8f0;
  min-height: 420px;
  max-height: min(68vh, 560px);
  overflow: auto;
  padding: var(--fonu-space-4);
  font-family: var(--fonu-mono);
  font-size: 12px;
  line-height: 1.7;
}

html[data-theme='dark'] .proxy-log-box,
html.dark .proxy-log-box {
  background: #020617;
}

.proxy-log-line {
  white-space: pre-wrap;
  word-break: break-all;
}

.proxy-log-line.is-warn {
  color: #fbbf24;
}

.proxy-log-line.is-error {
  color: #f87171;
}

.proxy-log-empty {
  padding: var(--fonu-space-6);
  text-align: center;
  color: #94a3b8;
}

.traffic-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 12px;
  line-height: 1.5;
}

.traffic-line {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.traffic-dir {
  width: 12px;
  color: var(--fonu-text-muted);
}

.traffic-rate {
  color: var(--fonu-text-muted);
  font-size: 11px;
}

.traffic-conn-btn {
  margin-top: 2px;
  padding: 0 4px !important;
  height: 22px !important;
}

.clients-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.clients-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--fonu-bg);
}

.clients-empty {
  padding: var(--fonu-space-6);
  text-align: center;
  color: var(--fonu-text-muted);
}
</style>
