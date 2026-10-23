<template>
  <div class="cert-export-wrap">
    <div class="cert-export-panel">
      <div class="cert-export-panel__head">
        <div class="cert-export-panel__intro">
          <div class="cert-export-panel__title">证书映射</div>
          <div class="cert-export-panel__sub">复制到本机其他目录（可选）</div>
        </div>
        <n-switch v-model:value="form.enabled" size="small" />
      </div>

      <n-collapse-transition :show="form.enabled">
        <div class="cert-export-panel__body">
          <n-form-item v-if="sourceDir" label="证书源目录" :show-feedback="false" class="cert-export-panel__item">
            <n-input :value="sourceDir" readonly />
          </n-form-item>

          <div class="cert-export-panel__block">
            <div class="cert-export-panel__label">目标目录</div>
            <p class="cert-export-panel__tip">
              填写 Fonu 进程能访问的绝对路径。Docker 部署时请把宿主机目录挂载到该路径，否则证书只会写在容器内。
            </p>
            <div class="cert-export__targets">
              <div v-for="(_, index) in form.targets" :key="index" class="cert-export__target-row">
                <DirectoryPathInput
                  :value="form.targets[index]"
                  placeholder="/etc/app/certs"
                  size="small"
                  @update:value="form.targets[index] = $event"
                />
                <n-button
                  quaternary
                  circle
                  type="error"
                  size="small"
                  :disabled="form.targets.length <= 1"
                  title="删除"
                  @click="removeTarget(index)"
                >
                  <template #icon>
                    <n-icon :component="TrashOutline" />
                  </template>
                </n-button>
              </div>
              <n-button dashed size="small" block @click="addTarget">
                <template #icon>
                  <n-icon :component="AddOutline" />
                </template>
                添加目录
              </n-button>
            </div>
          </div>

          <div class="cert-export-panel__block cert-export-panel__block--hook">
            <div class="cert-export-panel__label">重载命令（选填）</div>
            <n-input
              v-model:value="form.on_change_shell"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 2 }"
              placeholder="复制完成后执行，如 docker exec nginx nginx -s reload"
              size="small"
            />
          </div>

          <div
            v-if="showSyncStatus && (form.last_sync_at || form.last_error)"
            class="cert-export-panel__status"
          >
            <span v-if="form.last_sync_at">本地上次同步：{{ formatSyncTime(form.last_sync_at) }}</span>
            <span v-if="form.last_error" class="cert-export-panel__status-error">{{ form.last_error }}</span>
          </div>
        </div>
      </n-collapse-transition>
    </div>

    <div class="cert-export-panel">
      <div class="cert-export-panel__head">
        <div class="cert-export-panel__intro">
          <div class="cert-export-panel__title">上传到 SSH 主机</div>
          <div class="cert-export-panel__sub">通过 SFTP 复制到远程目录（可选）</div>
        </div>
        <n-switch v-model:value="form.ssh.enabled" size="small" />
      </div>

      <n-collapse-transition :show="form.ssh.enabled">
        <div class="cert-export-panel__body">
          <div class="cert-export-panel__grid">
            <div class="cert-export-panel__field">
              <div class="cert-export-panel__label">主机</div>
              <n-input v-model:value="form.ssh.host" placeholder="IP 或域名" size="small" />
            </div>
            <div class="cert-export-panel__field cert-export-panel__field--port">
              <div class="cert-export-panel__label">端口</div>
              <n-input-number
                v-model:value="form.ssh.port"
                :min="1"
                :max="65535"
                size="small"
                class="cert-export-port"
              />
            </div>
          </div>

          <n-form-item label="用户名" :show-feedback="false">
            <n-input v-model:value="form.ssh.username" placeholder="root" size="small" />
          </n-form-item>

          <n-form-item label="认证方式" :show-feedback="false">
            <n-radio-group v-model:value="form.ssh.auth_type" size="small" class="ssh-auth-buttons">
              <n-radio-button value="password">密码</n-radio-button>
              <n-radio-button value="private_key">私钥</n-radio-button>
            </n-radio-group>
          </n-form-item>

          <n-form-item v-if="form.ssh.auth_type === 'password'" label="密码" :show-feedback="false">
            <n-input
              v-model:value="form.ssh.password"
              type="password"
              show-password-on="click"
              :placeholder="form.ssh.has_password ? '已保存的密码显示为 ********' : 'SSH 登录密码'"
              size="small"
            />
          </n-form-item>

          <template v-else>
            <n-form-item label="私钥内容" :show-feedback="false">
              <div class="pem-field">
                <n-input
                  v-model:value="form.ssh.private_key"
                  type="textarea"
                  :autosize="{ minRows: 3, maxRows: 5 }"
                  :placeholder="sshKeyPlaceholder"
                  size="small"
                  class="pem-input"
                />
                <n-button text type="primary" size="small" class="pem-field__file" @click="sshKeyPasteInput?.click()">
                  <template #icon>
                    <n-icon :component="CloudUploadOutline" :size="16" />
                  </template>
                  从文件读取
                </n-button>
                <input
                  ref="sshKeyPasteInput"
                  type="file"
                  accept=".pem,.key,.txt"
                  class="file-input"
                  @change="onSSHKeyPasteFile"
                />
              </div>
            </n-form-item>
            <n-form-item label="私钥口令" :show-feedback="false">
              <n-input
                v-model:value="form.ssh.private_key_passphrase"
                type="password"
                show-password-on="click"
                :placeholder="form.ssh.has_passphrase ? '已保存口令显示为 ********' : '私钥未加密可留空'"
                size="small"
              />
            </n-form-item>
          </template>

          <div class="ssh-proxy-row">
            <div class="ssh-proxy-row__text">
              <div class="ssh-proxy-row__title">使用代理</div>
              <div class="ssh-proxy-row__sub">通过 SOCKS5 或 HTTP 代理连接目标主机</div>
            </div>
            <n-switch v-model:value="form.ssh.proxy_enabled" size="small" />
          </div>

          <n-collapse-transition :show="form.ssh.proxy_enabled">
            <div class="ssh-proxy-fields">
              <n-form-item label="代理类型" :show-feedback="false">
                <n-radio-group v-model:value="form.ssh.proxy_type" size="small" class="ssh-auth-buttons">
                  <n-radio-button value="socks5">SOCKS5</n-radio-button>
                  <n-radio-button value="http">HTTP</n-radio-button>
                </n-radio-group>
              </n-form-item>
              <div class="cert-export-panel__grid">
                <div class="cert-export-panel__field">
                  <div class="cert-export-panel__label">代理主机</div>
                  <n-input v-model:value="form.ssh.proxy_host" placeholder="IP 或域名" size="small" />
                </div>
                <div class="cert-export-panel__field cert-export-panel__field--port">
                  <div class="cert-export-panel__label">端口</div>
                  <n-input-number
                    v-model:value="form.ssh.proxy_port"
                    :min="1"
                    :max="65535"
                    size="small"
                    class="cert-export-port"
                  />
                </div>
              </div>
              <n-form-item label="代理用户名" :show-feedback="false">
                <n-input v-model:value="form.ssh.proxy_username" placeholder="无认证可留空" size="small" />
              </n-form-item>
              <n-form-item label="代理密码" :show-feedback="false">
                <n-input
                  v-model:value="form.ssh.proxy_password"
                  type="password"
                  show-password-on="click"
                  :placeholder="form.ssh.has_proxy_password ? '已保存的密码显示为 ********' : '无认证可留空'"
                  size="small"
                />
              </n-form-item>
            </div>
          </n-collapse-transition>

          <n-form-item label="远程目录" :show-feedback="false">
            <n-input v-model:value="form.ssh.remote_dir" placeholder="/etc/nginx/certs" size="small" />
          </n-form-item>

          <n-form-item label="上传后执行（选填）" :show-feedback="false">
            <n-input
              v-model:value="form.ssh.on_upload_shell"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 2 }"
              placeholder="systemctl reload nginx"
              size="small"
            />
          </n-form-item>

          <div
            v-if="showSyncStatus && (form.ssh.last_sync_at || form.ssh.last_error)"
            class="cert-export-panel__status"
          >
            <span v-if="form.ssh.last_sync_at">SSH 上次同步：{{ formatSyncTime(form.ssh.last_sync_at) }}</span>
            <span v-if="form.ssh.last_error" class="cert-export-panel__status-error">{{ form.ssh.last_error }}</span>
          </div>
        </div>
      </n-collapse-transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { AddOutline, CloudUploadOutline, TrashOutline } from '@vicons/ionicons5'
