<template>
  <PageHeader
    title="HTTPS 证书"
    description="通过 ACME 自动申请、续签，或导入已有证书，用于为域名提供安全的 HTTPS 访问"
  >
    <template #actions>
      <n-button @click="openImport">导入证书</n-button>
      <n-button type="primary" @click="openApply">
        <template #icon><n-icon :component="AddOutline" /></template>
        申请证书
      </n-button>
    </template>
  </PageHeader>

  <LoadError v-if="loadError" :message="loadError" @retry="load" />

  <template v-else>
    <div class="stats-row">
      <div class="stat-card">
        <n-icon :component="FolderOpenOutline" class="stat-card__icon stat-card__icon--blue" />
        <div class="stat-card__value">{{ records.length }}</div>
        <div class="stat-card__label">证书总数</div>
        <div class="stat-card__sub">当前已配置的证书</div>
      </div>

      <div class="stat-card">
        <n-icon :component="CheckmarkCircleOutline" class="stat-card__icon stat-card__icon--green" />
        <div class="stat-card__value stat-card__value--green">{{ normalCount }}</div>
        <div class="stat-card__label">正常证书</div>
        <div class="stat-card__sub">运行正常</div>
      </div>

      <div class="stat-card">
        <n-icon :component="AlertCircleOutline" class="stat-card__icon stat-card__icon--amber" />
        <div class="stat-card__value stat-card__value--amber">{{ expiringCount }}</div>
        <div class="stat-card__label">即将过期</div>
        <div class="stat-card__sub">30 天内到期</div>
      </div>

      <div class="stat-card">
        <n-icon :component="CloseCircleOutline" class="stat-card__icon stat-card__icon--red" />
        <div class="stat-card__value stat-card__value--red">{{ expiredCount }}</div>
        <div class="stat-card__label">已过期</div>
        <div class="stat-card__sub">需要处理</div>
      </div>
    </div>

    <FonuCard flush class="cert-panel">
      <div class="cert-toolbar">
        <n-input
          v-model:value="searchQuery"
          size="small"
          placeholder="搜索证书名称或关联域名..."
          clearable
          class="cert-toolbar__search"
        >
          <template #prefix>
            <n-icon :component="SearchOutline" />
          </template>
        </n-input>
        <n-select
          v-model:value="statusFilter"
          size="small"
          :options="statusFilterOptions"
          class="cert-toolbar__filter"
        />
        <n-select
          v-model:value="issuerFilter"
          size="small"
          :options="issuerFilterOptions"
          class="cert-toolbar__filter"
        />
        <div class="cert-toolbar__spacer" />
      </div>

      <div v-if="loading" class="cert-loading">
        <n-spin size="medium" />
      </div>

      <div v-else-if="filteredRecords.length > 0" class="cert-table-wrap">
        <n-data-table
          class="cert-table"
          :columns="columns"
          :data="paginatedRecords"
          :bordered="false"
          size="small"
          :scroll-x="1100"
        />
        <div v-if="filteredRecords.length > pageSize" class="cert-pagination">
          <n-pagination
            v-model:page="page"
            :page-size="pageSize"
            :item-count="filteredRecords.length"
            size="small"
          />
        </div>
      </div>

      <EmptyState
        v-else
        :title="records.length === 0 ? '还没有证书' : '没有匹配的证书'"
        :description="
          records.length === 0
            ? '申请证书时选择 DNS API 完成验证即可；本地测试也可直接导入已有证书。'
            : '试试调整搜索关键词或筛选条件。'
        "
      >
        <template v-if="records.length === 0" #action>
          <n-space>
            <n-button @click="openImport">导入证书</n-button>
            <n-button type="primary" @click="openApply">申请证书</n-button>
          </n-space>
        </template>
      </EmptyState>
    </FonuCard>

    <div class="tip-box">
      <n-icon :component="InformationCircleOutline" class="tip-box__icon" />
      <div>
        <strong>使用提示：</strong>
        证书属于域名本身；DNS 服务商仅作为申请/续签时的 DNS-01 验证通道，不会在证书上长期绑定。一张证书可包含多个域名和通配符，续签时系统会自动选用可用的 DNS 凭证完成验证。
      </div>
    </div>
  </template>

  <!-- 申请（三步向导） -->
  <n-modal
    v-model:show="showApply"
    preset="card"
    :mask-closable="applyStep === 3 && applyFinished"
    :close-on-esc="applyStep !== 3 || applyFinished"
    :style="{ width: applyStep === 3 ? 'min(720px, 96vw)' : 'min(640px, 92vw)' }"
    :segmented="{ content: true, footer: 'soft' }"
    :content-style="{ paddingTop: '8px' }"
    @after-leave="resetApplyWizard"
  >
    <template #header>
      <div class="modal-header">
        <div class="modal-header__title">申请证书</div>
        <div class="modal-header__desc">通过 DNS-01 验证自动申请证书，证书归属于域名而非 DNS 服务商</div>
      </div>
    </template>

    <div class="apply-steps">
      <div class="apply-step" :class="{ 'apply-step--active': applyStep === 1, 'apply-step--done': applyStep > 1 }">
        <span class="apply-step__dot">1</span>
        <span class="apply-step__label">填写信息</span>
      </div>
      <div class="apply-step__line" :class="{ 'apply-step__line--done': applyStep > 1 }" />
      <div class="apply-step" :class="{ 'apply-step--active': applyStep === 2, 'apply-step--done': applyStep > 2 }">
        <span class="apply-step__dot">2</span>
        <span class="apply-step__label">DNS 验证</span>
      </div>
      <div class="apply-step__line" :class="{ 'apply-step__line--done': applyStep > 2 }" />
      <div class="apply-step" :class="{ 'apply-step--active': applyStep === 3, 'apply-step--done': applyFinished && applyResult?.ok }">
        <span class="apply-step__dot">3</span>
        <span class="apply-step__label">申请完成</span>
      </div>
    </div>

    <n-form v-if="applyStep === 1" label-placement="top" class="cert-form">
      <n-form-item label="证书名称" feedback="仅用于本地识别，可自定义" :show-feedback="true" required>
        <n-input v-model:value="applyCertName" placeholder="例如 example" />
      </n-form-item>

      <n-form-item
        label="域名"
        feedback="每行一个域名，支持通配符（例如：*.example.com）"
        :show-feedback="true"
        required
      >
        <n-input
          v-model:value="applyDomainsText"
          class="domains-input"
          type="textarea"
          :rows="5"
          placeholder="example.com&#10;*.example.com&#10;api.example.com"
          @update:value="syncCertNameFromDomains"
        />
      </n-form-item>

      <n-form-item label="颁发机构" :show-feedback="false">
        <div class="ca-field">
          <div class="ca-cards">
            <button
              v-for="opt in applyCAOptions"
              :key="opt.value"
              type="button"
              class="ca-card"
              :class="{ 'ca-card--active': applyCA === opt.value }"
              @click="applyCA = opt.value"
            >
              <span class="ca-card__check" aria-hidden="true" />
              <div class="ca-card__logo">
                <img v-if="caLogo(opt.value)" :src="caLogo(opt.value)" :alt="caCardTitle(opt.value)" />
                <span v-else class="ca-card__fallback">{{ caCardTitle(opt.value) }}</span>
              </div>
              <div class="ca-card__sub">{{ caCardSub(opt.value) }}</div>
            </button>
          </div>
          <p v-if="applyCA === 'zerossl' && !zerosslApiKey" class="ca-hint ca-hint--warn">
            使用 ZeroSSL 需在「设置」中配置 API Key
          </p>
          <p v-if="applyCA === 'buypass'" class="ca-hint">
            Buypass 不支持通配符，单张证书最多 5 个域名
          </p>
        </div>
      </n-form-item>

      <div class="apply-auto-renew">
        <div>
          <div class="apply-auto-renew__title">自动续期</div>
          <div class="apply-auto-renew__sub">在证书到期前自动续期</div>
        </div>
        <n-switch v-model:value="applyAutoRenew" />
      </div>

      <n-form-item
        v-if="!applyEmailConfigured"
        label="ACME 邮箱"
        feedback="用于账户注册与到期提醒，保存后记住"
        :show-feedback="true"
        required
      >
        <n-input v-model:value="applyEmail" placeholder="admin@example.com" />
      </n-form-item>

      <n-alert v-if="applyError" type="error" :bordered="false" class="form-alert" :title="applyError" />
    </n-form>

    <div v-else-if="applyStep === 2" class="verify-step">
      <p class="verify-step__intro">
        选择用于本次 DNS-01 验证的 DNS 任务。这只是验证通道，选错会在下一步报错，不会与证书长期绑定。
      </p>

      <div v-if="applyDdnsConfigs.length === 0" class="verify-empty">
        <p v-if="ddnsConfigs.length === 0">还没有 DNS 任务。</p>
        <p v-else>没有已启用的 DNS 任务。</p>
        <p class="verify-empty__sub">
          {{ ddnsConfigs.length === 0
            ? '请先在 DDNS 页面添加 DNS API 凭证，然后返回继续申请。'
            : '请先在 DDNS 页面启用至少一个 DNS 任务，然后返回继续申请。' }}
        </p>
      </div>

      <div v-else class="verify-options">
        <button
          v-for="cfg in applyDdnsConfigs"
          :key="cfg.id"
          type="button"
          class="verify-option"
          :class="{ 'verify-option--active': applyDdnsConfigId === cfg.id }"
          @click="selectVerificationChannel(cfg)"
        >
          <span class="provider-logo">
            <img :src="providerIcon(cfg.provider)" :alt="providerLabel(cfg.provider)" />
          </span>
          <div class="verify-option__body">
            <div class="verify-option__title">{{ providerLabel(cfg.provider) }}</div>
            <div class="verify-option__sub">上次同步 {{ formatRelativeTime(cfg.last_updated_at) || '从未' }}</div>
            <div class="verify-option__zones">{{ ddnsTaskSummary(cfg) }}</div>
          </div>
          <span class="verify-option__check" />
        </button>
      </div>

      <n-alert v-if="applyError" type="error" :bordered="false" class="form-alert" :title="applyError" />
    </div>

    <div v-else class="apply-done-step">
      <p v-if="!applyFinished" class="apply-done-step__hint">
        DNS 验证与证书签发可能需要 1～3 分钟，请保持窗口打开
      </p>

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
        <div class="apply-result-detail">
          <div><span class="apply-result-k">证书目录</span><code>{{ applyResult.cert_dir || '-' }}</code></div>
          <div><span class="apply-result-k">证书文件</span><code>{{ applyResult.cert_path || '-' }}</code></div>
          <div><span class="apply-result-k">私钥文件</span><code>{{ applyResult.key_path || '-' }}</code></div>
          <div><span class="apply-result-k">到期时间</span>{{ formatDate(applyResult.expires_at) }}</div>
        </div>
      </n-result>

      <n-result
        v-else-if="applyFinished && applyResult && !applyResult.ok"
        status="error"
        title="证书申请失败"
        :description="applyResult.error || '未知错误'"
        class="apply-result"
      />
    </div>

    <template #footer>
      <n-space v-if="applyStep === 1" :key="1" justify="end">
        <n-button @click="showApply = false">取消</n-button>
        <n-button type="primary" @click="goApplyStep2">下一步</n-button>
      </n-space>
      <n-space v-else-if="applyStep === 2" :key="2" justify="end">
        <n-button @click="goApplyStep1">上一步</n-button>
        <n-button
          type="primary"
          :loading="applying"
          :disabled="applyDdnsConfigs.length === 0 || !applyDdnsConfigId"
          @click="submitApply"
        >
          开始申请
        </n-button>
      </n-space>
      <n-space v-else-if="applyStep === 3" :key="3" justify="end">
        <n-button v-if="!applyFinished" :loading="true" disabled>申请进行中…</n-button>
        <n-button v-else type="primary" @click="showApply = false">完成</n-button>
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
import { computed, h, onMounted, onUnmounted, ref, watch, type VNode } from 'vue'
import {
  NAlert,
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NPagination,
  NRadioButton,
  NRadioGroup,
  NResult,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NTag,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import {
  AddOutline,
  AlertCircleOutline,
  CheckmarkCircleOutline,
  CloseCircleOutline,
  FolderOpenOutline,
  InformationCircleOutline,
  SearchOutline,
} from '@vicons/ionicons5'
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
import buypassLogo from '../assets/brand/ca/buypass.png'
import letsencryptLogo from '../assets/brand/ca/letsencrypt.png'
import zerosslLogo from '../assets/brand/ca/zerossl.png'
import aliyunIcon from '../assets/brand/dns/aliyun.png'
import cloudflareIcon from '../assets/brand/dns/cloudflare.png'
import dnspodIcon from '../assets/brand/dns/dnspod.png'
import { formatDate, formatRelativeTime } from '../utils/format'
import { renderTableRowActions } from '../utils/tableActions'

type ImportMode = 'paste' | 'path'

const message = useMessage()
const dialog = useDialog()
const records = ref<CertificateRecord[]>([])
const loading = ref(false)
const downloadingDomain = ref('')
const loadError = ref('')
const applying = ref(false)
const renewingDomain = ref('')
const importing = ref(false)
const showImport = ref(false)
const showApply = ref(false)
const caOptions = ref<CertificateCAOption[]>([])
const applyCAOptions = [
  { value: 'letsencrypt', label: "Let's Encrypt" },
  { value: 'zerossl', label: 'ZeroSSL' },
  { value: 'buypass', label: 'Buypass' },
]
const applyCA = ref('letsencrypt')
const zerosslApiKey = ref('')
const applyEmail = ref('')
const applyCertName = ref('')
const applyDomainsText = ref('')
const applyAutoRenew = ref(true)
const applyStep = ref<1 | 2 | 3>(1)
const applyDdnsConfigId = ref<number | null>(null)
const applyError = ref('')
const applyLogLines = ref<{ level: string; text: string }[]>([])
const applyResult = ref<CertificateJobDone | null>(null)
const applyFinished = ref(false)
const applyStreaming = ref(false)
const applyLogBox = ref<HTMLElement | null>(null)
let applyEventSource: EventSource | null = null
const ddnsConfigs = ref<DDNSConfig[]>([])
const applyDdnsConfigs = computed(() => ddnsConfigs.value.filter((c) => c.enabled))
const certFileInput = ref<HTMLInputElement | null>(null)
const keyFileInput = ref<HTMLInputElement | null>(null)
const importMode = ref<ImportMode>('paste')
const importForm = ref({
  certificate: '',
  private_key: '',
  cert_path: '',
  key_path: '',
})
const searchQuery = ref('')
const statusFilter = ref('all')
const issuerFilter = ref('all')
const page = ref(1)
const pageSize = 10

const statusFilterOptions = [
  { label: '全部状态', value: 'all' },
  { label: '正常', value: 'ok' },
  { label: '即将过期', value: 'expiring' },
  { label: '已过期', value: 'expired' },
  { label: '异常', value: 'error' },
]

const providerMap: Record<string, { label: string; icon: string }> = {
  cloudflare: { label: 'Cloudflare', icon: cloudflareIcon },
  dnspod: { label: 'DNSPod', icon: dnspodIcon },
  alidns: { label: '阿里云 DNS', icon: aliyunIcon },
}

const caLogoMap: Record<string, string> = {
  letsencrypt: letsencryptLogo,
  'letsencrypt-staging': letsencryptLogo,
  zerossl: zerosslLogo,
  buypass: buypassLogo,
}

const caCardMeta: Record<string, { title: string; sub: string }> = {
  letsencrypt: { title: "Let's Encrypt", sub: '免费 · 自动续期' },
  'letsencrypt-staging': { title: "Let's Encrypt 测试", sub: '仅用于验证流程' },
  zerossl: { title: 'ZeroSSL', sub: '免费 · 稳定' },
  buypass: { title: 'Buypass', sub: '免费 · 备用' },
}

function providerLabel(v: string) {
  return providerMap[v]?.label ?? v
}

function providerIcon(v: string) {
  return providerMap[v]?.icon ?? ''
}

function caCardTitle(value: string) {
  return caCardMeta[value]?.title ?? caOptions.value.find((o) => o.value === value)?.label ?? value
}

function caCardSub(value: string) {
  return caCardMeta[value]?.sub ?? 'ACME 证书'
}

function caLogo(value: string) {
  return caLogoMap[value] ?? ''
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

function ddnsRecordCount(cfg: DDNSConfig) {
  if (cfg.domain_records?.length) return cfg.domain_records.length
  if (cfg.record_names?.length) return cfg.record_names.length
  return cfg.record_name ? 1 : 0
}

function ddnsTaskSummary(cfg: DDNSConfig) {
  const zones = ddnsManagedZones(cfg)
  const zone = zones[0] ?? cfg.root_domain
  if (!zone) return '未配置域名'
  const count = ddnsRecordCount(cfg)
  return count > 0 ? `${zone}（共 ${count} 条记录）` : zone
}

type CertLifeStatus = 'ok' | 'expiring' | 'expired' | 'error'

function certLifeStatus(row: CertificateRecord): CertLifeStatus {
  if (row.status === 'error' && !row.expires_at) return 'error'
  if (row.expires_at && row.days_left <= 0) return 'expired'
  if (row.expires_at && row.days_left <= 30) return 'expiring'
  if (row.status === 'error') return 'error'
  return 'ok'
}

function certLifeLabel(status: CertLifeStatus) {
  if (status === 'expiring') return '即将过期'
  if (status === 'expired') return '已过期'
  if (status === 'error') return '异常'
  return '正常'
}

function certLifeTagType(status: CertLifeStatus): 'success' | 'warning' | 'error' | 'default' {
  if (status === 'expiring') return 'warning'
  if (status === 'expired' || status === 'error') return 'error'
  return 'success'
}

function certDisplayName(domain: string) {
  const base = domain.replace(/^\*\./, '').split('.')[0]
  return base || domain
}

function certSubtitle(row: CertificateRecord) {
  const domains = row.domains?.length ? row.domains : [row.domain]
  if (row.wildcard || domains.some((d) => d.startsWith('*.'))) return '通配符证书'
  if (domains.length > 1) return '多域名证书'
  return '主证书'
}

function issuerFilterOptionsComputed() {
  const issuers = new Set<string>()
  for (const row of records.value) {
    issuers.add(row.acme_ca || 'unknown')
  }
  const options = [{ label: '全部颁发机构', value: 'all' }]
  for (const value of issuers) {
    options.push({ label: caLabel(value), value })
  }
  return options
}

const issuerFilterOptions = computed(() => issuerFilterOptionsComputed())

const normalCount = computed(() => records.value.filter((r) => certLifeStatus(r) === 'ok').length)
const expiringCount = computed(() => records.value.filter((r) => certLifeStatus(r) === 'expiring').length)
const expiredCount = computed(() => records.value.filter((r) => certLifeStatus(r) === 'expired').length)

const filteredRecords = computed(() => {
  let list = records.value
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter((row) => {
      const hay = [row.domain, certDisplayName(row.domain), certSubtitle(row), ...(row.domains ?? []), caLabel(row.acme_ca)]
        .join(' ')
        .toLowerCase()
      return hay.includes(q)
    })
  }
  if (statusFilter.value !== 'all') {
    list = list.filter((row) => certLifeStatus(row) === statusFilter.value)
  }
  if (issuerFilter.value !== 'all') {
    list = list.filter((row) => (row.acme_ca || 'unknown') === issuerFilter.value)
  }
  return list
})

const paginatedRecords = computed(() => {
  const start = (page.value - 1) * pageSize
  return filteredRecords.value.slice(start, start + pageSize)
})

watch([searchQuery, statusFilter, issuerFilter], () => {
  page.value = 1
})

function caLabel(value?: string) {
  return caOptions.value.find((o) => o.value === value)?.label ?? (value === 'imported' ? '手动导入' : value || '-')
}

function renderDomainTags(row: CertificateRecord) {
  const domains = row.domains?.length ? row.domains : [row.domain]
  const visible = domains.slice(0, 2)
  const extra = domains.length - visible.length
  const tags: VNode[] = visible.map((domain) =>
    h(NTag, { size: 'small', round: true, bordered: false, type: 'info' }, () => domain),
  )
  if (extra > 0) {
    tags.push(h(NTag, { size: 'small', round: true, bordered: false }, () => `+${extra}`))
  }
  return h('div', { class: 'table-tags' }, tags)
}

function renderIssuer(row: CertificateRecord) {
  const label = caLabel(row.acme_ca)
  const logo = caLogo(row.acme_ca || '')
  const icon = logo
    ? h('span', { class: 'issuer-logo' }, [h('img', { src: logo, alt: label })])
    : h(
        'span',
        { class: `issuer-badge issuer-badge--${issuerTone(row.acme_ca)}` },
        label.slice(0, 1).toUpperCase(),
      )
  return h('div', { class: 'issuer-cell' }, [icon, h('span', { class: 'issuer-cell__label' }, label)])
}

function issuerTone(ca?: string) {
  if (ca === 'letsencrypt' || ca === 'letsencrypt-staging') return 'le'
  if (ca === 'zerossl') return 'zero'
  if (ca === 'buypass') return 'buypass'
  if (ca === 'imported') return 'imported'
  return 'default'
}

const applyEmailConfigured = computed(() => applyEmail.value.trim().length > 0)

function validateApplyCA(domains: string[]) {
  if (applyCA.value === 'buypass') {
    if (domains.some((d) => d.startsWith('*.'))) {
      return 'Buypass 不支持通配符证书'
    }
    if (domains.length > 5) {
      return 'Buypass 单张证书最多支持 5 个域名'
    }
  }
  if (applyCA.value === 'zerossl' && !zerosslApiKey.value.trim()) {
    return '请先在设置中配置 ZeroSSL API Key'
  }
  return ''
}

function autoRenewEnabled(row: CertificateRecord) {
  return row.acme_ca !== 'imported' && row.status !== 'error'
}

const columns: DataTableColumns<CertificateRecord> = [
  {
    title: '证书名称',
    key: 'domain',
    minWidth: 130,
    render: (row) =>
      h('div', { class: 'cert-name-cell' }, [
        h('div', { class: 'cert-name-cell__title' }, certDisplayName(row.domain)),
        h('div', { class: 'cert-name-cell__sub' }, certSubtitle(row)),
      ]),
  },
  {
    title: '关联域名',
    key: 'domains',
    minWidth: 200,
    render: (row) => renderDomainTags(row),
  },
  {
    title: '颁发机构',
    key: 'acme_ca',
    minWidth: 140,
    render: (row) => renderIssuer(row),
  },
  {
    title: '状态',
    key: 'status',
    width: 108,
    render: (row) => {
      const life = certLifeStatus(row)
      return h(
        NTag,
        { size: 'small', round: true, bordered: false, type: certLifeTagType(life) },
        () => certLifeLabel(life),
      )
    },
  },
  {
    title: '到期时间',
    key: 'expires_at',
    minWidth: 170,
    render: (row) => {
      if (!row.expires_at) return h('span', { class: 'text-muted' }, '-')
      const daysText = row.days_left >= 0 ? `剩余 ${row.days_left} 天` : '已过期'
      return h('div', { class: 'expire-cell' }, [
        h('div', { class: 'expire-cell__date' }, formatDate(row.expires_at)),
        h('div', { class: 'expire-cell__days' }, daysText),
      ])
    },
  },
  {
    title: '自动续期',
    key: 'auto_renew',
    width: 88,
    render: (row) =>
      h(NSwitch, {
        value: autoRenewEnabled(row),
        size: 'small',
        disabled: true,
      }),
  },
  {
    title: '操作',
    key: 'actions',
    width: 168,
    fixed: 'right',
    render: (row) => {
      const canDownload = row.status !== 'error' || !!row.expires_at
      return renderTableRowActions([
        {
          label: '续签',
          show: row.acme_ca !== 'imported',
          onClick: () => confirmRenew(row),
        },
        {
          label: '下载',
          show: canDownload,
          onClick: () => downloadCert(row, 'zip'),
        },
        {
          label: '删除',
          type: 'error',
          onClick: () => confirmDelete(row),
        },
      ])
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

function confirmRenew(row: CertificateRecord) {
  if (row.acme_ca === 'imported') {
    message.warning('手动导入的证书请重新导入，无法续签')
    return
  }
  const domains = row.domains?.length ? row.domains.join('、') : row.domain
  dialog.warning({
    title: '续签证书',
    content: `确定续签 ${domains} 的证书？系统将自动选用可用的 DNS 凭证完成验证。`,
    positiveText: '续签',
    negativeText: '取消',
    onPositiveClick: async () => {
      renewingDomain.value = row.domain
      try {
        const record = await api.renewCertificate(row.domain, row.acme_ca || 'letsencrypt')
        records.value = [record, ...records.value.filter((r) => r.id !== record.id)]
        message.success('证书续签成功')
      } catch (error) {
        message.error(error instanceof Error ? error.message : '续签失败')
      } finally {
        renewingDomain.value = ''
      }
    },
  })
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

function parseDomainsInput(text: string) {
  return text
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function syncCertNameFromDomains() {
  if (applyCertName.value.trim()) return
  const first = parseDomainsInput(applyDomainsText.value)[0]
  if (!first) return
  applyCertName.value = first.replace(/^\*\./, '').split('.')[0] ?? ''
}

function selectVerificationChannel(cfg: DDNSConfig) {
  if (!cfg.enabled) return
  applyDdnsConfigId.value = cfg.id
}

function resetApplyWizard() {
  applyStep.value = 1
  applyCertName.value = ''
  applyDomainsText.value = ''
  applyDdnsConfigId.value = null
  applyAutoRenew.value = true
  applyError.value = ''
  resetApplyProgress()
}

function goApplyStep1() {
  applyStep.value = 1
  applying.value = false
  applyError.value = ''
}

function goApplyStep2() {
  applyError.value = ''
  if (!applyCertName.value.trim()) {
    applyError.value = '请填写证书名称'
    return
  }
  const domains = parseDomainsInput(applyDomainsText.value)
  if (domains.length === 0) {
    applyError.value = '请填写至少一个域名'
    return
  }
  if (!applyEmail.value.trim()) {
    applyError.value = '请填写 ACME 邮箱'
    return
  }
  const caError = validateApplyCA(domains)
  if (caError) {
    applyError.value = caError
    return
  }
  if (applyDdnsConfigs.value.length === 1) {
    applyDdnsConfigId.value = applyDdnsConfigs.value[0].id
  } else {
    applyDdnsConfigId.value = null
  }
  applyStep.value = 2
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
  zerosslApiKey.value = settings.zerossl_api_key ?? ''
  const defaultCA = settings.acme_ca || 'letsencrypt'
  applyCA.value = applyCAOptions.some((o) => o.value === defaultCA) ? defaultCA : 'letsencrypt'
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
  resetApplyWizard()
  try {
    await loadCAOptions()
  } catch {
    message.error('加载配置失败')
    return
  }
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

async function submitApply() {
  applyError.value = ''
  const cfg = applyDdnsConfigs.value.find((c) => c.id === applyDdnsConfigId.value)
  if (!cfg) {
    applyError.value = '请选择已启用的 DNS 任务用于 DNS-01 验证'
    return
  }
  const domains = parseDomainsInput(applyDomainsText.value)
  if (domains.length === 0) {
    applyError.value = '请填写至少一个域名'
    return
  }
  if (!applyEmail.value.trim()) {
    applyError.value = '请填写 ACME 邮箱'
    return
  }
  const caError = validateApplyCA(domains)
  if (caError) {
    applyError.value = caError
    return
  }
  applying.value = true
  try {
    const { job_id } = await api.applyCertificate({
      ddns_config_id: Number(cfg.id),
      domains,
      ca: applyCA.value,
      email: applyEmail.value.trim(),
    })
    applyStep.value = 3
    resetApplyProgress()
    startApplyStream(job_id)
  } catch (error) {
    const text = error instanceof Error ? error.message : '申请失败'
    applyStep.value = 3
    resetApplyProgress()
    applyFinished.value = true
    applyResult.value = {
      ok: false,
      error: text,
      domain: domains[0],
    }
    message.error(text)
  } finally {
    applying.value = false
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

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.stat-card {
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  padding: var(--fonu-space-4) var(--fonu-space-5);
  min-height: 118px;
}

.stat-card__icon { font-size: 18px; margin-bottom: 4px; }
.stat-card__icon--blue { color: #3b82f6; }
.stat-card__icon--green { color: #10b981; }
.stat-card__icon--amber { color: #f59e0b; }
.stat-card__icon--red { color: #ef4444; }

.stat-card__value {
  margin-top: 4px;
  font-size: 30px;
  font-weight: 700;
  line-height: 1.1;
  color: var(--fonu-text);
  letter-spacing: -0.03em;
}

.stat-card__value--green { color: #10b981; }
.stat-card__value--amber { color: #f59e0b; }
.stat-card__value--red { color: #ef4444; }

.stat-card__label {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text);
}

.stat-card__sub {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.cert-panel {
  min-width: 0;
}

.cert-panel :deep(.fonu-card__body) {
  padding-left: 0;
  padding-right: 0;
  padding-bottom: 0;
}

.cert-toolbar {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-4);
  border-bottom: 1px solid var(--fonu-border);
  flex-wrap: wrap;
}

.cert-toolbar__search {
  width: 240px;
  max-width: 100%;
  flex-shrink: 0;
}

.cert-toolbar__filter {
  width: 132px;
  flex-shrink: 0;
}

.cert-toolbar__spacer {
  flex: 1;
  min-width: 0;
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

.cert-table { width: 100%; }

.cert-table :deep(.n-data-table-td) {
  vertical-align: middle;
}

.cert-table :deep(.cert-name-cell__title) {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.cert-table :deep(.cert-name-cell__sub) {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.cert-pagination {
  display: flex;
  justify-content: flex-end;
  padding: var(--fonu-space-3) var(--fonu-space-4) var(--fonu-space-4);
}

.cert-name-cell__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.cert-name-cell__sub {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.cert-table :deep(.issuer-cell) {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cert-table :deep(.issuer-badge) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
}

.cert-table :deep(.issuer-badge--le) { background: rgba(16, 185, 129, 0.15); color: #059669; }
.cert-table :deep(.issuer-badge--zero) { background: rgba(59, 130, 246, 0.15); color: #2563eb; }
.cert-table :deep(.issuer-badge--buypass) { background: rgba(37, 99, 235, 0.15); color: #1d4ed8; }
.cert-table :deep(.issuer-badge--imported) { background: rgba(100, 116, 139, 0.15); color: #64748b; }
.cert-table :deep(.issuer-badge--default) { background: var(--fonu-bg); color: var(--fonu-text-secondary); }

.cert-table :deep(.issuer-logo) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  border-radius: 6px;
  background-color: #fff;
  border: 1px solid rgba(15, 23, 42, 0.08);
  overflow: hidden;
}

.cert-table :deep(.issuer-logo img) {
  width: 100%;
  height: 100%;
  max-width: 28px;
  max-height: 28px;
  object-fit: contain;
  display: block;
}

.cert-table :deep(.issuer-cell__label) {
  font-size: 13px;
  color: var(--fonu-text);
  white-space: nowrap;
}

.expire-cell__date {
  font-size: 13px;
  color: var(--fonu-text);
}

.expire-cell__days {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.text-muted { color: var(--fonu-text-muted); }

.tip-box {
  display: flex;
  gap: 10px;
  margin-top: var(--fonu-space-4);
  padding: 14px 16px;
  border-radius: 12px;
  background: rgba(59, 130, 246, 0.08);
  border: 1px solid rgba(59, 130, 246, 0.16);
  color: #1d4ed8;
  font-size: 13px;
  line-height: 1.6;
}

.tip-box__icon {
  font-size: 18px;
  flex-shrink: 0;
  margin-top: 2px;
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

.ca-field {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}

.ca-hint {
  margin: 0;
  width: 100%;
  font-size: 12px;
  line-height: 1.5;
  color: var(--fonu-text-muted);
}

.ca-hint--warn {
  color: #d97706;
}

.apply-steps {
  display: flex;
  align-items: center;
  gap: 0;
  margin-bottom: var(--fonu-space-5);
  padding: 0 4px;
}

.apply-step {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--fonu-text-muted);
  font-size: 13px;
  white-space: nowrap;
}

.apply-step--active {
  color: var(--fonu-text);
  font-weight: 600;
}

.apply-step--done {
  color: #10b981;
}

.apply-step__dot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid currentColor;
  font-size: 12px;
  font-weight: 600;
}

.apply-step--active .apply-step__dot,
.apply-step--done .apply-step__dot {
  background: #10b981;
  border-color: #10b981;
  color: #fff;
}

.apply-step__line {
  flex: 1;
  height: 2px;
  margin: 0 12px;
  background: var(--fonu-border);
  min-width: 24px;
}

.apply-step__line--done {
  background: #10b981;
}

.ca-cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  width: 100%;
}

.ca-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 10px 10px 12px;
  border: 1px solid var(--fonu-border);
  border-radius: 12px;
  background: var(--fonu-surface);
  cursor: pointer;
  text-align: center;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.ca-card:hover {
  border-color: rgba(16, 185, 129, 0.45);
}

.ca-card--active {
  border-color: #10b981;
  box-shadow: 0 0 0 1px rgba(16, 185, 129, 0.18);
}

.ca-card__check {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 2px solid #d1d5db;
  background: #fff;
  box-sizing: border-box;
}

.ca-card--active .ca-card__check {
  border-color: #10b981;
  background: #10b981;
}

.ca-card--active .ca-card__check::after {
  content: '';
  position: absolute;
  left: 5px;
  top: 2px;
  width: 5px;
  height: 9px;
  border: solid #fff;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

.ca-card__logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  aspect-ratio: 1;
  max-height: 108px;
  padding: 8px;
  border-radius: 10px;
  background-color: #fff;
  border: 1px solid rgba(15, 23, 42, 0.06);
  overflow: hidden;
}

.ca-card__logo img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  display: block;
}

.ca-card__fallback {
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text);
}

.ca-card__sub {
  width: 100%;
  font-size: 12px;
  line-height: 1.4;
  color: var(--fonu-text-muted);
}

.apply-auto-renew {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  padding: 14px 16px;
  margin-top: 4px;
  border: 1px solid var(--fonu-border);
  border-radius: 12px;
  background: var(--fonu-bg);
}

.apply-auto-renew__title {
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.apply-auto-renew__sub {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

.verify-step__intro {
  margin: 0 0 var(--fonu-space-4);
  font-size: 13px;
  line-height: 1.6;
  color: var(--fonu-text-secondary);
}

.verify-empty {
  padding: var(--fonu-space-5);
  text-align: center;
  border: 1px dashed var(--fonu-border);
  border-radius: 12px;
  color: var(--fonu-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.verify-empty__sub {
  margin-top: 8px;
  color: var(--fonu-text-muted);
  font-size: 12px;
}

.verify-options {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-3);
}

.verify-option {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--fonu-border);
  border-radius: 12px;
  background: var(--fonu-surface);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s;
}

.verify-option:hover {
  border-color: rgba(16, 185, 129, 0.4);
}

.verify-option--active {
  border-color: #10b981;
  box-shadow: 0 0 0 1px rgba(16, 185, 129, 0.2);
}

.provider-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 10px;
  background: var(--fonu-bg);
  overflow: hidden;
}

.provider-logo img {
  width: 28px;
  height: 28px;
  object-fit: contain;
}

.verify-option__body {
  flex: 1;
  min-width: 0;
}

.verify-option__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.verify-option__sub {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.verify-option__zones {
  margin-top: 6px;
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.verify-option__check {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 2px solid var(--fonu-border);
  flex-shrink: 0;
}

.verify-option--active .verify-option__check {
  border-color: #10b981;
  background: #10b981;
  box-shadow: inset 0 0 0 4px #fff;
}

.apply-done-step__hint {
  margin: 0 0 var(--fonu-space-3);
  font-size: 13px;
  color: var(--fonu-text-secondary);
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

@media (max-width: 1199px) {
  .stats-row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 767px) {
  .stats-row { grid-template-columns: 1fr; }
  .cert-toolbar__filter { width: 100%; }
  .ca-cards { grid-template-columns: 1fr; }
  .cert-table :deep(.n-data-table-base-table) {
    min-width: 900px;
  }
}
</style>
