<template>
  <PageHeader title="系统设置" description="配置界面、域名策略与系统维护" />

  <LoadError v-if="pageError" :message="pageError" @retry="load" />

  <n-spin v-else :show="loading">
    <div class="settings-grid">
      <FonuCard title="外观" subtitle="界面主题与时区">
        <div class="settings-block">
          <div class="settings-block__label">主题模式</div>
          <div class="theme-cards">
            <button
              v-for="item in themeOptions"
              :key="item.value"
              type="button"
              class="theme-card"
              :class="{ 'theme-card--active': form.theme === item.value }"
              @click="selectTheme(item.value)"
            >
              <n-icon :component="item.icon" class="theme-card__icon" />
              <span class="theme-card__text">{{ item.label }}</span>
              <n-icon
                v-if="form.theme === item.value"
                :component="CheckmarkCircle"
                class="theme-card__check"
              />
            </button>
          </div>
          <p class="field-hint">切换后立即预览，保存后写入配置</p>
        </div>

        <div class="settings-block settings-block--spaced">
          <div class="settings-block__label">时区</div>
          <n-select v-model:value="form.timezone" :options="timezoneOptions" />
          <p class="field-hint">影响日志、证书到期时间等显示</p>
        </div>
      </FonuCard>

      <FonuCard title="自动任务" subtitle="DDNS 与证书维护策略">
        <div class="settings-fields">
          <div class="settings-field">
            <div class="settings-field__row">
              <span class="settings-field__label">DDNS 检查周期（分钟）</span>
              <n-input-number v-model:value="ddnsInterval" :min="1" :max="1440" class="settings-field__input" />
            </div>
            <p class="field-hint">每隔 N 分钟检查公网 IP 是否变化并更新 DNS</p>
          </div>
          <div class="settings-field">
            <div class="settings-field__row">
              <span class="settings-field__label">证书续签检查（天）</span>
              <n-input-number v-model:value="certThreshold" :min="1" :max="90" class="settings-field__input" />
            </div>
            <p class="field-hint">当证书剩余有效期少于此天数时自动续签</p>
          </div>
          <div class="settings-field">
            <div class="settings-field__row">
              <span class="settings-field__label">日志保留天数</span>
              <n-input-number v-model:value="logRetention" :min="1" :max="365" class="settings-field__input" />
            </div>
            <p class="field-hint">超过保留天数的日志将被自动清理</p>
          </div>
        </div>
      </FonuCard>

      <FonuCard title="反代安全" subtitle="全局 IP 策略与真实客户端 IP">
        <p class="field-hint">
          IP 黑白名单、仅中国大陆等策略依赖正确识别客户端 IP。若域名经 CDN/多层反代，请在此配置信任代理。
        </p>
        <div class="form-switch-row">
          <div class="form-switch-row__text">
            <div class="form-switch-row__label">启用信任代理</div>
            <div class="form-switch-row__hint">从请求头解析真实客户端 IP</div>
          </div>
          <n-switch v-model:value="trustedProxyEnabled" />
        </div>
        <div v-if="trustedProxyEnabled" class="settings-fields">
          <div class="settings-field">
            <div class="settings-field__label">预设</div>
            <n-select v-model:value="trustedProxyPreset" :options="trustedProxyPresets" />
          </div>
          <div v-if="trustedProxyPreset === 'custom'" class="settings-field">
            <div class="settings-field__label">信任 CIDR（每行一个）</div>
            <n-input v-model:value="trustedProxyCIDRs" type="textarea" :rows="4" placeholder="203.0.113.0/24" />
          </div>
          <div v-if="trustedProxyPreset === 'custom'" class="settings-field">
            <div class="settings-field__label">IP 头字段</div>
            <n-select v-model:value="trustedProxyHeader" :options="trustedProxyHeaders" />
          </div>
        </div>
        <div class="settings-field">
          <div class="settings-field__label">全局 IP 黑名单（每行一个，优先于所有规则）</div>
          <n-input v-model:value="globalIPBlacklistText" type="textarea" :rows="3" placeholder="1.2.3.4&#10;5.6.7.0/24" />
        </div>
        <div class="settings-field china-cidr-field">
          <div class="settings-field__row">
            <span class="settings-field__label">中国 IP 段</span>
            <StatusBadge :kind="chinaCIDRBadgeKind" :text="chinaCIDRBadgeText" />
            <n-button
              size="small"
              :loading="refreshingCIDR || chinaCIDR.updating"
              :disabled="chinaCIDR.updating && !refreshingCIDR"
              @click="refreshChinaCIDR"
            >
              {{ refreshingCIDR || chinaCIDR.updating ? '更新中…' : '立即更新' }}
            </n-button>
          </div>
          <p class="field-hint">
            <template v-if="chinaCIDR.ready">
              上次更新：{{ formatDate(chinaCIDR.updated_at) }}（{{ formatRelativeTime(chinaCIDR.updated_at) }}） · IPv4
              {{ chinaCIDR.entry_count_v4 }} 条 · IPv6 {{ chinaCIDR.entry_count_v6 }} 条
            </template>
            <template v-else-if="chinaCIDR.updated_at">
              上次更新：{{ formatDate(chinaCIDR.updated_at) }} · IPv4 {{ chinaCIDR.entry_count_v4 }} 条 · IPv6
              {{ chinaCIDR.entry_count_v6 }} 条（当前不可用，请重新更新）
            </template>
            <template v-else-if="refreshingCIDR || chinaCIDR.updating">
              正在下载 IPv4 / IPv6 段并校验 Nginx 配置，通常需要 10–60 秒…
            </template>
            <template v-else>尚未下载中国 IP 段，启用「仅中国大陆」前请先更新</template>
          </p>
          <p v-if="chinaCIDR.last_error && !refreshingCIDR && !chinaCIDR.updating" class="field-hint field-hint--error">
            {{ chinaCIDR.last_error }}
          </p>
          <div class="settings-field">
            <div class="settings-field__label">更新间隔（小时）</div>
            <n-input-number v-model:value="chinaCIDRHours" :min="1" :max="168" class="settings-field__input" />
          </div>
        </div>
      </FonuCard>

      <FonuCard title="安全" subtitle="管理员账户" class="settings-card settings-card--security">
        <div class="security-form">
          <div class="settings-fields">
            <div class="settings-field">
              <div class="settings-field__label">当前密码</div>
              <n-input v-model:value="passwordForm.old_password" type="password" show-password-on="click" />
            </div>
            <div class="settings-field">
              <div class="settings-field__label">新密码</div>
              <n-input v-model:value="passwordForm.new_password" type="password" show-password-on="click" />
            </div>
            <div class="settings-field">
              <div class="settings-field__label">确认新密码</div>
              <n-input v-model:value="passwordForm.confirm_password" type="password" show-password-on="click" />
            </div>
          </div>
          <n-button type="primary" class="password-btn" :loading="changingPassword" @click="changePassword">
            修改管理员密码
          </n-button>
        </div>
      </FonuCard>

      <FonuCard title="通知" subtitle="Webhook 告警">
        <div class="settings-field">
          <div class="settings-field__label">Webhook URL</div>
          <n-input v-model:value="form.notify_webhook_url" placeholder="https://example.com/hook" />
          <p class="field-hint">接收告警通知的 HTTP 地址，留空则不发送</p>
        </div>
        <div class="form-switch-list">
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">DDNS 失败通知</div>
              <div class="form-switch-row__hint">当 DDNS 更新失败时发送 Webhook 通知</div>
            </div>
            <n-switch v-model:value="notifyDDNS" />
          </div>
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">证书失败通知</div>
              <div class="form-switch-row__hint">当证书申请或续签失败时发送 Webhook 通知</div>
            </div>
            <n-switch v-model:value="notifyCert" />
          </div>
        </div>
      </FonuCard>

      <FonuCard title="ACME 证书" subtitle="申请与续签所需账户信息">
        <div class="settings-fields">
          <div class="settings-field">
            <div class="settings-field__label">ACME 邮箱</div>
            <n-input v-model:value="acmeEmail" placeholder="admin@example.com" />
            <p class="field-hint">用于 Let's Encrypt / ZeroSSL / Buypass 账户注册</p>
          </div>
          <div class="settings-field">
            <div class="settings-field__label">ZeroSSL API Key</div>
            <n-input
              v-model:value="zerosslApiKey"
              type="password"
              show-password-on="click"
              placeholder="留空表示不使用 ZeroSSL"
            />
            <p class="field-hint">申请 ZeroSSL 证书时必填，可在 ZeroSSL 控制台获取</p>
          </div>
        </div>
      </FonuCard>

      <FonuCard title="数据" subtitle="备份与恢复">
        <div class="data-actions">
          <button type="button" class="data-action-card" :disabled="exporting" @click="exportBackup">
            <n-icon :component="DownloadOutline" class="data-action-card__icon" />
            <div class="data-action-card__title">导出配置</div>
            <div class="data-action-card__hint">下载数据库、证书与 Nginx 配置</div>
          </button>
          <n-upload
            class="data-upload"
            :show-file-list="false"
            accept=".tar.gz,.tgz"
            @change="onRestoreFile"
          >
            <div
              class="data-action-card"
              :class="{ 'data-action-card--disabled': restoring }"
              role="button"
              tabindex="0"
            >
              <n-icon :component="CloudUploadOutline" class="data-action-card__icon" />
              <div class="data-action-card__title">恢复配置</div>
              <div class="data-action-card__hint">从备份文件还原系统配置</div>
            </div>
          </n-upload>
        </div>
        <p class="field-hint data-note">恢复配置将覆盖当前数据，操作前请确保已备份。</p>
      </FonuCard>
    </div>

    <div class="save-bar">
      <n-button type="primary" size="large" :loading="saving" @click="save">保存设置</n-button>
    </div>
  </n-spin>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NIcon,
  NInput,
  NInputNumber,
  NSelect,
  NSpin,
  NSwitch,
  NUpload,
  useMessage,
  type UploadFileInfo,
} from 'naive-ui'
import {
  CheckmarkCircle,
  CloudUploadOutline,
  DesktopOutline,
  DownloadOutline,
  MoonOutline,
  SunnyOutline,
} from '@vicons/ionicons5'
import { api } from '../api/client'
import type { ChinaCIDRStatus } from '../api/types'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useTheme } from '../composables/useTheme'
import { formatDate, formatRelativeTime } from '../utils/format'
import type { StatusKind } from '../utils/status'

