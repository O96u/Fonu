<template>
  <PageHeader title="系统设置" description="管理应用配置与运行环境">
    <template #actions><n-button secondary :type="settingsTab==='version'?'primary':'default'" :disabled="saving || restoring" @click="switchTab('version')"><template #icon><n-icon :component="InformationCircleOutline" /></template>关于 Fonu</n-button></template>
  </PageHeader>
  <LoadError v-if="pageError" :message="pageError" @retry="load" />
  <n-spin v-else :show="loading">
    <div class="settings-workspace">
      <nav class="settings-navigation" aria-label="设置分类">
        <button v-for="item in categories" :key="item.key" type="button" class="settings-nav-item" :class="{'is-active':settingsTab===item.key}" :aria-current="settingsTab===item.key?'page':undefined" :disabled="saving || restoring" :data-testid="'settings-tab-'+item.key" @click="switchTab(item.key)">
          <n-icon :component="item.icon" /><span>{{ item.label }}</span><span v-if="dirtyGroups[item.key]" class="dirty-dot" title="未保存修改" />
        </button>
      </nav>
      <div class="settings-category-head"><div><h2>{{ categoryTitle }}</h2><p>{{ categoryDescription }}</p></div><StatusBadge v-if="currentCanSave" :kind="currentDirty?'warning':'success'" :text="currentDirty?'未保存修改':'配置已保存'"/></div>
      <fieldset class="settings-content" :disabled="saving || restoring">
        <div v-if="settingsTab==='general'" class="settings-grid settings-grid--single"><FonuCard title="外观" subtitle="界面主题与时区">
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
          <div class="settings-block__label">主题颜色</div>
          <div class="accent-options">
            <button v-for="preset in themePresets" :key="preset.value" type="button"
              class="accent-option" :aria-pressed="themeAccent === preset.value"
              @click="setThemeAccent(preset.value)">
              <span class="accent-swatch" :style="{ background: isDark ? preset.dark : preset.light }">
                <span v-if="themeAccent === preset.value" :style="{ color: isDark ? '#171c1a' : '#fff' }">✓</span>
              </span>
              <span>{{ preset.label }}</span>
            </button>
          </div>
          <p class="field-hint">立即应用并保存在当前浏览器，不影响其他设备。</p>
        </div>
        <div class="settings-block settings-block--spaced">
          <div class="settings-block__label">时区</div>
          <n-select v-model:value="form.timezone" :options="timezoneOptions" />
          <p class="field-hint">影响日志、证书到期时间等显示</p>
        </div>
      </FonuCard><FonuCard title="自动任务" subtitle="公网 IP 检查">
        <div class="settings-fields">
          <div class="settings-field">
            <div class="settings-field__row">
              <span class="settings-field__label">DDNS 检查周期（分钟）</span>
              <n-input-number v-model:value="ddnsInterval" :min="1" :max="1440" class="settings-field__input" />
            </div>
            <p class="field-hint">每隔 N 分钟检查公网 IP 是否变化并更新 DNS</p>
          </div>

        </div>
      </FonuCard></div>
        <div v-else-if="settingsTab==='account'" class="settings-grid settings-grid--single"><FonuCard title="管理员账户" subtitle="修改登录用户名与密码" class="settings-card settings-card--security">
        <div class="security-form">
          <div class="settings-fields">
            <div class="settings-field">
              <div class="settings-field__label">登录用户名</div>
              <n-input v-model:value="usernameForm.new_username" placeholder="用户名" />
            </div>
            <div class="settings-field">
              <div class="settings-field__label">当前密码</div>
              <n-input
                v-model:value="usernameForm.password"
                type="password"
                show-password-on="click"
                placeholder="修改用户名时需验证"
              />
            </div>
          </div>
          <n-button
            type="primary"
            class="password-btn"
            :loading="changingUsername"
            :disabled="!usernameChanged || changingPassword || changingUsername"
            @click="changeUsername"
          >
            保存用户名
          </n-button>
          <div class="settings-fields settings-fields--password">
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
          <n-button type="primary" class="password-btn" :loading="changingPassword" :disabled="changingUsername || changingPassword" @click="changePassword">
            修改管理员密码
          </n-button>
        </div>
      </FonuCard></div>
        <div v-else-if="settingsTab==='security'" class="settings-grid settings-grid--single"><FonuCard title="反代安全" subtitle="全局 IP 策略与真实客户端 IP">
        <div class="security-form proxy-security-form">
          <p class="field-hint proxy-security-form__intro">
            IP 黑白名单、仅中国大陆等策略依赖正确识别客户端 IP。若域名经 CDN/多层反代，请在此配置信任代理。
          </p>
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">启用信任代理</div>
              <div class="form-switch-row__hint">仅在 Fonu 前方存在可信 HTTP 代理时开启</div>
            </div>
            <n-switch v-model:value="trustedProxyEnabled" />
          </div>
          <n-alert type="info" :bordered="false">直接公网访问、DDNS 或 Cloudflare 灰云保持关闭。橙云选择 Cloudflare；自建代理仅填写实际连接 Fonu 的代理 IP。不要信任整个公网或不受控制的内网段。</n-alert>
          <div v-if="trustedProxyEnabled" class="settings-fields settings-fields--compact">
            <div class="settings-field">
              <div class="settings-field__label">预设</div>
              <n-select v-model:value="trustedProxyPreset" :options="trustedProxyPresets" />
            </div>
            <p v-if="trustedProxyPreset==='cloudflare'" class="field-hint">自动使用 Cloudflare 预置 IP 段及 CF-Connecting-IP 请求头。仅信任来自这些代理的请求头。</p>
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
      </FonuCard></div>
        <div v-else-if="settingsTab==='notify'" class="settings-grid settings-grid--single"><FonuCard title="通知" subtitle="告警通道与事件（仅可启用一种方式）">
        <div class="security-form notify-form"><div class="notify-channel">
          <div class="settings-field">
            <div class="settings-field__label">通知渠道</div><p class="field-hint">选择一种通知方式</p>
            <div class="notify-channel-grid">
              <button v-for="item in channelOptions" :key="item.key" type="button" class="notify-channel-choice" :class="{'is-active':selectedChannel===item.key}" :aria-pressed="selectedChannel===item.key" :data-testid="'notify-channel-'+item.key" @click="selectChannel(item.key)">
                <span class="notify-channel-icon" :style="{color:item.color}"><n-icon :component="item.icon" /></span><span>{{item.label}}</span>
              </button>
            </div>
          </div>
          <div class="form-switch-row"><div class="form-switch-row__text"><div class="form-switch-row__label">启用通知</div><div class="form-switch-row__hint">关闭后保留配置，不再推送告警</div></div><n-switch v-model:value="notifyEnabled" data-testid="notify-enabled" /></div>
          <div class="notify-config-panel">
          <h3>{{channelOptions.find(item=>item.key===selectedChannel)?.label}} 配置</h3>
          <div v-if="selectedNotifyType === 'email'" class="settings-fields settings-fields--compact">
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

          <div v-else-if="selectedNotifyType === 'webhook'" class="settings-fields settings-fields--compact">
            <template v-if="notifyWebhook.provider === 'bark'">
              <div class="settings-field">
                <div class="settings-field__label">Bark 服务地址</div>
                <n-input v-model:value="notifyWebhook.server" :placeholder="DEFAULT_BARK_SERVER" />
              </div>
              <ConfiguredSecretField
                v-model="notifyWebhookKey"
                label="Device Key"
                :configured="hasSavedWebhookSecret"
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
                :configured="hasSavedWebhookSecret"
                placeholder="Bearer Token"
              />
            </template>
            <template v-else-if="['wecom','dingtalk','feishu','wecom_app','feishu_app'].includes(notifyWebhook.provider)">
              <div v-if="selectedChannel==='wecom'||selectedChannel==='feishu'" class="settings-field">
                <div class="settings-field__label">接入模式</div>
                <n-radio-group :value="isAppMode?'app':'webhook'" @update:value="changeNotifyMode"><n-radio value="webhook">群机器人 Webhook</n-radio><n-radio value="app">自建应用</n-radio></n-radio-group>
              </div>
              <template v-if="isAppMode">
                <div class="settings-field"><div class="settings-field__label">{{selectedChannel==='wecom'?'企业 ID（CorpID）':'App ID'}}</div><n-input v-model:value="notifyWebhook.app_id" placeholder="应用标识" /></div>
                <div v-if="selectedChannel==='wecom'" class="settings-field"><div class="settings-field__label">应用 AgentId</div><n-input v-model:value="notifyWebhook.agent_id" placeholder="1000002"/></div>
                <ConfiguredSecretField v-model="notifyWebhookKey" label="应用 Secret" :configured="hasSavedWebhookSecret" placeholder="请输入应用密钥" />
                <div v-if="selectedChannel==='feishu'" class="settings-field"><div class="settings-field__label">接收对象</div><n-radio-group v-model:value="notifyWebhook.recipient_type"><n-radio value="open_id">指定用户</n-radio><n-radio value="chat_id">指定群聊</n-radio></n-radio-group></div>
                <div class="settings-field"><div class="settings-field__label">{{selectedChannel==='wecom'?'接收成员 UserID（多个用 | 分隔）':notifyWebhook.recipient_type==='chat_id'?'群聊 Chat ID':'用户 Open ID'}}</div><n-input v-model:value="notifyWebhook.recipient" :placeholder="selectedChannel==='wecom'?'zhangsan|lisi':notifyWebhook.recipient_type==='chat_id'?'oc_xxxxx':'ou_xxxxx'"/></div>
                <div v-if="selectedChannel==='wecom'" class="settings-field"><div class="settings-field__label">API 转发代理（可选）</div><n-input v-model:value="notifyWebhook.api_base" placeholder="https://qyapi.weixin.qq.com"/><p class="field-hint">仅填写你信任的 HTTPS 企业微信 API 转发服务，它将接触应用密钥。</p></div>
                <p class="field-hint">应用须已启用消息发送权限，接收对象须在应用可见范围内。仅主动通知，不接收回调或远程命令。</p>
              </template>
              <template v-else>
                <ConfiguredSecretField v-model="notifyWebhookKey" label="机器人 Webhook 地址" :configured="hasSavedWebhookSecret" placeholder="粘贴完整 HTTPS Webhook 地址" />
                <template v-if="selectedChannel!=='wecom'"><div class="form-switch-row"><div class="form-switch-row__text">启用签名校验</div><n-switch v-model:value="notifyWebhook.signed"/></div><ConfiguredSecretField v-if="notifyWebhook.signed" v-model="notifySignature" label="签名密钥" :configured="hasSavedSignature" placeholder="机器人安全设置中的签名密钥" /></template>
                <p class="field-hint">群机器人仅向指定群发送消息。使用关键词校验时可设置 Fonu。</p>
              </template>
            </template>
            <template v-else-if="notifyWebhook.provider === 'gotify'">
              <div class="settings-field">
                <div class="settings-field__label">Gotify 服务地址</div>
                <n-input v-model:value="notifyWebhook.server" placeholder="https://gotify.example.com" />
              </div>
              <ConfiguredSecretField
                v-model="notifyWebhookKey"
                label="App Token"
                :configured="hasSavedWebhookSecret"
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

          <div v-else-if="selectedNotifyType === 'telegram'" class="settings-fields settings-fields--compact">
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

          <div class="notify-test-row"><n-button :loading="notifyTesting" :disabled="!notifyEnabled" @click="testNotify">测试通知</n-button><p class="field-hint">使用当前表单配置发送测试消息，无需先保存</p></div>
          </div></div><div class="notify-events" :class="{'notify-events-disabled':!notifyEnabled}">
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


        </div>
      </FonuCard><NotifyHistory/></div>
        <div v-else-if="settingsTab==='acme'" class="settings-grid settings-grid--single"><FonuCard title="续签策略" subtitle="自动检查与续签">
            <div class="settings-fields">
              <div class="settings-field">
                <div class="settings-field__row">
                  <span class="settings-field__label">证书续签检查（天）</span>
                  <n-input-number v-model:value="certThreshold" :min="1" :max="90" class="settings-field__input" />
                </div>
                <p class="field-hint">当证书剩余有效期少于此天数时自动续签</p>
              </div>
            </div>
          </FonuCard><FonuCard title="DNS-01 验证" subtitle="传播检查与超时（高级）">
            <div class="settings-fields">
              <div class="settings-field">
                <div class="settings-field__row">
                  <span class="settings-field__label">DNS 传播超时（秒）</span>
                  <n-input-number v-model:value="acmeDnsPropagationTimeout" :min="30" :max="900" class="settings-field__input" />
                </div>
                <p class="field-hint">等待 TXT 记录在全球 DNS 生效的最长时间，默认 180</p>
              </div>
              <div class="settings-switches settings-switches--col">
                <div class="settings-switch">
                  <span>跳过权威 NS 传播检查</span>
                  <n-switch v-model:value="acmeDnsDisableAuthNS" />
                </div>
                <div class="settings-switch">
                  <span>忽略传播结果（仅等待后提交验证）</span>
                  <n-switch v-model:value="acmeDnsIgnorePropagation" />
                </div>
              </div>
            </div>
          </FonuCard><FonuCard title="ACME 账户" subtitle="申请与续签所需凭据">
            <div class="settings-fields">
              <div class="settings-field">
                <div class="settings-field__label">ACME 邮箱</div>
                <n-input
                  v-model:value="acmeEmail"
                  placeholder="admin@example.com"
                  :status="acmeEmailInputStatus"
                />
                <p class="field-hint">用于 Let's Encrypt / ZeroSSL / Buypass 账户注册；首次可在「申请证书」中填写并保存</p>
              </div>

              <div class="provider-list">
  <div class="provider-row"><span><strong>Let's Encrypt</strong><small>无需 EAB 凭据</small></span><StatusBadge kind="success" text="可使用" /></div>
  <div v-for="item in acmeProviders" :key="item.key" class="provider-row">
    <span><strong>{{ item.label }}</strong><small>{{ item.hint }}</small></span>
    <StatusBadge :kind="item.draft?'warning':item.ready?'success':'disabled'" :text="item.draft?'待保存':item.ready?'已配置':'未配置'" />
    <n-button size="small" :disabled="saving" @click="activeProvider=item.key">{{ item.ready?'编辑':'配置' }}</n-button>
  </div>
