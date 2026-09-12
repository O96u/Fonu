<template>
  <PageHeader title="DDNS" description="自动同步公网 IP 到 DNS 解析，支持多个域名">
    <template #actions>
      <n-button :loading="updatingAll" @click="updateAll">全部更新</n-button>
      <n-button type="primary" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        添加任务
      </n-button>
    </template>
  </PageHeader>

  <LoadError v-if="loadError" :message="loadError" @retry="init" />

  <template v-else>
    <FonuCard flush class="interval-card">
      <div class="interval-row">
        <span class="interval-label">全局检查周期（分钟）</span>
        <n-input-number v-model:value="updateInterval" :min="1" :max="1440" style="width: 140px" />
        <n-button size="small" :loading="savingInterval" @click="saveInterval">保存周期</n-button>
        <span v-if="refreshingDNS" class="refresh-hint">正在刷新 DNS 解析值…</span>
      </div>
    </FonuCard>

    <div v-if="loading" class="loading-wrap">
      <n-spin size="medium" />
    </div>

    <EmptyState
      v-else-if="configs.length === 0"
      title="还没有 DDNS 配置"
      description="添加你的第一个任务，Fonu 会自动同步公网 IP。"
    >
      <template #action>
        <n-button type="primary" @click="openCreate">添加任务</n-button>
      </template>
    </EmptyState>

    <div v-else class="task-list">
      <article
        v-for="cfg in configs"
        :key="cfg.id"
        class="task-card"
        :class="{ 'task-card--disabled': !cfg.enabled }"
      >
        <div class="task-header">
          <div class="task-header__left">
            <div class="task-title-row">
              <h3 class="task-provider">{{ providerLabel(cfg.provider) }}</h3>
              <n-switch
                :value="cfg.enabled"
                size="small"
                :loading="togglingId === cfg.id"
                @update:value="(v: boolean) => toggleEnabled(cfg, v)"
              />
              <StatusBadge
                :value="cfg.enabled ? cfg.last_status : 'disabled'"
                :text="statusLabel(cfg.last_status)"
              />
            </div>
            <div class="task-meta">
              <span class="meta-item">
                <span class="meta-label">上次同步</span>
                {{ formatRelativeTime(cfg.last_updated_at) || '从未' }}
              </span>
              <span v-if="cfg.last_error" class="meta-item meta-item--error">{{ cfg.last_error }}</span>
            </div>
          </div>
          <div class="task-header__actions">
            <n-button size="small" quaternary type="primary" @click="openEdit(cfg)">编辑</n-button>
            <n-button
              size="small"
              quaternary
              :loading="updatingId === cfg.id"
              @click="updateOne(cfg)"
            >立即更新</n-button>
            <n-button size="small" quaternary type="error" @click="confirmDelete(cfg)">删除</n-button>
          </div>
        </div>

        <div v-if="cfg.ipv4_enabled || cfg.ipv6_enabled" class="task-ip-bar">
          <span class="task-ip-bar__label">同步目标（本机公网）</span>
          <div class="task-ip-bar__values">
            <code v-if="cfg.ipv4_enabled && cfg.last_ipv4" class="ip-chip">IPv4 {{ cfg.last_ipv4 }}</code>
            <code v-if="cfg.ipv6_enabled && cfg.last_ipv6" class="ip-chip ip-chip--v6">IPv6 {{ cfg.last_ipv6 }}</code>
            <span
              v-if="!(cfg.ipv4_enabled && cfg.last_ipv4) && !(cfg.ipv6_enabled && cfg.last_ipv6)"
              class="ip-empty"
            >未检测到公网 IP（请检查网络或点击立即更新）</span>
          </div>
        </div>

        <div class="record-panel">
          <div class="record-panel__title">域名记录</div>
          <table class="record-table">
            <thead>
              <tr>
                <th>域名</th>
                <th>类型</th>
                <th>解析值</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rec in domainRecordsOf(cfg)" :key="rec.domain">
                <td class="record-domain" data-label="域名">{{ rec.domain }}</td>
                <td data-label="类型">
                  <span v-if="cfg.ipv4_enabled" class="type-tag">A</span>
                  <span v-if="cfg.ipv6_enabled" class="type-tag">AAAA</span>
                </td>
                <td class="record-ip-cell" data-label="解析值">
                  <div v-if="cfg.ipv4_enabled && rec.ipv4" class="ip-line">
                    <span class="ip-line__label">v4</span>
                    <code>{{ rec.ipv4 }}</code>
                  </div>
                  <div v-if="cfg.ipv6_enabled && rec.ipv6" class="ip-line">
                    <span class="ip-line__label">v6</span>
                    <code>{{ rec.ipv6 }}</code>
                  </div>
                  <span
                    v-if="!(cfg.ipv4_enabled && rec.ipv4) && !(cfg.ipv6_enabled && rec.ipv6)"
                    class="ip-empty"
                  >-</span>
                </td>
                <td data-label="状态">
                  <n-tag size="small" round :type="recordTagType(rec.status)" :bordered="false">
                    {{ rec.message || recordStatusLabel(rec.status) }}
                  </n-tag>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>
    </div>

    <div class="info-tip">
      <n-icon :component="InformationCircleOutline" />
      <span>每个任务对应一套 DNS 凭证。域名记录分别显示当前 DNS 解析值，同步时逐条对比并更新。</span>
    </div>
  </template>

  <n-modal v-model:show="showModal" :mask-closable="false" transform-origin="center">
    <div class="ddns-modal">
      <h3 class="modal-title">{{ editing ? '编辑 DDNS' : '添加 DDNS' }}</h3>
      <n-form label-placement="top">
        <n-form-item label="DNS Provider">
          <n-select v-model:value="form.provider" :options="providerOptions" />
        </n-form-item>
        <n-form-item v-if="form.provider === 'dnspod'" label="Token ID">
          <n-input v-model:value="form.api_token_id" placeholder="DNSPod ID" />
        </n-form-item>
        <n-form-item :label="credentialLabel">
          <n-input
            v-model:value="form.api_token"
            type="password"
            show-password-on="click"
            :placeholder="credentialPlaceholder"
          />
          <template v-if="editing?.has_token" #feedback>
            <span class="token-hint">已配置 · 留空则保持不变</span>
          </template>
        </n-form-item>
        <n-form-item v-if="form.provider === 'alidns'" label="AccessKey Secret">
          <n-input v-model:value="form.api_secret" type="password" show-password-on="click" placeholder="留空则保持不变" />
        </n-form-item>
        <div class="switch-row">
          <n-form-item label="IPv4"><n-switch v-model:value="form.ipv4_enabled" /></n-form-item>
          <n-form-item label="IPv6"><n-switch v-model:value="form.ipv6_enabled" /></n-form-item>
          <n-form-item label="启用"><n-switch v-model:value="form.enabled" /></n-form-item>
        </div>
        <n-form-item label="解析域名">
          <n-input
            v-model:value="form.domainsText"
            type="textarea"
            :rows="5"
            placeholder="每行一个完整域名，例如：&#10;s.example.com&#10;www.example.com&#10;www.other.com"
          />
          <template #feedback>
            <span class="domain-hint">可跨不同主域名，同一 AccessKey 只需配置一次</span>
          </template>
        </n-form-item>
      </n-form>
      <div class="modal-actions">
        <n-button @click="showModal = false">取消</n-button>
        <n-button :loading="testing" @click="test">测试连接</n-button>
        <n-button type="primary" :loading="saving" @click="save">{{ editing ? '保存' : '创建' }}</n-button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NSelect,
  NSpin,
  NSwitch,
  NTag,
  useDialog,
  useMessage,
} from 'naive-ui'
import { AddOutline, InformationCircleOutline } from '@vicons/ionicons5'
import { api, asList } from '../api/client'
import type { DDNSConfig, DDNSDomainRecord } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatRelativeTime } from '../utils/format'
import { statusLabel } from '../utils/status'

