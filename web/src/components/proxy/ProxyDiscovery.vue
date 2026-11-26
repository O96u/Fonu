<template>
  <n-modal :show="showDiscoveryModal" @update:show="emit('update:showDiscoveryModal', $event)" :mask-closable="false" transform-origin="center">
    <div class="discovery-modal">
      <div class="discovery-modal__header">
        <div>
          <h3 class="modal-title">发现内网服务</h3>
          <p class="discovery-modal__subtitle">
            {{ discoveryTarget ? '扫描常见端口并识别 HTTP 服务，勾选后填写前端域名添加到当前入口' : '扫描常见端口并识别 HTTP 服务，勾选后批量创建反向代理规则' }}
          </p>
        </div>
        <n-button size="small" quaternary @click="emit('update:showDiscoveryModal', false)">
          <template #icon><n-icon :component="CloseOutline" /></template>
        </n-button>
      </div>

      <div class="discovery-modal__toolbar">
        <n-input
          :value="discoveryHost" @update:value="emit('update:discoveryHost', $event)"
          size="small"
          placeholder="扫描地址，如 127.0.0.1"
          class="discovery-modal__host"
          :disabled="discoveryLoading"
          @keydown.enter.prevent="emit('scan')"
        />
        <n-button size="small" type="primary" :loading="discoveryLoading" @click="emit('scan')">
          {{ discoveryLoading ? '扫描中…' : '开始发现' }}
        </n-button>
        <span v-if="discoveryLoading" class="discovery-modal__status discovery-modal__status--loading">
          <n-spin size="small" />
          正在扫描 {{ discoveryScanHost }} …
        </span>
        <span v-else-if="discoverySummary" class="discovery-modal__summary">{{ discoverySummary }}</span>
      </div>

      <div class="discovery-modal__content">
        <div v-if="discoveryLoading && discoveryRows.length === 0" class="discovery-modal__loading">
          <n-spin size="medium" />
          <span>正在扫描 {{ discoveryScanHost }} 的常见 Web 端口…</span>
        </div>
        <n-empty
          v-else-if="!discoveryLoading && discoveryRows.length === 0"
          description="暂未发现可添加的服务，请确认目标地址后重试"
        />
        <div v-else class="discovery-modal__list-wrap">
          <div v-if="discoveryLoading" class="discovery-modal__scan-overlay">
            <n-spin size="medium" />
            <span>正在重新扫描 {{ discoveryScanHost }} …</span>
          </div>
          <div class="discovery-modal__list">
            <div class="discovery-modal__list-head">
              <n-checkbox
                class="discovery-modal__select-all"
                :checked="discoverySelectAllChecked"
                :indeterminate="discoverySelectAllIndeterminate"
                :disabled="discoveryLoading"
                aria-label="全选"
                @update:checked="emit('selectAll', $event)"
              />
              <span>服务</span>
              <span>规则名称</span>
              <span>前端域名</span>
            </div>
            <div
              v-for="row in discoveryRows"
              :key="row.id"
              class="discovery-item"
            >
              <n-checkbox :checked="row.selected" @update:checked="emit('patchRow', row.id, { selected: $event })" :disabled="discoveryLoading" />
              <div class="discovery-item__info">
                <div class="discovery-item__title mono">{{ row.service.upstream }}</div>
                <div class="discovery-item__meta">{{ row.service.name }}</div>
                <div v-if="row.service.suggestion" class="discovery-item__hint">{{ row.service.suggestion }}</div>
              </div>
              <n-input
                :value="row.ruleName" @update:value="emit('patchRow', row.id, { ruleName: $event })"
                size="small"
                maxlength="100"
                placeholder="规则名称"
                :disabled="!row.selected || discoveryLoading"
              />
              <n-input
                :value="row.hostname" @update:value="emit('patchRow', row.id, { hostname: $event })"
                size="small"
                placeholder="例如 nas.example.com"
                :disabled="!row.selected || discoveryLoading"
              />
            </div>
          </div>
        </div>
      </div>

      <div class="modal-footer discovery-modal__footer">
        <n-button @click="emit('update:showDiscoveryModal', false)" :disabled="discoveryLoading">取消</n-button>
        <n-button :loading="discoveryLoading" @click="emit('scan')">
          {{ discoveryLoading ? '扫描中…' : '重新扫描' }}
        </n-button>
        <n-button
          type="primary"
          :loading="discoveryAdding"
          :disabled="!hasSelectedDiscoveryRows || discoveryLoading"
          @click="emit('submit')"
        >
          {{ discoveryTarget ? '添加为子规则' : '批量创建规则' }}
        </n-button>
      </div>
    </div>
  </n-modal>
