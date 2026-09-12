<template>
  <PageHeader title="HTTPS 证书" description="通过 ACME 自动申请、续签，或导入已有证书">
    <template #actions>
      <n-button @click="openImport">导入证书</n-button>
      <n-button type="primary" @click="openApply">申请证书</n-button>
    </template>
  </PageHeader>

  <LoadError v-if="loadError" :message="loadError" @retry="load" />

  <template v-else>
    <FonuCard v-if="primaryCert" class="cert-overview">
      <div class="overview-grid">
        <div class="overview-stat">
          <span class="overview-label">状态</span>
          <StatusBadge :value="primaryCert.status" text="正常" />
        </div>
        <div class="overview-stat">
          <span class="overview-label">证书数量</span>
          <span class="overview-value">{{ records.length }} 张</span>
        </div>
        <div class="overview-stat">
          <span class="overview-label">最近到期</span>
          <span class="overview-value">{{ primaryCert.days_left }} 天</span>
        </div>
        <div class="overview-stat">
          <span class="overview-label">到期时间</span>
          <span class="overview-value">{{ formatDate(primaryCert.expires_at) }}</span>
        </div>
      </div>
    </FonuCard>

    <FonuCard title="证书列表" flush>
      <div v-if="records.length > 0" class="cert-table-wrap">
        <n-data-table
          class="cert-table"
          :columns="columns"
          :data="records"
          :loading="loading"
          :bordered="false"
        />
      </div>
      <EmptyState
        v-if="!loading && records.length === 0"
        title="还没有证书"
        description="先在 DDNS 配置 DNS 凭证，再申请 Let's Encrypt 证书；本地测试可直接导入自签证书。"
      >
        <template #action>
          <n-space>
            <n-button @click="openImport">导入证书</n-button>
            <n-button type="primary" @click="openApply">申请证书</n-button>
          </n-space>
        </template>
      </EmptyState>
    </FonuCard>
  </template>

  <!-- 申请 -->
  <n-modal
    v-model:show="showApply"
    preset="card"
    :style="{ width: 'min(560px, 92vw)' }"
    :segmented="{ content: true, footer: 'soft' }"
    :content-style="{ paddingTop: '8px' }"
  >
    <template #header>
      <div class="modal-header">
        <div class="modal-header__title">申请证书</div>
        <div class="modal-header__desc">通过 DNS-01 验证自动申请 Let's Encrypt 证书</div>
      </div>
    </template>

    <n-form label-placement="top" class="cert-form">
      <n-form-item label="DNS 凭证" :feedback="applyDnsHint" :show-feedback="!!applyDnsHint">
        <n-select
          v-model:value="applyDnsZone"
          :options="ddnsDomainOptions"
          placeholder="选择已配置的根域名"
          @update:value="onApplyDnsZoneChange"
        />
      </n-form-item>

      <n-form-item
        label="证书域名"
        feedback="每行一个域名，支持通配符（*.example.com），也可用逗号分隔"
        :show-feedback="true"
      >
        <n-input
          v-model:value="applyDomainsText"
          class="domains-input"
          type="textarea"
          :rows="5"
          placeholder="example.com&#10;*.example.com&#10;nas.example.com"
        />
      </n-form-item>

      <n-form-item label="ACME 邮箱" feedback="用于账户注册与到期提醒，填写后会记住" :show-feedback="true">
        <n-input v-model:value="applyEmail" placeholder="admin@example.com" />
      </n-form-item>

      <n-form-item label="颁发机构" :show-feedback="false">
        <div class="ca-field">
          <n-radio-group v-model:value="applyCA" class="ca-buttons">
            <n-radio-button v-for="opt in caOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </n-radio-button>
          </n-radio-group>
          <n-alert
            v-if="applyCA === 'letsencrypt-staging'"
            type="warning"
            :bordered="false"
            class="form-alert"
          >
            测试证书不受浏览器信任，仅用于验证 DNS 流程
          </n-alert>
        </div>
      </n-form-item>
    </n-form>

    <template #footer>
      <n-space justify="end">
        <n-button @click="showApply = false">取消</n-button>
        <n-button type="primary" :loading="applying" @click="submitApply">开始申请</n-button>
      </n-space>
    </template>
  </n-modal>

  <!-- 续签 -->
  <n-modal
    v-model:show="showRenew"
    preset="card"
    title="续签证书"
    :style="{ width: 'min(480px, 92vw)' }"
    :segmented="{ content: true, footer: 'soft' }"
  >
    <n-form label-placement="top" class="cert-form">
      <n-form-item label="主域名">
        <n-input :value="renewDomain" readonly />
      </n-form-item>
      <n-form-item v-if="renewDomainsPreview.length > 0" label="续签域名" :show-feedback="false">
        <div class="domain-preview">
          <n-tag v-for="domain in renewDomainsPreview" :key="domain" size="small" :bordered="false">
            {{ domain }}
          </n-tag>
        </div>
      </n-form-item>
      <n-form-item label="颁发机构" :show-feedback="false">
        <n-radio-group v-model:value="renewCA" class="ca-buttons">
          <n-radio-button v-for="opt in caOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </n-radio-button>
        </n-radio-group>
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="showRenew = false">取消</n-button>
        <n-button type="primary" :loading="renewing" @click="submitRenew">开始续签</n-button>
      </n-space>
    </template>
  </n-modal>

  <!-- 导入 -->
  <n-modal
    v-model:show="showImport"
    preset="card"
    :style="{ width: 'min(560px, 92vw)' }"
    :segmented="{ content: true, footer: 'soft' }"
    :content-style="{ paddingTop: '8px' }"
  >
    <template #header>
      <div class="modal-header">
        <div class="modal-header__title">导入证书</div>
        <div class="modal-header__desc">提供证书与私钥后，自动解析域名、通配符和到期时间</div>
      </div>
    </template>

    <n-form label-placement="top" class="cert-form">
      <n-form-item label="导入方式" :show-feedback="false">
        <n-radio-group v-model:value="importMode" class="ca-buttons">
          <n-radio-button value="paste">粘贴 PEM</n-radio-button>
          <n-radio-button value="path">文件路径</n-radio-button>
        </n-radio-group>
      </n-form-item>

      <template v-if="importMode === 'paste'">
        <n-form-item label="证书 (PEM)" :show-feedback="false">
          <n-input
            v-model:value="importForm.certificate"
            class="pem-input"
            type="textarea"
            :rows="4"
            placeholder="-----BEGIN CERTIFICATE-----"
          />
          <input ref="certFileInput" type="file" accept=".pem,.crt,.cer" class="file-input" @change="onCertFile" />
          <n-button text type="primary" class="file-btn" @click="certFileInput?.click()">从本机文件读取</n-button>
        </n-form-item>
        <n-form-item label="私钥 (PEM)" :show-feedback="false">
          <n-input
            v-model:value="importForm.private_key"
            class="pem-input"
            type="textarea"
            :rows="4"
            placeholder="-----BEGIN PRIVATE KEY-----"
          />
          <input ref="keyFileInput" type="file" accept=".pem,.key" class="file-input" @change="onKeyFile" />
          <n-button text type="primary" class="file-btn" @click="keyFileInput?.click()">从本机文件读取</n-button>
        </n-form-item>
      </template>

      <template v-else>
        <n-form-item label="证书文件路径">
          <n-input v-model:value="importForm.cert_path" placeholder=".data/certs/example.com/fullchain.pem" />
        </n-form-item>
        <n-form-item
          label="私钥文件路径"
          feedback="填写 Fonu 服务可访问的路径，导入后会复制到证书目录"
          :show-feedback="true"
        >
          <n-input v-model:value="importForm.key_path" placeholder=".data/certs/example.com/privatekey.pem" />
        </n-form-item>
      </template>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="showImport = false">取消</n-button>
        <n-button type="primary" :loading="importing" @click="submitImport">导入</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NRadioButton,
  NRadioGroup,
  NSelect,
  NSpace,
  NTag,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { api, asList } from '../api/client'