const message = useMessage()
const { setThemeMode } = useTheme()
const saving = ref(false)
const loading = ref(false)
const pageError = ref('')
const changingPassword = ref(false)
const exporting = ref(false)
const restoring = ref(false)

const form = reactive({
  theme: 'system',
  timezone: 'Asia/Shanghai',
  notify_webhook_url: '',
})

const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' })
const notifyDDNS = ref(true)
const notifyCert = ref(true)
const ddnsInterval = ref(5)
const certThreshold = ref(30)
const logRetention = ref(30)
const acmeEmail = ref('')
const zerosslApiKey = ref('')
const trustedProxyEnabled = ref(false)
const trustedProxyPreset = ref('cloudflare')
const trustedProxyCIDRs = ref('')
const trustedProxyHeader = ref('X-Forwarded-For')
const globalIPBlacklistText = ref('')
const chinaCIDRHours = ref(24)
const refreshingCIDR = ref(false)
const chinaCIDR = ref<ChinaCIDRStatus>({ entry_count_v4: 0, entry_count_v6: 0 })
let chinaCIDRPollTimer: ReturnType<typeof setInterval> | undefined

const chinaCIDRBadgeKind = computed((): StatusKind => {
  if (refreshingCIDR.value || chinaCIDR.value.updating) return 'warning'
  if (chinaCIDR.value.ready) return 'success'
  if (chinaCIDR.value.last_error) return 'error'
  return 'disabled'
})

