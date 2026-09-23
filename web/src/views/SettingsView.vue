<template>
  <PageHeader title="系统设置" description="配置界面、域名策略与系统维护" />

  <LoadError v-if="pageError" :message="pageError" @retry="load" />

  <n-spin v-else :show="loading">
    <n-tabs v-model:value="settingsTab" type="line" animated class="settings-tabs">
      <n-tab-pane name="general" tab="常规">
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

      <FonuCard title="ACME 证书" subtitle="申请与续签所需账户信息">
        <div class="settings-fields">
          <div class="settings-field">
            <div class="settings-field__label">ACME 邮箱</div>
            <n-input v-model:value="acmeEmail" placeholder="admin@example.com" />
            <p class="field-hint">用于 Let's Encrypt / ZeroSSL / Buypass 账户注册</p>
          </div>
          <ConfiguredSecretField
            v-model="zerosslApiKey"
            label="ZeroSSL API Key"
            :configured="zerosslHasKey"
            placeholder="申请 ZeroSSL 证书时填写"
          />
          <p class="field-hint zerossl-key-hint">申请 ZeroSSL 证书时必填，可在 ZeroSSL 控制台获取</p>
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
      </n-tab-pane>

      <n-tab-pane name="notify" tab="通知">
        <div class="settings-grid settings-grid--single">
      <FonuCard title="通知" subtitle="告警通道与事件（仅可启用一种方式）">
        <div class="security-form notify-form">
          <div class="settings-field">
            <div class="settings-field__label">通知方式</div>
            <n-radio-group v-model:value="notifyType" class="notify-type-group">
              <n-radio v-for="item in NOTIFY_TYPE_OPTIONS" :key="item.value || 'off'" :value="item.value">
                {{ item.label }}
              </n-radio>
            </n-radio-group>
          </div>

          <div v-if="notifyType === 'email'" class="settings-fields settings-fields--compact">
            <div class="settings-field notify-smtp-endpoint">
              <div class="notify-smtp-endpoint__col">
                <div class="settings-field__label">SMTP 主机</div>
                <n-input v-model:value="notifyEmail.host" placeholder="smtp.example.com" />
              </div>
              <div class="notify-smtp-endpoint__col notify-smtp-endpoint__col--port">
                <div class="settings-field__label">端口</div>
                <n-input-number v-model:value="notifyEmail.port" :min="1" :max="65535" />
              </div>
            </div>
            <div class="settings-field">
              <div class="settings-field__label">用户名</div>
              <n-input v-model:value="notifyEmail.username" placeholder="user@example.com" />
            </div>
            <ConfiguredSecretField
              v-model="notifySMTPPassword"
              label="密码"
              :configured="notifyEmail.has_password"
              placeholder="SMTP 密码"
            />
            <div class="settings-field">
              <div class="settings-field__label">发件人</div>
              <n-input v-model:value="notifyEmail.from" placeholder="fonu@example.com" />
            </div>
            <div class="settings-field">
              <div class="settings-field__label">收件人（每行一个）</div>
              <n-input v-model:value="notifyEmailToText" type="textarea" :rows="2" placeholder="admin@example.com" />
            </div>
            <div class="form-switch-row">
              <div class="form-switch-row__text">
                <div class="form-switch-row__label">启用 TLS</div>
              </div>
              <n-switch v-model:value="notifyEmail.tls" />
            </div>
          </div>

          <div v-else-if="notifyType === 'webhook'" class="settings-fields settings-fields--compact">
            <div class="settings-field">
              <div class="settings-field__label">Webhook 预设</div>
              <n-select v-model:value="notifyWebhook.provider" :options="WEBHOOK_PROVIDER_OPTIONS" />
            </div>
            <template v-if="notifyWebhook.provider === 'bark'">
              <div class="settings-field">
                <div class="settings-field__label">Bark 服务地址</div>
                <n-input v-model:value="notifyWebhook.server" :placeholder="DEFAULT_BARK_SERVER" />
              </div>
              <ConfiguredSecretField
                v-model="notifyWebhookKey"
                label="Device Key"
                :configured="notifyWebhook.has_secret"
                placeholder="Bark Device Key"
              />
            </template>
            <template v-else-if="notifyWebhook.provider === 'ntfy'">
              <div class="settings-field">
                <div class="settings-field__label">ntfy 服务地址</div>
                <n-input v-model:value="notifyWebhook.server" :placeholder="DEFAULT_NTFY_SERVER" />
              </div>
              <div class="settings-field">
                <div class="settings-field__label">Topic</div>
                <n-input v-model:value="notifyWebhook.topic" placeholder="fonu-alerts" />
              </div>
              <ConfiguredSecretField
                v-model="notifyWebhookKey"
                label="访问令牌（可选）"
                :configured="notifyWebhook.has_secret"
                placeholder="Bearer Token"
              />
            </template>
            <template v-else-if="notifyWebhook.provider === 'gotify'">
              <div class="settings-field">
                <div class="settings-field__label">Gotify 服务地址</div>
                <n-input v-model:value="notifyWebhook.server" placeholder="https://gotify.example.com" />
              </div>
              <ConfiguredSecretField
                v-model="notifyWebhookKey"
                label="App Token"
                :configured="notifyWebhook.has_secret"
                placeholder="Gotify App Token"
              />
            </template>
            <template v-else>
              <div class="settings-field">
                <div class="settings-field__label">Webhook URL</div>
                <n-input v-model:value="notifyWebhook.url" placeholder="https://example.com/hook" />
                <p class="field-hint">将以 JSON 发送：event、category、icon、title、message、formatted、time</p>
              </div>
            </template>
          </div>

          <div v-else-if="notifyType === 'telegram'" class="settings-fields settings-fields--compact">
            <ConfiguredSecretField
              v-model="notifyTelegramToken"
              label="Bot Token"
              :configured="notifyTelegram.has_bot_token"
              placeholder="123456:ABC-DEF"
            />
            <div class="settings-field">
              <div class="settings-field__label">Chat ID</div>
              <n-input v-model:value="notifyTelegram.chat_id" placeholder="-1001234567890" />
            </div>
            <div class="settings-field">
              <div class="settings-field__label">代理地址（可选）</div>
              <n-input v-model:value="notifyTelegram.proxy_url" placeholder="http://127.0.0.1:7890" />
              <p class="field-hint">无法直连时填写，如 127.0.0.1:7890 或 http://127.0.0.1:7890（支持 socks5）</p>
            </div>
          </div>

          <div v-if="notifyType" class="notify-events">
            <div class="notify-events__title">告警事件</div>
            <div v-for="group in NOTIFY_EVENT_GROUPS" :key="group.title" class="notify-event-group">
              <div class="notify-event-group__title">{{ group.title }}</div>
              <div class="form-switch-list">
                <template v-for="event in group.events" :key="event.key">
                  <div class="form-switch-row">
                    <div class="form-switch-row__text">
                      <div class="form-switch-row__label">{{ event.label }}</div>
                      <div v-if="event.hint" class="form-switch-row__hint">{{ event.hint }}</div>
                    </div>
                    <n-switch v-model:value="notifyEventFlags[event.key]" />
                  </div>
                  <div
                    v-if="event.hasThreshold && notifyEventFlags[event.key]"
                    class="settings-fields settings-fields--compact notify-threshold-fields"
                  >
                    <template v-if="event.thresholdKey === 'ip_frequent'">
                      <div class="settings-field">
                        <label class="settings-field__label">访问次数阈值</label>
                        <n-input-number v-model:value="notifyIPFrequentThreshold" :min="10" :max="10000" class="settings-field__input" />
                      </div>
                      <div class="settings-field">
                        <label class="settings-field__label">统计窗口（秒）</label>
                        <n-input-number v-model:value="notifyIPFrequentWindowSec" :min="10" :max="3600" class="settings-field__input" />
                      </div>
                    </template>
                    <template v-else-if="event.thresholdKey === 'login_failure'">
                      <div class="settings-field">
                        <label class="settings-field__label">失败次数阈值</label>
                        <n-input-number v-model:value="notifyLoginFailureThreshold" :min="3" :max="100" class="settings-field__input" />
                      </div>
                      <div class="settings-field">
                        <label class="settings-field__label">统计窗口（秒）</label>
                        <n-input-number v-model:value="notifyLoginFailureWindowSec" :min="10" :max="3600" class="settings-field__input" />
                      </div>
                    </template>
                  </div>
                </template>
              </div>
            </div>
          </div>

          <div v-if="notifyType" class="notify-test-row">
            <n-button :loading="notifyTesting" @click="testNotify">测试通知</n-button>
            <p class="field-hint">使用当前表单配置发送测试消息，无需先保存</p>
          </div>
        </div>
      </FonuCard>
        </div>
      </n-tab-pane>

      <n-tab-pane name="security" tab="安全">
        <div class="settings-grid">
      <FonuCard title="反代安全" subtitle="全局 IP 策略与真实客户端 IP">
        <div class="security-form proxy-security-form">
          <p class="field-hint proxy-security-form__intro">
            IP 黑白名单、仅中国大陆等策略依赖正确识别客户端 IP。若域名经 CDN/多层反代，请在此配置信任代理。
          </p>
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">启用信任代理</div>
              <div class="form-switch-row__hint">从请求头解析真实客户端 IP</div>
            </div>
            <n-switch v-model:value="trustedProxyEnabled" />
          </div>
          <div v-if="trustedProxyEnabled" class="settings-fields settings-fields--compact">
            <div class="settings-field">
              <div class="settings-field__label">预设</div>
              <n-select v-model:value="trustedProxyPreset" :options="trustedProxyPresets" />
            </div>
            <template v-if="trustedProxyPreset === 'custom'">
              <div class="settings-field">
                <div class="settings-field__label">信任 CIDR（每行一个）</div>
                <n-input v-model:value="trustedProxyCIDRs" type="textarea" :rows="3" placeholder="203.0.113.0/24" />
              </div>
              <div class="settings-field">
                <div class="settings-field__label">IP 头字段</div>
                <n-select v-model:value="trustedProxyHeader" :options="trustedProxyHeaders" />
              </div>
            </template>
          </div>
          <div class="ip-policy-grid">
            <div class="settings-field ip-policy-field">
              <div class="settings-field__label">全局 IP 黑名单</div>
              <div class="ip-policy-field__input">
                <n-input
                  v-model:value="globalIPBlacklistText"
                  type="textarea"
                  :rows="6"
                  placeholder="1.2.3.4&#10;5.6.7.0/24"
                />
              </div>
              <p class="field-hint">每行一个，优先于所有规则拦截</p>
            </div>
            <div class="settings-field ip-policy-field">
              <div class="settings-field__label">全局 IP 白名单</div>
              <div class="ip-policy-field__input">
                <n-input
                  v-model:value="globalIPWhitelistText"
                  type="textarea"
                  :rows="6"
                  placeholder="203.0.113.10&#10;203.0.113.0/24"
                />
              </div>
              <p class="field-hint">内网段已默认放行（已预填），可在下方追加额外 IP，每行一个</p>
            </div>
          </div>
          <div class="china-cidr-panel">
            <div class="settings-field__row china-cidr-panel__head">
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
          </div>
          <div class="settings-field">
            <div class="settings-field__row">
              <span class="settings-field__label">更新间隔（小时）</span>
              <n-input-number v-model:value="chinaCIDRHours" :min="1" :max="168" class="settings-field__input" />
            </div>
            <p class="field-hint">自动检查并更新中国 IP 段库的间隔</p>
          </div>
        </div>
      </FonuCard>

      <FonuCard title="管理员账户" subtitle="修改登录密码" class="settings-card settings-card--security">
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
        </div>
      </n-tab-pane>

      <n-tab-pane name="advanced" tab="高级">
        <div class="settings-grid settings-grid--single">
      <FonuCard title="Nginx 全局配置" subtitle="http 块内自定义片段">
        <p class="field-hint">
          仅作用于 <code>http {}</code> 内部、各反代 <code>server {}</code> 之前。单条规则的 server 块请在反代详情 → Nginx 中编辑。
        </p>
        <details class="nginx-framework">
          <summary>查看自动生成的框架说明</summary>
          <NginxCodeEditor
            :model-value="globalNginxFramework"
            readonly
            min-height="240px"
            class="nginx-framework__editor"
          />
        </details>
        <div class="log-panel-head global-nginx-head">
          <div class="nginx-mode-toggle">
            <n-button
              size="tiny"
              quaternary
              :type="globalNginxEditMode === 'auto' ? 'primary' : 'default'"
              @click="setGlobalNginxEditMode('auto')"
            >
              自动生成
            </n-button>
            <n-button
              size="tiny"
              quaternary
              :type="globalNginxEditMode === 'custom' ? 'primary' : 'default'"
              @click="setGlobalNginxEditMode('custom')"
            >
              手动编辑
            </n-button>
          </div>
          <div class="log-panel-actions">
            <n-button
              v-if="globalNginxEditMode === 'custom'"
              size="tiny"
              quaternary
              type="primary"
              :loading="globalNginxSaving"
              :disabled="!globalNginxDirty"
              @click="saveGlobalNginx"
            >
              保存
            </n-button>
            <n-button
              v-if="globalNginxEditMode === 'custom' && globalNginxBackups.length > 0"
              size="tiny"
              quaternary
              :loading="globalNginxSaving"
              @click="rollbackGlobalNginx()"
            >
              回滚
            </n-button>
            <n-button
              v-if="globalNginxEditMode === 'custom' && globalNginxServerMode === 'custom'"
              size="tiny"
              quaternary
              :loading="globalNginxSaving"
              @click="resetGlobalNginxAuto"
            >
              恢复自动生成
            </n-button>
          </div>
        </div>
        <NginxCodeEditor
          v-model="globalNginxDraft"
          class="nginx-editor global-nginx-editor"
          :readonly="globalNginxEditMode === 'auto'"
          min-height="280px"
          placeholder="# 在此添加 http 块内的自定义指令&#10;# 例如：gzip on;"
          @update:model-value="onGlobalNginxDraftInput"
        />
        <p class="global-nginx-upload-hint field-hint">
          上传大小默认 50M。如需针对单条规则单独设置，请到对应反代 → 编辑 → Nginx 中修改
          <code>client_max_body_size</code>。
        </p>
        <n-alert type="warning" :bordered="false" class="global-nginx-warn">
          小白勿碰。若保存后 Nginx 启动失败，请点「恢复自动生成」或「回滚」还原配置。
        </n-alert>
      </FonuCard>
        </div>
      </n-tab-pane>
    </n-tabs>

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
  NRadio,
  NRadioGroup,
  NSelect,
  NSpin,
  NSwitch,
  NTabPane,
  NTabs,
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
import type {
  ChinaCIDRStatus,
  GlobalNginxView,
  NotifyEmailConfig,
  NotifyTelegramConfig,
  NotifyTestPayload,
  NotifyType,
  NotifyWebhookConfig,
  WebhookProvider,
} from '../api/types'
import ConfiguredSecretField from '../components/ConfiguredSecretField.vue'
import FonuCard from '../components/FonuCard.vue'
import NginxCodeEditor from '../components/NginxCodeEditor.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { DEFAULT_PRIVATE_CIDRS, mergeIPList, stripDefaultIPList } from '../constants/ipPolicy'
import {
  DEFAULT_LOGIN_FAILURE_THRESHOLD,
  DEFAULT_LOGIN_FAILURE_WINDOW_SEC,
  NOTIFY_EVENT_GROUPS,
} from '../constants/notifyEvents'
import {
  DEFAULT_BARK_SERVER,
  DEFAULT_NTFY_SERVER,
  MASKED_SECRET,
  NOTIFY_TYPE_OPTIONS,
  WEBHOOK_PROVIDER_OPTIONS,
} from '../constants/notifyProviders'
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
})

