<template>
<n-form label-placement="top" class="proxy-modal__pane">
              <n-form-item label="名称">
                <n-input
                  :value="form.name" @update:value="emit('patch', { name: $event })"
                  maxlength="100"
                  show-count
                  placeholder="选填；留空保存时默认使用对应前端域名"
                />
              </n-form-item>

              <n-form-item required>
                <template #label>
                  <span class="form-label">
                    前端域名
                    <n-tooltip trigger="hover">
                      <template #trigger>
                        <n-icon :component="HelpCircleOutline" class="form-label__help" />
                      </template>
                      每行一个域名；如需单独端口可写 example.com:6893
                    </n-tooltip>
                  </span>
                </template>
                <div class="field-stack">
                  <n-select
                    :value="form.hostsSelected" @update:value="updateHosts"
                    multiple
                    filterable
                    tag
                    :loading="ddnsHostsLoading"
                    :options="hostSelectOptions"
                    :filter="filterDdnsHostOption"
                    placeholder="输入域名后回车，或从 DDNS 列表选择（可多选）"
                    class="hosts-select"
                    @focus="emit('loadHosts')"
                  />
                  <p class="field-hint">
                    可自定义填写，不必在 Fonu 配置 DDNS；每选一个域名各生成一条规则（共用下方目标地址），表格中各占一行。
                    <template v-if="ddnsHostOptionsEmpty && !ddnsHostsLoading">
                      <span class="field-hint__sep"> </span>
                      <router-link class="field-hint__link" :to="{ name: 'ddns' }">在 Fonu 管理 DDNS</router-link>
                      可选。
                    </template>
                  </p>
                </div>
              </n-form-item>

              <div v-if="!bindEntryListen" class="listen-row">
                <div class="listen-col listen-col--port">
                  <div class="listen-col__label">监听端口 <span class="required-mark">*</span></div>
                  <div class="listen-col__control">
                    <n-input-number :value="form.listen_port" @update:value="emit('patch', { listen_port: $event ?? 0 })" :min="1" :max="65535" class="port-input" />
                  </div>
                </div>
                <div class="listen-col listen-col--protocol">
                  <div class="listen-col__label">监听协议</div>
                  <div class="listen-col__control">
                    <div class="listen-types">
                      <n-checkbox :checked="form.listen_ipv4" @update:checked="emit('patch', { listen_ipv4: Boolean($event) })">IPv4</n-checkbox>
                      <n-checkbox :checked="form.listen_ipv6" @update:checked="emit('patch', { listen_ipv6: Boolean($event) })">IPv6</n-checkbox>
                    </div>
                  </div>
                </div>
              </div>
              <div v-else class="listen-row listen-row--readonly">
                <div class="listen-col listen-col--port">
                  <div class="listen-col__label">监听入口</div>
                  <div class="listen-col__control">
                    <n-tag size="small" round :bordered="false">
                      {{ entryGroupDisplayName({ key: '', name: '', listen: form, rules: [], enabledCount: 0, portHasMixedEntries: false }) }}
                    </n-tag>
                  </div>
                </div>
              </div>

              <n-form-item label="目标地址" required>
                <div class="field-stack">
                  <n-input :value="form.upstream" @update:value="emit('patch', { upstream: $event })" placeholder="例如：http://192.168.1.100:5173" />
                  <p class="field-hint">仅支持 http:// 或 https:// 开头的内网地址。</p>
                </div>
              </n-form-item>

              <div v-if="!bindEntryListen" class="form-switch-list form-switch-list--compact">
                <div class="form-switch-row">
                  <div class="form-switch-row__text">
                    <div class="form-switch-row__label">启用 HTTPS</div>
                    <div class="form-switch-row__hint">为前端域名启用 HTTPS 访问</div>
                  </div>
                  <n-switch :value="form.https_enabled" @update:value="emit('patch', { https_enabled: Boolean($event) })" />
                </div>
                <div class="form-switch-row">
                  <div class="form-switch-row__text">
                    <div class="form-switch-row__label">HTTP 跳转 HTTPS</div>
                    <div class="form-switch-row__hint">将 HTTP 请求自动跳转为 HTTPS</div>
                  </div>
                  <n-switch :value="form.http_redirect" @update:value="emit('patch', { http_redirect: Boolean($event) })" :disabled="!form.https_enabled" />
                </div>
                <div class="form-switch-row">
                  <div class="form-switch-row__text">
                    <div class="form-switch-row__label">启用规则</div>
                    <div class="form-switch-row__hint">保存后立即开始转发请求</div>
                  </div>
                  <n-switch :value="form.enabled" @update:value="emit('patch', { enabled: Boolean($event) })" />
                </div>
              </div>
              <div v-else-if="editing" class="form-switch-list form-switch-list--compact">
                <div class="form-switch-row">
                  <div class="form-switch-row__text">
                    <div class="form-switch-row__label">启用规则</div>
                    <div class="form-switch-row__hint">保存后立即开始转发请求</div>
                  </div>
                  <n-switch :value="form.enabled" @update:value="emit('patch', { enabled: Boolean($event) })" />
                </div>
              </div>
            </n-form>