const chinaCIDRBadgeText = computed(() => {
  if (refreshingCIDR.value || chinaCIDR.value.updating) return '更新中'
  if (chinaCIDR.value.ready) return '已就绪'
  if (chinaCIDR.value.last_error) return '更新失败'
  return '未下载'
})

const trustedProxyPresets = [
  { label: 'Cloudflare', value: 'cloudflare' },
  { label: '自定义', value: 'custom' },
]
const trustedProxyHeaders = [
  { label: 'X-Forwarded-For', value: 'X-Forwarded-For' },
  { label: 'X-Real-IP', value: 'X-Real-IP' },
  { label: 'CF-Connecting-IP', value: 'CF-Connecting-IP' },
]

const themeOptions = [
  { value: 'system', label: '跟随系统', icon: DesktopOutline },
  { value: 'light', label: '浅色', icon: SunnyOutline },
  { value: 'dark', label: '深色', icon: MoonOutline },
]

const timezoneOptions = [
  { label: 'Asia/Shanghai (UTC+8)', value: 'Asia/Shanghai' },
  { label: 'Asia/Tokyo (UTC+9)', value: 'Asia/Tokyo' },
  { label: 'UTC', value: 'UTC' },
  { label: 'America/New_York (UTC-5/-4)', value: 'America/New_York' },
  { label: 'Europe/London (UTC+0/+1)', value: 'Europe/London' },
]

function selectTheme(mode: string) {
  form.theme = mode
  setThemeMode(mode)
}