const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' })
const settingsTab = ref('general')
const notifyType = ref<NotifyType>('')
const notifyEventFlags = reactive<Record<string, boolean>>({
  on_ddns_ip_change: true,
  on_ddns_failure: true,
  on_cert_expiry: true,
  on_cert_renew_success: true,
  on_cert_renew_failure: true,
  on_ip_frequent_access: false,
  on_login_failure: false,
  on_nginx_reload_failure: true,
})
const notifyIPFrequentThreshold = ref(100)
const notifyIPFrequentWindowSec = ref(60)
const notifyLoginFailureThreshold = ref(DEFAULT_LOGIN_FAILURE_THRESHOLD)
const notifyLoginFailureWindowSec = ref(DEFAULT_LOGIN_FAILURE_WINDOW_SEC)
const notifyTesting = ref(false)
const notifySMTPPassword = ref('')
const notifyWebhookKey = ref('')
const notifyTelegramToken = ref('')
const notifyEmailToText = ref('')
const notifyEmail = reactive({
  host: '',
  port: 587,
  username: '',
  from: '',
  tls: true,
  has_password: false,
})
const notifyWebhook = reactive({
  provider: 'bark' as WebhookProvider,
  server: DEFAULT_BARK_SERVER,
  topic: '',
  url: '',
  has_secret: false,
})
const notifyTelegram = reactive({
  chat_id: '',
  proxy_url: '',
  has_bot_token: false,
})
const ddnsInterval = ref(5)
const certThreshold = ref(30)
const logRetention = ref(30)
const acmeEmail = ref('')
const zerosslApiKey = ref('')
const zerosslHasKey = ref(false)
const trustedProxyEnabled = ref(false)
const trustedProxyPreset = ref('cloudflare')
const trustedProxyCIDRs = ref('')
const trustedProxyHeader = ref('X-Forwarded-For')
const globalIPBlacklistText = ref('')
const globalIPWhitelistText = ref('')
const chinaCIDRHours = ref(24)
const refreshingCIDR = ref(false)
const chinaCIDR = ref<ChinaCIDRStatus>({ entry_count_v4: 0, entry_count_v6: 0 })
let chinaCIDRPollTimer: ReturnType<typeof setInterval> | undefined

