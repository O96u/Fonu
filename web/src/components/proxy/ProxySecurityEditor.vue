<template>
  <div class="security-section">
              <div v-if="section==='security'&&customNginx" class="security-header">当前规则采用手动 Nginx 配置，自动生成的安全策略需切换回自动生成模式后才会应用。</div>
              <div class="security-header" v-if="section==='security'">
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
                  IP 策略依赖正确的客户端 IP；<n-button text type="primary" size="tiny" @click="emit('settings')">信任代理设置（新窗口）</n-button>。白名单 IP 可豁免「仅中国大陆」限制。
                </p>
              </div>

              <n-collapse :expanded-names="section==='advanced'?['advanced']:securityExpanded" @update:expanded-names="updateExpanded" class="security-collapse">
                <n-collapse-item title="IP 访问控制" name="ip" v-if="section==='security'">
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
                        <ProxyDraftInput
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
                        <ProxyDraftInput
                          :value="form.ip_whitelist_text" @update:value="emit('patch', { ip_whitelist_text: $event })"
                          type="textarea"
                          :rows="2"
                          placeholder="每行一个 IP 或 CIDR"
                        />
                      </n-form-item>
                    </div>
                  </div>
                </n-collapse-item>

                <n-collapse-item title="访问认证" name="auth" v-if="section==='security'">
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
                      <ProxyDraftInput :value="form.basic_auth_username" @update:value="emit('patch', { basic_auth_username: $event })" placeholder="用户名" />
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
                      <ProxyDraftInput
                        :value="form.basic_auth_password" @update:value="emit('patch', { basic_auth_password: $event })"
                        type="password"
                        label="认证密码"
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

                <n-collapse-item title="流量控制" name="traffic" v-if="section==='security'">
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

                <n-collapse-item v-if="section==='advanced'" title="上游代理设置" name="advanced">
                  <div class="security-panel">
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">忽略后端 TLS 证书</div>
                      <div class="security-option__hint">仅在明确接受自签证书风险时使用；默认验证上游 TLS 证书。</div>
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
                  </div>
                </n-collapse-item>
                <n-collapse-item v-if="section==='security'" title="请求防护" name="request">
                  <div class="security-panel">
                    <div class="security-option"><div class="security-option__text"><div class="security-option__label">敏感路径保护</div><div class="security-option__hint">阻止访问 .env、.git、.svn 等敏感路径；不影响 .well-known。</div></div><n-switch :value="form.block_sensitive_paths" @update:value="emit('patch',{block_sensitive_paths:$event})" size="small"/></div>
                    <n-form-item label="允许的 HTTP 请求方法（留空不限制）"><n-select multiple clearable :value="form.allowed_methods" :options="methodOptions" @update:value="emit('patch',{allowed_methods:$event})" placeholder="默认允许所有方法"/></n-form-item>
                    <div class="security-option__hint">WebDAV、API 等服务需要额外方法，请勿盲目限制。</div>
                    <n-form-item label="请求体大小上限（MB，0 继承全局配置）"><n-input-number :value="form.max_body_mb" :min="0" :max="1048576" :precision="0" class="w-full" @update:value="emit('patch',{max_body_mb:$event??0})"/></n-form-item>
                  </div>
                </n-collapse-item>
                <n-collapse-item v-if="section==='security'" title="安全响应头" name="headers">
                  <div class="security-panel">
                    <div class="security-option"><div class="security-option__text"><div class="security-option__label">基础安全响应头</div><div class="security-option__hint">X-Content-Type-Options、Referrer-Policy，不强制 HTTPS。</div></div><n-switch :value="form.safe_headers" @update:value="emit('patch',{safe_headers:$event})" size="small"/></div>
                    <div class="security-option"><div class="security-option__text"><div class="security-option__label">严格 HTTPS 与嵌入保护</div><div class="security-option__hint">兼容旧版安全响应头：HSTS（含子域名）、X-Frame-Options。可能影响 HTTP 子域名或嵌入页面，按需开启。</div></div><n-switch :value="form.security_headers" @update:value="emit('patch',{security_headers:$event})" size="small" :disabled="!form.https_enabled"/></div>
                  </div>
                </n-collapse-item>
              </n-collapse>
              </div>
</template>
<script setup lang="ts">
import { NButton,NIcon,NTag,NCollapse,NCollapseItem,NSwitch,NFormItem,NInputNumber,NSelect } from 'naive-ui'
import ProxyDraftInput from './ProxyDraftInput.vue'
import { FlashOutline } from '@vicons/ionicons5'
import { CONFIGURED_SECRET_PLACEHOLDER, CONFIGURED_SECRET_TAG } from '../../constants/secretField'
import type { ProxySecurityDraft } from '../../composables/proxy/securityDraft'
defineProps<{
  section:'security'|'advanced'
  form: ProxySecurityDraft & { https_enabled: boolean }
  activeSecurityFeatures: string[]
  securityExpanded: string[]
  hasPassword: boolean
  customNginx?: boolean
}>()
const emit = defineEmits<{
  patch: [value: Partial<ProxySecurityDraft>]
  'update:securityExpanded': [value: string[]]
  preset: []
  settings: []
}>()
const methodOptions=['GET','HEAD','POST','PUT','PATCH','DELETE','OPTIONS','PROPFIND','PROPPATCH','MKCOL','COPY','MOVE','LOCK','UNLOCK','REPORT','SEARCH'].map(value=>({label:value,value}))
function updateExpanded(value: string | number | (string | number)[] | null | undefined) {
  emit('update:securityExpanded', (value == null ? [] : Array.isArray(value) ? value : [value]).map(String))
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
.security-section{padding:0!important;min-width:0;text-align:left}.security-header{background:var(--fonu-bg-muted);border:1px solid var(--fonu-border);border-radius:8px;padding:12px}.security-option{flex-wrap:wrap;gap:8px}.security-option__text{min-width:0;flex:1 1 180px}.security-panel__fields{min-width:0}.security-collapse :deep(.n-collapse-item__header){background:var(--fonu-surface)}.security-panel :deep(.n-select){width:100%;min-width:0}@container proxy-editor(max-width:520px){.security-fields-grid{grid-template-columns:1fr}}
</style>