function applySettingsToForm(settings: Record<string, string>) {
  Object.assign(form, {
    theme: settings.theme ?? 'system',
    timezone: settings.timezone ?? 'Asia/Shanghai',
    notify_webhook_url: settings.notify_webhook_url ?? '',
  })
  notifyDDNS.value = settings.notify_on_ddns_error !== '0' && settings.notify_on_ddns_error !== 'false'
  notifyCert.value = settings.notify_on_cert_error !== '0' && settings.notify_on_cert_error !== 'false'
  ddnsInterval.value = Number(settings.ddns_check_interval_minutes ?? 5)
  certThreshold.value = Number(settings.cert_renew_threshold_days ?? 30)
  logRetention.value = Number(settings.log_retention_days ?? 30)
  acmeEmail.value = settings.acme_email ?? ''
  zerosslApiKey.value = settings.zerossl_api_key ?? ''
  chinaCIDRHours.value = Number(settings.china_cidr_update_interval_hours ?? 24)
  try {
    const tp = JSON.parse(settings.trusted_proxy_json || '{}') as {
      enabled?: boolean
      preset?: string
      cidrs?: string[]
      header?: string
    }
    trustedProxyEnabled.value = tp.enabled ?? false
    trustedProxyPreset.value = tp.preset || 'cloudflare'
    trustedProxyCIDRs.value = (tp.cidrs ?? []).join('\n')
    trustedProxyHeader.value = tp.header || 'X-Forwarded-For'
  } catch {
    trustedProxyEnabled.value = false
  }
  try {
    const bl = JSON.parse(settings.global_ip_blacklist || '[]') as string[]
    globalIPBlacklistText.value = bl.join('\n')
  } catch {
    globalIPBlacklistText.value = ''
  }
}

function buildTrustedProxyJSON() {
  const cidrs = trustedProxyCIDRs.value
    .split('\n')
    .map((l) => l.trim())
    .filter(Boolean)
  return JSON.stringify({
    enabled: trustedProxyEnabled.value,
    preset: trustedProxyPreset.value,
    cidrs,
    header: trustedProxyHeader.value,
    recursive: trustedProxyHeader.value === 'X-Forwarded-For',
  })
}

function buildGlobalBlacklistJSON() {
  const list = globalIPBlacklistText.value
    .split('\n')
    .map((l) => l.trim())
    .filter(Boolean)
  return JSON.stringify(list)
}

async function loadChinaCIDRStatus() {
  try {
    chinaCIDR.value = await api.getChinaCIDRStatus()
    if (chinaCIDR.value.updating) {
      startChinaCIDRPoll()
    } else {
      stopChinaCIDRPoll()
    }
  } catch {
    // optional on load
  }
}

function stopChinaCIDRPoll() {
  if (chinaCIDRPollTimer) {
    clearInterval(chinaCIDRPollTimer)
    chinaCIDRPollTimer = undefined
  }
}

function startChinaCIDRPoll() {
  if (chinaCIDRPollTimer) return
  chinaCIDRPollTimer = setInterval(() => void loadChinaCIDRStatus(), 2000)
}

