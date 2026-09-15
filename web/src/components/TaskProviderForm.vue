<template>
  <div class="provider-form">
    <div class="provider-form__head">
      <span class="provider-form__title">{{ editing ? '编辑服务商' : '添加服务商' }}</span>
    </div>

    <div class="provider-form__field">
      <label class="provider-form__label">DNS 服务商</label>
      <div class="provider-select">
        <span class="provider-logo provider-logo--md">
          <img :src="providerIcon(form.provider)" :alt="providerLabel(form.provider)" />
        </span>
        <n-select
          v-model:value="form.provider"
          :options="providerOptions"
          :disabled="!!editing"
          size="small"
          class="provider-select__input"
        />
      </div>
    </div>

    <div v-if="form.provider === 'dnspod'" class="provider-form__field">
      <label class="provider-form__label">Token ID</label>
      <n-input v-model:value="form.api_token_id" placeholder="DNSPod ID" size="small" />
    </div>

    <div class="provider-form__field">
      <label class="provider-form__label">{{ credentialLabel }}</label>
      <n-input
        v-model:value="form.api_token"
        type="password"
        show-password-on="click"
        :placeholder="credentialPlaceholder"
        size="small"
      />
      <span v-if="editing?.has_token" class="token-hint">已配置 · 留空则保持不变</span>
    </div>

    <div v-if="form.provider === 'alidns' || form.provider === 'tencentcloud'" class="provider-form__field">
      <label class="provider-form__label">{{ secretLabel }}</label>
      <n-input
        v-model:value="form.api_secret"
        type="password"
        show-password-on="click"
        placeholder="留空则保持不变"
        size="small"
      />
    </div>

    <div class="provider-form__switches">
      <div class="settings-switch">
        <span>IPv4</span>
        <n-switch v-model:value="form.ipv4_enabled" size="small" />
      </div>
      <div class="settings-switch">
        <span>IPv6</span>
        <n-switch v-model:value="form.ipv6_enabled" size="small" />
      </div>
      <div class="settings-switch">
        <span>启用</span>
        <n-switch v-model:value="form.enabled" size="small" />
      </div>
    </div>

    <div class="provider-form__actions">
      <n-button size="small" quaternary @click="emit('cancel')">取消</n-button>
      <n-button size="small" :loading="testing" @click="emit('test')">测试连接</n-button>
      <n-button size="small" type="primary" :loading="saving" @click="emit('save')">保存</n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NInput, NSelect, NSwitch } from 'naive-ui'
import aliyunIcon from '../assets/brand/dns/aliyun.png'
import cloudflareIcon from '../assets/brand/dns/cloudflare.png'
import dnspodIcon from '../assets/brand/dns/dnspod.png'
import tencentcloudIcon from '../assets/brand/dns/tencentcloud.png'
import type { DDNSConfig } from '../api/types'

export type ProviderForm = {
  provider: string
  api_token: string
  api_token_id: string
  api_secret: string
  ipv4_enabled: boolean
  ipv6_enabled: boolean
  enabled: boolean
}

const props = defineProps<{
  form: ProviderForm
  editing?: DDNSConfig | null
  saving?: boolean
  testing?: boolean
}>()

const emit = defineEmits<{
  save: []
  test: []
  cancel: []
}>()

const providerOptions = [
  { label: 'Cloudflare', value: 'cloudflare' },
  { label: 'DNSPod', value: 'dnspod' },
  { label: '阿里云 DNS', value: 'alidns' },
  { label: '腾讯云 DNS', value: 'tencentcloud' },
]

const providerMap: Record<string, { label: string; icon: string }> = {
  cloudflare: { label: 'Cloudflare', icon: cloudflareIcon },
  dnspod: { label: 'DNSPod', icon: dnspodIcon },
  alidns: { label: '阿里云 DNS', icon: aliyunIcon },
  tencentcloud: { label: '腾讯云 DNS', icon: tencentcloudIcon },
}

const credentialLabel = computed(() => {
  if (props.form.provider === 'alidns') return 'AccessKey ID'
  if (props.form.provider === 'tencentcloud') return 'SecretId'
  if (props.form.provider === 'dnspod') return 'Token'
  return 'API Token'
})

const secretLabel = computed(() => {
  if (props.form.provider === 'tencentcloud') return 'SecretKey'
  return 'AccessKey Secret'
})

const credentialPlaceholder = computed(() => {
  if (props.editing?.has_token) return '留空则保持不变'
  if (props.form.provider === 'alidns') return 'AccessKey ID'
  if (props.form.provider === 'tencentcloud') return '腾讯云 SecretId'
  if (props.form.provider === 'dnspod') return 'DNSPod Token'
  return 'Cloudflare API Token'
})

function providerLabel(v: string) {
  return providerMap[v]?.label ?? v
}

function providerIcon(v: string) {
  return providerMap[v]?.icon ?? ''
}
</script>

<style scoped>
.provider-form__head {
  margin-bottom: 10px;
}

.provider-form__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.provider-form__field {
  margin-bottom: 10px;
}

.provider-form__label {
  display: block;
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.provider-select {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

.provider-select__input {
  flex: 1;
  min-width: 0;
}

.provider-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
  background: #fff;
  border: 1px solid var(--fonu-border);
}

.provider-logo--md {
  width: 32px;
  height: 32px;
}

.provider-logo img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.provider-form__switches {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin: 8px 0 12px;
}

.settings-switch {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--fonu-text);
}

.provider-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  flex-wrap: wrap;
}

.token-hint {
  display: block;
  margin-top: 4px;
  color: var(--fonu-success);
  font-size: 11px;
}
</style>