import {
  NButton,
  NCollapseTransition,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NRadioButton,
  NRadioGroup,
  NSwitch,
} from 'naive-ui'
import DirectoryPathInput from './DirectoryPathInput.vue'
import { formatDate } from '../utils/format'
import type { CertExportFormModel } from '../utils/certExportForm'

const form = defineModel<CertExportFormModel>('form', { required: true })
const sshKeyPasteInput = ref<HTMLInputElement | null>(null)

withDefaults(
  defineProps<{
    sourceDir?: string
    showSyncStatus?: boolean
  }>(),
  {
    sourceDir: '',
    showSyncStatus: false,
  },
)

const sshKeyPlaceholder = computed(() => {
  if (form.value.ssh.has_private_key) {
    return '已保存私钥；不修改则留空'
  }
  return '-----BEGIN OPENSSH PRIVATE KEY-----'
})

function addTarget() {
  form.value.targets.push('')
}

function removeTarget(index: number) {
  form.value.targets.splice(index, 1)
  if (form.value.targets.length === 0) {
    form.value.targets.push('')
  }
}

function formatSyncTime(value: string) {
  return formatDate(value) || value
}

async function onSSHKeyPasteFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  form.value.ssh.private_key = await file.text()
  ;(event.target as HTMLInputElement).value = ''
}

