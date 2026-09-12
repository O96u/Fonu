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
      <div v-if="loading" class="cert-loading">
        <n-spin size="medium" />
      </div>
      <div v-else-if="records.length > 0" class="cert-table-wrap">
        <n-data-table
          class="cert-table"
          :columns="columns"
          :data="records"
          :bordered="false"
        />
      </div>
      <EmptyState
        v-else
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

      <n-alert v-if="applyError" type="error" :bordered="false" class="form-alert" :title="applyError" />
    </n-form>

    <template #footer>
      <n-space justify="end">
        <n-button @click="showApply = false">取消</n-button>
        <n-button type="primary" :loading="applying" @click="submitApply">开始申请</n-button>
      </n-space>
    </template>
  </n-modal>

  <n-modal
    v-model:show="showApplyProgress"
    preset="card"
    :mask-closable="applyFinished"
    :close-on-esc="applyFinished"
    :style="{ width: 'min(720px, 96vw)' }"
    :segmented="{ content: true, footer: 'soft' }"
    :content-style="{ paddingTop: '8px' }"
    @after-leave="resetApplyProgress"
  >
    <template #header>
      <div class="modal-header">
        <div class="modal-header__title">{{ applyFinished ? (applyResult?.ok ? '申请成功' : '申请失败') : '正在申请证书' }}</div>
        <div class="modal-header__desc">
          {{ applyFinished ? '以下是本次申请的详细结果' : 'DNS 验证与证书签发可能需要 1～3 分钟，请保持窗口打开' }}
        </div>
      </div>
    </template>

    <div ref="applyLogBox" class="apply-log-box" :class="{ paused: !applyStreaming }">
      <div
        v-for="(line, i) in applyLogLines"
        :key="i"
        class="apply-log-line"
        :class="`apply-log-line--${line.level}`"
      >
        {{ line.text }}
      </div>
      <div v-if="applyLogLines.length === 0 && !applyFinished" class="apply-log-empty">等待日志输出…</div>
    </div>

    <n-result
      v-if="applyFinished && applyResult?.ok"
      status="success"
      title="证书申请成功"
      class="apply-result"
    >
      <template #footer>
        <div class="apply-result-detail">
          <div><span class="apply-result-k">证书目录</span><code>{{ applyResult.cert_dir || '-' }}</code></div>
          <div><span class="apply-result-k">证书文件</span><code>{{ applyResult.cert_path || '-' }}</code></div>
          <div><span class="apply-result-k">私钥文件</span><code>{{ applyResult.key_path || '-' }}</code></div>
          <div><span class="apply-result-k">到期时间</span>{{ formatDate(applyResult.expires_at) }}</div>
        </div>
      </template>
    </n-result>

    <n-result
      v-else-if="applyFinished && applyResult && !applyResult.ok"
      status="error"
      title="证书申请失败"
      :description="applyResult.error || '未知错误'"
      class="apply-result"
    />

    <template #footer>
      <n-space justify="end">
        <n-button v-if="!applyFinished" :loading="true" disabled>申请进行中…</n-button>
        <n-button v-else type="primary" @click="closeApplyProgress">关闭</n-button>
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
import { computed, h, onMounted, onUnmounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NDataTable,
  NDropdown,
  NForm,
  NFormItem,
  NInput,
  NRadioButton,
  NRadioGroup,
  NResult,
  NSelect,
  NSpace,
  NSpin,
  NTag,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { api, asList } from '../api/client'
import type {
  CertificateCAOption,
  CertificateJobDone,
  CertificateJobEvent,
  CertificateRecord,
  DDNSConfig,
} from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatDate } from '../utils/format'
import { statusLabel } from '../utils/status'

type ImportMode = 'paste' | 'path'

const message = useMessage()
const dialog = useDialog()
const records = ref<CertificateRecord[]>([])
const loading = ref(false)
const downloadingDomain = ref('')
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
const applyError = ref('')
const showApplyProgress = ref(false)
const applyLogLines = ref<{ level: string; text: string }[]>([])
const applyResult = ref<CertificateJobDone | null>(null)
const applyFinished = ref(false)
const applyStreaming = ref(false)
const applyLogBox = ref<HTMLElement | null>(null)
let applyEventSource: EventSource | null = null
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