</div>
<p class="field-hint">提示：同一 Fonu 实例内多组域名证书可共用同一组 EAB 凭据；多个 Fonu 实例之间不可共用同一组 EAB 凭据</p>
            </div>
          </FonuCard></div>
        <div v-else-if="settingsTab==='logs'" class="settings-grid settings-grid--single"><FonuCard title="日志保留" subtitle="日志文件留存策略"><div class="settings-fields">          <div class="settings-field">
            <div class="settings-field__row">
              <span class="settings-field__label">日志保留天数</span>
              <n-input-number v-model:value="logRetention" :min="1" :max="365" class="settings-field__input" />
            </div>
            <p class="field-hint">按天清理已轮转的日志归档，不删除正在写入的日志；系统日志另受最多 10 份归档的限制。</p>
          </div></div></FonuCard><FonuCard title="访问统计" subtitle="SQLite 入库与仪表盘口径">
        <div class="settings-fields">
          <div class="settings-field">
            <div class="settings-field__row">
              <span class="settings-field__label">统计保留天数</span>
              <n-input-number v-model:value="analyticsRetention" :min="1" :max="365" class="settings-field__input" />
            </div>
            <p class="field-hint">超过保留天数的 access_facts 记录将被自动清理（默认 30 天）</p>
          </div>
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">UV 排除内网 IP</div>
              <div class="form-switch-row__hint">独立 IP 统计不包含内网/本机地址</div>
            </div>
            <n-switch v-model:value="analyticsExcludePrivate" />
          </div>
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">跳过历史 access.log 导入</div>
              <div class="form-switch-row__hint">仅影响首次历史导入；不会删除已经入库的记录</div>
            </div>
            <n-switch v-model:value="analyticsSkipBackfill" />
          </div>
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">远程 IP 归属地补齐</div>
              <div class="form-switch-row__hint">需出站访问外网 API，默认关闭；离线 ip2region 优先</div>
            </div>
            <n-switch v-model:value="analyticsGeoRemote" />
          </div>
          <div v-if="analyticsGeoRemote" class="settings-field">
            <div class="settings-field__label">远程批量查询 URL</div>
            <n-input v-model:value="analyticsGeoRemoteURL" placeholder="http://ip-api.com/batch?..." />
          </div>
        </div>
      </FonuCard></div>
        <div v-else-if="settingsTab==='backup'" class="settings-grid settings-grid--single"><FonuCard title="备份与恢复" subtitle="导出系统配置或恢复已有备份">
        <div class="data-actions">
          <button type="button" class="data-action-card" :disabled="exporting" @click="exportBackup">
            <n-icon :component="DownloadOutline" class="data-action-card__icon" />
            <div class="data-action-card__title">导出配置</div>
            <div class="data-action-card__hint">下载数据库、证书、密钥与 Nginx 配置</div>
          </button>
          <n-upload
            class="data-upload"
            :show-file-list="false" :disabled="restoring || exporting"
            accept=".tar.gz,.tgz"
            :custom-request="restoreRequest"
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
        <n-alert type="warning" :bordered="false" class="backup-warning">备份包含账号数据和敏感凭据，请妥善保管，不要上传到不受信任的服务。</n-alert><p class="field-hint data-note">恢复前会先确认。完成后会退出登录，请使用备份中的账号重新登录。</p>
      </FonuCard></div>
        <div v-else-if="settingsTab==='advanced'" class="settings-grid settings-grid--single"><FonuCard title="Nginx 全局配置" subtitle="http 块内自定义片段"><div v-if="nginxLoadError" class="settings-load-error"><span>{{ nginxLoadError }}</span><n-button size="small" @click="loadGlobalNginx">重试</n-button></div><n-spin :show="nginxLoading"><fieldset :disabled="!nginxLoaded || globalNginxSaving" class="settings-content">
        <NginxApplyBanner/>
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
              size="small"
              :type="globalNginxEditMode === 'auto' ? 'primary' : 'default'"
              @click="setGlobalNginxEditMode('auto')"
            >
              自动生成
            </n-button>
            <n-button
              size="small"
              :type="globalNginxEditMode === 'custom' ? 'primary' : 'default'"
              @click="setGlobalNginxEditMode('custom')"
            >
              手动编辑
            </n-button>
          </div>
          <div class="log-panel-actions"><n-select v-if="globalNginxEditMode==='custom' && globalNginxBackups.length" v-model:value="selectedNginxBackup" :options="globalNginxBackups.map(item=>({label:formatDate(item.created_at)+' · '+item.name,value:item.name}))" placeholder="选择历史版本" clearable size="small" style="width:230px" />
            <n-button
              v-if="globalNginxEditMode === 'custom'"
              size="small"
              type="primary"
              :loading="globalNginxSaving"
              :disabled="!globalNginxDirty"
              @click="saveGlobalNginx"
            >
              保存配置
            </n-button>
            <n-button
              v-if="globalNginxEditMode === 'custom' && globalNginxBackups.length > 0"
              size="small"
              :loading="globalNginxSaving"
              @click="confirmNginxRollback()"
            >
              回滚
            </n-button>
            <n-button
              v-if="globalNginxEditMode === 'custom' && globalNginxServerMode === 'custom'"
              size="small"
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
          手动配置会影响所有反向代理。保存前请确认指令有效；如需恢复，可选择自动生成或历史版本。
        </n-alert></fieldset></n-spin>
      </FonuCard></div>
        <div v-else class="settings-grid settings-grid--single"><SettingsVersionCard /></div>
      </fieldset>
      <div v-if="currentCanSave" class="save-bar" data-testid="settings-save-bar">
        <div><strong>{{ currentDirty?'当前分类有未保存修改':'当前配置已保存' }}</strong><small>仅保存当前分类，其他分类草稿保持不变。</small></div>
        <div class="save-bar-actions"><n-button :disabled="!currentDirty || saving" @click="discardCurrent">取消修改</n-button><n-button type="primary" :loading="saving" :disabled="!currentDirty || saving || loading" @click="save">保存设置</n-button></div>
      </div>
    </div>
  </n-spin>
  <n-modal :show="!!activeProvider" :mask-closable="false" @update:show="value => { if (!value) activeProvider = '' }" preset="card" :title="providerTitle" style="width:600px;max-height:calc(100dvh - 24px);display:flex;flex-direction:column" class="provider-modal">
  <div class="provider-editor"><template v-if="activeProvider==='custom'"><div class="settings-field">
                <div class="settings-field__label">自定义 ACME Directory URL</div>
                <n-input v-model:value="customAcmeDirectoryUrl" placeholder="https://acme.example.com/directory" />
                <p class="field-hint">任意 ACME v2 服务器（内网 step-ca、Vault、EJBCA 或其他新 CA），申请时选择「自定义 ACME」</p>
              </div>
              <ConfiguredSecretField
                v-model="customAcmeEabKid"
                label="自定义 ACME EAB Kid（可选）"
                :configured="customAcmeEabKidHasKey"
                placeholder="服务器要求 EAB 时填写"
              />
              <ConfiguredSecretField
                v-model="customAcmeEabHmac"
                label="自定义 ACME EAB Hmac（可选）"
                :configured="customAcmeEabHmacHasKey"
                placeholder="与 Kid 配对"
              />
              </template>
  <div v-if="activeProvider==='zerossl'" class="acme-provider-block provider-edit-block">
                <div class="acme-provider-block__head">
                  <AcmeApplyLink :url="ACME_APPLY_LINKS.zerossl.url" :link-text="ACME_APPLY_LINKS.zerossl.linkText" />
                </div>
                <ConfiguredSecretField
                  v-model="zerosslEabKid"
                  label="EAB Kid"
                  :configured="zerosslEabKidHasKey"
                  placeholder="Developer 页面生成的 Kid"
                />
                <ConfiguredSecretField
                  v-model="zerosslEabHmac"
                  label="EAB Hmac"
                  :configured="zerosslEabHmacHasKey"
                  placeholder="与 Kid 配对"
                />
                <p class="field-hint">申请 ZeroSSL 证书时必填</p>
              </div>

              
