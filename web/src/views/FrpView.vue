<template>
  <PageHeader
    title="内网穿透"
    description="通过 VPS + FRP，让你即使没有公网 IP，也能随时访问 NAS 上的服务。"
  >
    <template #actions>
      <n-button quaternary @click="showHelp = true">
        <template #icon><n-icon :component="HelpCircleOutline" /></template>
        使用说明
      </n-button>
    </template>
  </PageHeader>

  <LoadError v-if="pageError" :message="pageError" @retry="load" />

  <n-spin v-else :show="loading">
    <div class="frp-page">
      <div class="top-grid">
        <FonuCard class="status-card" flush>
          <div class="status-card__body">
            <div class="status-hero" :class="statusHeroClass">
              <div class="status-hero__main">
                <div class="status-hero__icon-wrap">
                  <n-icon :component="statusHeroIcon" />
                </div>
                <div>
                  <div class="status-hero__title">{{ connectionTitle }}</div>
                  <div class="status-hero__sub">{{ connectionSubtitle }}</div>
                </div>
              </div>
              <n-button size="small" secondary :disabled="!frpForm.enabled" @click="openLogs">
                <template #icon><n-icon :component="DocumentTextOutline" /></template>
                查看日志
              </n-button>
            </div>

            <div class="status-flow">
              <span class="status-flow__node">外网用户</span>
              <span class="status-flow__arrow">→</span>
              <span class="status-flow__node status-flow__node--accent">frps</span>
              <span class="status-flow__arrow">→</span>
              <span class="status-flow__node">frpc</span>
              <span class="status-flow__arrow">→</span>
              <span class="status-flow__node">Fonu Nginx</span>
            </div>

            <div class="status-tags">
              <span class="status-tag" :class="{ 'status-tag--on': frpForm.enabled }">
                {{ frpForm.enabled ? '穿透已启用' : '穿透未启用' }}
              </span>
              <span class="status-tag" :class="{ 'status-tag--on': frpForm.tls_enabled }">
                TLS {{ frpForm.tls_enabled ? '已开启' : '未开启' }}
              </span>
              <span class="status-tag" :class="{ 'status-tag--on': domainCount > 0 }">
                {{ domainCount }} 个域名
              </span>
            </div>

            <div class="status-metrics">
              <div class="status-metric">
                <div class="status-metric__label">服务器地址</div>
                <code class="status-metric__value">{{ displayServerEndpoint }}</code>
              </div>
              <div class="status-metric">
                <div class="status-metric__label">连接时间</div>
                <div class="status-metric__value">{{ connectedAtLabel }}</div>
              </div>
              <div class="status-metric">
                <div class="status-metric__label">运行时长</div>
                <div class="status-metric__value">{{ uptimeLabel }}</div>
              </div>
              <div class="status-metric">
                <div class="status-metric__label">客户端版本</div>
                <div class="status-metric__value">{{ frpStatus.client_version || '—' }}</div>
              </div>
            </div>

            <div class="status-gateways">
              <div class="status-gateway" :class="{ 'status-gateway--on': httpGatewayOn }">
                <span class="status-gateway__dot" />
                <div>
                  <div class="status-gateway__title">HTTP 网关</div>
                  <div class="status-gateway__hint">{{ httpGatewayOn ? '转发到 Nginx HTTP' : '等待配置域名' }}</div>
                </div>
              </div>
              <div class="status-gateway" :class="{ 'status-gateway--on': httpsGatewayOn }">
                <span class="status-gateway__dot" />
                <div>
                  <div class="status-gateway__title">HTTPS 网关</div>
                  <div class="status-gateway__hint">{{ httpsGatewayOn ? '转发到 Nginx HTTPS' : '等待配置域名' }}</div>
                </div>
              </div>
            </div>

            <div v-if="domainPreview.length" class="status-domains">
              <div class="status-domains__label">穿透域名</div>
              <div class="status-domains__list">
                <span v-for="domain in domainPreview" :key="domain" class="status-domain-chip">{{ domain }}</span>
                <span v-if="domainCount > domainPreview.length" class="status-domain-more">
                  +{{ domainCount - domainPreview.length }}
                </span>
              </div>
            </div>
          </div>
        </FonuCard>

        <FonuCard subtitle="填写你的 FRP 服务器信息，保存后即可连接。">
          <template #title>
            <n-icon :component="SettingsOutline" class="card-title-icon" />
            <span>连接设置</span>
          </template>

          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">启用内网穿透</div>
            </div>
            <n-switch v-model:value="frpForm.enabled" />
          </div>

          <div class="settings-fields">
            <div class="settings-field">
              <div class="settings-field__label">服务器地址 <span class="required">*</span></div>
              <n-input
                v-model:value="frpForm.server_addr"
                placeholder="127.0.0.1 或 frp.example.com"
                :disabled="!frpForm.enabled"
              />
            </div>
            <div class="settings-field">
              <div class="settings-field__label">服务器端口 <span class="required">*</span></div>
              <n-input-number
                v-model:value="frpForm.server_port"
                :min="1"
                :max="65535"
                class="settings-field__input-full"
                :disabled="!frpForm.enabled"
              />
            </div>
            <div class="settings-field">
              <div class="settings-field__label">
                Token <span class="required">*</span>
                <span v-if="frpForm.has_auth_token" class="token-saved-hint">已保存</span>
              </div>
              <n-input
                v-model:value="frpForm.auth_token"
                type="password"
                show-password-on="click"
                :disabled="!frpForm.enabled"
                :placeholder="frpForm.has_auth_token ? '已保存，留空不修改' : '与 frps 配置一致'"
              />
              <p v-if="frpForm.has_auth_token && !frpForm.auth_token" class="field-hint field-hint--inline">
                Token 已加密保存，出于安全不显示明文；如需更换请直接输入新 Token。
              </p>
            </div>
            <div class="form-switch-row">
              <div class="form-switch-row__text">
                <div class="form-switch-row__label">启用 TLS</div>
                <div class="form-switch-row__hint">使用 TLS 连接到 frps，建议开启</div>
              </div>
              <n-switch v-model:value="frpForm.tls_enabled" :disabled="!frpForm.enabled" />
            </div>
          </div>

          <div class="save-row">
            <n-button type="primary" :loading="saving" @click="save">
              <template #icon><n-icon :component="LinkOutline" /></template>
              {{ frpForm.enabled ? '保存并连接' : '保存并关闭' }}
            </n-button>
          </div>
        </FonuCard>
      </div>

      <FonuCard>
        <template #title>
          <n-icon :component="GlobeOutline" class="card-title-icon card-title-icon--blue" />
          <span>Web 穿透</span>
        </template>
        <template #header>
          <n-button size="small" :loading="syncing" @click="syncDomains">
            <template #icon><n-icon :component="RefreshOutline" /></template>
            同步域名
          </n-button>
        </template>

        <p class="section-desc">
          将当前启用的反向代理域名同步到 FRP Web 网关，所有域名将通过 Fonu Nginx 进行分流。点击「已同步域名」可查看或编辑。
        </p>

        <div class="gateway-grid">
          <div class="gateway-item">
            <span class="gateway-dot" :class="{ 'gateway-dot--on': httpGatewayOn }" />
            <div>
              <div class="gateway-item__title">HTTP 网关</div>
              <div class="gateway-item__state">{{ httpGatewayOn ? '已启用' : '未启用' }}</div>
              <div class="gateway-item__hint">转发到 Fonu Nginx (HTTP)</div>
            </div>
          </div>
          <div class="gateway-item">
            <span class="gateway-dot" :class="{ 'gateway-dot--on': httpsGatewayOn }" />
            <div>
              <div class="gateway-item__title">HTTPS 网关</div>
              <div class="gateway-item__state">{{ httpsGatewayOn ? '已启用' : '未启用' }}</div>
              <div class="gateway-item__hint">转发到 Fonu Nginx (HTTPS)</div>
            </div>
          </div>
          <button type="button" class="gateway-item gateway-item--action" @click="showDomains = true">
            <n-icon :component="LinkOutline" class="gateway-item__icon" />
            <div>
              <div class="gateway-item__title">已同步域名</div>
              <div class="gateway-item__state gateway-item__state--count">{{ domainCount }}</div>
            </div>
            <n-icon :component="ChevronForwardOutline" class="gateway-item__chevron" />
          </button>
        </div>
      </FonuCard>

      <FonuCard>
        <template #title>
          <n-icon :component="OptionsOutline" class="card-title-icon" />
          <span>高级设置</span>
        </template>
        <p class="section-desc">
          在 VPS 部署 frps 后，将下方配置保存为 <code>frps.toml</code> 并启动。配置会随上方连接设置实时更新，保存 Fonu 后同步生效。
        </p>

        <div class="settings-field">
          <div class="settings-field__label">VPS 服务端配置 (frps.toml)</div>
          <n-input type="textarea" :rows="12" readonly :value="frpsConfigPreview" class="frps-config-input" />
          <p class="field-hint field-hint--inline">
            网关端口与 Fonu Nginx 一致（HTTP {{ nginxHttpPort }} / HTTPS {{ nginxHttpsPort }}）。VPS 防火墙需放行
            {{ frpForm.server_port || 7000 }}、{{ nginxHttpPort }}、{{ nginxHttpsPort }}。
          </p>
          <div class="frps-config-actions">
            <n-button secondary @click="copyFrpsConfig">复制配置</n-button>
            <n-button quaternary @click="openLogs">
              <template #icon><n-icon :component="TerminalOutline" /></template>
              查看 frpc 日志
            </n-button>
          </div>
        </div>
      </FonuCard>
    </div>
  </n-spin>

  <n-modal v-model:show="showHelp" preset="card" title="内网穿透使用说明" style="width: min(640px, 92vw)">
    <div class="help-content">
      <p>1. 在 VPS 部署 frps，放行 7000、80、443 端口。</p>
      <p>2. 在本页填写 frps 地址、端口与 Token，保存并连接。</p>
      <p>3. 点击「同步域名」，从反代规则导入域名到 FRP Web 网关。</p>
      <p>4. 在 <router-link :to="{ name: 'ddns' }">DDNS</router-link> 将域名解析到 VPS 公网 IP。</p>
      <p>5. HTTPS 证书推荐继续使用 DNS-01，由 Fonu Nginx 终结 TLS。</p>
    </div>
  </n-modal>

  <n-modal v-model:show="showDomains" preset="card" title="穿透域名" style="width: min(520px, 92vw)">
    <n-input
      v-model:value="frpDomainsText"
      type="textarea"
      :rows="8"
      placeholder="每行一个域名，支持 *.example.com"
      :disabled="!frpForm.enabled"
    />
    <p class="field-hint field-hint--inline">可手动编辑，或点击「同步域名」从已启用的反向代理规则导入。</p>
    <div class="modal-actions">
      <n-button @click="showDomains = false">取消</n-button>
      <n-button type="primary" :loading="saving" :disabled="!frpForm.enabled" @click="saveDomains">保存</n-button>
    </div>
  </n-modal>

  <n-modal v-model:show="showLogs" preset="card" title="frpc 运行日志" style="width: min(860px, 92vw)">
    <n-spin :show="loadingLogs">
      <pre class="log-view">{{ logText || '暂无日志' }}</pre>
    </n-spin>
    <n-button class="modal-action" quaternary :loading="loadingLogs" @click="fetchLogs">刷新</n-button>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NSpin,
  NSwitch,
  useMessage,
} from 'naive-ui'
import {
  AlertCircleOutline,
  CheckmarkCircleOutline,
  ChevronForwardOutline,
  DocumentTextOutline,
  GlobeOutline,
  HelpCircleOutline,
  LinkOutline,
  OptionsOutline,
  PauseCircleOutline,
  RefreshOutline,
  SettingsOutline,
  TerminalOutline,
} from '@vicons/ionicons5'
import { api } from '../api/client'
import type { FRPStatus } from '../api/types'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import { formatDate } from '../utils/format'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const syncing = ref(false)
const loadingLogs = ref(false)
const pageError = ref('')
const showHelp = ref(false)
const showDomains = ref(false)
const showLogs = ref(false)