function ddnsManagedZones(cfg: DDNSConfig): string[] {
  const zones = new Set<string>()
  const add = (zone?: string) => {
    const value = zone?.trim().toLowerCase().replace(/^\*\./, '')
    if (value) zones.add(value)
  }
  add(cfg.root_domain)
  for (const name of cfg.record_names ?? []) {
    add(name)
  }
  if (zones.size === 0 && cfg.record_name) {
    add(cfg.record_name)
  }
  return [...zones]
}

const ddnsDomainOptions = computed(() =>
  ddnsConfigs.value.map((c) => {
    const zones = ddnsManagedZones(c)
    const zoneLabel = zones.length > 1 ? zones.join('、') : zones[0] ?? c.root_domain
    return {
      label: `${zoneLabel} · ${providerLabels[c.provider] ?? c.provider}`,
      value: c.root_domain,
    }
  }),
)

const applyDnsHint = computed(() => {
  const cfg = ddnsConfigs.value.find((c) => c.root_domain === applyDnsZone.value)
  if (!cfg) return ''
  const zones = ddnsManagedZones(cfg)
  const zoneText = zones.length > 1 ? `覆盖 ${zones.join('、')}` : zones[0] ?? cfg.root_domain
  return `将使用 ${providerLabels[cfg.provider] ?? cfg.provider} API 完成 DNS-01 验证（${zoneText}）`
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
  {
    title: '错误信息',
    key: 'last_error',
    minWidth: 200,
    ellipsis: { tooltip: true },
    render: (row) => row.last_error || '-',
  },
  {
    title: '剩余',
    key: 'days_left',
    width: 72,
    render: (row) => (row.status === 'error' && !row.expires_at ? '-' : `${row.days_left} 天`),
  },
  { title: '到期', key: 'expires_at', minWidth: 140, render: (row) => formatDate(row.expires_at) },
  {
    title: '操作',
    key: 'actions',
    width: 168,
    render: (row) => {
      const canDownload = row.status !== 'error' || !!row.expires_at
      const buttons = []
      if (canDownload) {
        buttons.push(
          h(
            NDropdown,
            {
              trigger: 'click',
              options: [
                { label: '打包下载 (ZIP)', key: 'zip' },
                { label: '证书 (fullchain.pem)', key: 'cert' },
                { label: '私钥 (privatekey.pem)', key: 'key' },
              ],
              onSelect: (key: string) => downloadCert(row, key as 'zip' | 'cert' | 'key'),
            },
            () =>
              h(
                NButton,
                { size: 'small', quaternary: true, loading: downloadingDomain === row.domain },
                () => '下载',
              ),
          ),
        )
      }
      if (row.acme_ca !== 'imported') {
        buttons.push(
          h(
            NButton,
            { size: 'small', quaternary: true, type: 'primary', onClick: () => openRenewFor(row.domain) },
            () => '续签',
          ),
        )
      }
      buttons.push(
        h(
          NButton,
          { size: 'small', quaternary: true, type: 'error', onClick: () => confirmDelete(row) },
          () => '删除',
        ),
      )
      return h(NSpace, { size: 4, wrap: false, align: 'center', inline: true }, () => buttons)
    },
  },
]

async function downloadCert(row: CertificateRecord, part: 'zip' | 'cert' | 'key') {
  downloadingDomain.value = row.domain
  try {
    await api.downloadCertificate(row.domain, part)
    message.success(part === 'zip' ? '证书包已下载' : '文件已下载')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '下载失败')
  } finally {
    downloadingDomain.value = ''
  }
}

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
    api.listDDNSLite(),
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
    records.value = asList(await api.listCertificates())
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
  loadCAOptions().catch(() => {})
}

