<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-brand">
        <div class="auth-brand__logo">F</div>
        <div>
          <h1>Fonu</h1>
          <p>Simple Access for Your NAS</p>
        </div>
      </div>
      <h2 class="auth-title">创建管理员账户</h2>
      <n-form @submit.prevent="submit">
        <n-form-item label="管理员用户名">
          <n-input v-model:value="username" placeholder="admin" size="large" />
        </n-form-item>
        <n-form-item label="管理员密码">
          <n-input
            v-model:value="password"
            type="password"
            show-password-on="click"
            placeholder="至少 8 位"
            size="large"
          />
        </n-form-item>
        <n-button type="primary" block size="large" :loading="loading" attr-type="submit">创建管理员</n-button>
      </n-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NForm, NFormItem, NInput, useMessage } from 'naive-ui'
import { api } from '../api/client'

const router = useRouter()
const message = useMessage()
const username = ref('admin')
const password = ref('')
const loading = ref(false)

async function submit() {
  loading.value = true
  try {
    await api.setup(username.value, password.value)
    message.success('管理员已创建')
    router.push({ name: 'dashboard' })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '初始化失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: var(--fonu-space-5);
  background: var(--fonu-bg);
}

.auth-card {
  width: 100%;
  max-width: 420px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow-md);
  padding: var(--fonu-space-6) var(--fonu-space-5);
}

.auth-brand {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-5);
}

.auth-brand__logo {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: var(--fonu-brand);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  font-weight: 700;
}

.auth-brand h1 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
}

.auth-brand p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.auth-title {
  margin: 0 0 var(--fonu-space-5);
  font-size: 16px;
  font-weight: 600;
  color: var(--fonu-text);
}
</style>
