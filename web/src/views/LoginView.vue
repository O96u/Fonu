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
        <n-button block size="large" :loading="fnosLoading" :disabled="fnosLoading" @click="loginWithFnos">
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
import { useRoute, useRouter } from 'vue-router'
import { NButton, NForm, NFormItem, NInput, useMessage } from 'naive-ui'
import { api } from '../api/client'
import type { FnosLoginStatus } from '../api/types'
import logoImg from '../assets/brand/logo.png'
import { appBasePath } from '../utils/appBase'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const username = ref('')
const password = ref('')
const loading = ref(false)
const fnosLoading = ref(false)
const showFnos = ref(false)
const fnos = ref<FnosLoginStatus | null>(null)

/** True only after this click has landed on the fnOS gateway. */
const returnedFromGateway = computed(() => appBasePath() !== '' && route.query.nas === '1')

const fnosHint = computed(() => {
  const status = fnos.value
  if (!status?.available || !returnedFromGateway.value || status.bound) return ''
  return '飞牛网关已确认当前账号，但还没绑定 Fonu。请填写上方的 Fonu 管理员账号和密码，再点一次。'
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
    if (status.available && status.gateway && status.bound && returnedFromGateway.value) {
      await finishFnosLogin(true)
    }
  } catch {
    showFnos.value = false
  }
}

async function loginWithFnos() {
  if (fnosLoading.value) return
  const status = fnos.value
  if (!status?.available) return
  if (!returnedFromGateway.value || !status.gateway) {
    if (returnedFromGateway.value && !status.gateway) {
      message.error('无法验证当前飞牛用户，请重新登录飞牛 fnOS。')
      return
    }
    const target = gatewayTarget(status)
    if (!target) {
      message.error('无法打开飞牛网关，请从飞牛桌面打开 Fonu。')
      return
    }
    window.location.assign(target)
    return
  }
  if (!status.bound && (!username.value.trim() || !password.value)) {
    message.info('请填写上方的 Fonu 管理员账号和密码，用来绑定当前飞牛账号。')
    return
  }
  await finishFnosLogin(status.bound)
}

async function finishFnosLogin(alreadyBound: boolean) {
  fnosLoading.value = true
  try {
    await api.fnosLogin(alreadyBound ? '' : username.value.trim(), alreadyBound ? '' : password.value)
    message.success('登录成功')
    await router.replace({ name: 'dashboard' })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '登录失败')
  } finally {
    fnosLoading.value = false
  }
}

function gatewayTarget(status: FnosLoginStatus) {
  if (status.redirect_url && isGatewayURL(status.redirect_url)) {
    const url = new URL(status.redirect_url)
    url.searchParams.set('nas', '1')
    return url.toString()
  }
  if (appBasePath()) {
    const url = new URL(window.location.href)
    url.pathname = `${appBasePath()}/login`
    url.searchParams.set('nas', '1')
    url.hash = ''
    return url.toString()
  }
  return ''
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
