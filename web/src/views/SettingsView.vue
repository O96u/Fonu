<template>
  <PageHeader title="设置" description="配置系统参数与运行策略" />

  <LoadError v-if="pageError" :message="pageError" @retry="load" />

  <n-spin v-else :show="loading">
    <div class="settings-sections">
      <FonuCard title="常规设置" subtitle="界面与基础信息">
        <n-form label-placement="top" class="settings-form">
          <n-form-item label="主题">
            <n-select v-model:value="form.theme" :options="themeOptions" />
          </n-form-item>
          <n-form-item label="时区">
            <n-input v-model:value="form.timezone" />
          </n-form-item>
          <n-form-item label="主域名">
            <n-input v-model:value="form.root_domain" placeholder="example.com" />
          </n-form-item>
          <n-form-item label="ACME 邮箱">
            <n-input v-model:value="form.acme_email" placeholder="admin@example.com" />
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

      <FonuCard title="数据" subtitle="备份与恢复">
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
      <n-button type="primary" size="large" :loading="saving" @click="save">保存设置</n-button>
    </div>
  </n-spin>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import {
  NButton,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
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
const { themeMode } = useTheme()
const saving = ref(false)
const loading = ref(false)
const pageError = ref('')
const changingPassword = ref(false)
const exporting = ref(false)
const restoring = ref(false)

const form = reactive({
  theme: 'system',
  timezone: 'Asia/Shanghai',
  root_domain: '',
  acme_email: '',
  notify_webhook_url: '',
})

const passwordForm = reactive({ old_password: '', new_password: '' })
const notifyDDNS = ref(true)
const notifyCert = ref(true)
const ddnsInterval = ref(5)
const certThreshold = ref(30)
const logRetention = ref(30)

const themeOptions = [
  { label: '跟随系统', value: 'system' },
  { label: '浅色', value: 'light' },
  { label: '深色', value: 'dark' },
]

watch(() => form.theme, (v) => { themeMode.value = v })

async function load() {
  loading.value = true
  pageError.value = ''
  try {
    const settings = await api.getSettings()
    Object.assign(form, {
      theme: settings.theme ?? 'system',
      timezone: settings.timezone ?? 'Asia/Shanghai',
      root_domain: settings.root_domain ?? '',
      acme_email: settings.acme_email ?? '',
      notify_webhook_url: settings.notify_webhook_url ?? '',
    })
    notifyDDNS.value = settings.notify_on_ddns_error !== '0' && settings.notify_on_ddns_error !== 'false'
    notifyCert.value = settings.notify_on_cert_error !== '0' && settings.notify_on_cert_error !== 'false'
    ddnsInterval.value = Number(settings.ddns_check_interval_minutes ?? 5)
    certThreshold.value = Number(settings.cert_renew_threshold_days ?? 30)
    logRetention.value = Number(settings.log_retention_days ?? 30)
    themeMode.value = form.theme
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await api.saveSettings({
      theme: form.theme,
      timezone: form.timezone,
      root_domain: form.root_domain,
      acme_email: form.acme_email,
      notify_webhook_url: form.notify_webhook_url,
      notify_on_ddns_error: notifyDDNS.value ? '1' : '0',
      notify_on_cert_error: notifyCert.value ? '1' : '0',
      ddns_check_interval_minutes: String(ddnsInterval.value),
      cert_renew_threshold_days: String(certThreshold.value),
      log_retention_days: String(logRetention.value),
    })
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
.settings-sections {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-5);
  max-width: 720px;
}

.settings-form {
  max-width: 480px;
}

.switch-inline {
  display: flex;
  gap: var(--fonu-space-5);
}

.data-hint {
  margin: var(--fonu-space-3) 0 0;
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.save-bar {
  margin-top: var(--fonu-space-6);
  padding-top: var(--fonu-space-5);
  border-top: 1px solid var(--fonu-border);
}
</style>