const frpForm = reactive({
  enabled: false,
  server_addr: '',
  server_port: 7000,
  auth_token: '',
  has_auth_token: false,
  tls_enabled: false,
})
const frpDomainsText = ref('')
const frpsConfigSaved = ref('')
const nginxHttpPort = ref(80)
const nginxHttpsPort = ref(443)
const frpStatus = ref<FRPStatus>({ enabled: false, connected: false, message: '未启用' })
const logLines = ref<string[]>([])

const domainCount = computed(() => parseDomains().length)
const logText = computed(() => logLines.value.join('\n'))
const httpGatewayOn = computed(() => frpForm.enabled && (frpStatus.value.http_gateway_enabled ?? false))
const httpsGatewayOn = computed(() => frpForm.enabled && (frpStatus.value.https_gateway_enabled ?? false))

const statusHeroClass = computed(() => {
  if (!frpForm.enabled) return 'status-hero--off'
  if (frpStatus.value.connected) return 'status-hero--on'
  return 'status-hero--warn'
})

const statusHeroIcon = computed(() => {
  if (!frpForm.enabled) return PauseCircleOutline
  if (frpStatus.value.connected) return CheckmarkCircleOutline
  return AlertCircleOutline
})

const displayServerEndpoint = computed(() => {
  if (frpStatus.value.server_endpoint) return frpStatus.value.server_endpoint
  const addr = frpForm.server_addr.trim()
  if (!addr) return '—'
  const port = frpForm.server_port > 0 ? frpForm.server_port : 7000
  return `${addr}:${port}`
})

