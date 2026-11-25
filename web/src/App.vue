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

html,
body,
#app {
  height: 100%;
}

body {
  margin: 0;
  overflow: hidden;
  font-family: var(--fonu-font);
  background: var(--fonu-bg);
  color: var(--fonu-text);
}

#app > .n-config-provider {
  height: 100%;
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

.page-shell,
.page-content {
  width: 100%;
}

.page-narrow {
  width: 100%;
  max-width: var(--fonu-form-max);
  margin-inline: auto;
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
  padding-top: var(--fonu-space-4);
}

.n-data-table .n-data-table-th {
  background: var(--fonu-bg-muted) !important;
  font-weight: 500;
  font-size: 13px;
  color: var(--fonu-text-secondary);
}
.page-body { min-width: 0; }
.page-body .n-data-table { font-variant-numeric: tabular-nums; }
.page-body .n-data-table-td { overflow-wrap: anywhere; }
.page-body .n-data-table-th { white-space: nowrap; }
.page-body .n-tabs-nav { min-width: 0; }
.page-body .n-button:focus-visible { outline: 2px solid var(--fonu-brand); outline-offset: 2px; }
@media (max-width: 760px) { .page-body .toolbar { gap: 8px; } }
</style>
