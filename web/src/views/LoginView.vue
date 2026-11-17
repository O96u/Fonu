<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-brand">
        <img class="auth-brand__logo" :src="logoImg" alt="Fonu" />
        <div>
          <h1>Fonu</h1>
          <p>让 NAS 访问更简单</p>
        </div>
      </div>

      <h2 class="auth-title">登录管理后台</h2>

      <n-form @submit.prevent="submit">
        <n-form-item label="用户名">
          <n-input v-model:value="username" size="large" placeholder="请输入用户名" autocomplete="username" />
        </n-form-item>
        <n-form-item label="密码">
          <n-input v-model:value="password" type="password" show-password-on="click" size="large" placeholder="请输入密码" autocomplete="current-password" />
        </n-form-item>
        <n-button type="primary" block size="large" :loading="loading" :disabled="loading" attr-type="submit">登录</n-button>
      </n-form>

      <div v-if="showFnos" class="nas-login">
        <div class="nas-login__or"><span>或</span></div>
        <n-button block size="large" secondary :loading="fnosLoading" :disabled="fnosLoading" @click="loginWithFnos">
          使用 NAS 登录
        </n-button>
        <p v-if="fnosHint" class="auth-hint auth-hint--tight">{{ fnosHint }}</p>
      </div>

      <p class="auth-hint">登录后可在右上角账户菜单中修改用户名和密码。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NForm, NFormItem, NInput, useMessage } from 'naive-ui'
import { api } from '../api/client'
import type { FnosLoginStatus } from '../api/types'
import logoImg from '../assets/brand/logo.png'

const router = useRouter()
const message = useMessage()
const username = ref('')
const password = ref('')
const loading = ref(false)
const fnosLoading = ref(false)
const showFnos = ref(false)
const fnos = ref<FnosLoginStatus | null>(null)
const fnosHint = computed(() => {
  const status = fnos.value
  if (!status?.available || status.bound) return ''
  if (status.login_method === 'redirect') return '检测到飞牛。点击后打开飞牛里的 Fonu，再用当前账号登录。'
  return '这个飞牛账号还没绑定。填写上方的 Fonu 账号和密码后，再点一次即可绑定并登录。'
})

onMounted(() => {
  if (history.state?.restored) {
    message.info('备份已恢复，请使用备份中的账号登录')
  }
  void loadFnos()
})

async function loadFnos() {
  try {
    const status = await api.fnosStatus()
    fnos.value = status
    showFnos.value = status.available
  } catch {
    showFnos.value = false
  }
}

async function loginWithFnos() {
  if (fnosLoading.value) return
  const status = fnos.value
  if (!status?.available) return
  if (status.login_method === 'redirect') {
    if (!isGatewayURL(status.redirect_url)) {
      message.error('无法连接飞牛认证网关，请检查 NAS 地址或网络连接。')
      return
    }
    window.location.assign(status.redirect_url)
    return
  }
  if (!status.bound && (!username.value.trim() || !password.value)) {
    message.warning('这个飞牛账号还没绑定。请先填写上方的 Fonu 账号和密码。')
    return
  }
  fnosLoading.value = true
  try {
    await api.fnosLogin(status.bound ? '' : username.value.trim(), status.bound ? '' : password.value)
    message.success('登录成功')
    await router.push({ name: 'dashboard' })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '登录失败')
  } finally {
    fnosLoading.value = false
  }
}

function isGatewayURL(value: string | undefined): value is string {
  if (!value) return false
  try {
    const url = new URL(value)
    return (url.protocol === 'http:' || url.protocol === 'https:') && url.pathname.startsWith('/app/fonu')
  } catch {
    return false
  }
}

async function submit() {
  if (loading.value) return
  loading.value = true
  try {
    await api.login(username.value, password.value)
    message.success('登录成功')
    await router.push({ name: 'dashboard' })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  min-height: 100%;
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
  width: 48px;
  height: 48px;
  border-radius: 14px;
  object-fit: cover;
  box-shadow: 0 6px 16px rgba(16, 185, 129, 0.22);
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

.auth-hint {
  margin: var(--fonu-space-4) 0 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--fonu-text-muted);
}

.auth-hint--tight {
  margin-top: 10px;
}

.nas-login {
  margin-top: 16px;
}

.nas-login__or {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  color: var(--fonu-text-muted);
  font-size: 13px;
}

.nas-login__or::before,
.nas-login__or::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--fonu-border);
}
</style>