<div v-if="activeProvider==='google'" class="acme-provider-block provider-edit-block">
                <div class="acme-provider-block__head">
                  <AcmeApplyLink :url="ACME_APPLY_LINKS.google.url" :link-text="ACME_APPLY_LINKS.google.linkText" />
                </div>
                <ConfiguredSecretField
                  v-model="googleEabKid"
                  label="EAB Kid"
                  :configured="googleEabKidHasKey"
                  placeholder="Certificate Manager 申请的 Kid"
                />
                <ConfiguredSecretField
                  v-model="googleEabHmac"
                  label="EAB Hmac"
                  :configured="googleEabHmacHasKey"
                  placeholder="与 Kid 配对"
                />
              </div>

              
<div v-if="activeProvider==='sslcom'" class="acme-provider-block provider-edit-block">
                <div class="acme-provider-block__head">
                  <AcmeApplyLink :url="ACME_APPLY_LINKS.sslcom.url" :link-text="ACME_APPLY_LINKS.sslcom.linkText" />
                </div>
                <ConfiguredSecretField
                  v-model="sslcomEabKid"
                  label="EAB Kid"
                  :configured="sslcomEabKidHasKey"
                  placeholder="控制台 ACME 凭据"
                />
                <ConfiguredSecretField
                  v-model="sslcomEabHmac"
                  label="EAB Hmac"
                  :configured="sslcomEabHmacHasKey"
                  placeholder="与 Kid 配对"
                />
              </div>

              
<div v-if="activeProvider==='freessl'" class="acme-provider-block provider-edit-block">
                <div class="acme-provider-block__head">
                  <AcmeApplyLink :url="ACME_APPLY_LINKS.freessl.url" :link-text="ACME_APPLY_LINKS.freessl.linkText" />
                </div>
                <ConfiguredSecretField
                  v-model="freesslEabKid"
                  label="EAB Kid"
                  :configured="freesslEabKidHasKey"
                  placeholder="EAB 管理页生成"
                />
                <ConfiguredSecretField
                  v-model="freesslEabHmac"
                  label="EAB Hmac"
                  :configured="freesslEabHmacHasKey"
                  placeholder="与 Kid 配对"
                />
                <ConfiguredSecretField
                  v-model="freesslAutomationToken"
                  label="Automation Token"
                  :configured="freesslAutomationTokenHasKey"
                  placeholder="freessl.cn 自动化 Token，可自动获取 EAB"
                />
                <div class="settings-field">
                  <div class="settings-field__label">ACME 地址（可选）</div>
                  <n-input v-model:value="freesslDirectoryUrl" placeholder="留空默认 https://acme.litessl.com/acme/v2/directory" />
                  <p class="field-hint">若分配了专属 Directory URL（含路径），粘贴到此处；填写 Automation Token 时可自动拼接旧版路径</p>
                </div>
              </div>

              
<div v-if="activeProvider==='actalis'" class="acme-provider-block provider-edit-block">
                <div class="acme-provider-block__head">
                  <AcmeApplyLink :url="ACME_APPLY_LINKS.actalis.url" :link-text="ACME_APPLY_LINKS.actalis.linkText" />
                </div>
                <ConfiguredSecretField
                  v-model="actalisEabKid"
                  label="EAB Kid"
                  :configured="actalisEabKidHasKey"
                  placeholder="客户区 ACME Credentials"
                />
                <ConfiguredSecretField
                  v-model="actalisEabHmac"
                  label="EAB Hmac"
                  :configured="actalisEabHmacHasKey"
                  placeholder="与 Kid 配对"
                />
                <p class="field-hint">免费套餐为单域名 90 天证书，不支持通配符</p>
              </div>
              
  <p class="field-hint">关闭后保留本次输入；在 ACME 证书分类中保存后生效。留空保留已保存的密钥。</p></div>
  <template #footer><div class="provider-modal-actions"><n-button @click="activeProvider=''">完成编辑</n-button></div></template>
