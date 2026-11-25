<template>
  <div class="security-section">
              <div class="security-header">
                <div class="security-header__row">
                  <div class="security-header__status">
                    <span class="security-header__status-label">已启用</span>
                    <div v-if="activeSecurityFeatures.length" class="security-header__tags">
                      <n-tag
                        v-for="tag in activeSecurityFeatures"
                        :key="tag"
                        size="small"
                        round
                        :bordered="false"
                      >
                        {{ tag }}
                      </n-tag>
                    </div>
                    <span v-else class="security-header__empty">暂无</span>
                  </div>
                  <n-button type="primary" size="tiny" class="security-header__preset" @click="emit('preset')">
                    <template #icon><n-icon :component="FlashOutline" :size="14" /></template>
                    一键推荐
                  </n-button>
                </div>
                <p class="security-header__note">
                  IP 策略依赖「设置 → 信任代理」；白名单 IP 可豁免「仅中国大陆」限制。
                </p>
              </div>

              <n-collapse :expanded-names="securityExpanded" @update:expanded-names="updateExpanded" class="security-collapse">
                <n-collapse-item title="IP 访问控制" name="ip">
                  <div class="security-panel">
                    <div class="security-option">
                      <div class="security-option__text">
                        <div class="security-option__label">仅中国大陆 IP</div>
                        <div class="security-option__hint">
                          需先在设置页更新中国 IP 段；内网（10/8、172.16/12、192.168/16 等）默认放行
                        </div>
                      </div>
                      <n-switch :value="form.china_only" @update:value="emit('patch', { china_only: Boolean($event) })" size="small" />
                    </div>

                    <div class="security-option">
                      <div class="security-option__text">
                        <div class="security-option__label">黑名单模式</div>
                        <div class="security-option__hint">启用后拒绝列表中的 IP 访问</div>
                      </div>
                      <n-switch :value="form.ip_blacklist_mode" @update:value="emit('patch', { ip_blacklist_mode: Boolean($event) })" size="small" />
                    </div>
                    <div v-if="form.ip_blacklist_mode" class="security-panel__fields">
                      <n-form-item label="IP 黑名单">
                        <n-input
                          :value="form.ip_blacklist_text" @update:value="emit('patch', { ip_blacklist_text: $event })"
                          type="textarea"
                          :rows="2"
                          placeholder="每行一个 IP 或 CIDR"
                        />
                      </n-form-item>
                    </div>

                    <div class="security-option">
                      <div class="security-option__text">
                        <div class="security-option__label">白名单模式</div>
                        <div class="security-option__hint">启用后仅允许白名单 IP 访问</div>
                      </div>
                      <n-switch :value="form.ip_whitelist_mode" @update:value="emit('patch', { ip_whitelist_mode: Boolean($event) })" size="small" />
                    </div>
                    <div v-if="form.ip_whitelist_mode" class="security-panel__fields">
                      <n-form-item label="IP 白名单">
                        <n-input
                          :value="form.ip_whitelist_text" @update:value="emit('patch', { ip_whitelist_text: $event })"
                          type="textarea"
                          :rows="2"
                          placeholder="每行一个 IP 或 CIDR"
                        />
                      </n-form-item>
                    </div>
                  </div>
                </n-collapse-item>

                <n-collapse-item title="认证" name="auth">
                  <div class="security-panel">
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">Basic Auth</div>
                      <div class="security-option__hint">浏览器弹窗认证</div>
                    </div>
                    <n-switch :value="form.basic_auth_enabled" @update:value="emit('patch', { basic_auth_enabled: Boolean($event) })" size="small" />
                  </div>
                  <div v-if="form.basic_auth_enabled" class="security-fields-grid">
                    <n-form-item label="用户名">
                      <n-input :value="form.basic_auth_username" @update:value="emit('patch', { basic_auth_username: $event })" placeholder="用户名" />
                    </n-form-item>
                    <n-form-item>
                      <template #label>
                        <span class="proxy-secret-label">
                          密码
                          <n-tag
                            v-if="hasPassword"
                            size="small"
                            type="success"
                            :bordered="false"
                          >
                            {{ CONFIGURED_SECRET_TAG }}
                          </n-tag>
                        </span>
                      </template>
                      <n-input
                        :value="form.basic_auth_password" @update:value="emit('patch', { basic_auth_password: $event })"
                        type="password"
                        show-password-on="click"
                        :placeholder="
                          hasPassword
                            ? CONFIGURED_SECRET_PLACEHOLDER
                            : '至少 8 位'
                        "
                      />
                    </n-form-item>
                  </div>
                  </div>
                </n-collapse-item>

                <n-collapse-item title="流量控制" name="traffic">
                  <div class="security-panel">
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">请求限流</div>
                      <div class="security-option__hint">超出速率返回 429</div>
                    </div>
                    <n-switch :value="form.rate_limit_enabled" @update:value="emit('patch', { rate_limit_enabled: Boolean($event) })" size="small" />
                  </div>
                  <div v-if="form.rate_limit_enabled" class="security-fields-grid">
                    <n-form-item label="每秒请求数">
                      <n-input-number :value="form.rate_limit_rate" @update:value="emit('patch', { rate_limit_rate: $event ?? 0 })" :min="1" :max="10000" class="w-full" />
                    </n-form-item>
                    <n-form-item label="突发上限">
                      <n-input-number :value="form.rate_limit_burst" @update:value="emit('patch', { rate_limit_burst: $event ?? 0 })" :min="1" :max="100000" class="w-full" />
                    </n-form-item>
                  </div>

                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">连接数限制</div>
                      <div class="security-option__hint">每 IP 最大并发连接</div>
                    </div>
                    <n-switch :value="form.conn_limit_enabled" @update:value="emit('patch', { conn_limit_enabled: Boolean($event) })" size="small" />
                  </div>
                  <div v-if="form.conn_limit_enabled" class="security-panel__fields">
                    <n-form-item label="最大连接数">
                      <n-input-number :value="form.conn_limit_max" @update:value="emit('patch', { conn_limit_max: $event ?? 0 })" :min="1" :max="10000" class="conn-limit-input" />
                    </n-form-item>
                  </div>
                  </div>
                </n-collapse-item>

                <n-collapse-item title="高级" name="advanced">
                  <div class="security-panel">
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">忽略后端 TLS 证书</div>
                      <div class="security-option__hint">上游为自签 https:// 时使用</div>
                    </div>
                    <n-switch :value="form.proxy_ssl_verify_off" @update:value="emit('patch', { proxy_ssl_verify_off: Boolean($event) })" size="small" />
                  </div>
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">使用目标 Host 头</div>
                      <div class="security-option__hint">Host 改为上游地址。1Panel、LitePan 这类会校验来源的应用会因此拒绝请求</div>
                    </div>
                    <n-switch :value="form.proxy_host_upstream" @update:value="emit('patch', { proxy_host_upstream: Boolean($event) })" size="small" />
                  </div>
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">仅 TLS 1.3</div>
                    </div>
                    <n-switch :value="form.tls_min_13_only" @update:value="emit('patch', { tls_min_13_only: Boolean($event) })" size="small" :disabled="!form.https_enabled" />
                  </div>
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">安全响应头</div>
                      <div class="security-option__hint">HSTS、X-Frame-Options 等</div>
                    </div>
                    <n-switch :value="form.security_headers" @update:value="emit('patch', { security_headers: Boolean($event) })" size="small" :disabled="!form.https_enabled" />
                  </div>
                  </div>
                </n-collapse-item>
              </n-collapse>
              </div>