async function refreshChinaCIDR() {
  refreshingCIDR.value = true
  stopChinaCIDRPoll()
  try {
    chinaCIDR.value = await api.refreshChinaCIDR()
    if (chinaCIDR.value.ready) {
      message.success(
        `中国 IP 段已更新：IPv4 ${chinaCIDR.value.entry_count_v4} 条，IPv6 ${chinaCIDR.value.entry_count_v6} 条`,
      )
    } else {
      message.warning('更新完成，但数据尚未就绪，请查看下方错误信息')
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
    await loadChinaCIDRStatus()
  } finally {
    refreshingCIDR.value = false
  }
}

onBeforeUnmount(() => {
  stopChinaCIDRPoll()
})

function buildSavePayload() {
  return {
    theme: form.theme,
    timezone: form.timezone,
    notify_webhook_url: form.notify_webhook_url,
    notify_on_ddns_error: notifyDDNS.value ? '1' : '0',
    notify_on_cert_error: notifyCert.value ? '1' : '0',
    ddns_check_interval_minutes: String(ddnsInterval.value),
    cert_renew_threshold_days: String(certThreshold.value),
    log_retention_days: String(logRetention.value),
    acme_email: acmeEmail.value.trim(),
    zerossl_api_key: zerosslApiKey.value.trim(),
    trusted_proxy_json: buildTrustedProxyJSON(),
    global_ip_blacklist: buildGlobalBlacklistJSON(),
    china_cidr_update_interval_hours: String(chinaCIDRHours.value),
  }
}

async function load() {
  loading.value = true
  pageError.value = ''
  try {
    applySettingsToForm(await api.getSettings())
    await loadChinaCIDRStatus()
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const saved = await api.saveSettings(buildSavePayload())
    const savedTheme = saved.theme ?? form.theme
    form.theme = savedTheme
    setThemeMode(savedTheme)
    message.success('设置已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function changePassword() {
  if (!passwordForm.old_password || !passwordForm.new_password) {
    message.warning('请填写当前密码和新密码')
    return
  }
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    message.warning('两次输入的新密码不一致')
    return
  }
  changingPassword.value = true
  try {
    await api.changePassword(passwordForm.old_password, passwordForm.new_password)
    passwordForm.old_password = ''
    passwordForm.new_password = ''
    passwordForm.confirm_password = ''
    message.success('密码已更新')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新密码失败')
  } finally {
    changingPassword.value = false
  }
}

async function exportBackup() {
  exporting.value = true
  try {
    await api.exportBackup()
    message.success('备份已开始下载')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

async function onRestoreFile({ file }: { file: UploadFileInfo }) {
  if (!file.file) return
  restoring.value = true
  try {
    const result = await api.restoreBackup(file.file)
    message.success(result.message)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '恢复失败')
  } finally {
    restoring.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--fonu-space-5);
  align-items: stretch;
}

.settings-grid :deep(.fonu-card) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.settings-grid :deep(.fonu-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.settings-block--spaced {
  margin-top: var(--fonu-space-5);
}

.settings-block__label,
.settings-field__label {
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.settings-block__label,
.settings-field > .settings-field__label {
  margin-bottom: 8px;
}

.theme-cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--fonu-space-3);
}

.theme-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 80px;
  padding: 12px 8px;
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  background: var(--fonu-bg);
  cursor: pointer;
  transition: border-color 0.15s;
}

.theme-card:hover {
  border-color: color-mix(in srgb, #10b981 40%, var(--fonu-border));
}

.theme-card--active {
  border-color: #10b981;
  background: color-mix(in srgb, #10b981 6%, var(--fonu-bg));
}

.theme-card__icon {
  font-size: 20px;
  color: var(--fonu-text-secondary);
}

.theme-card--active .theme-card__icon {
  color: #10b981;
}

.theme-card__text {
  font-size: 13px;
  font-weight: 500;
}

.theme-card__check {
  position: absolute;
  top: 8px;
  right: 8px;
  font-size: 16px;
  color: #10b981;
}

.field-hint--error {
  color: var(--n-error-color, #e11d48);
}

.field-hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

.settings-fields {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.settings-field {
  min-width: 0;
}

.settings-field__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.settings-field__row .settings-field__label {
  margin-bottom: 0;
  flex: 1;
  min-width: 0;
}

.settings-field__input {
  width: 120px;
  flex-shrink: 0;
}

.security-form {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
}

.security-form .settings-fields {
  width: 100%;
}

.security-form .settings-field {
  width: 100%;
}

.security-form :deep(.n-input) {
  width: 100%;
}

.password-btn {
  margin-top: var(--fonu-space-4);
  align-self: flex-start;
  flex-shrink: 0;
}

.form-switch-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: var(--fonu-space-4);
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
  line-height: 1.5;
}

.data-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--fonu-space-3);
  align-items: stretch;
}

.data-upload {
  display: flex;
  min-width: 0;
}

.data-upload :deep(.n-upload) {
  width: 100%;
  display: flex;
}

.data-upload :deep(.n-upload-trigger) {
  width: 100%;
  display: flex;
}

.data-action-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: flex-start;
  gap: 4px;
  width: 100%;
  min-height: 96px;
  height: 100%;
  padding: 14px;
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  background: var(--fonu-bg);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s;
  box-sizing: border-box;
  font: inherit;
  color: inherit;
}

.data-action-card:hover:not(:disabled):not(.data-action-card--disabled) {
  border-color: color-mix(in srgb, #10b981 35%, var(--fonu-border));
}

.data-action-card:disabled,
.data-action-card--disabled {
  opacity: 0.6;
  cursor: not-allowed;
  pointer-events: none;
}

.data-action-card__icon {
  font-size: 18px;
  color: #10b981;
}

.data-action-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.data-action-card__hint {
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.45;
}

.data-note {
  margin-top: auto;
  padding-top: var(--fonu-space-3);
}

.save-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--fonu-space-5);
  padding-bottom: var(--fonu-space-2);
}

@media (max-width: 960px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }

  .theme-cards {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .data-actions {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .theme-cards {
    grid-template-columns: 1fr;
  }

  .settings-field__row {
    flex-direction: column;
    align-items: stretch;
  }

  .settings-field__input {
    width: 100%;
  }
}
</style>