const globalNginxFramework = ref('')
const globalNginxGeneratedSnippet = ref('')
const globalNginxDraft = ref('')
const globalNginxEditMode = ref<'auto' | 'custom'>('auto')
const globalNginxServerMode = ref<'auto' | 'custom'>('auto')
const globalNginxDirty = ref(false)
const globalNginxSaving = ref(false)
const globalNginxBackups = ref<{ name: string; created_at: string }[]>([])

const globalNginxCustomHint = `# 在此添加 http 块内的自定义指令
# 上传大小限制（全局默认 50M，单条规则可在 server 块覆盖）
client_max_body_size 50m;
# 例如：gzip on;
`

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
  })
  applyNotifySettings(settings)
  ddnsInterval.value = Number(settings.ddns_check_interval_minutes ?? 5)
  certThreshold.value = Number(settings.cert_renew_threshold_days ?? 30)
  logRetention.value = Number(settings.log_retention_days ?? 30)
  acmeEmail.value = settings.acme_email ?? ''
  zerosslHasKey.value = Boolean(settings.zerossl_api_key?.trim())
  zerosslApiKey.value = ''
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
  try {
    const wl = JSON.parse(settings.global_ip_whitelist || '[]') as string[]
    globalIPWhitelistText.value = mergeIPList(DEFAULT_PRIVATE_CIDRS, wl).join('\n')
  } catch {
    globalIPWhitelistText.value = DEFAULT_PRIVATE_CIDRS.join('\n')
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

function buildGlobalIPListJSON(text: string) {
  const list = text
    .split('\n')
    .map((l) => l.trim())
    .filter(Boolean)
  return JSON.stringify(list)
}

function buildGlobalWhitelistJSON() {
  const list = globalIPWhitelistText.value
    .split('\n')
    .map((l) => l.trim())
    .filter(Boolean)
  return JSON.stringify(stripDefaultIPList(DEFAULT_PRIVATE_CIDRS, list))
}

function applyGlobalNginxView(view: GlobalNginxView) {
  globalNginxFramework.value = view.generated_framework
  globalNginxGeneratedSnippet.value = view.generated_snippet || globalNginxCustomHint
  globalNginxServerMode.value = view.mode
  globalNginxEditMode.value = view.mode
  globalNginxBackups.value = view.backups ?? []
  globalNginxDraft.value =
    view.mode === 'auto' ? globalNginxGeneratedSnippet.value : view.content || globalNginxCustomHint
  globalNginxDirty.value = false
}

async function loadGlobalNginx() {
  try {
    applyGlobalNginxView(await api.getGlobalNginx())
  } catch {
    // optional on load
  }
}

function setGlobalNginxEditMode(mode: 'auto' | 'custom') {
  if (mode === globalNginxEditMode.value) return
  if (mode === 'custom') {
    if (!globalNginxDraft.value.trim() || globalNginxDraft.value === globalNginxGeneratedSnippet.value) {
      globalNginxDraft.value = globalNginxCustomHint
      globalNginxDirty.value = false
    } else {
      globalNginxDirty.value = globalNginxServerMode.value !== 'custom'
    }
  } else {
    globalNginxDraft.value = globalNginxGeneratedSnippet.value
    globalNginxDirty.value = false
  }
  globalNginxEditMode.value = mode
}

function onGlobalNginxDraftInput() {
  if (globalNginxEditMode.value === 'custom') {
    globalNginxDirty.value = true
  }
}

async function saveGlobalNginx() {
  globalNginxSaving.value = true
  try {
    applyGlobalNginxView(await api.saveGlobalNginx({
      mode: 'custom',
      content: globalNginxDraft.value,
    }))
    message.success('全局 Nginx 配置已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    globalNginxSaving.value = false
  }
}

async function rollbackGlobalNginx(backup?: string) {
  globalNginxSaving.value = true
  try {
    applyGlobalNginxView(await api.rollbackGlobalNginx(backup))
    message.success('已回滚到上一版本')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '回滚失败')
  } finally {
    globalNginxSaving.value = false
  }
}

async function resetGlobalNginxAuto() {
  globalNginxSaving.value = true
  try {
    applyGlobalNginxView(await api.saveGlobalNginx({ mode: 'auto' }))
    message.success('已恢复自动生成')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '操作失败')
  } finally {
    globalNginxSaving.value = false
  }
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

function applyNotifySettings(settings: Record<string, string>) {
  notifyType.value = (settings.notify_type as NotifyType) || ''
  notifySMTPPassword.value = ''
  notifyWebhookKey.value = ''
  notifyTelegramToken.value = ''

  try {
    const email = JSON.parse(settings.notify_email_json || '{}') as NotifyEmailConfig
    notifyEmail.host = email.host ?? ''
    notifyEmail.port = email.port ?? 587
    notifyEmail.username = email.username ?? ''
    notifyEmail.from = email.from ?? ''
    notifyEmail.tls = email.tls ?? true
    notifyEmail.has_password = email.has_password ?? false
    notifyEmailToText.value = (email.to ?? []).join('\n')
  } catch {
    notifyEmailToText.value = ''
  }

  try {
    const webhook = JSON.parse(settings.notify_webhook_json || '{}') as NotifyWebhookConfig
    notifyWebhook.provider = webhook.provider ?? 'bark'
    notifyWebhook.server = webhook.server ?? (webhook.provider === 'ntfy' ? DEFAULT_NTFY_SERVER : DEFAULT_BARK_SERVER)
    notifyWebhook.topic = webhook.topic ?? ''
    notifyWebhook.url = webhook.url ?? ''
    notifyWebhook.has_secret = webhook.has_secret ?? false
  } catch {
    notifyWebhook.provider = 'bark'
    notifyWebhook.server = DEFAULT_BARK_SERVER
  }

  try {
    const telegram = JSON.parse(settings.notify_telegram_json || '{}') as NotifyTelegramConfig
    notifyTelegram.chat_id = telegram.chat_id ?? ''
    notifyTelegram.proxy_url = telegram.proxy_url ?? ''
    notifyTelegram.has_bot_token = telegram.has_bot_token ?? false
  } catch {
    notifyTelegram.chat_id = ''
    notifyTelegram.proxy_url = ''
  }

  if (!notifyType.value && settings.notify_webhook_url) {
    notifyType.value = 'webhook'
    notifyWebhook.provider = 'custom'
    notifyWebhook.url = settings.notify_webhook_url
  }

  const flagOn = (key: string) => settings[key] !== '0' && settings[key] !== 'false'
  const flagOptIn = (key: string) => settings[key] === '1' || settings[key] === 'true'

  const hasNewNotifyEvents =
    settings.notify_on_ddns_ip_change != null ||
    settings.notify_on_ddns_failure != null ||
    settings.notify_on_cert_expiry != null ||
    settings.notify_on_cert_renew_success != null ||
    settings.notify_on_cert_renew_failure != null ||
    settings.notify_on_ip_frequent_access != null ||
    settings.notify_on_login_failure != null ||
    settings.notify_on_nginx_reload_failure != null

  if (hasNewNotifyEvents) {
    notifyEventFlags.on_ddns_ip_change = flagOn('notify_on_ddns_ip_change')
    notifyEventFlags.on_ddns_failure = flagOn('notify_on_ddns_failure')
    notifyEventFlags.on_cert_expiry = flagOn('notify_on_cert_expiry')
    notifyEventFlags.on_cert_renew_success = flagOn('notify_on_cert_renew_success')
    notifyEventFlags.on_cert_renew_failure = flagOn('notify_on_cert_renew_failure')
    notifyEventFlags.on_ip_frequent_access = flagOptIn('notify_on_ip_frequent_access')
    notifyEventFlags.on_login_failure = flagOptIn('notify_on_login_failure')
    notifyEventFlags.on_nginx_reload_failure = flagOn('notify_on_nginx_reload_failure')
  } else {
    const legacyDDNS = flagOn('notify_on_ddns_error')
    const legacyCert = flagOn('notify_on_cert_error')
    notifyEventFlags.on_ddns_failure = legacyDDNS
    notifyEventFlags.on_cert_expiry = legacyCert
    notifyEventFlags.on_cert_renew_failure = legacyCert
    notifyEventFlags.on_ddns_ip_change = true
    notifyEventFlags.on_cert_renew_success = true
    notifyEventFlags.on_nginx_reload_failure = true
    notifyEventFlags.on_ip_frequent_access = false
    notifyEventFlags.on_login_failure = false
  }
  notifyIPFrequentThreshold.value = Number(settings.notify_ip_frequent_threshold || 100)
  notifyIPFrequentWindowSec.value = Number(settings.notify_ip_frequent_window_sec || 60)
  notifyLoginFailureThreshold.value = Number(settings.notify_login_failure_threshold || DEFAULT_LOGIN_FAILURE_THRESHOLD)
  notifyLoginFailureWindowSec.value = Number(settings.notify_login_failure_window_sec || DEFAULT_LOGIN_FAILURE_WINDOW_SEC)
}

function buildNotifyEmailJSON(): string {
  const payload: NotifyEmailConfig = {
    host: notifyEmail.host.trim(),
    port: notifyEmail.port || 587,
    username: notifyEmail.username.trim(),
    from: notifyEmail.from.trim(),
    to: notifyEmailToText.value
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean),
    tls: notifyEmail.tls,
    has_password: notifyEmail.has_password || Boolean(notifySMTPPassword.value.trim()),
  }
  return JSON.stringify(payload)
}