const domainPreview = computed(() => parseDomains().slice(0, 3))

const frpsConfigPreview = computed(() =>
  buildFrpsConfig({
    bindPort: frpForm.server_port > 0 ? frpForm.server_port : 7000,
    httpPort: nginxHttpPort.value,
    httpsPort: nginxHttpsPort.value,
    token: resolveFrpsTokenPreview(),
  }),
)

const connectionTitle = computed(() => {
  if (!frpForm.enabled) return '未启用'
  if (frpStatus.value.connected) return '已连接'
  return frpStatus.value.message || '未连接'
})

const connectionSubtitle = computed(() => {
  if (!frpForm.enabled) return '内网穿透未开启'
  if (frpStatus.value.connected) return 'FRP 客户端运行正常'
  return '请检查服务器配置或查看运行日志'
})

const connectedAtLabel = computed(() => {
  if (!frpStatus.value.connected_at) return '—'
  return formatDate(frpStatus.value.connected_at)
})

const uptimeLabel = computed(() => formatUptime(frpStatus.value.uptime_seconds ?? 0))

function formatUptime(seconds: number) {
  if (!seconds || seconds <= 0) return '—'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days} 天 ${hours} 小时`
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`
  return `${minutes} 分钟`
}

function buildFrpsConfig(opts: { bindPort: number; httpPort: number; httpsPort: number; token: string }) {
  const token = opts.token.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
  return `bindAddr = "0.0.0.0"
bindPort = ${opts.bindPort}

vhostHTTPPort = ${opts.httpPort}
vhostHTTPSPort = ${opts.httpsPort}

[auth]
method = "token"
token = "${token}"
`
}

