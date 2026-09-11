<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { NConfigProvider, NDialogProvider, NMessageProvider } from 'naive-ui'
import { useTheme } from './composables/useTheme'
import './styles/variables.css'

const { naiveTheme, themeOverrides } = useTheme()
</script>

<style>
* {
  box-sizing: border-box;
}

body {
  margin: 0;
  font-family: var(--fonu-font);
  background: var(--fonu-bg);
  color: var(--fonu-text);
}

.mono {
  font-family: var(--fonu-mono);
}

.text-secondary {
  color: var(--fonu-text-secondary);
}

.text-muted {
  color: var(--fonu-text-muted);
}

.page-content {
  max-width: var(--fonu-content-max);
  margin: 0 auto;
}

.form-narrow {
  max-width: var(--fonu-form-max);
}

.toolbar {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  flex-wrap: wrap;
  margin-bottom: var(--fonu-space-4);
  padding: var(--fonu-space-4) var(--fonu-space-5) 0;
}

.n-data-table .n-data-table-th {
  background: #f9fafb !important;
  font-weight: 500;
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

html[data-theme='dark'] .n-data-table .n-data-table-th,
html.dark .n-data-table .n-data-table-th {
  background: #1e293b !important;
}
</style>