function buildNotifyWebhookJSON(): string {
  const payload: NotifyWebhookConfig = {
    provider: notifyWebhook.provider,
    server: notifyWebhook.server.trim(),
    key: '',
    topic: notifyWebhook.topic.trim(),
    url: notifyWebhook.url.trim(),
    has_secret: notifyWebhook.has_secret || Boolean(notifyWebhookKey.value.trim()),
  }
  return JSON.stringify(payload)
}

function buildNotifyTelegramJSON(): string {
  const payload: NotifyTelegramConfig = {
    chat_id: notifyTelegram.chat_id.trim(),
    proxy_url: notifyTelegram.proxy_url.trim(),
    has_bot_token: notifyTelegram.has_bot_token || Boolean(notifyTelegramToken.value.trim()),
  }
  return JSON.stringify(payload)
}

function resolveSecretInput(value: string, hasExisting: boolean) {
  const trimmed = value.trim()
  if (trimmed) return trimmed
  if (hasExisting) return MASKED_SECRET
  return ''
}

function buildNotifyTestPayload(): NotifyTestPayload {
  return {
    type: notifyType.value,
    email: JSON.parse(buildNotifyEmailJSON()) as NotifyEmailConfig,
    webhook: JSON.parse(buildNotifyWebhookJSON()) as NotifyWebhookConfig,
    telegram: JSON.parse(buildNotifyTelegramJSON()) as NotifyTelegramConfig,
    on_ddns_ip_change: notifyEventFlags.on_ddns_ip_change,
    on_ddns_failure: notifyEventFlags.on_ddns_failure,
    on_cert_expiry: notifyEventFlags.on_cert_expiry,
    on_cert_renew_success: notifyEventFlags.on_cert_renew_success,
    on_cert_renew_failure: notifyEventFlags.on_cert_renew_failure,
    on_ip_frequent_access: notifyEventFlags.on_ip_frequent_access,
    on_login_failure: notifyEventFlags.on_login_failure,
    on_nginx_reload_failure: notifyEventFlags.on_nginx_reload_failure,
    ip_frequent_threshold: notifyIPFrequentThreshold.value,
    ip_frequent_window_sec: notifyIPFrequentWindowSec.value,
    login_failure_threshold: notifyLoginFailureThreshold.value,
    login_failure_window_sec: notifyLoginFailureWindowSec.value,
    smtp_password: resolveSecretInput(notifySMTPPassword.value, notifyEmail.has_password),
    webhook_secret: resolveSecretInput(notifyWebhookKey.value, notifyWebhook.has_secret),
    telegram_token: resolveSecretInput(notifyTelegramToken.value, notifyTelegram.has_bot_token),
  }
}