</template>
<script setup lang="ts">
import { NModal, NButton, NIcon, NInput, NCheckbox, NSpin, NEmpty } from 'naive-ui'
import { CloseOutline } from '@vicons/ionicons5'
import type { DiscoveredService } from '../../api/types'
import type { ProxyEntryGroup } from '../../utils/proxyEntries'
interface DiscoveryRow {
  id: string
  selected: boolean
  service: DiscoveredService
  ruleName: string
  hostname: string
}


defineProps<{
  showDiscoveryModal: boolean
  discoveryHost: string
  discoveryTarget: ProxyEntryGroup | null
  discoveryLoading: boolean
  discoveryScanHost: string
  discoverySummary: string
  discoveryRows: DiscoveryRow[]
  discoverySelectAllChecked: boolean
  discoverySelectAllIndeterminate: boolean
  discoveryAdding: boolean
  hasSelectedDiscoveryRows: boolean
}>()
const emit = defineEmits<{
  'update:showDiscoveryModal': [value: boolean]
  'update:discoveryHost': [value: string]
  scan: []
  selectAll: [checked: boolean]
  patchRow: [id: string, patch: Partial<DiscoveryRow>]
  submit: []
}>()
</script>
<style scoped>
.modal-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-4) var(--fonu-space-5) var(--fonu-space-5);
  border-top: 1px solid var(--fonu-border);
  margin-top: var(--fonu-space-2);
}

.mono { font-family: var(--fonu-mono); }
.discovery-modal {
  width: min(760px, 94vw);
  max-height: 82vh;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: var(--fonu-space-4);
  background: var(--fonu-surface, #fff);
  border-radius: var(--fonu-radius-lg, 12px);
}

.discovery-modal__header,
.discovery-modal__toolbar,
.discovery-modal__footer {
  flex-shrink: 0;
}

.discovery-modal__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  margin-bottom: var(--fonu-space-3);
}

.discovery-modal__subtitle {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.discovery-modal__toolbar {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-2);
  flex-wrap: wrap;
  margin-bottom: var(--fonu-space-3);
}

.discovery-modal__host {
  width: 180px;
}

.discovery-modal__summary {
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.discovery-modal__list-head .discovery-modal__select-all {
  justify-self: start;
}

.discovery-modal__status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.discovery-modal__status--loading {
  color: var(--fonu-accent, #18a058);
}

.discovery-modal__content {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.discovery-modal__loading {
  display: flex;
  flex: 1;
  min-height: 120px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-6) 0;
  color: var(--fonu-text-secondary);
  font-size: 13px;
}

.discovery-modal__list-wrap {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.discovery-modal__scan-overlay {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--fonu-space-3);
  background: color-mix(in srgb, var(--fonu-surface, #fff) 82%, transparent);
  backdrop-filter: blur(1px);
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.discovery-modal__list {
  flex: 1;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius-md, 8px);
}

.discovery-modal__list-head,
.discovery-item {
  display: grid;
  grid-template-columns: 36px minmax(180px, 1.2fr) minmax(120px, 0.9fr) minmax(160px, 1fr);
  gap: var(--fonu-space-2);
  align-items: center;
  padding: 10px 12px;
}

.discovery-modal__list-head {
  position: sticky;
  top: 0;
  z-index: 1;
  font-size: 12px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
  background: var(--fonu-bg-subtle, rgba(0, 0, 0, 0.02));
  border-bottom: 1px solid var(--fonu-border);
}

.discovery-item + .discovery-item {
  border-top: 1px solid var(--fonu-border);
}

.discovery-item--offline {
  opacity: 0.55;
}

.discovery-item__title {
  font-size: 12px;
  color: var(--fonu-text-secondary);
  word-break: break-all;
}

.discovery-item__meta {
  margin-top: 4px;
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text-primary);
}

.discovery-item__hint {
  margin-top: 4px;
  font-size: 11px;
  color: var(--fonu-text-secondary);
}

.discovery-modal__footer {
  margin-top: var(--fonu-space-4);
}
</style>