function resolveFrpsTokenPreview() {
  const typed = frpForm.auth_token.trim()
  if (typed) return typed
  if (frpForm.has_auth_token && frpsConfigSaved.value) {
    const match = frpsConfigSaved.value.match(/token\s*=\s*"([^"]*)"/)
    if (match?.[1]) return match[1]
  }
  return '请填写 Token 后保存'
}

function applyFrpsMeta(data: { frps_config?: string; nginx_http_port?: number; nginx_https_port?: number }) {
  frpsConfigSaved.value = data.frps_config ?? ''
  if (data.nginx_http_port && data.nginx_http_port > 0) nginxHttpPort.value = data.nginx_http_port
  if (data.nginx_https_port && data.nginx_https_port > 0) nginxHttpsPort.value = data.nginx_https_port
}

function tokenForSave() {
  const token = frpForm.auth_token.trim()
  if (!token) return undefined
  return token
}

function applyData(data: Awaited<ReturnType<typeof api.getFRP>>) {
  frpForm.enabled = data.enabled
  frpForm.server_addr = data.server_addr ?? ''
  frpForm.server_port = data.server_port > 0 ? data.server_port : 7000
  frpForm.has_auth_token = data.has_auth_token
  frpForm.auth_token = ''
  frpForm.tls_enabled = data.tls_enabled
  frpDomainsText.value = (data.custom_domains ?? []).join('\n')
  applyFrpsMeta(data)
  frpStatus.value = data.status
}

