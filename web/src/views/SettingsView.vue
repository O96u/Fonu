<template>
  <PageHeader title="设置" description="配置界面、域名策略与系统维护" />

  <LoadError v-if="pageError" :message="pageError" @retry="load" />

  <n-spin v-else :show="loading">
    <div class="settings-grid">
      <FonuCard title="外观" subtitle="界面主题与时区">
        <n-form label-placement="top" class="settings-form">
          <n-form-item label="主题">
            <n-radio-group v-model:value="form.theme" class="theme-group" @update:value="onThemePreview">
              <n-radio-button value="system">跟随系统</n-radio-button>
              <n-radio-button value="light">浅色</n-radio-button>
              <n-radio-button value="dark">深色</n-radio-button>
            </n-radio-group>
            <p class="field-hint">切换后立即预览；点击「保存设置」后写入配置。进入本页不会自动改主题。</p>
          </n-form-item>
          <n-form-item label="时区">
            <n-input v-model:value="form.timezone" placeholder="Asia/Shanghai" />
          </n-form-item>
        </n-form>
      </FonuCard>

      <FonuCard title="自动任务" subtitle="DDNS 与证书维护策略">
        <n-form label-placement="top" class="settings-form">
          <n-form-item label="DDNS 检查周期（分钟）">
            <n-input-number v-model:value="ddnsInterval" :min="1" :max="1440" style="width: 100%" />
          </n-form-item>
          <n-form-item label="证书续签阈值（天）">
            <n-input-number v-model:value="certThreshold" :min="1" :max="90" style="width: 100%" />
          </n-form-item>
          <n-form-item label="日志保留天数">
            <n-input-number v-model:value="logRetention" :min="1" :max="365" style="width: 100%" />
          </n-form-item>
        </n-form>
      </FonuCard>

      <FonuCard title="安全" subtitle="管理员账户">
        <n-form label-placement="top" class="settings-form">
          <n-form-item label="当前密码">
            <n-input v-model:value="passwordForm.old_password" type="password" show-password-on="click" />
          </n-form-item>
          <n-form-item label="新密码">
            <n-input v-model:value="passwordForm.new_password" type="password" show-password-on="click" />
          </n-form-item>
        </n-form>
        <n-button type="primary" :loading="changingPassword" @click="changePassword">修改管理员密码</n-button>
      </FonuCard>

      <FonuCard title="通知" subtitle="Webhook 告警">
        <n-form label-placement="top" class="settings-form">
          <n-form-item label="Webhook URL">
            <n-input v-model:value="form.notify_webhook_url" placeholder="https://example.com/hook" />
          </n-form-item>
          <div class="switch-inline">
            <n-form-item label="DDNS 失败"><n-switch v-model:value="notifyDDNS" /></n-form-item>
            <n-form-item label="证书失败"><n-switch v-model:value="notifyCert" /></n-form-item>
          </div>
        </n-form>
      </FonuCard>

      <FonuCard title="数据" subtitle="备份与恢复" class="settings-span-full">
        <n-space>
          <n-button :loading="exporting" @click="exportBackup">导出配置</n-button>
          <n-upload :show-file-list="false" accept=".tar.gz,.tgz" @change="onRestoreFile">
            <n-button :loading="restoring">恢复配置</n-button>
          </n-upload>
        </n-space>
        <p class="data-hint">备份包含数据库、证书与 Nginx 配置，不包含日志。</p>
      </FonuCard>
    </div>

    <div class="save-bar">
      <span class="save-hint">修改设置后请点击保存</span>
      <n-button type="primary" size="large" :loading="saving" @click="save">保存设置</n-button>
    </div>
  </n-spin>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NRadioButton,
  NRadioGroup,
  NSpace,
  NSpin,
  NSwitch,
  NUpload,
  useMessage,
  type UploadFileInfo,
} from 'naive-ui'
import { api } from '../api/client'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import { useTheme } from '../composables/useTheme'

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

const passwordForm = reactive({ old_password: '', new_password: '' })
const notifyDDNS = ref(true)
const notifyCert = ref(true)
const ddnsInterval = ref(5)
const certThreshold = ref(30)
const logRetention = ref(30)

function onThemePreview(mode: string) {
  setThemeMode(mode)
}

async function load() {
  loading.value = true
  pageError.value = ''
  try {
    const settings = await api.getSettings()
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
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    const saved = await api.saveSettings({
      theme: form.theme,
      timezone: form.timezone,
      notify_webhook_url: form.notify_webhook_url,
      notify_on_ddns_error: notifyDDNS.value ? '1' : '0',
      notify_on_cert_error: notifyCert.value ? '1' : '0',
      ddns_check_interval_minutes: String(ddnsInterval.value),
      cert_renew_threshold_days: String(certThreshold.value),
      log_retention_days: String(logRetention.value),
    })
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
  changingPassword.value = true
  try {
    await api.changePassword(passwordForm.old_password, passwordForm.new_password)
    passwordForm.old_password = ''
    passwordForm.new_password = ''
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
}

.settings-span-full {
  grid-column: 1 / -1;
}

.settings-form {
  max-width: 100%;
}

.theme-group {
  display: inline-flex;
  flex-wrap: wrap;
}

.field-hint,
.data-hint {
  margin: var(--fonu-space-2) 0 0;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

.switch-inline {
  display: flex;
  gap: var(--fonu-space-5);
  flex-wrap: wrap;
}

.save-bar {
  position: sticky;
  bottom: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  margin-top: var(--fonu-space-6);
  padding: var(--fonu-space-4) var(--fonu-space-5);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  background: color-mix(in srgb, var(--fonu-surface) 92%, transparent);
  backdrop-filter: blur(8px);
}

.save-hint {
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

@media (max-width: 960px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }

  .save-bar {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
