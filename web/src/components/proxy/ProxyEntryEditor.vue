<template>
  <n-modal :show="showEntryModal" @update:show="emit('update:showEntryModal', $event)" :mask-closable="false" transform-origin="center">
    <div class="proxy-entry-edit-modal">
      <div class="proxy-entry-edit-modal__header">
        <h3 class="modal-title">{{ entryModalTitle }}</h3>
        <n-button size="small" quaternary @click="emit('update:showEntryModal', false)">
          <template #icon><n-icon :component="CloseOutline" /></template>
        </n-button>
      </div>
      <n-form label-placement="top" class="proxy-entry-edit-modal__body">
        <n-form-item label="名称">
          <n-input :value="entryForm.name" @update:value="emit('patch', { name: $event })" maxlength="100" show-count placeholder="选填；留空时标题自动显示为「端口 · HTTPS · IPv4」" />
        </n-form-item>
        <div class="listen-row">
          <div class="listen-col listen-col--port">
            <div class="listen-col__label">监听端口 <span class="required-mark">*</span></div>
            <div class="listen-col__control">
              <n-input-number :value="entryForm.listen_port" @update:value="emit('patch', { listen_port: $event ?? 0 })" :min="1" :max="65535" class="port-input" />
            </div>
          </div>
          <div class="listen-col listen-col--protocol">
            <div class="listen-col__label">监听协议</div>
            <div class="listen-col__control">
              <div class="listen-types">
                <n-checkbox :checked="entryForm.listen_ipv4" @update:checked="emit('patch', { listen_ipv4: Boolean($event) })">IPv4</n-checkbox>
                <n-checkbox :checked="entryForm.listen_ipv6" @update:checked="emit('patch', { listen_ipv6: Boolean($event) })">IPv6</n-checkbox>
              </div>
            </div>
          </div>
        </div>
        <div class="form-switch-list form-switch-list--compact">
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">启用 HTTPS</div>
            </div>
            <n-switch :value="entryForm.https_enabled" @update:value="emit('patch', { https_enabled: Boolean($event) })" />
          </div>
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">HTTP 跳转 HTTPS</div>
            </div>
            <n-switch :value="entryForm.http_redirect" @update:value="emit('patch', { http_redirect: Boolean($event) })" :disabled="!entryForm.https_enabled" />
          </div>
        </div>
        <p v-if="cloningSourceId" class="field-hint">
          保存后将新建监听入口，并把原入口下的 {{ cloneRuleCount }} 条规则（前端域名、上游地址与安全设置）一并复制到新端口；复制后两份规则相互独立。
        </p>
        <p v-else-if="editingEntryId" class="field-hint">
          仅修改入口名称、监听与 HTTPS；域名、目标与安全策略请在下方列表中编辑各条规则。
        </p>
        <p v-else class="field-hint">
          先创建空入口，再在下方列表添加规则；批量改域名、目标或安全策略时请点「编辑入口」。
        </p>
      </n-form>
      <div class="modal-footer">
        <n-button @click="emit('update:showEntryModal', false)">取消</n-button>
        <n-button type="primary" :loading="savingEntry" @click="emit('save')">保存</n-button>
      </div>
    </div>
  </n-modal>
</template>
<script setup lang="ts">
import { NModal, NButton, NIcon, NForm, NFormItem, NInput, NInputNumber, NCheckbox, NSwitch } from 'naive-ui'
import { CloseOutline } from '@vicons/ionicons5'
interface EntryDraft {
  name: string
  listen_port: number
  listen_ipv4: boolean
  listen_ipv6: boolean
  https_enabled: boolean
  http_redirect: boolean
}
defineProps<{
  showEntryModal: boolean
  entryModalTitle: string
  entryForm: EntryDraft
  cloningSourceId: number | null
  editingEntryId: number | null
  cloneRuleCount: number
  savingEntry: boolean
}>()
const emit = defineEmits<{
  'update:showEntryModal': [value: boolean]
  patch: [value: Partial<EntryDraft>]
  save: []
}>()
</script>
<style scoped>
.proxy-entry-edit-modal {
  width: min(480px, 92vw);
  padding: var(--fonu-space-4);
  background: var(--fonu-surface, #fff);
  border-radius: var(--fonu-radius-lg, 12px);
}
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

.form-switch-list--compact {
  margin-bottom: 0;
}

.field-hint {
  margin: 0;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
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

.required-mark {
  color: #d03050;
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

.modal-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-4) var(--fonu-space-5) var(--fonu-space-5);
  border-top: 1px solid var(--fonu-border);
  margin-top: var(--fonu-space-2);
}

.proxy-entry-edit-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--fonu-space-4);
}

.proxy-entry-edit-modal__body {
  margin-bottom: var(--fonu-space-4);
}

.listen-col__label {
  font-size: 14px;
  line-height: 1.25;
  color: var(--fonu-text);
}

.listen-col__control {
  min-height: 34px;
  display: flex;
  align-items: center;
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
@media(max-width:640px) {
  .listen-row { flex-direction:column; align-items:stretch; gap:var(--fonu-space-4); }
  .listen-col--port { width:100%; }
}
</style>