</n-modal>
  <n-modal :show="restoring" :mask-closable="false" :close-on-esc="false" preset="card" :title="restoreTitle" :closable="false" style="width:420px"><div class="restore-progress"><n-spin size="small" /><p>{{ restoreDetail }}</p></div></n-modal>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import {
  NAlert,
  NButton,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NRadio,
  NRadioGroup,
  NSelect,
  NSpin,
  NSwitch,
  NUpload,
  useDialog,
  useMessage,
  type UploadCustomRequestOptions,
} from 'naive-ui'
import {
  MailOutline, ChatbubbleEllipsesOutline, PaperPlaneOutline, SendOutline, NotificationsCircleOutline, LinkOutline, TerminalOutline,
  CheckmarkCircle,
  CloudUploadOutline,
  DesktopOutline,
  DownloadOutline,
  MoonOutline,
  SunnyOutline,
  SettingsOutline, ShieldCheckmarkOutline, ShieldOutline, NotificationsOutline, LockClosedOutline, StatsChartOutline, ServerOutline, CodeSlashOutline, InformationCircleOutline,
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
import AcmeApplyLink from '../components/AcmeApplyLink.vue'
import ConfiguredSecretField from '../components/ConfiguredSecretField.vue'
import { ACME_APPLY_LINKS } from '../constants/acmeApplyLinks'
import FonuCard from '../components/FonuCard.vue'
import NotifyHistory from '../components/NotifyHistory.vue'
import SettingsVersionCard from '../components/SettingsVersionCard.vue'
import NginxCodeEditor from '../components/NginxCodeEditor.vue'
import NginxApplyBanner from '../components/NginxApplyBanner.vue'
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
} from '../constants/notifyProviders'
import { invalidateAppTimezoneCache } from '../composables/useAppTimezone'
import { useTheme, themePresets } from '../composables/useTheme'
import { isValidAcmeEmail, validateAcmeEmailMessage } from '../utils/acmeEmail'
import { formatDate, formatRelativeTime } from '../utils/format'
import type { StatusKind } from '../utils/status'

const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const route = useRoute()

const categories = [
 {key:'general',label:'常规设置',icon:SettingsOutline,description:'界面外观、时区与自动任务'},
 {key:'account',label:'账户安全',icon:ShieldCheckmarkOutline,description:'管理员用户名与密码'},
 {key:'security',label:'反代安全',icon:ShieldOutline,description:'信任代理、访问策略与中国 IP 段库'},
 {key:'notify',label:'通知管理',icon:NotificationsOutline,description:'通知通道、告警事件与测试'},
 {key:'acme',label:'ACME 证书',icon:LockClosedOutline,description:'自动续签、DNS 验证及 CA 凭据'},
 {key:'logs',label:'日志与统计',icon:StatsChartOutline,description:'数据保留、访问统计与 IP 归属地'},
 {key:'backup',label:'备份与恢复',icon:ServerOutline,description:'系统配置导出与恢复'},
 {key:'advanced',label:'高级配置',icon:CodeSlashOutline,description:'全局 Nginx 配置及历史回滚'},
] as const
type SettingsGroup='general'|'security'|'notify'|'acme'|'logs'
const saveGroups:SettingsGroup[]=['general','security','notify','acme','logs']
const SETTINGS_TAB_NAMES = new Set<string>([...categories.map(c=>c.key),'version'])
const activeProvider=ref('')
const selectedNginxBackup=ref<string|null>(null)
const savedSettings=ref<Record<string,string>>({})
const baseline=ref<Partial<Record<SettingsGroup,Record<string,string>>>>({})
const initialized=ref(false)
const { setThemeMode, themeMode, themeAccent, setThemeAccent, isDark } = useTheme()
const saving = ref(false)
const loading = ref(false)
const pageError = ref('')
const changingPassword = ref(false)
const changingUsername = ref(false)
const adminUsername = ref('')
const usernameForm = reactive({ new_username: '', password: '' })
const usernameChanged = computed(
  () => usernameForm.new_username.trim() !== '' && usernameForm.new_username.trim() !== adminUsername.value,
)
const exporting = ref(false)
const restoring = ref(false)
const restorePhase = ref<'upload' | 'check'>('upload')
const restorePercent = ref(0)
const restoreTitle = computed(() => (restorePhase.value === 'upload' ? '正在上传备份' : '正在校验数据库'))
const restoreDetail = computed(() => {
  if (restorePhase.value === 'upload') return `已上传 ${restorePercent.value}%。`
  return '正在检查备份是否完整，数据库较大时需要一些时间。'
})

const form = reactive({
  theme: 'system',
  timezone: 'Asia/Shanghai',
})

const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' })
const settingsTab = ref('general')
const categoryTitle=computed(()=>categories.find(c=>c.key===settingsTab.value)?.label || '关于 Fonu')
const categoryDescription=computed(()=>categories.find(c=>c.key===settingsTab.value)?.description || '版本信息与更新日志')
const currentCanSave=computed(()=>saveGroups.includes(settingsTab.value as SettingsGroup))
const dirtyGroups=computed<Record<string,boolean>>(()=>Object.fromEntries(saveGroups.map(group=>[group,initialized.value && JSON.stringify(groupPayload(group))!==JSON.stringify(baseline.value[group]) ])))
const currentDirty=computed(()=>Boolean(dirtyGroups.value[settingsTab.value]))
const anyDirty=computed(()=>Object.values(dirtyGroups.value).some(Boolean)||globalNginxDirty.value||usernameChanged.value||!!usernameForm.password||!!passwordForm.old_password||!!passwordForm.new_password||!!passwordForm.confirm_password)
function switchTab(tab:string){if(saving.value||restoring.value)return;settingsTab.value=tab;void router.replace({name:'settings',query:{...route.query,tab}})}
const notifyType = ref<NotifyType>('')
const selectedNotifyType=ref<NotifyType>('webhook')
const notifyEnabled=computed({get:()=>Boolean(notifyType.value),set:(on:boolean)=>{notifyType.value=on?selectedNotifyType.value:''}})
const channelOptions=[
 {key:'wecom',label:'企业微信',icon:ChatbubbleEllipsesOutline,color:'#1484ef'},
 {key:'dingtalk',label:'钉钉',icon:SendOutline,color:'#1688ff'},
 {key:'feishu',label:'飞书',icon:PaperPlaneOutline,color:'#16a6a0'},
 {key:'telegram',label:'Telegram',icon:PaperPlaneOutline,color:'#229ed9'},
 {key:'bark',label:'Bark',icon:NotificationsCircleOutline,color:'#64748b'},
 {key:'ntfy',label:'ntfy',icon:TerminalOutline,color:'#30998c'},
 {key:'gotify',label:'Gotify',icon:NotificationsOutline,color:'#48a1c7'},
 {key:'email',label:'邮件 SMTP',icon:MailOutline,color:'var(--fonu-text-secondary)'},
 {key:'custom',label:'自定义 Webhook',icon:LinkOutline,color:'var(--fonu-text-secondary)'},
] as const
type ChannelKey=typeof channelOptions[number]['key']
const notifySignature=ref('')
const savedWebhookAppID=ref('')
const savedWebhookAPIBase=ref('')
const savedWebhookSigned=ref(false)