async function load() {
  loading.value = true
  pageError.value = ''
  try {
    applyData(await api.getFRP())
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

function normalizeServerAddr(addr: string) {
  return addr
    .trim()
    .replace(/^https?:\/\//i, '')
    .replace(/[/?#].*$/, '')
    .trim()
}

function parseDomains() {
  return frpDomainsText.value
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

async function ensureDomains() {
  const existing = parseDomains()
  if (existing.length > 0) return existing
  try {
    const result = await api.syncFRPDomains()
    frpDomainsText.value = (result.domains ?? []).join('\n')
    return result.domains ?? []
  } catch {
    return []
  }
}

async function save(): Promise<boolean> {
  saving.value = true
  try {
    frpForm.server_addr = normalizeServerAddr(frpForm.server_addr)
    let domains = parseDomains()
    if (frpForm.enabled && domains.length === 0) {
      domains = await ensureDomains()
    }
    if (frpForm.enabled && domains.length === 0) {
      message.warning('请填写穿透域名，或先在反向代理中启用规则后点击「同步域名」')
      return false
    }
    const result = await api.saveFRP({
      enabled: frpForm.enabled,
      server_addr: frpForm.server_addr,
      server_port: frpForm.server_port || 7000,
      auth_token: tokenForSave(),
      tls_enabled: frpForm.tls_enabled,
      custom_domains: domains,
    })
    frpForm.has_auth_token = result.config.has_auth_token
    frpForm.auth_token = ''
    frpDomainsText.value = (result.config.custom_domains ?? []).join('\n')
    applyFrpsMeta(result)
    frpStatus.value = result.status
    message.success(result.message || (frpForm.enabled ? '已保存并连接' : '已关闭内网穿透'))
    return true
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
    return false
  } finally {
    saving.value = false
  }
}

async function saveDomains() {
  if (await save()) {
    showDomains.value = false
  }
}

async function syncDomains() {
  syncing.value = true
  try {
    const result = await api.syncFRPDomains()
    frpDomainsText.value = (result.domains ?? []).join('\n')
    applyFrpsMeta(result)
    frpStatus.value = result.status
    message.success(result.message || `已同步 ${result.domains.length} 个域名`)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '同步失败')
  } finally {
    syncing.value = false
  }
}

async function fetchLogs() {
  loadingLogs.value = true
  try {
    logLines.value = await api.getFRPLogs(300)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '读取日志失败')
  } finally {
    loadingLogs.value = false
  }
}

async function openLogs() {
  showLogs.value = true
  await fetchLogs()
}

async function copyFrpsConfig() {
  const typed = frpForm.auth_token.trim()
  if (!typed && !frpForm.has_auth_token) {
    message.warning('请先填写 Token 并保存')
    return
  }
  const text =
    !typed && frpForm.has_auth_token && frpsConfigSaved.value
      ? frpsConfigSaved.value
      : frpsConfigPreview.value
  try {
    await navigator.clipboard.writeText(text)
    message.success('已复制 frps 配置')
  } catch {
    message.error('复制失败')
  }
}

onMounted(load)
</script>

<style scoped>
.frp-page {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-5);
}

.top-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.15fr);
  gap: var(--fonu-space-5);
  align-items: stretch;
}

.card-title-icon {
  font-size: 18px;
  margin-right: 8px;
  color: var(--fonu-text-secondary);
  vertical-align: -3px;
}

.card-title-icon--blue {
  color: #3b82f6;
}

.status-card :deep(.fonu-card__body) {
  height: 100%;
}

.status-card__body {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
  height: 100%;
  padding: var(--fonu-space-5);
}

.status-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  padding: 16px;
  border-radius: 12px;
  border: 1px solid var(--fonu-border);
  background: var(--fonu-bg);
}