const message = useMessage()
const dialog = useDialog()
const configs = ref<DDNSConfig[]>([])
const loading = ref(false)
const refreshingDNS = ref(false)
const loadError = ref('')
const saving = ref(false)
const testing = ref(false)
const updatingAll = ref(false)
const updatingId = ref<number | null>(null)
const savingInterval = ref(false)
const togglingId = ref<number | null>(null)
const showModal = ref(false)
const editing = ref<DDNSConfig | null>(null)
const updateInterval = ref(5)

const providerOptions = [
  { label: 'Cloudflare', value: 'cloudflare' },
  { label: 'DNSPod', value: 'dnspod' },
  { label: '阿里云 DNS', value: 'alidns' },
]

const form = reactive({
  provider: 'cloudflare',
  domainsText: '',
  api_token: '',
  api_token_id: '',
  api_secret: '',
  ipv4_enabled: true,
  ipv6_enabled: false,
  enabled: true,
})

const providerLabel = (v: string) =>
  providerOptions.find((o) => o.value === v)?.label ?? v

const credentialLabel = computed(() => {
  if (form.provider === 'alidns') return 'AccessKey ID'
  if (form.provider === 'dnspod') return 'Token'
  return 'API Token'
})

const credentialPlaceholder = computed(() => {
  if (editing.value?.has_token) return '留空则保持不变'
  if (form.provider === 'alidns') return 'AccessKey ID'
  if (form.provider === 'dnspod') return 'DNSPod Token'
  return 'Cloudflare API Token'
})