async function testNotify() {
  if (!notifyType.value) {
    message.warning('请先选择通知方式')
    return
  }
  notifyTesting.value = true
  try {
    const result = await api.testNotify(buildNotifyTestPayload())
    message.success(result.message || '测试通知已发送')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '测试通知失败')
  } finally {
    notifyTesting.value = false
  }
}

function buildSavePayload() {
  const payload: Record<string, string> = {
    theme: form.theme,
    timezone: form.timezone,
    notify_type: notifyType.value,
    notify_email_json: buildNotifyEmailJSON(),
    notify_smtp_password: resolveSecretInput(notifySMTPPassword.value, notifyEmail.has_password),
    notify_webhook_json: buildNotifyWebhookJSON(),
    notify_webhook_secret: resolveSecretInput(notifyWebhookKey.value, notifyWebhook.has_secret),
    notify_telegram_json: buildNotifyTelegramJSON(),
    notify_telegram_token: resolveSecretInput(notifyTelegramToken.value, notifyTelegram.has_bot_token),
    notify_on_ddns_ip_change: notifyEventFlags.on_ddns_ip_change ? '1' : '0',
    notify_on_ddns_failure: notifyEventFlags.on_ddns_failure ? '1' : '0',
    notify_on_cert_expiry: notifyEventFlags.on_cert_expiry ? '1' : '0',
    notify_on_cert_renew_success: notifyEventFlags.on_cert_renew_success ? '1' : '0',
    notify_on_cert_renew_failure: notifyEventFlags.on_cert_renew_failure ? '1' : '0',
    notify_on_ip_frequent_access: notifyEventFlags.on_ip_frequent_access ? '1' : '0',
    notify_on_login_failure: notifyEventFlags.on_login_failure ? '1' : '0',
    notify_on_nginx_reload_failure: notifyEventFlags.on_nginx_reload_failure ? '1' : '0',
    notify_ip_frequent_threshold: String(notifyIPFrequentThreshold.value || 100),
    notify_ip_frequent_window_sec: String(notifyIPFrequentWindowSec.value || 60),
    notify_login_failure_threshold: String(notifyLoginFailureThreshold.value || DEFAULT_LOGIN_FAILURE_THRESHOLD),
    notify_login_failure_window_sec: String(notifyLoginFailureWindowSec.value || DEFAULT_LOGIN_FAILURE_WINDOW_SEC),
    ddns_check_interval_minutes: String(ddnsInterval.value),
    cert_renew_threshold_days: String(certThreshold.value),
    log_retention_days: String(logRetention.value),
    acme_email: acmeEmail.value.trim(),
    trusted_proxy_json: buildTrustedProxyJSON(),
    global_ip_blacklist: buildGlobalIPListJSON(globalIPBlacklistText.value),
    global_ip_whitelist: buildGlobalWhitelistJSON(),
    china_cidr_update_interval_hours: String(chinaCIDRHours.value),
  }
  if (notifyType.value === 'webhook' && notifyWebhook.provider === 'custom') {
    payload.notify_webhook_url = notifyWebhook.url.trim()
  }
  const zerosslKey = zerosslApiKey.value.trim()
  if (zerosslKey) {
    payload.zerossl_api_key = zerosslKey
  }
  return payload
}

