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

          <n-input v-model:value="username" size="large" />

        </n-form-item>

        <n-form-item label="密码">

          <n-input v-model:value="password" type="password" show-password-on="click" size="large" />

        </n-form-item>

        <n-button type="primary" block size="large" :loading="loading" attr-type="submit">登录</n-button>

      </n-form>

      <p class="auth-hint">首次使用请查看系统日志中的初始密码，登录后请在「设置」中修改。</p>

    </div>

  </div>

</template>



<script setup lang="ts">

import { ref } from 'vue'

import { useRouter } from 'vue-router'

import { NButton, NForm, NFormItem, NInput, useMessage } from 'naive-ui'

import { api } from '../api/client'
import logoImg from '../assets/brand/logo.png'



const router = useRouter()

const message = useMessage()

const username = ref('admin')

const password = ref('')

const loading = ref(false)



async function submit() {

  loading.value = true

  try {

    await api.login(username.value, password.value)

    message.success('登录成功')

    router.push({ name: 'dashboard' })

  } catch (error) {

    message.error(error instanceof Error ? error.message : '登录失败')

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

</style>