function recordNamesOf(row: DDNSConfig): string[] {
  return row.record_names?.length ? row.record_names : [row.record_name || '@']
}

function formatDomain(root: string, record: string): string {
  if (!record || record === '@') return root
  if (record === '*') return `*.${root}`
  if (record.includes('.')) return record
  return `${record}.${root}`
}

function domainsOf(row: DDNSConfig): string[] {
  return recordNamesOf(row).map((name) => formatDomain(row.root_domain, name))
}

function domainRecordsOf(row: DDNSConfig): DDNSDomainRecord[] {
  if (row.domain_records?.length) return row.domain_records
  return domainsOf(row).map((domain) => ({ domain, status: 'unknown' }))
}

function recordStatusLabel(status: string): string {
  if (status === 'unchanged') return '记录未变化'
  if (status === 'ok') return '已更新'
  if (status === 'error') return '失败'
  return statusLabel(status)
}

function recordTagType(status: string): 'success' | 'warning' | 'error' | 'default' {
  if (status === 'ok') return 'success'
  if (status === 'unchanged') return 'default'
  if (status === 'error') return 'error'
  return 'warning'
}

function parseDomainsText(text: string): string[] {
  const seen = new Set<string>()
  const domains: string[] = []
  for (const line of text.split('\n')) {
    const domain = line.trim().toLowerCase()
    if (!domain || seen.has(domain)) continue
    seen.add(domain)
    domains.push(domain)
  }
  return domains
}

function domainsToText(row: DDNSConfig): string {
  return domainsOf(row).join('\n')
}

function resetForm() {
  Object.assign(form, {
    provider: 'cloudflare',
    domainsText: '',
    api_token: '',
    api_token_id: '',
    api_secret: '',
    ipv4_enabled: true,
    ipv6_enabled: false,
    enabled: true,
  })
}

function openCreate() {
  editing.value = null
  resetForm()
  showModal.value = true
}

function openEdit(row: DDNSConfig) {
  editing.value = row
  Object.assign(form, {
    provider: row.provider || 'cloudflare',
    domainsText: domainsToText(row),
    api_token: '',
    api_token_id: '',
    api_secret: '',
    ipv4_enabled: row.ipv4_enabled,
    ipv6_enabled: row.ipv6_enabled,
    enabled: row.enabled,
  })
  showModal.value = true
}

async function loadCached() {
  const [list, settings] = await Promise.all([api.listDDNSLite(), api.getSettings()])
  configs.value = asList(list)
  updateInterval.value = Number(settings.ddns_check_interval_minutes ?? 5)
}

async function refreshLiveDNS() {
  refreshingDNS.value = true
  try {
    configs.value = asList(await api.listDDNS())
  } catch {
    // 保留已展示的缓存数据
  } finally {
    refreshingDNS.value = false
  }
}

async function load() {
  await loadCached()
  refreshLiveDNS()
}