const notifyEventFlags = reactive<Record<string, boolean>>({
  on_ddns_ip_change: true,
  on_ddns_failure: true,
  on_cert_expiry: true,
  on_cert_renew_success: true,
  on_cert_renew_failure: true,
  on_ip_frequent_access: false,
  on_scan:false,
  on_cc:false,
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
const savedWebhookProvider=ref<WebhookProvider>('bark')
const notifyWebhook = reactive({
  provider: 'bark' as WebhookProvider,
  app_id:'',agent_id:'',recipient:'',recipient_type:'open_id',api_base:'',signed:false,
  server: DEFAULT_BARK_SERVER,
  topic: '',
  url: '',
  has_secret: false,
})
const isAppMode=computed(()=>notifyWebhook.provider==='wecom_app'||notifyWebhook.provider==='feishu_app')
const selectedChannel=computed<ChannelKey>(()=>selectedNotifyType.value==='email'?'email':selectedNotifyType.value==='telegram'?'telegram':notifyWebhook.provider==='wecom_app'?'wecom':notifyWebhook.provider==='feishu_app'?'feishu':notifyWebhook.provider)
const hasSavedWebhookSecret = computed(() => notifyWebhook.has_secret && notifyWebhook.provider === savedWebhookProvider.value && (!isAppMode.value||(notifyWebhook.app_id.trim()===savedWebhookAppID.value && notifyWebhook.api_base.trim()===savedWebhookAPIBase.value)))
const hasSavedSignature=computed(()=>hasSavedWebhookSecret.value&&savedWebhookSigned.value)
function setWebhookProvider(provider:WebhookProvider){
 if(notifyWebhook.provider===provider)return
 notifyWebhook.provider=provider;notifyWebhookKey.value='';notifySignature.value=''
 Object.assign(notifyWebhook,{app_id:'',agent_id:'',recipient:'',recipient_type:'open_id',api_base:'',signed:false,server:provider==='bark'?DEFAULT_BARK_SERVER:provider==='ntfy'?DEFAULT_NTFY_SERVER:'',topic:'',url:''})
}
function selectChannel(key:ChannelKey){
 if(key===selectedChannel.value)return
 const enabled=notifyEnabled.value
 selectedNotifyType.value=key==='email'||key==='telegram'?key:'webhook'
 if(selectedNotifyType.value==='webhook' && !(key==='wecom'&&notifyWebhook.provider==='wecom_app') && !(key==='feishu'&&notifyWebhook.provider==='feishu_app'))setWebhookProvider(key as WebhookProvider)
 if(enabled)notifyType.value=selectedNotifyType.value
}
function changeNotifyMode(mode:string){const platform=selectedChannel.value;if(platform==='wecom'||platform==='feishu')setWebhookProvider((mode==='app'?platform+'_app':platform) as WebhookProvider)}
function buildWebhookSecret(){
 const primary=resolveSecretInput(notifyWebhookKey.value,hasSavedWebhookSecret.value)
 if(notifyWebhook.signed && !isAppMode.value && (notifyWebhook.provider==='feishu'||notifyWebhook.provider==='dingtalk'))return 'fonu-robot-v1:'+JSON.stringify({primary,signature:resolveSecretInput(notifySignature.value,hasSavedSignature.value)})
 return primary
}
const notifyTelegram = reactive({
  chat_id: '',
  proxy_url: '',
  has_bot_token: false,
})
const ddnsInterval = ref(5)
const certThreshold = ref(30)
const logRetention = ref(30)
const analyticsRetention = ref(30)
const analyticsExcludePrivate = ref(true)
const analyticsSkipBackfill = ref(false)
const analyticsGeoRemote = ref(false)
const analyticsGeoRemoteURL = ref('')
const acmeEmail = ref('')
const acmeEmailInputStatus = computed(() => {
  const email = acmeEmail.value.trim()
  if (!email) return undefined
  return isValidAcmeEmail(email) ? undefined : 'error'
})
const zerosslApiKey = ref('')
const zerosslHasKey = ref(false)
const zerosslEabKid = ref('')
const zerosslEabKidHasKey = ref(false)
const zerosslEabHmac = ref('')
const zerosslEabHmacHasKey = ref(false)
const googleEabKid = ref('')
const googleEabKidHasKey = ref(false)
const googleEabHmac = ref('')
const googleEabHmacHasKey = ref(false)
const sslcomEabKid = ref('')
const sslcomEabKidHasKey = ref(false)
const sslcomEabHmac = ref('')
const sslcomEabHmacHasKey = ref(false)
const freesslEabKid = ref('')
const freesslEabKidHasKey = ref(false)
const freesslEabHmac = ref('')
const freesslEabHmacHasKey = ref(false)
const freesslDirectoryUrl = ref('')
const freesslAutomationToken = ref('')
const freesslAutomationTokenHasKey = ref(false)
const acmeDnsPropagationTimeout = ref(180)
const acmeDnsDisableAuthNS = ref(false)
const acmeDnsIgnorePropagation = ref(false)
const actalisEabKid = ref('')
const actalisEabKidHasKey = ref(false)
const actalisEabHmac = ref('')
const actalisEabHmacHasKey = ref(false)
const customAcmeDirectoryUrl = ref('')
const customAcmeEabKid = ref('')
const customAcmeEabKidHasKey = ref(false)
const customAcmeEabHmac = ref('')
const customAcmeEabHmacHasKey = ref(false)
const acmeProviders=computed(()=>[
 {key:'zerossl',label:'ZeroSSL',hint:'EAB Kid / Hmac',ready:zerosslEabKidHasKey.value&&zerosslEabHmacHasKey.value,draft:!!(zerosslEabKid.value||zerosslEabHmac.value)},
 {key:'google',label:'Google Trust Services',hint:'EAB Kid / Hmac',ready:googleEabKidHasKey.value&&googleEabHmacHasKey.value,draft:!!(googleEabKid.value||googleEabHmac.value)},
 {key:'sslcom',label:'SSL.com',hint:'EAB Kid / Hmac',ready:sslcomEabKidHasKey.value&&sslcomEabHmacHasKey.value,draft:!!(sslcomEabKid.value||sslcomEabHmac.value)},
 {key:'freessl',label:'FreeSSL',hint:'EAB 或 Automation Token',ready:freesslAutomationTokenHasKey.value||(freesslEabKidHasKey.value&&freesslEabHmacHasKey.value),draft:!!(freesslEabKid.value||freesslEabHmac.value||freesslAutomationToken.value)||freesslDirectoryUrl.value!==String(savedSettings.value.freessl_directory_url??'')},
 {key:'actalis',label:'Actalis',hint:'EAB Kid / Hmac',ready:actalisEabKidHasKey.value&&actalisEabHmacHasKey.value,draft:!!(actalisEabKid.value||actalisEabHmac.value)},
 {key:'custom',label:'自定义 ACME',hint:'Directory URL / 可选 EAB',ready:!!savedSettings.value.custom_acme_directory_url,draft:!!(customAcmeEabKid.value||customAcmeEabHmac.value)||customAcmeDirectoryUrl.value!==String(savedSettings.value.custom_acme_directory_url??'')},
])
const providerTitle=computed(()=>acmeProviders.value.find(item=>item.key===activeProvider.value)?.label || 'ACME 凭据')
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
let chinaCIDRLoading=false

const globalNginxFramework = ref('')
const globalNginxGeneratedSnippet = ref('')
const globalNginxDraft = ref('')
const globalNginxEditMode = ref<'auto' | 'custom'>('auto')
const globalNginxServerMode = ref<'auto' | 'custom'>('auto')
const nginxLoaded=ref(false),nginxLoading=ref(false),nginxLoadError=ref('')
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

function applySettingsToForm(settings: Record<string, string>, group?:SettingsGroup) {
  const applies=(key:SettingsGroup)=>!group||group===key
  if(applies('general')){
    Object.assign(form,{theme:settings.theme ?? themeMode.value,timezone:settings.timezone ?? 'Asia/Shanghai'})
    ddnsInterval.value=Number(settings.ddns_check_interval_minutes??5)
  }
  if(applies('notify'))applyNotifySettings(settings)
  if(applies('logs')){
  logRetention.value = Number(settings.log_retention_days ?? 30)
  analyticsRetention.value = Number(settings.analytics_retention_days ?? 30)
  analyticsExcludePrivate.value = settings.analytics_exclude_private_ips !== 'false'
  analyticsSkipBackfill.value = settings.analytics_skip_history_backfill === 'true'
  analyticsGeoRemote.value = settings.analytics_geo_remote_enabled === 'true'
  analyticsGeoRemoteURL.value =
    settings.analytics_geo_remote_url ??
    'http://ip-api.com/batch?fields=status,country,regionName,city,query&lang=zh-CN'
  }
  if(applies('acme')){
  certThreshold.value = Number(settings.cert_renew_threshold_days ?? 30)
  acmeEmail.value = settings.acme_email ?? ''
  zerosslHasKey.value = Boolean(settings.zerossl_api_key?.trim())
  zerosslApiKey.value = ''
  zerosslEabKidHasKey.value = Boolean(settings.zerossl_eab_kid?.trim())
  zerosslEabKid.value = ''
  zerosslEabHmacHasKey.value = Boolean(settings.zerossl_eab_hmac_key?.trim())
  zerosslEabHmac.value = ''
  googleEabKidHasKey.value = Boolean(settings.google_eab_kid?.trim())
  googleEabKid.value = ''
  googleEabHmacHasKey.value = Boolean(settings.google_eab_hmac?.trim())
  googleEabHmac.value = ''
  sslcomEabKidHasKey.value = Boolean(settings.sslcom_eab_kid?.trim())
  sslcomEabKid.value = ''
  sslcomEabHmacHasKey.value = Boolean(settings.sslcom_eab_hmac?.trim())
  sslcomEabHmac.value = ''
  freesslEabKidHasKey.value = Boolean(settings.freessl_eab_kid?.trim())
  freesslEabKid.value = ''
  freesslEabHmacHasKey.value = Boolean(settings.freessl_eab_hmac?.trim())
  freesslEabHmac.value = ''
  freesslDirectoryUrl.value = settings.freessl_directory_url ?? ''
  freesslAutomationTokenHasKey.value = Boolean(settings.freessl_automation_token?.trim())
  freesslAutomationToken.value = ''
  acmeDnsPropagationTimeout.value = Number(settings.acme_dns_propagation_timeout_sec || 180) || 180
  acmeDnsDisableAuthNS.value = settings.acme_dns_disable_auth_ns === '1'
  acmeDnsIgnorePropagation.value = settings.acme_dns_ignore_propagation === '1'
  actalisEabKidHasKey.value = Boolean(settings.actalis_eab_kid?.trim())
  actalisEabKid.value = ''
  actalisEabHmacHasKey.value = Boolean(settings.actalis_eab_hmac?.trim())
  actalisEabHmac.value = ''
  customAcmeDirectoryUrl.value = settings.custom_acme_directory_url ?? ''
  customAcmeEabKidHasKey.value = Boolean(settings.custom_acme_eab_kid?.trim())
  customAcmeEabKid.value = ''
  customAcmeEabHmacHasKey.value = Boolean(settings.custom_acme_eab_hmac?.trim())
  customAcmeEabHmac.value = ''
  }
  if(applies('security')){
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
    header: trustedProxyPreset.value==='cloudflare'?'CF-Connecting-IP':trustedProxyHeader.value,
    recursive: trustedProxyPreset.value!=='cloudflare'&&trustedProxyHeader.value==='X-Forwarded-For',
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
  selectedNginxBackup.value = null
  globalNginxDraft.value =
    view.mode === 'auto' ? globalNginxGeneratedSnippet.value : view.content || globalNginxCustomHint
  globalNginxDirty.value = false
}

async function loadGlobalNginx() {
  if(nginxLoading.value||globalNginxDirty.value)return
  nginxLoading.value=true;nginxLoadError.value=''
  try{applyGlobalNginxView(await api.getGlobalNginx());nginxLoaded.value=true}
  catch(error){nginxLoadError.value=error instanceof Error?error.message:'读取 Nginx 配置失败'}
  finally{nginxLoading.value=false}
}

function setGlobalNginxEditMode(mode: 'auto' | 'custom') {
  if (mode === globalNginxEditMode.value) return
  if (mode === 'custom') {
    if (!globalNginxDraft.value.trim() || globalNginxDraft.value === globalNginxGeneratedSnippet.value) {
      globalNginxDraft.value = globalNginxCustomHint
      globalNginxDirty.value = globalNginxServerMode.value !== 'custom'
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
  if(!nginxLoaded.value||globalNginxSaving.value)return
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

function confirmNginxRollback(){
  if(globalNginxSaving.value)return
  dialog.warning({title:'回滚 Nginx 配置',content:'将恢复所选历史版本（未选择时恢复上一版本），并重新应用 Nginx。',positiveText:'确认回滚',negativeText:'取消',onPositiveClick:()=>rollbackGlobalNginx(selectedNginxBackup.value||undefined)})
}
async function rollbackGlobalNginx(backup?: string) {
  if(!nginxLoaded.value||globalNginxSaving.value)return
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
  if(!nginxLoaded.value||globalNginxSaving.value)return
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
  if(disposed||chinaCIDRLoading||settingsTab.value!=='security')return
  chinaCIDRLoading=true
  try {
    const status=await api.getChinaCIDRStatus()
    if(disposed||settingsTab.value!=='security')return
    chinaCIDR.value = status
    if (chinaCIDR.value.updating) {
      startChinaCIDRPoll()
    } else {
      stopChinaCIDRPoll()
    }
  } catch {
    // Status remains available for a manual retry.
  } finally {chinaCIDRLoading=false}
}

function stopChinaCIDRPoll() {
  if (chinaCIDRPollTimer) {
    clearInterval(chinaCIDRPollTimer)
    chinaCIDRPollTimer = undefined
  }
}

function startChinaCIDRPoll() {
  if (chinaCIDRPollTimer||disposed||settingsTab.value!=='security') return
  chinaCIDRPollTimer = setInterval(() => {if(!document.hidden)void loadChinaCIDRStatus()}, 2000)
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
  disposed=true
  stopChinaCIDRPoll()
})

function applyNotifySettings(settings: Record<string, string>) {
  notifyType.value = (settings.notify_type as NotifyType) || ''
  notifySMTPPassword.value = ''
  notifyWebhookKey.value = ''
  notifySignature.value=''
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
    savedWebhookProvider.value=notifyWebhook.provider
    Object.assign(notifyWebhook,{app_id:webhook.app_id??'',agent_id:webhook.agent_id??'',recipient:webhook.recipient??'',recipient_type:webhook.recipient_type??'open_id',api_base:webhook.api_base??'',signed:webhook.signed??false})
    savedWebhookAppID.value=notifyWebhook.app_id.trim()
    savedWebhookAPIBase.value=notifyWebhook.api_base.trim()
    savedWebhookSigned.value=notifyWebhook.signed
    selectedNotifyType.value=notifyType.value||webhook.preferred_type||'webhook'
    notifyWebhook.server = webhook.server ?? (webhook.provider === 'ntfy' ? DEFAULT_NTFY_SERVER : DEFAULT_BARK_SERVER)
    notifyWebhook.topic = webhook.topic ?? ''
    notifyWebhook.url = webhook.url ?? ''
    notifyWebhook.has_secret = webhook.has_secret ?? false
  } catch {
    Object.assign(notifyWebhook,{provider:'bark',app_id:'',agent_id:'',recipient:'',recipient_type:'open_id',api_base:'',signed:false,has_secret:false,server:DEFAULT_BARK_SERVER,topic:'',url:''})
    savedWebhookProvider.value='bark'
    savedWebhookAppID.value='';savedWebhookAPIBase.value='';savedWebhookSigned.value=false
    selectedNotifyType.value=notifyType.value||'webhook'
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

  if (settings.notify_type === undefined && settings.notify_webhook_url) {
    notifyType.value = 'webhook'
    notifyWebhook.provider = 'custom'
    savedWebhookProvider.value='custom'
    notifyWebhook.url = settings.notify_webhook_url
    selectedNotifyType.value='webhook'
  }

  const flagOn = (key: string) => settings[key] !== '0' && settings[key] !== 'false'
  const flagOptIn = (key: string) => settings[key] === '1' || settings[key] === 'true'

  notifyEventFlags.on_scan=flagOptIn('notify_on_scan')
  notifyEventFlags.on_cc=flagOptIn('notify_on_cc')
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
    app_id:notifyWebhook.app_id.trim(),agent_id:notifyWebhook.agent_id.trim(),recipient:notifyWebhook.recipient.trim(),recipient_type:notifyWebhook.recipient_type,api_base:notifyWebhook.api_base.trim(),signed:notifyWebhook.signed,preferred_type:selectedNotifyType.value,
    server: notifyWebhook.server.trim(),
    key: '',
    topic: notifyWebhook.topic.trim(),
    url: notifyWebhook.url.trim(),
    has_secret: hasSavedWebhookSecret.value || Boolean(notifyWebhookKey.value.trim()),
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
    on_scan:notifyEventFlags.on_scan,
    on_cc:notifyEventFlags.on_cc,
    on_login_failure: notifyEventFlags.on_login_failure,
    on_nginx_reload_failure: notifyEventFlags.on_nginx_reload_failure,
    ip_frequent_threshold: notifyIPFrequentThreshold.value,
    ip_frequent_window_sec: notifyIPFrequentWindowSec.value,
    login_failure_threshold: notifyLoginFailureThreshold.value,
    login_failure_window_sec: notifyLoginFailureWindowSec.value,
    smtp_password: resolveSecretInput(notifySMTPPassword.value, notifyEmail.has_password),
    webhook_secret: buildWebhookSecret(),
    telegram_token: resolveSecretInput(notifyTelegramToken.value, notifyTelegram.has_bot_token),
  }
}

async function testNotify() {
  if(notifyTesting.value||saving.value)return
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
    notify_webhook_secret: buildWebhookSecret(),
    notify_telegram_json: buildNotifyTelegramJSON(),
    notify_telegram_token: resolveSecretInput(notifyTelegramToken.value, notifyTelegram.has_bot_token),
    notify_on_ddns_ip_change: notifyEventFlags.on_ddns_ip_change ? '1' : '0',
    notify_on_ddns_failure: notifyEventFlags.on_ddns_failure ? '1' : '0',
    notify_on_cert_expiry: notifyEventFlags.on_cert_expiry ? '1' : '0',
    notify_on_cert_renew_success: notifyEventFlags.on_cert_renew_success ? '1' : '0',
    notify_on_cert_renew_failure: notifyEventFlags.on_cert_renew_failure ? '1' : '0',
    notify_on_ip_frequent_access: notifyEventFlags.on_ip_frequent_access ? '1' : '0',
    notify_on_scan:notifyEventFlags.on_scan?'1':'0',
    notify_on_cc:notifyEventFlags.on_cc?'1':'0',
    notify_on_login_failure: notifyEventFlags.on_login_failure ? '1' : '0',
    notify_on_nginx_reload_failure: notifyEventFlags.on_nginx_reload_failure ? '1' : '0',
    notify_ip_frequent_threshold: String(notifyIPFrequentThreshold.value || 100),
    notify_ip_frequent_window_sec: String(notifyIPFrequentWindowSec.value || 60),
    notify_login_failure_threshold: String(notifyLoginFailureThreshold.value || DEFAULT_LOGIN_FAILURE_THRESHOLD),
    notify_login_failure_window_sec: String(notifyLoginFailureWindowSec.value || DEFAULT_LOGIN_FAILURE_WINDOW_SEC),
    ddns_check_interval_minutes: String(ddnsInterval.value),
    cert_renew_threshold_days: String(certThreshold.value),
    log_retention_days: String(logRetention.value),
    analytics_retention_days: String(analyticsRetention.value),
    analytics_exclude_private_ips: analyticsExcludePrivate.value ? 'true' : 'false',
    analytics_skip_history_backfill: analyticsSkipBackfill.value ? 'true' : 'false',
    analytics_geo_remote_enabled: analyticsGeoRemote.value ? 'true' : 'false',
    analytics_geo_remote_url: analyticsGeoRemoteURL.value.trim(),
    acme_email: acmeEmail.value.trim(),
    trusted_proxy_json: buildTrustedProxyJSON(),
    global_ip_blacklist: buildGlobalIPListJSON(globalIPBlacklistText.value),
    global_ip_whitelist: buildGlobalWhitelistJSON(),
    china_cidr_update_interval_hours: String(chinaCIDRHours.value),
  }
  if (notifyType.value === 'webhook' && notifyWebhook.provider === 'custom') {
    payload.notify_webhook_url = notifyWebhook.url.trim()
  }
  const zKid = zerosslEabKid.value.trim()
  if (zKid) {
    payload.zerossl_eab_kid = zKid
  }
  const zHmac = zerosslEabHmac.value.trim()
  if (zHmac) {
    payload.zerossl_eab_hmac_key = zHmac
  }
  const gKid = googleEabKid.value.trim()
  if (gKid) {
    payload.google_eab_kid = gKid
  }
  const gHmac = googleEabHmac.value.trim()
  if (gHmac) {
    payload.google_eab_hmac = gHmac
  }
  const sKid = sslcomEabKid.value.trim()
  if (sKid) {
    payload.sslcom_eab_kid = sKid
  }
  const sHmac = sslcomEabHmac.value.trim()
  if (sHmac) {
    payload.sslcom_eab_hmac = sHmac
  }
  const fKid = freesslEabKid.value.trim()
  if (fKid) {
    payload.freessl_eab_kid = fKid
  }
  const fHmac = freesslEabHmac.value.trim()
  if (fHmac) {
    payload.freessl_eab_hmac = fHmac
  }
  const fDir = freesslDirectoryUrl.value.trim().replace(/`/g, '')
  payload.freessl_directory_url = fDir
  const fAuto = freesslAutomationToken.value.trim()
  if (fAuto) {
    payload.freessl_automation_token = fAuto
  }
  payload.acme_dns_propagation_timeout_sec = String(acmeDnsPropagationTimeout.value || 180)
  payload.acme_dns_disable_auth_ns = acmeDnsDisableAuthNS.value ? '1' : '0'
  payload.acme_dns_ignore_propagation = acmeDnsIgnorePropagation.value ? '1' : '0'
  const aKid = actalisEabKid.value.trim()
  if (aKid) {
    payload.actalis_eab_kid = aKid
  }
  const aHmac = actalisEabHmac.value.trim()
  if (aHmac) {
    payload.actalis_eab_hmac = aHmac
  }
  const cDir = customAcmeDirectoryUrl.value.trim().replace(/`/g, '')
  payload.custom_acme_directory_url = cDir
  const cKid = customAcmeEabKid.value.trim()
  if (cKid) {
    payload.custom_acme_eab_kid = cKid
  }
  const cHmac = customAcmeEabHmac.value.trim()
  if (cHmac) {
    payload.custom_acme_eab_hmac = cHmac
  }
  return payload
}

function groupPayload(group:SettingsGroup):Record<string,string>{
  const belongs=(key:string)=>{
    if(group==='general')return ['theme','timezone','ddns_check_interval_minutes'].includes(key)
    if(group==='security')return ['trusted_proxy_json','global_ip_blacklist','global_ip_whitelist','china_cidr_update_interval_hours'].includes(key)
    if(group==='notify')return key.startsWith('notify_')
    if(group==='logs')return key==='log_retention_days'||key.startsWith('analytics_')
    return key==='cert_renew_threshold_days'||/^(acme_|zerossl_|google_|sslcom_|freessl_|actalis_|custom_acme_)/.test(key)
  }
  return Object.fromEntries(Object.entries(buildSavePayload()).filter(([key])=>belongs(key)))
}
function captureBaseline(group:SettingsGroup){baseline.value={...baseline.value,[group]:groupPayload(group)}}
function discardCurrent(){
  if(!currentCanSave.value||saving.value)return
  const group=settingsTab.value as SettingsGroup
  applySettingsToForm(savedSettings.value,group)
  if(group==='general')setThemeMode(form.theme)
  captureBaseline(group)
}
async function loadAdminUsername() {
  const status = await api.authStatus()
  adminUsername.value = status.username ?? ''
  usernameForm.new_username = adminUsername.value
  usernameForm.password = ''
}

async function load() {
  loading.value = true
  pageError.value = ''
  try {
    savedSettings.value=await api.getSettings()
    applySettingsToForm(savedSettings.value)
    // Local theme remains the initial preview; server theme is changed only explicitly.
    form.theme=themeMode.value
    savedSettings.value={...savedSettings.value,theme:form.theme}
    for(const group of saveGroups)captureBaseline(group)
    initialized.value=true
    void loadCategory(settingsTab.value)
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

async function save(){
  if(saving.value||loading.value||!currentCanSave.value||!currentDirty.value)return
  const group=settingsTab.value as SettingsGroup
  if(group==='acme'){
    const err=acmeEmail.value.trim()?validateAcmeEmailMessage(acmeEmail.value):''
    if(err){message.warning(err);return}
  }
  const draft=groupPayload(group),original=baseline.value[group]||{}
  const changes=Object.fromEntries(Object.entries(draft).filter(([key,value])=>value!==original[key]))
  if(!Object.keys(changes).length)return
  saving.value=true
  try{
    // Notify is a cohesive config; the API resolves unchanged secrets server-side.
    const saved=await api.saveSettings(group==='notify'?draft:changes)
    savedSettings.value=saved
    applySettingsToForm(saved,group)
    if(group==='general'){setThemeMode(form.theme);invalidateAppTimezoneCache()}
    captureBaseline(group)
    message.success('当前分类已保存')
  }catch(error){message.error(error instanceof Error?error.message:'保存失败')}
  finally{saving.value=false}
}

async function changeUsername() {
  if(changingUsername.value||changingPassword.value)return
  const newUsername = usernameForm.new_username.trim()
  if (!newUsername) {
    message.warning('请填写登录用户名')
    return
  }
  if (!usernameForm.password) {
    message.warning('修改用户名需填写当前密码')
    return
  }
  changingUsername.value = true
  try {
    await api.changeUsername(usernameForm.password, newUsername)
    adminUsername.value = newUsername
    usernameForm.new_username = newUsername
    usernameForm.password = ''
    message.success('用户名已更新')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新用户名失败')
  } finally {
    changingUsername.value = false
  }
}

async function changePassword() {
  if(changingUsername.value||changingPassword.value)return
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
    message.success('密码已更新，请重新登录')
    initialized.value=false
    await router.replace({ name: 'login' })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新密码失败')
  } finally {
    changingPassword.value = false
  }
}

async function exportBackup() {
  if(exporting.value||restoring.value)return
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

function restoreRequest({ file, onFinish, onError }: UploadCustomRequestOptions) {
  const raw = file.file
  if(restoring.value||exporting.value){onError();return}
  if (!raw) {
    message.error('请选择备份文件')
    onError()
    return
  }
  if(!/\.(tar\.gz|tgz)$/i.test(raw.name)||raw.size===0||raw.size>511*1024*1024){message.warning('请选择非空的 tar.gz / tgz 备份文件（最大 511 MiB）');onError();return}
  dialog.warning({
    title: '恢复配置',
    content: '文件：'+raw.name+'（'+(raw.size/1024/1024).toFixed(2)+' MiB）。将校验备份并覆盖当前数据库、证书和 Nginx 配置，完成后需要重新登录。',
    positiveText: '恢复',
    negativeText: '取消',
    onNegativeClick: () => {
      onError()
    },
    onPositiveClick: () => {
      void runRestore(raw, onFinish, onError)
    },
  })
}

async function runRestore(
  raw: File,
  onFinish: () => void,
  onError: () => void,
) {
  restorePhase.value = 'upload'
  restorePercent.value = 0
  restoring.value = true
  try {
    await api.restoreBackup(raw, (percent) => {
      restorePercent.value = percent
      if (percent >= 100) restorePhase.value = 'check'
    })
    onFinish()
    restoring.value = false
    initialized.value=false
    await router.push({ name: 'login', state: { restored: true } })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '恢复失败')
    onError()
    restoring.value = false
  }
}

function applySettingsTab(tab: unknown) {
  if (typeof tab === 'string' && SETTINGS_TAB_NAMES.has(tab)) {
    settingsTab.value = tab
  }
}

let adminLoaded=false,adminLoading=false,disposed=false
async function loadCategory(tab:string){
 if(disposed||!initialized.value)return
 if(tab==='security')void loadChinaCIDRStatus()
 else stopChinaCIDRPoll()
 if(tab==='advanced'&&!nginxLoaded.value)void loadGlobalNginx()
 if(tab==='account'&&!adminLoaded&&!adminLoading){
  adminLoading=true
  try{await loadAdminUsername();adminLoaded=true}
  catch(error){message.error(error instanceof Error?error.message:'读取管理员账户失败')}
  finally{adminLoading=false}
 }
}
watch(settingsTab,tab=>void loadCategory(tab))
watch(() => route.query.tab, applySettingsTab)
onBeforeRouteLeave(()=>{if(saving.value||restoring.value)return false;if(!initialized.value||!anyDirty.value)return true;return new Promise<boolean>(resolve=>dialog.warning({title:'离开系统设置？',content:'尚有未保存的设置，离开后将丢失草稿。',positiveText:'离开',negativeText:'继续编辑',onPositiveClick:()=>{if(dirtyGroups.value.general)setThemeMode(savedSettings.value.theme||'system');resolve(true)},onNegativeClick:()=>resolve(false),onClose:()=>resolve(false),onMaskClick:()=>resolve(false)}))})

onMounted(() => {
  applySettingsTab(route.query.tab)
  void load()
})
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

.settings-switches--col {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: 2px;
}

.settings-switch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 32px;
  font-size: 13px;
  color: var(--fonu-text);
}

.settings-switch > span {
  flex: 1;
  min-width: 0;
  line-height: 1.45;
  color: var(--fonu-text-secondary);
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

.acme-provider-block {
  padding: var(--fonu-space-4);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  background: var(--fonu-bg);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.acme-provider-block__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 16px;
}

.acme-provider-block__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
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

.settings-fields--password {
  margin-top: var(--fonu-space-6);
  padding-top: var(--fonu-space-6);
  border-top: 1px solid var(--fonu-border-subtle);
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

.restore-progress {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.restore-progress p {
  margin: 2px 0 0;
  line-height: 1.6;
  color: var(--fonu-text-secondary);
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
.accent-options { display: flex; flex-wrap: wrap; gap: 8px; }
.accent-option { display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 8px; min-width: 60px; border: 1px solid transparent; border-radius: var(--fonu-radius-sm); background: transparent; color: var(--fonu-text-secondary); cursor: pointer; font: inherit; font-size: 12px; }
.accent-option[aria-pressed="true"] { border-color: var(--fonu-brand); background: var(--fonu-brand-soft); color: var(--fonu-text); }
.accent-option:focus-visible { outline: 2px solid var(--fonu-brand); outline-offset: 2px; }
.accent-swatch { width: 28px; height: 28px; border-radius: 50%; display: grid; place-items: center; font-weight: 700; }
@media (max-width: 760px) {
  .settings-grid { min-width: 0; }
}
.settings-grid { gap:12px; align-items:start; }
.settings-grid :deep(.fonu-card) { height:auto; }
.settings-grid :deep(.fonu-card__header) { padding-bottom:12px; border-bottom:1px solid var(--fonu-border); }
.theme-card { min-height:64px; border-radius:var(--fonu-radius-sm); color:var(--fonu-text); }
.theme-card:hover { border-color:var(--fonu-brand-hover); }
.theme-card--active { border-color:var(--fonu-brand); background:var(--fonu-brand-soft); }
.theme-card--active .theme-card__icon,.theme-card__check { color:var(--fonu-brand); }
.workspace-settings :deep(.n-tabs-nav) { border-bottom:1px solid var(--fonu-border); }
@media(max-width:960px) { .settings-grid { grid-template-columns:minmax(0,1fr); } }

.settings-workspace { display:grid; gap:14px; min-width:0; container-type:inline-size; }
.settings-navigation { display:grid; grid-template-columns:repeat(8,minmax(0,1fr)); gap:6px; padding:8px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); background:var(--fonu-surface); }
.settings-nav-item { position:relative; display:flex; align-items:center; justify-content:center; flex-direction:column; gap:6px; min-width:0; min-height:64px; border:1px solid transparent; border-radius:8px; background:transparent; color:var(--fonu-text-secondary); cursor:pointer; font:inherit; font-size:12px; padding:8px 4px; white-space:nowrap; }
.settings-nav-item .n-icon { font-size:21px; }
.settings-nav-item:hover { background:var(--fonu-bg-muted); }
.settings-nav-item.is-active { color:var(--fonu-brand-text); background:var(--fonu-brand-soft); border-color:color-mix(in srgb,var(--fonu-brand) 28%,transparent); font-weight:600; }
.dirty-dot { position:absolute; right:7px; top:7px; height:7px; width:7px; border-radius:50%; background:var(--fonu-warning); }
.settings-category-head { display:flex; align-items:center; justify-content:space-between; gap:12px; }
.settings-category-head h2 { font-size:18px; margin:0; color:var(--fonu-text); }
.settings-category-head p { font-size:12px; color:var(--fonu-text-secondary); margin:4px 0 0; }
.settings-content { padding:0; margin:0; border:0; min-width:0; }
.settings-grid { gap:12px; align-items:start; }
.settings-grid :deep(.fonu-card) { height:auto; }
.settings-grid :deep(.fonu-card__body) { gap:10px; }
.settings-block--spaced { margin-top:10px; padding-top:12px; border-top:1px solid var(--fonu-border); }
.theme-cards { max-width:560px; }
.theme-card { min-height:58px; flex-direction:row; padding:10px; }
.theme-card--active { color:var(--fonu-brand-text); background:var(--fonu-brand-soft); border-color:var(--fonu-brand); }
.theme-card__check { position:static; margin-left:auto; color:var(--fonu-brand); }
.settings-fields { gap:14px; }
.save-bar { position:sticky; bottom:0; z-index:2; display:flex; align-items:center; justify-content:space-between; gap:12px; flex-wrap:wrap; margin:0; padding:12px 16px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); background:var(--fonu-surface); box-shadow:0 -3px 12px #00000008; }
.save-bar strong { font-size:13px; color:var(--fonu-text); }
.save-bar small { display:block; font-size:11px; margin-top:3px; color:var(--fonu-text-secondary); }
.save-bar-actions { display:flex; gap:8px; margin-left:auto; }
.provider-list { display:grid; }
.provider-row { display:flex; align-items:center; gap:12px; padding:12px 0; border-bottom:1px solid var(--fonu-border); }
.provider-row:last-child { border-bottom:0; }
.provider-row > span:first-child { flex:1; min-width:0; }
.provider-row strong { font-size:14px; display:block; overflow-wrap:anywhere; }
.provider-row small { display:block; color:var(--fonu-text-secondary); font-size:12px; margin-top:3px; }
.provider-editor { display:grid; gap:14px; max-height:calc(100dvh - 160px); overflow:auto; padding:0 2px; }
.provider-edit-block { border:0; padding:0; margin:0; }
.provider-modal-actions { display:flex; justify-content:flex-end; }
.notify-form { display:grid; grid-template-columns:minmax(0,1fr) minmax(0,1fr); gap:16px; align-items:start; }
.notify-channel { display:grid; gap:14px; min-width:0; }
.notify-events { margin-top:0; min-width:0; }
.notify-test-row { grid-column:1/-1; }
.backup-warning { margin-top:8px; }
@container(max-width:960px){.settings-navigation{grid-template-columns:repeat(4,minmax(0,1fr));}}
@container(max-width:730px){.notify-form{grid-template-columns:minmax(0,1fr)}.notify-test-row{grid-column:auto}}
@container(max-width:420px){.settings-navigation{gap:3px;padding:5px}.settings-nav-item{font-size:11px;min-height:57px}.settings-nav-item .n-icon{font-size:17px}.provider-row{gap:6px}.save-bar{padding:10px}.theme-cards{grid-template-columns:repeat(3,minmax(0,1fr))}.theme-card{flex-direction:column;min-height:56px}.theme-card__check{display:none}}
.settings-load-error{display:flex;align-items:center;gap:10px;color:var(--fonu-error);margin-bottom:12px}
.provider-editor .provider-edit-block {background:transparent;padding:0;gap:16px;display:flex;flex-direction:column;}
.provider-edit-block .acme-provider-block__head {justify-content:flex-end;padding:4px 0 10px;border-bottom:1px solid var(--fonu-border);margin-bottom:4px;}
.provider-modal-actions {padding-top:4px;}
.global-nginx-head {padding:10px;border:1px solid var(--fonu-border);border-radius:8px;background:var(--fonu-bg);margin:12px 0 10px;}
.log-panel-actions {display:flex;align-items:center;flex-wrap:wrap;gap:8px;}
.nginx-mode-toggle {gap:6px;}
.notify-channel-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px;margin-top:8px}
.notify-channel-choice{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:7px;min-height:72px;padding:8px 4px;border:1px solid var(--fonu-border);background:var(--fonu-surface);border-radius:8px;color:var(--fonu-text);font:inherit;font-size:12px;cursor:pointer;min-width:0;overflow-wrap:anywhere}
.notify-channel-choice.is-active{background:var(--fonu-brand-soft);border-color:var(--fonu-brand);color:var(--fonu-brand-text)}
.notify-channel-icon{font-size:23px;display:flex}
.notify-config-panel{display:grid;gap:12px;border:1px solid var(--fonu-border);border-radius:8px;padding:12px;min-width:0;background:var(--fonu-surface)}
.notify-config-panel h3{font-size:15px;margin:0}
.notify-config-panel .notify-test-row{display:flex;flex-direction:column;align-items:flex-end;gap:4px;margin:0}
.notify-events-disabled{opacity:.7}
</style>
