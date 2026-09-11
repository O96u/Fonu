<template>
  <PageHeader title="DDNS" description="自动同步公网 IP 到 DNS 解析" />

  <LoadError v-if="loadError" :message="loadError" @retry="init" />

  <template v-else>
    <div class="ddns-layout">
      <FonuCard title="当前状态">
        <div class="status-panel">
          <div class="status-panel__head">
            <StatusBadge :value="config?.last_status || 'unknown'" text="连接正常" />
            <n-button size="small" :loading="updating" @click="updateNow">立即更新</n-button>
          </div>
          <div class="status-grid">
            <div class="status-item">
              <span class="label">IPv4</span>
              <span class="value mono">{{ config?.last_ipv4 || '-' }}</span>
            </div>
            <div class="status-item">
              <span class="label">IPv6</span>
              <span class="value mono">{{ config?.last_ipv6 || '-' }}</span>
            </div>
            <div class="status-item">
              <span class="label">上次更新</span>
              <span class="value">{{ formatRelativeTime(config?.last_updated_at) }}</span>
            </div>
          </div>
        </div>
      </FonuCard>

      <FonuCard title="DDNS 配置">
        <n-form label-placement="top" class="ddns-form">
          <n-form-item label="DNS Provider">
            <n-select v-model:value="form.provider" :options="providerOptions" />
          </n-form-item>
          <div class="form-row">
            <n-form-item label="主域名" class="form-row__item">
              <n-input v-model:value="form.root_domain" placeholder="example.com" />
            </n-form-item>
            <n-form-item label="子域名" class="form-row__item">
              <n-input v-model:value="form.record_name" placeholder="*" />
            </n-form-item>
          </div>
          <n-form-item v-if="form.provider === 'dnspod'" label="Token ID">
            <n-input v-model:value="form.api_token_id" placeholder="DNSPod ID" />
          </n-form-item>
          <n-form-item :label="credentialLabel">
            <n-input
              v-model:value="form.api_token"
              type="password"
              show-password-on="click"
              :placeholder="credentialPlaceholder"
            />
            <template v-if="config?.has_token" #feedback>
              <span class="token-hint">已配置 · 重新填写以更新</span>
            </template>
          </n-form-item>
          <n-form-item v-if="form.provider === 'alidns'" label="AccessKey Secret">
            <n-input v-model:value="form.api_secret" type="password" show-password-on="click" placeholder="重新填写以更新" />
          </n-form-item>
          <div class="switch-row">
            <n-form-item label="IPv4"><n-switch v-model:value="form.ipv4_enabled" /></n-form-item>
            <n-form-item label="IPv6"><n-switch v-model:value="form.ipv6_enabled" /></n-form-item>
            <n-form-item label="启用"><n-switch v-model:value="form.enabled" /></n-form-item>
          </div>
          <n-form-item label="更新周期（分钟）">
            <n-input-number v-model:value="updateInterval" :min="1" :max="1440" style="width: 100%" />
          </n-form-item>
        </n-form>
        <div class="actions">
          <n-button :loading="testing" @click="test">测试连接</n-button>
          <n-button type="primary" :loading="saving" @click="save">保存配置</n-button>
        </div>
      </FonuCard>
    </div>

    <div class="info-tip">
      <n-icon :component="InformationCircleOutline" />
      <span>Fonu 会定时检测公网 IP 变化，并自动更新 DNS 记录，无需手动干预。</span>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NSelect,
  NSwitch,
  useMessage,
} from 'naive-ui'
import { InformationCircleOutline } from '@vicons/ionicons5'
import { api } from '../api/client'
import type { DDNSConfig } from '../api/types'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatRelativeTime } from '../utils/format'

const message = useMessage()
const config = ref<DDNSConfig | null>(null)
const loading = ref(false)
const loadError = ref('')
const saving = ref(false)
const testing = ref(false)
const updating = ref(false)
const updateInterval = ref(5)

const providerOptions = [
  { label: 'Cloudflare', value: 'cloudflare' },
  { label: 'DNSPod', value: 'dnspod' },
  { label: '阿里云 DNS', value: 'alidns' },
]

const form = reactive({
  provider: 'cloudflare',
  root_domain: '',
  record_name: '*',
  api_token: '',
  api_token_id: '',
  api_secret: '',
  ipv4_enabled: true,
  ipv6_enabled: false,
  enabled: true,
})

const credentialLabel = computed(() => {
  if (form.provider === 'alidns') return 'AccessKey ID'
  if (form.provider === 'dnspod') return 'Token'
  return 'API Token'
})

const credentialPlaceholder = computed(() => {
  if (form.provider === 'alidns') return 'AccessKey ID'
  if (form.provider === 'dnspod') return 'DNSPod Token'
  return '重新填写以更新'
})

async function load() {
  const [data, settings] = await Promise.all([api.getDDNS(), api.getSettings()])
  updateInterval.value = Number(settings.ddns_check_interval_minutes ?? 5)
  if (data.configured === false) return
  config.value = data
  form.provider = data.provider || 'cloudflare'
  form.root_domain = data.root_domain
  form.record_name = data.record_name || '*'
  form.ipv4_enabled = data.ipv4_enabled
  form.ipv6_enabled = data.ipv6_enabled
  form.enabled = data.enabled
}

async function init() {
  loading.value = true
  loadError.value = ''
  try {
    await load()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    config.value = await api.saveDDNS({ ...form })
    await api.saveSettings({ ddns_check_interval_minutes: String(updateInterval.value) })
    form.api_token = ''
    form.api_token_id = ''
    form.api_secret = ''
    message.success('DDNS 配置已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function test() {
  testing.value = true
  try {
    await api.testDDNS({
      provider: form.provider,
      api_token: form.api_token,
      api_token_id: form.api_token_id,
      api_secret: form.api_secret,
    })
    message.success('连接成功')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '测试失败')
  } finally {
    testing.value = false
  }
}

async function updateNow() {
  updating.value = true
  try {
    config.value = await api.updateDDNS()
    message.success('DDNS 已更新')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
  } finally {
    updating.value = false
  }
}

onMounted(init)
</script>

<style scoped>
.ddns-layout {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: var(--fonu-space-5);
  align-items: start;
}

.status-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--fonu-space-4);
}

.status-grid {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-3);
}

.status-item {
  display: flex;
  justify-content: space-between;
  font-size: 14px;
}

.status-item .label { color: var(--fonu-text-secondary); }

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--fonu-space-4);
}

.form-row__item { margin-bottom: 0; }

.switch-row {
  display: flex;
  gap: var(--fonu-space-5);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--fonu-space-3);
  margin-top: var(--fonu-space-4);
  padding-top: var(--fonu-space-4);
  border-top: 1px solid var(--fonu-border);
}

.token-hint { color: var(--fonu-success); font-size: 12px; }

.info-tip {
  display: flex;
  align-items: flex-start;
  gap: var(--fonu-space-2);
  margin-top: var(--fonu-space-5);
  padding: var(--fonu-space-4);
  background: rgba(59, 130, 246, 0.08);
  border-radius: var(--fonu-radius-sm);
  font-size: 13px;
  color: var(--fonu-info);
}

@media (max-width: 991px) {
  .ddns-layout { grid-template-columns: 1fr; }
  .form-row { grid-template-columns: 1fr; }
}
</style>