import type { CertificateCAOption, CertificateRecord, DDNSConfig } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatDate } from '../utils/format'
import { statusLabel } from '../utils/status'
import { renderTableRowActions } from '../utils/tableActions'

type ImportMode = 'paste' | 'path'

const message = useMessage()
const dialog = useDialog()
const records = ref<CertificateRecord[]>([])
const loading = ref(false)
const loadError = ref('')
const applying = ref(false)
const renewing = ref(false)
const importing = ref(false)
const showImport = ref(false)
const showApply = ref(false)
const showRenew = ref(false)
const caOptions = ref<CertificateCAOption[]>([])
const applyCA = ref('letsencrypt')
const applyEmail = ref('')
const applyDnsZone = ref('')
const applyDomainsText = ref('')
const renewCA = ref('letsencrypt')
const renewDomain = ref('')
const ddnsConfigs = ref<DDNSConfig[]>([])
const certFileInput = ref<HTMLInputElement | null>(null)
const keyFileInput = ref<HTMLInputElement | null>(null)
const importMode = ref<ImportMode>('paste')
const importForm = ref({
  certificate: '',
  private_key: '',
  cert_path: '',
  key_path: '',
})

const providerLabels: Record<string, string> = {
  cloudflare: 'Cloudflare',
  dnspod: 'DNSPod',
  alidns: '阿里云',
}