</script>

<style scoped>
.cert-export-wrap {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-3);
  margin-top: var(--fonu-space-4);
}

.cert-export-panel {
  padding: 10px 14px;
  border-radius: 12px;
  border: 1px solid var(--fonu-border);
  background: var(--fonu-bg);
}

.cert-export-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
}

.cert-export-panel__title {
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.cert-export-panel__sub {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--fonu-text-muted);
}

.cert-export-panel__body {
  margin-top: var(--fonu-space-3);
  padding-top: var(--fonu-space-3);
  border-top: 1px solid var(--fonu-border);
}

.cert-export-panel__item {
  margin-bottom: var(--fonu-space-3);
}

.cert-export-panel__block {
  margin-bottom: var(--fonu-space-3);
}

.cert-export-panel__block--hook {
  margin-bottom: 0;
}

.cert-export-panel__label {
  margin-bottom: var(--fonu-space-2);
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.cert-export-panel__tip {
  margin: 0 0 var(--fonu-space-2);
  font-size: 12px;
  line-height: 1.45;
  color: var(--fonu-text-muted);
}

.cert-export-panel__grid {
  display: flex;
  align-items: flex-end;
  gap: var(--fonu-space-3);
  margin-bottom: var(--fonu-space-3);
}

.cert-export-panel__field {
  flex: 1;
  min-width: 0;
}

.cert-export-panel__field--port {
  flex: 0 0 112px;
}

.cert-export-port {
  width: 100%;
}

.ssh-proxy-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  margin-bottom: var(--fonu-space-3);
}

.ssh-proxy-row__title {
  font-size: 13px;
  font-weight: 500;
  color: var(--fonu-text);
}

.ssh-proxy-row__sub {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.45;
  color: var(--fonu-text-muted);
}

.ssh-proxy-fields {
  margin-bottom: var(--fonu-space-3);
  padding-bottom: var(--fonu-space-3);
  border-bottom: 1px solid var(--fonu-border);
}

.ssh-auth-buttons {
  display: flex;
  width: 100%;
}

.ssh-auth-buttons :deep(.n-radio-button) {
  flex: 1;
  justify-content: center;
}

.cert-export__targets {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-2);
  width: 100%;
}

.cert-export__target-row {
  display: flex;
  gap: var(--fonu-space-2);
  align-items: center;
}

.cert-export__target-row :deep(.n-input) {
  flex: 1;
}

.cert-export-panel__status {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: var(--fonu-space-3);
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.cert-export-panel__status-error {
  color: var(--fonu-danger);
}

.pem-input :deep(textarea) {
  font-family: var(--fonu-mono);
  font-size: 12px;
}

.pem-field {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--fonu-space-2);
  width: 100%;
}

.pem-field__file {
  padding-left: 0;
  font-size: 13px;
}

.file-input {
  display: none;
}

</style>
