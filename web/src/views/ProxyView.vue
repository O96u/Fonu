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
      :scroll-x="1100"
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
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
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
  NSwitch,
  NTag,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { AddOutline, SearchOutline } from '@vicons/ionicons5'
import { api, asList } from '../api/client'
import type { ProxyRule, ProxySavePayload } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
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
    title: '操作',
    key: 'actions',
    width: 132,
    render: (row) =>
      renderTableRowActions([
        { label: '编辑', type: 'primary', onClick: () => openEdit(row) },
        { label: '删除', type: 'error', onClick: () => confirmDelete(row) },
      ]),
  },
]

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

onMounted(load)
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
</style>