async function init() {
  loading.value = true
  loadError.value = ''
  try {
    await loadCached()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
  refreshLiveDNS()
}

async function save() {
  const domains = parseDomainsText(form.domainsText)
  if (domains.length === 0) {
    message.error('至少需要一个域名')
    return
  }
  saving.value = true
  try {
    const { domainsText: _, ...rest } = form
    const payload = { ...rest, domains }
    if (editing.value) {
      const updated = await api.updateDDNS(editing.value.id, payload)
      configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
      message.success('DDNS 配置已更新')
    } else {
      const created = await api.createDDNS(payload)
      configs.value = [...configs.value, created]
      message.success('DDNS 配置已创建')
    }
    showModal.value = false
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(row: DDNSConfig, enabled: boolean) {
  togglingId.value = row.id
  try {
    const updated = await api.updateDDNS(row.id, {
      provider: row.provider,
      domains: domainsOf(row),
      enabled,
      ipv4_enabled: row.ipv4_enabled,
      ipv6_enabled: row.ipv6_enabled,
    })
    configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
  } finally {
    togglingId.value = null
  }
}

async function test() {
  testing.value = true
  try {
    await api.testDDNS({
      config_id: editing.value?.id,
      provider: form.provider,
      api_token: form.api_token,
      api_token_id: form.api_token_id,
      api_secret: form.api_secret,
    })
    message.success('连接成功')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '测试失败')
  } finally {
    testing.value = false
  }
}

function syncFeedbackMessage(cfg: DDNSConfig): { type: 'success' | 'warning' | 'error'; text: string } {
  if (cfg.last_status === 'error') {
    return { type: 'error', text: cfg.last_error || '同步失败' }
  }

  const records = domainRecordsOf(cfg)
  const updated = records.filter((r) => r.status === 'ok').length
  const unchanged = records.filter((r) => r.status === 'unchanged').length
  const failed = records.filter((r) => r.status === 'error').length

  if (cfg.last_status === 'warning' || failed > 0) {
    const parts: string[] = []
    if (updated > 0) parts.push(`${updated} 条已更新`)
    if (unchanged > 0) parts.push(`${unchanged} 条未变化`)
    if (failed > 0) parts.push(`${failed} 条失败`)
    return {
      type: 'warning',
      text: parts.join('，') || cfg.last_error || '部分域名同步失败',
    }
  }

  if (updated > 0 && unchanged === 0) {
    return { type: 'success', text: `已更新 ${updated} 条记录` }
  }
  if (updated > 0) {
    return { type: 'success', text: `已更新 ${updated} 条，${unchanged} 条未变化` }
  }
  if (unchanged > 0) {
    return { type: 'success', text: `同步完成，${unchanged} 条记录未变化` }
  }
  return { type: 'success', text: '同步完成' }
}

async function updateOne(row: DDNSConfig) {
  updatingId.value = row.id
  try {
    const updated = await api.updateDDNSOne(row.id)
    configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
    const fb = syncFeedbackMessage(updated)
    message[fb.type](fb.text)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
    await load()
  } finally {
    updatingId.value = null
  }
}

async function updateAll() {
  updatingAll.value = true
  try {
    configs.value = await api.updateAllDDNS()
    const errors = configs.value.filter((c) => c.enabled && c.last_status === 'error')
    const warnings = configs.value.filter((c) => c.enabled && c.last_status === 'warning')
    if (errors.length > 0) {
      message.error(errors.length === 1 ? errors[0].last_error || '部分任务更新失败' : `${errors.length} 个任务更新失败`)
    } else if (warnings.length > 0) {
      message.warning(warnings.length === 1 ? warnings[0].last_error || '部分域名同步失败' : `${warnings.length} 个任务存在域名同步失败`)
    } else {
      message.success('全部 DDNS 已更新')
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
    await load()
  } finally {
    updatingAll.value = false
  }
}

async function saveInterval() {
  savingInterval.value = true
  try {
    await api.saveSettings({ ddns_check_interval_minutes: String(updateInterval.value) })
    message.success('检查周期已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    savingInterval.value = false
  }
}

function confirmDelete(row: DDNSConfig) {
  const names = domainsOf(row).join('、')
  dialog.warning({
    title: '删除 DDNS 配置',
    content: `确定删除 ${names} 的 DDNS 配置吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await api.deleteDDNS(row.id)
      configs.value = configs.value.filter((c) => c.id !== row.id)
      message.success('已删除')
    },
  })
}

onMounted(init)
</script>

<style scoped>
.interval-card {
  margin-bottom: var(--fonu-space-3);
}

.interval-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-3) var(--fonu-space-4);
}

.refresh-hint {
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.interval-label {
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.loading-wrap {
  display: flex;
  justify-content: center;
  padding: var(--fonu-space-6);
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-3);
}

.task-card {
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  overflow: hidden;
  border-left: 3px solid var(--fonu-brand);
}

.task-card--disabled {
  border-left-color: var(--fonu-disabled);
  opacity: 0.88;
}

.task-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-3) var(--fonu-space-4);
  background: linear-gradient(180deg, var(--fonu-brand-soft) 0%, transparent 100%);
  border-bottom: 1px solid var(--fonu-border);
}

.task-header__left {
  min-width: 0;
  flex: 1;
}

.task-title-row {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-2);
  flex-wrap: wrap;
}

.task-provider {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--fonu-text);
}

.task-meta {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.meta-label {
  color: var(--fonu-text-muted);
}

.meta-item--error {
  color: var(--fonu-error);
}

.task-header__actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}

.task-ip-bar {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  padding: 8px var(--fonu-space-4);
  background: var(--fonu-bg-muted);
  border-bottom: 1px solid var(--fonu-border);
  font-size: 12px;
}

.task-ip-bar__label {
  flex-shrink: 0;
  color: var(--fonu-text-muted);
  font-weight: 500;
}

.task-ip-bar__values {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.ip-chip {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  font-family: var(--fonu-mono);
  font-size: 12px;
  color: var(--fonu-text);
}

.ip-chip--v6 {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ip-empty {
  color: var(--fonu-text-muted);
  font-size: 12px;
}

.record-panel {
  padding: var(--fonu-space-3) var(--fonu-space-4) var(--fonu-space-4);
}

.record-panel__title {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--fonu-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.record-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.record-table th {
  padding: 6px 10px;
  text-align: left;
  font-size: 12px;
  font-weight: 500;
  color: var(--fonu-text-muted);
  border-bottom: 1px solid var(--fonu-border);
  white-space: nowrap;
}

.record-table th:last-child {
  width: 108px;
}

.record-table td {
  padding: 8px 10px;
  vertical-align: middle;
  border-bottom: 1px solid var(--fonu-border);
}

.record-table tbody tr:last-child td {
  border-bottom: none;
}

.record-table tbody tr:hover td {
  background: var(--fonu-bg-muted);
}

.record-domain {
  font-family: var(--fonu-mono);
  font-size: 12px;
  word-break: break-all;
}

.type-tag {
  display: inline-block;
  margin-right: 4px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--fonu-bg-muted);
  border: 1px solid var(--fonu-border);
  font-size: 11px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
  font-family: var(--fonu-mono);
}

.record-ip-cell {
  min-width: 0;
}

.ip-line {
  display: flex;
  align-items: baseline;
  gap: 6px;
  line-height: 1.5;
}

.ip-line + .ip-line {
  margin-top: 2px;
}

.ip-line__label {
  flex-shrink: 0;
  width: 18px;
  font-size: 10px;
  font-weight: 600;
  color: var(--fonu-text-muted);
  text-transform: uppercase;
}

.ip-line code {
  font-family: var(--fonu-mono);
  font-size: 12px;
  color: var(--fonu-text);
  word-break: break-all;
}

.switch-row {
  display: flex;
  gap: var(--fonu-space-5);
  flex-wrap: wrap;
}

.token-hint,
.domain-hint {
  color: var(--fonu-text-secondary);
  font-size: 12px;
}

.token-hint {
  color: var(--fonu-success);
}

.info-tip {
  display: flex;
  align-items: flex-start;
  gap: var(--fonu-space-2);
  margin-top: var(--fonu-space-4);
  padding: var(--fonu-space-3) var(--fonu-space-4);
  background: var(--fonu-info-soft);
  border-radius: var(--fonu-radius-sm);
  font-size: 12px;
  line-height: 1.5;
  color: var(--fonu-info);
}

.ddns-modal {
  width: min(520px, 92vw);
  padding: var(--fonu-space-5);
  background: var(--fonu-surface);
  border-radius: var(--fonu-radius);
}

.modal-title {
  margin: 0 0 var(--fonu-space-4);
  font-size: 18px;
  font-weight: 600;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--fonu-space-3);
  margin-top: var(--fonu-space-4);
}

@media (max-width: 767px) {
  .task-header {
    flex-direction: column;
  }

  .task-header__actions {
    width: 100%;
    justify-content: flex-end;
  }

  .task-ip-bar {
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
  }

  .record-table thead {
    display: none;
  }

  .record-table tr {
    display: block;
    padding: 8px 0;
    border-bottom: 1px solid var(--fonu-border);
  }

  .record-table tr:last-child {
    border-bottom: none;
  }

  .record-table td {
    display: flex;
    justify-content: space-between;
    gap: var(--fonu-space-3);
    padding: 4px 0;
    border: none;
  }

  .record-table td::before {
    content: attr(data-label);
    flex-shrink: 0;
    color: var(--fonu-text-muted);
    font-size: 12px;
  }
}
</style>