</template>
<script setup lang="ts">
import { NForm,NFormItem,NInput,NInputNumber,NSelect,NCheckbox,NSwitch,NTag,NTooltip,NIcon,type SelectGroupOption } from 'naive-ui'
import { HelpCircleOutline } from '@vicons/ionicons5'
import { entryGroupDisplayName } from '../../utils/proxyEntries'
import type { DdnsHostSelectOption } from '../../utils/ddnsHosts'
interface BasicDraft {
  name:string
  hostsSelected:string[]
  listen_port:number
  listen_ipv4:boolean
  listen_ipv6:boolean
  https_enabled:boolean
  http_redirect:boolean
  upstream:string
  enabled:boolean
}
defineProps<{
  form:BasicDraft
  bindEntryListen:boolean
  editing:boolean
  ddnsHostsLoading:boolean
  ddnsHostOptionsEmpty:boolean
  hostSelectOptions:SelectGroupOption[]
  filterDdnsHostOption:(pattern:string,option:DdnsHostSelectOption)=>boolean
}>()
const emit=defineEmits<{patch:[value:Partial<BasicDraft>];loadHosts:[]}>()
function updateHosts(value:unknown){emit('patch',{hostsSelected:Array.isArray(value)?value.map(String):[]})}
</script>
<style scoped>
.listen-row--readonly {
  padding: 8px 0 4px;
}

.listen-row--readonly .listen-col__label {
  margin-bottom: 6px;
}

.listen-types {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-5);
}

.proxy-modal__pane {
  width: 100%;
  box-sizing: border-box;
  padding: var(--fonu-space-4) var(--fonu-space-5) 0;
}

.proxy-modal__pane--nginx {
  display: flex;
  flex-direction: column;
  min-height: min(480px, 55vh);
}

.proxy-modal__pane--nginx .nginx-editor-spin {
  flex: 1;
  min-height: 0;
}

.proxy-modal__pane :deep(.n-form) {
  width: 100%;
}

.form-switch-list--compact {
  margin-bottom: 0;
}

.form-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.form-label__help {
  font-size: 14px;
  color: var(--fonu-text-muted);
  cursor: help;
}

.field-stack {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
}

.field-hint {
  margin: 0;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

.field-hint__link {
  color: var(--fonu-primary);
  text-decoration: none;
}

.field-hint__link:hover {
  text-decoration: underline;
}

.hosts-select {
  width: 100%;
}

.hosts-select :deep(.n-base-select-group-header) {
  padding-top: 10px;
  padding-bottom: 4px;
}

.hosts-select :deep(.ddns-host-group-label__task) {
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text);
}

.hosts-select :deep(.ddns-host-group-label__dash) {
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text);
}

.hosts-select :deep(.ddns-host-group-label__provider) {
  font-size: 11px;
  font-weight: 400;
  color: var(--fonu-text-muted);
}

.proxy-modal__pane :deep(.n-form-item .n-form-item-blank) {
  display: block;
  width: 100%;
}

.listen-row {
  display: flex;
  align-items: flex-start;
  gap: 24px;
  margin-bottom: var(--fonu-space-5);
}

.listen-col {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.listen-col--port {
  width: 168px;
  flex-shrink: 0;
}

.listen-col--protocol {
  flex-shrink: 0;
}

.listen-col__label {
  font-size: 14px;
  line-height: 1.25;
  color: var(--fonu-text);
}

.required-mark {
  color: #d03050;
}

.listen-col__control {
  min-height: 34px;
  display: flex;
  align-items: center;
}

.port-input {
  width: 100%;
}

.form-switch-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-bottom: var(--fonu-space-4);
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
  font-weight: 600;
  color: var(--fonu-text);
}

.form-switch-row__hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}
@media(max-width:640px){
.listen-row{flex-direction:column;align-items:stretch;gap:var(--fonu-space-4)}
.listen-col--port{width:100%}
}
</style>