async function load() {
  loading.value = true
  pageError.value = ''
  try {
    applySettingsToForm(await api.getSettings())
    await Promise.all([loadChinaCIDRStatus(), loadGlobalNginx()])
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
    applySettingsToForm(saved)
    setThemeMode(saved.theme ?? form.theme)
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
.settings-tabs {
  margin-bottom: var(--fonu-space-4);
}

.settings-tabs :deep(.n-tabs-nav) {
  margin-bottom: var(--fonu-space-4);
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--fonu-space-5);
  align-items: stretch;
}

.settings-grid--single {
  grid-template-columns: minmax(0, 1fr);
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

.nginx-framework {
  margin: 12px 0;
}

.nginx-framework__editor {
  margin-top: 8px;
}

.global-nginx-head {
  margin-top: 12px;
}

.nginx-mode-toggle {
  display: inline-flex;
  gap: 4px;
}

.log-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.log-panel-actions {
  display: inline-flex;
  gap: 4px;
}

.log-panel-actions :deep(.n-button) {
  min-width: 56px;
}

.global-nginx-editor,
.nginx-editor {
  flex: 1;
  min-height: 0;
}

.global-nginx-upload-hint {
  margin: 10px 0 0;
}

.global-nginx-warn {
  margin-top: 12px;
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

.proxy-security-form {
  gap: 14px;
}

.proxy-security-form__intro {
  margin: 0;
}

.settings-fields--compact {
  gap: 12px;
}

.ip-policy-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  align-items: start;
}

.ip-policy-field {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.ip-policy-field__input {
  width: 100%;
}

.ip-policy-field__input :deep(.n-input) {
  width: 100%;
}

.ip-policy-field__input :deep(.n-input-wrapper) {
  width: 100%;
}

.ip-policy-field__input :deep(.n-input__textarea-el) {
  width: 100%;
  height: 152px;
  min-height: 152px;
  max-height: 152px;
  resize: none;
  box-sizing: border-box;
}

.ip-policy-field .field-hint {
  margin-top: 6px;
}

.china-cidr-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  background: var(--fonu-bg);
}

.china-cidr-panel__head {
  margin-bottom: 0;
}

.notify-type-group {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
}

.notify-smtp-endpoint {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 120px;
  gap: 12px;
  align-items: start;
}

.notify-smtp-endpoint__col--port :deep(.n-input-number) {
  width: 100%;
}

.notify-input-wide {
  flex: 1;
  min-width: 180px;
}

.notify-form {
  gap: 16px;
}

.notify-events {
  margin-top: var(--fonu-space-4);
}

.notify-events__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
  margin-bottom: 12px;
}

.notify-event-group {
  margin-bottom: 16px;
}

.notify-event-group__title {
  font-size: 12px;
  font-weight: 600;
  color: var(--fonu-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 8px;
}

.notify-threshold-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin: -4px 0 8px;
  padding: 0 14px;
}

.zerossl-key-hint {
  margin-top: -8px;
}

.notify-test-row {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
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

  .ip-policy-grid {
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
