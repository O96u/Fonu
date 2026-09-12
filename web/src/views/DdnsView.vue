<template>
  <PageHeader title="DDNS" description="自动同步公网 IP 到 DNS 解析，支持多个域名">
    <template #actions>
      <n-button :loading="updatingAll" @click="updateAll">全部更新</n-button>
      <n-button type="primary" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        添加域名
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
      </div>
    </FonuCard>

    <FonuCard title="域名列表" flush class="list-card">
      <n-data-table
        v-if="configs.length > 0"
        :columns="columns"
        :data="configs"
        :loading="loading"
        :bordered="false"
        :scroll-x="1000"
        :row-key="(r: DDNSConfig) => r.id"
      />
      <EmptyState
        v-if="!loading && configs.length === 0"
        title="还没有 DDNS 配置"
        description="添加你的第一个域名，Fonu 会自动同步公网 IP。"
      >
        <template #action>
          <n-button type="primary" @click="openCreate">添加域名</n-button>
        </template>
      </EmptyState>
    </FonuCard>

    <div class="info-tip">
      <n-icon :component="InformationCircleOutline" />
      <span>每个根域名独立配置 DNS 服务商与 API 凭证。申请证书时选择对应 DNS 凭证，用于完成 DNS-01 验证。</span>
    </div>
  </template>

  <n-modal v-model:show="showModal" :mask-closable="false" transform-origin="center">
    <div class="ddns-modal">
      <h3 class="modal-title">{{ editing ? '编辑 DDNS' : '添加 DDNS' }}</h3>
      <n-form label-placement="top">
        <n-form-item label="DNS Provider">
          <n-select v-model:value="form.provider" :options="providerOptions" />
        </n-form-item>
        <div class="form-row">
          <n-form-item label="主域名" class="form-row__item">
            <n-input v-model:value="form.root_domain" placeholder="example.com" />
          </n-form-item>
          <n-form-item label="子域名" class="form-row__item">
            <n-input v-model:value="form.record_name" placeholder="*" />
          </n-form-item>
        </div>
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
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NSelect,
  NSwitch,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { AddOutline, InformationCircleOutline } from '@vicons/ionicons5'
import { api } from '../api/client'
import type { DDNSConfig } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatRelativeTime } from '../utils/format'
import { statusLabel } from '../utils/status'
import { renderTableRowActions } from '../utils/tableActions'

const message = useMessage()
const dialog = useDialog()
const configs = ref<DDNSConfig[]>([])
const loading = ref(false)
const loadError = ref('')
const saving = ref(false)
const testing = ref(false)
const updatingAll = ref(false)
const savingInterval = ref(false)
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
  root_domain: '',
  record_name: '*',
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

const columns: DataTableColumns<DDNSConfig> = [
  { title: '主域名', key: 'root_domain', minWidth: 140 },
  { title: '子域名', key: 'record_name', width: 90 },
  { title: '服务商', key: 'provider', width: 110, render: (row) => providerLabel(row.provider) },
  {
    title: '状态',
    key: 'last_status',
    width: 100,
    render: (row) => h(StatusBadge, { value: row.enabled ? row.last_status : 'disabled', text: statusLabel(row.last_status) }),
  },
  { title: 'IPv4', key: 'last_ipv4', width: 130, render: (row) => row.last_ipv4 || '-' },
  { title: '上次更新', key: 'last_updated_at', width: 120, render: (row) => formatRelativeTime(row.last_updated_at) },
  {
    title: '操作',
    key: 'actions',
    width: 208,
    render: (row) =>
      renderTableRowActions([
        { label: '编辑', type: 'primary', onClick: () => openEdit(row) },
        { label: '立即更新', onClick: () => updateOne(row) },
        { label: '删除', type: 'error', onClick: () => confirmDelete(row) },
      ]),
  },
]

function resetForm() {
  Object.assign(form, {
    provider: 'cloudflare',
    root_domain: '',
    record_name: '*',
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
    root_domain: row.root_domain,
    record_name: row.record_name || '*',
    api_token: '',
    api_token_id: '',
    api_secret: '',
    ipv4_enabled: row.ipv4_enabled,
    ipv6_enabled: row.ipv6_enabled,
    enabled: row.enabled,
  })
  showModal.value = true
}

async function load() {
  const [list, settings] = await Promise.all([api.listDDNS(), api.getSettings()])
  configs.value = list
  updateInterval.value = Number(settings.ddns_check_interval_minutes ?? 5)
}

async function init() {
  loading.value = true
  loadError.value = ''
  try {
    await load()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const payload = { ...form }
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

async function updateOne(row: DDNSConfig) {
  try {
    const updated = await api.updateDDNSOne(row.id)
    configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
    message.success(`${row.root_domain} 已更新`)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
  }
}

async function updateAll() {
  updatingAll.value = true
  try {
    configs.value = await api.updateAllDDNS()
    message.success('全部 DDNS 已更新')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '部分更新失败')
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
  dialog.warning({
    title: '删除 DDNS 配置',
    content: `确定删除 ${row.root_domain} 的 DDNS 配置吗？`,
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
  margin-bottom: var(--fonu-space-4);
}

.interval-row {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  flex-wrap: wrap;
  padding: var(--fonu-space-4) var(--fonu-space-5);
}

.interval-label {
  font-size: 14px;
  color: var(--fonu-text-secondary);
}

.list-card {
  margin-bottom: 0;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--fonu-space-4);
}

.form-row__item {
  margin-bottom: 0;
}

.switch-row {
  display: flex;
  gap: var(--fonu-space-5);
  flex-wrap: wrap;
}

.token-hint {
  color: var(--fonu-success);
  font-size: 12px;
}

.info-tip {
  display: flex;
  align-items: flex-start;
  gap: var(--fonu-space-2);
  margin-top: var(--fonu-space-5);
  padding: var(--fonu-space-4);
  background: var(--fonu-info-soft);
  border-radius: var(--fonu-radius-sm);
  font-size: 13px;
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
  .form-row {
    grid-template-columns: 1fr;
  }
}
</style>