async function openApply() {
  if (ddnsDomainOptions.value.length === 0) {
    try {
      await loadCAOptions()
    } catch {
      message.error('加载 DNS 凭证失败')
      return
    }
  }
  if (ddnsDomainOptions.value.length === 0) {
    message.warning('请先在 DDNS 页面添加域名配置')
    return
  }
  applyError.value = ''
  resetApplyForm()
  showApply.value = true
}

function stopApplyStream() {
  applyEventSource?.close()
  applyEventSource = null
  applyStreaming.value = false
}

function appendApplyLog(level: string, text: string) {
  applyLogLines.value.push({ level, text })
  if (applyLogLines.value.length > 300) {
    applyLogLines.value = applyLogLines.value.slice(-250)
  }
  requestAnimationFrame(() => {
    applyLogBox.value?.scrollTo({ top: applyLogBox.value.scrollHeight })
  })
}

function handleApplyJobEvent(ev: CertificateJobEvent) {
  if (ev.type === 'log' && ev.message) {
    appendApplyLog(ev.level || 'info', ev.message)
    return
  }
  if (ev.type === 'done' && ev.result) {
    applyResult.value = ev.result
    applyFinished.value = true
    stopApplyStream()
    if (ev.result.ok) {
      message.success('证书申请成功')
      api.listCertificates().then((list) => {
        records.value = asList(list)
      })
    } else {
      message.error(ev.result.error || '证书申请失败')
      api.listCertificates().then((list) => {
        records.value = asList(list)
      })
    }
  }
}

function startApplyStream(jobId: string) {
  stopApplyStream()
  applyEventSource = new EventSource(api.certificateApplyStreamURL(jobId), { withCredentials: true })
  const onEvent = (event: Event) => {
    try {
      const ev = JSON.parse((event as MessageEvent).data) as CertificateJobEvent
      handleApplyJobEvent(ev)
    } catch {
      // ignore malformed events
    }
  }
  applyEventSource.addEventListener('log', onEvent)
  applyEventSource.addEventListener('done', onEvent)
  applyStreaming.value = true
}

function resetApplyProgress() {
  stopApplyStream()
  applyLogLines.value = []
  applyResult.value = null
  applyFinished.value = false
}

function closeApplyProgress() {
  showApplyProgress.value = false
}

async function submitApply() {
  applyError.value = ''
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
    const { job_id } = await api.applyCertificate({
      dns_zone: applyDnsZone.value,
      domains,
      ca: applyCA.value,
      email: applyEmail.value.trim(),
    })
    showApply.value = false
    resetApplyProgress()
    showApplyProgress.value = true
    startApplyStream(job_id)
  } catch (error) {
    const text = error instanceof Error ? error.message : '申请失败'
    applyError.value = text
    message.error(text)
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
onUnmounted(stopApplyStream)
</script>

<style scoped>
.apply-log-box {
  background: #0f172a;
  color: #e2e8f0;
  min-height: 220px;
  max-height: 320px;
  overflow: auto;
  padding: var(--fonu-space-4);
  border-radius: var(--fonu-radius);
  border: 1px solid var(--fonu-border);
  font-family: var(--fonu-mono);
  font-size: 12px;
  line-height: 1.7;
  margin-bottom: var(--fonu-space-4);
}

html[data-theme='dark'] .apply-log-box,
html.dark .apply-log-box {
  background: #020617;
}

.apply-log-line--warn {
  color: #fbbf24;
}

.apply-log-line--error {
  color: #f87171;
}

.apply-log-empty {
  text-align: center;
  color: #94a3b8;
  padding: var(--fonu-space-5) 0;
}

.apply-result {
  margin-top: var(--fonu-space-2);
}

.apply-result-detail {
  display: grid;
  gap: 10px;
  text-align: left;
  font-size: 13px;
  line-height: 1.6;
  max-width: 100%;
}

.apply-result-detail code {
  word-break: break-all;
}

.apply-result-k {
  display: inline-block;
  min-width: 72px;
  color: var(--fonu-text-muted);
  margin-right: 8px;
}

.cert-overview {
  margin-bottom: var(--fonu-space-5);
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--fonu-space-4);
}

.cert-loading {
  display: flex;
  justify-content: center;
  padding: var(--fonu-space-6) 0;
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