.status-hero--on {
  border-color: color-mix(in srgb, #10b981 35%, var(--fonu-border));
  background: color-mix(in srgb, #10b981 6%, var(--fonu-bg));
}

.status-hero--warn {
  border-color: color-mix(in srgb, #f59e0b 35%, var(--fonu-border));
  background: color-mix(in srgb, #f59e0b 6%, var(--fonu-bg));
}

.status-hero--off {
  border-color: var(--fonu-border);
  background: color-mix(in srgb, #94a3b8 6%, var(--fonu-bg));
}

.status-hero__main {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.status-hero__icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: var(--fonu-surface);
  color: var(--fonu-text-secondary);
  font-size: 22px;
  flex-shrink: 0;
}

.status-hero--on .status-hero__icon-wrap {
  color: #10b981;
}

.status-hero--warn .status-hero__icon-wrap {
  color: #f59e0b;
}

.status-hero__title {
  font-size: 20px;
  font-weight: 700;
  color: var(--fonu-text);
  line-height: 1.2;
}

.status-hero__sub {
  margin-top: 4px;
  font-size: 13px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

.status-flow {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  padding: 10px 12px;
  border-radius: 10px;
  background: color-mix(in srgb, #3b82f6 5%, var(--fonu-bg));
  border: 1px dashed color-mix(in srgb, #3b82f6 20%, var(--fonu-border));
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.status-flow__node {
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
}

.status-flow__node--accent {
  color: #2563eb;
  border-color: color-mix(in srgb, #3b82f6 30%, var(--fonu-border));
}

.status-flow__arrow {
  color: var(--fonu-text-muted);
}

.status-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.status-tag {
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
}

.status-tag--on {
  color: #047857;
  background: color-mix(in srgb, #10b981 8%, var(--fonu-bg));
  border-color: color-mix(in srgb, #10b981 25%, var(--fonu-border));
}

.status-metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.status-metric {
  padding: 12px;
  border-radius: 10px;
  border: 1px solid var(--fonu-border);
  background: var(--fonu-bg);
}

.status-metric__label {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.status-metric__value {
  margin-top: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
  word-break: break-all;
}

.status-gateways {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.status-gateway {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  border-radius: 10px;
  border: 1px solid var(--fonu-border);
  background: var(--fonu-bg);
}

.status-gateway--on {
  border-color: color-mix(in srgb, #10b981 25%, var(--fonu-border));
}

.status-gateway__dot {
  width: 8px;
  height: 8px;
  margin-top: 5px;
  border-radius: 50%;
  background: #cbd5e1;
  flex-shrink: 0;
}

.status-gateway--on .status-gateway__dot {
  background: #10b981;
}

.status-gateway__title {
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text);
}

.status-gateway__hint {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.status-domains {
  margin-top: auto;
  padding-top: 4px;
}

.status-domains__label {
  font-size: 12px;
  color: var(--fonu-text-muted);
  margin-bottom: 8px;
}

.status-domains__list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.status-domain-chip {
  padding: 4px 10px;
  border-radius: 8px;
  font-size: 12px;
  color: #1d4ed8;
  background: color-mix(in srgb, #3b82f6 8%, var(--fonu-bg));
  border: 1px solid color-mix(in srgb, #3b82f6 18%, var(--fonu-border));
}

.status-domain-more {
  padding: 4px 10px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.section-desc {
  margin: 0 0 var(--fonu-space-4);
  font-size: 13px;
  line-height: 1.6;
  color: var(--fonu-text-muted);
}

.field-hint--inline {
  margin: 8px 0 var(--fonu-space-4);
  font-size: 12px;
  line-height: 1.6;
  color: var(--fonu-text-muted);
}

.settings-fields {
  display: flex;
  flex-direction: column;
  gap: 18px;
  margin-top: var(--fonu-space-4);
}

.settings-field__label {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.token-saved-hint {
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  color: #047857;
  background: color-mix(in srgb, #10b981 10%, var(--fonu-bg));
  border: 1px solid color-mix(in srgb, #10b981 22%, var(--fonu-border));
}

.settings-field__input-full {
  width: 100%;
}

.required {
  color: #ef4444;
}

.form-switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  padding: 12px 14px;
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  background: var(--fonu-bg);
}

.form-switch-row__label {
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.form-switch-row__hint {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.save-row {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--fonu-space-5);
}

.gateway-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--fonu-space-4);
}

.gateway-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-height: 92px;
  padding: 16px;
  border: 1px solid var(--fonu-border);
  border-radius: 12px;
  background: var(--fonu-bg);
  text-align: left;
}

.gateway-item--action {
  align-items: center;
  cursor: pointer;
  transition: border-color 0.15s;
  font: inherit;
  color: inherit;
}

.gateway-item--action:hover {
  border-color: color-mix(in srgb, #3b82f6 35%, var(--fonu-border));
}

.gateway-dot {
  width: 8px;
  height: 8px;
  margin-top: 6px;
  border-radius: 50%;
  background: #cbd5e1;
  flex-shrink: 0;
}

.gateway-dot--on {
  background: #10b981;
}

.gateway-item__icon {
  font-size: 18px;
  color: #3b82f6;
}

.gateway-item__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.gateway-item__state {
  margin-top: 4px;
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.gateway-item__state--count {
  font-size: 24px;
  font-weight: 700;
  color: var(--fonu-text);
}

.gateway-item__hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.gateway-item__chevron {
  margin-left: auto;
  color: var(--fonu-text-muted);
}

.frps-config-input {
  margin-top: 8px;
}

.frps-config-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
}

.advanced-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.advanced-item {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--fonu-border);
  border-radius: 12px;
  background: var(--fonu-bg);
  cursor: pointer;
  text-align: left;
  font: inherit;
  color: inherit;
  transition: border-color 0.15s;
}

.advanced-item:hover {
  border-color: color-mix(in srgb, #10b981 30%, var(--fonu-border));
}

.advanced-item__icon {
  font-size: 18px;
  color: var(--fonu-text-secondary);
}

.advanced-item__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.advanced-item__hint {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

.advanced-item__chevron {
  margin-left: auto;
  color: var(--fonu-text-muted);
}

.help-content p {
  margin: 0 0 10px;
  line-height: 1.7;
  color: var(--fonu-text-secondary);
}

.modal-action {
  margin-top: var(--fonu-space-4);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: var(--fonu-space-4);
}

.log-view {
  max-height: 420px;
  overflow: auto;
  margin: 0;
  padding: 14px;
  border-radius: 10px;
  background: #0f172a;
  color: #e2e8f0;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.domain-list {
  margin: 0;
  padding-left: 18px;
  line-height: 1.8;
}

.empty-domains {
  color: var(--fonu-text-muted);
  font-size: 14px;
}

@media (max-width: 1080px) {
  .top-grid,
  .gateway-grid {
    grid-template-columns: 1fr;
  }
}
</style>