</template>
<script setup lang="ts">
import { NButton,NIcon,NTag,NCollapse,NCollapseItem,NSwitch,NFormItem,NInput,NInputNumber } from 'naive-ui'
import { FlashOutline } from '@vicons/ionicons5'
import { CONFIGURED_SECRET_PLACEHOLDER, CONFIGURED_SECRET_TAG } from '../../constants/secretField'
import type { ProxySecurityDraft } from '../../composables/proxy/securityDraft'
defineProps<{
  form: ProxySecurityDraft & { https_enabled: boolean }
  activeSecurityFeatures: string[]
  securityExpanded: string[]
  hasPassword: boolean
}>()
const emit = defineEmits<{
  patch: [value: Partial<ProxySecurityDraft>]
  'update:securityExpanded': [value: string[]]
  preset: []
}>()
function updateExpanded(value: string | number | (string | number)[]) {
  emit('update:securityExpanded', (Array.isArray(value) ? value : [value]).map(String))
}
</script>
<style scoped>
.security-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.security-header {
  padding: 14px 16px 16px;
  border-radius: 10px;
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
}

.security-header__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
}

.security-header__preset {
  flex-shrink: 0;
  font-weight: 500;
}

.security-header__status {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.security-header__status-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
  flex-shrink: 0;
}

.security-header__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.security-header__empty {
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.security-header__note {
  margin: 14px 0 0;
  padding-top: 14px;
  border-top: 1px solid var(--fonu-border);
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.6;
}

.security-collapse {
  width: 100%;
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  overflow: hidden;
  background: var(--fonu-surface);
}

.security-collapse :deep(.n-collapse-item) {
  margin: 0 !important;
  border: none !important;
  border-radius: 0 !important;
}

.security-collapse :deep(.n-collapse-item + .n-collapse-item) {
  border-top: 1px solid var(--fonu-border) !important;
}

.security-collapse :deep(.n-collapse-item__header) {
  padding: 12px 14px !important;
  min-height: 44px;
  box-sizing: border-box;
  font-weight: 600;
  background: var(--fonu-bg);
}

.security-collapse :deep(.n-collapse-item:first-child > .n-collapse-item__header) {
  padding-top: 12px !important;
}

.security-collapse :deep(.n-collapse-item__content-inner) {
  padding: 0 14px 14px;
}

.security-panel {
  padding-top: 2px;
}

.security-panel__fields {
  padding: 8px 0 16px;
  margin-bottom: 4px;
  border-bottom: 1px solid var(--fonu-border);
}

.security-panel__fields :deep(.n-form-item) {
  margin-bottom: 0;
}

.security-panel__fields :deep(.n-form-item-label) {
  padding-bottom: 6px;
}

.security-panel__fields:last-child {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 6px;
}

.security-panel .security-fields-grid {
  padding: 8px 0 16px;
  margin-bottom: 4px;
  border-bottom: 1px solid var(--fonu-border);
}

.security-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: 12px 0;
}

.security-panel .security-option + .security-option,
.security-panel__fields + .security-option,
.security-fields-grid + .security-option {
  border-top: 1px solid var(--fonu-border);
}

.security-option__label {
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.security-option__hint {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.45;
}

.security-fields-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--fonu-space-3);
  padding: 4px 0 8px;
}

.conn-limit-input {
  width: 160px;
}

.w-full {
  width: 100%;
}

.proxy-secret-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
</style>