const ddnsDomainOptions = computed(() =>
  ddnsConfigs.value.map((c) => ({
    label: `${c.root_domain} · ${providerLabels[c.provider] ?? c.provider}`,
    value: c.root_domain,
  })),
)

const applyDnsHint = computed(() => {
  const cfg = ddnsConfigs.value.find((c) => c.root_domain === applyDnsZone.value)
  if (!cfg) return ''
  return `将使用 ${providerLabels[cfg.provider] ?? cfg.provider} API 完成 DNS-01 验证`
})

const primaryCert = computed(() => {
  if (records.value.length === 0) return null
  return [...records.value].sort((a, b) => a.days_left - b.days_left)[0]
})

const renewDomainsPreview = computed(() => {
  const record = records.value.find((r) => r.domain === renewDomain.value)
  if (!record) return []
  return record.domains?.length ? record.domains : [record.domain]
})

function caLabel(value?: string) {
  return caOptions.value.find((o) => o.value === value)?.label ?? (value === 'imported' ? '手动导入' : value || '-')
}

function renderDomainTags(row: CertificateRecord) {
  const domains = row.domains?.length ? row.domains : [row.domain]
  return h(
    'div',
    { class: 'table-tags' },
    domains.map((domain) =>
      h(NTag, { size: 'small', bordered: false, type: domain.startsWith('*.') ? 'info' : 'default' }, () => domain),
    ),
  )
}

const columns: DataTableColumns<CertificateRecord> = [
  { title: '主域名', key: 'domain', minWidth: 120, ellipsis: { tooltip: true } },
  { title: '覆盖域名', key: 'domains', minWidth: 180, ellipsis: { tooltip: true }, render: (row) => renderDomainTags(row) },
  { title: '颁发机构', key: 'acme_ca', minWidth: 120, render: (row) => caLabel(row.acme_ca) },
  {
    title: '状态',
    key: 'status',
    width: 96,
    render: (row) => h(StatusBadge, { value: row.status, text: statusLabel(row.status) }),
  },
  { title: '剩余', key: 'days_left', width: 72, render: (row) => `${row.days_left} 天` },
  { title: '到期', key: 'expires_at', minWidth: 140, render: (row) => formatDate(row.expires_at) },
  {
    title: '操作',
    key: 'actions',
    width: 132,
    render: (row) =>
      renderTableRowActions([
        { label: '续签', type: 'primary', show: row.acme_ca !== 'imported', onClick: () => openRenewFor(row.domain) },
        { label: '删除', type: 'error', onClick: () => confirmDelete(row) },
      ]),
  },
]

function confirmDelete(row: CertificateRecord) {
  dialog.warning({
    title: '删除证书',
    content: `确定删除 ${row.domain} 的证书？相关文件将被移除，使用该证书的 HTTPS 代理可能失效。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.deleteCertificate(row.domain)
        records.value = records.value.filter((r) => r.id !== row.id)
        message.success('证书已删除')
      } catch (error) {
        message.error(error instanceof Error ? error.message : '删除失败')
      }
    },
  })
}

function defaultDomainsText(zone: string) {
  return `${zone}\n*.${zone}`
}

function parseDomainsInput(text: string) {
  return text
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function onApplyDnsZoneChange(zone: string) {
  applyDomainsText.value = defaultDomainsText(zone)
}

function resetApplyForm() {
  if (ddnsConfigs.value.length > 0) {
    applyDnsZone.value = ddnsConfigs.value[0].root_domain
    applyDomainsText.value = defaultDomainsText(ddnsConfigs.value[0].root_domain)
  } else {
    applyDomainsText.value = ''
  }
}

async function loadCAOptions() {
  const [options, settings, ddns] = await Promise.all([
    api.listCertificateCAOptions(),
    api.getSettings(),
    api.listDDNS(),
  ])
  caOptions.value = options
  ddnsConfigs.value = ddns
  applyEmail.value = settings.acme_email ?? ''
  const defaultCA = settings.acme_ca || 'letsencrypt'
  applyCA.value = defaultCA
  renewCA.value = defaultCA
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    await loadCAOptions()
    records.value = asList(await api.listCertificates())
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

function openApply() {
  if (ddnsDomainOptions.value.length === 0) {
    message.warning('请先在 DDNS 页面添加域名配置')
    return
  }
  resetApplyForm()
  showApply.value = true
}

async function submitApply() {
  if (!applyDnsZone.value) {
    message.warning('请选择 DNS 凭证')
    return
  }
  const domains = parseDomainsInput(applyDomainsText.value)
  if (domains.length === 0) {
    message.warning('请填写至少一个域名')
    return
  }
  if (!applyEmail.value.trim()) {
    message.warning('请填写 ACME 邮箱')
    return
  }
  applying.value = true
  try {
    records.value = await api.applyCertificate({
      dns_zone: applyDnsZone.value,
      domains,
      ca: applyCA.value,
      email: applyEmail.value.trim(),
    })
    showApply.value = false
    message.success('证书申请成功')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '申请失败')
  } finally {
    applying.value = false
  }
}

function openRenewFor(domain: string) {
  const current = records.value.find((r) => r.domain === domain)
  if (!current || current.acme_ca === 'imported') {
    message.warning('手动导入的证书请重新导入，无法续签')
    return
  }
  renewDomain.value = domain
  renewCA.value = current.acme_ca || applyCA.value
  showRenew.value = true
}

async function submitRenew() {
  renewing.value = true
  try {
    const record = await api.renewCertificate(renewDomain.value, renewCA.value)
    records.value = [record, ...records.value.filter((r) => r.id !== record.id)]
    showRenew.value = false
    message.success('证书续签成功')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '续签失败')
  } finally {
    renewing.value = false
  }
}

function openImport() {
  importMode.value = 'paste'
  importForm.value = {
    certificate: '',
    private_key: '',
    cert_path: '',
    key_path: '',
  }
  showImport.value = true
}

async function onCertFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  importForm.value.certificate = await file.text()
  input.value = ''
}

async function onKeyFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  importForm.value.private_key = await file.text()
  input.value = ''
}

async function submitImport() {
  if (importMode.value === 'paste') {
    if (!importForm.value.certificate.trim() || !importForm.value.private_key.trim()) {
      message.warning('请填写证书和私钥内容')
      return
    }
  } else if (!importForm.value.cert_path.trim() || !importForm.value.key_path.trim()) {
    message.warning('请填写证书和私钥文件路径')
    return
  }

  importing.value = true
  try {
    const payload =
      importMode.value === 'path'
        ? {
            cert_path: importForm.value.cert_path,
            key_path: importForm.value.key_path,
          }
        : {
            certificate: importForm.value.certificate,
            private_key: importForm.value.private_key,
          }
    const record = await api.importCertificate(payload)
    records.value = [record, ...records.value.filter((r) => r.id !== record.id)]
    showImport.value = false
    const domains = record.domains?.length ? record.domains.join(', ') : record.domain
    message.success(`证书导入成功：${domains}`)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '导入失败')
  } finally {
    importing.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.cert-overview {
  margin-bottom: var(--fonu-space-5);
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--fonu-space-4);
}

.cert-table-wrap {
  width: 100%;
  overflow-x: auto;
}

.cert-table {
  width: 100%;
}

.overview-stat {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-2);
}

.overview-label {
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.overview-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--fonu-text);
}

.modal-header__title {
  font-size: 18px;
  font-weight: 600;
  color: var(--fonu-text);
  line-height: 1.3;
}

.modal-header__desc {
  margin-top: 4px;
  font-size: 13px;
  font-weight: 400;
  color: var(--fonu-text-secondary);
  line-height: 1.5;
}

.cert-form :deep(.n-form-item) {
  margin-bottom: 20px;
}

.cert-form :deep(.n-form-item:last-child) {
  margin-bottom: 0;
}

.cert-form :deep(.n-form-item-label) {
  font-weight: 500;
  padding-bottom: 6px;
}

.cert-form :deep(.n-form-item-feedback-wrapper) {
  min-height: auto;
  padding-top: 6px;
}

.ca-field {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-3);
  width: 100%;
}

.ca-buttons {
  display: flex;
  width: 100%;
}

.ca-buttons :deep(.n-radio-button) {
  flex: 1;
  justify-content: center;
}

.domains-input :deep(textarea),
.pem-input :deep(textarea) {
  font-family: var(--fonu-mono);
  font-size: 13px;
  line-height: 1.6;
}

.form-alert {
  width: 100%;
}

.file-btn {
  margin-top: var(--fonu-space-2);
}

.domain-preview {
  display: flex;
  flex-wrap: wrap;
  gap: var(--fonu-space-2);
  min-height: 28px;
  padding: var(--fonu-space-3);
  border: 1px dashed var(--fonu-border);
  border-radius: var(--fonu-radius-sm);
  background: var(--fonu-bg);
}

.domain-preview__empty {
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.file-input {
  display: none;
}

:deep(.table-tags) {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

@media (max-width: 767px) {
  .cert-table :deep(.n-data-table-base-table) {
    min-width: 720px;
  }
}
</style>
