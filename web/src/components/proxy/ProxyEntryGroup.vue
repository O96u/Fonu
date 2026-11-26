<template>
<section
              class="proxy-entry-card"
              :data-entry-id="group.entryId"
              :class="{ 'proxy-entry-card--collapsed': collapsed }"
            >
              <header class="proxy-entry-card__head">
                <span
                  v-if="sortable"
                  class="proxy-entry-drag-handle"
                  title="拖动排序"
                  @click.stop
                >
                  <n-icon :component="ReorderThreeOutline" />
                </span>
                <button
                  type="button"
                  class="proxy-entry-card__toggle"
                  @click="emit('toggle')"
                >
                  <n-icon :component="collapsed ? ChevronForwardOutline : ChevronDownOutline" />
                </button>
                <div
                  class="proxy-entry-card__intro"
                  role="button"
                  tabindex="0"
                  @click="emit('toggle')"
                  @keydown.enter.prevent="emit('toggle')"
                  @keydown.space.prevent="emit('toggle')"
                >
                  <div class="proxy-entry-card__title-row">
                    <span class="proxy-entry-card__name">{{ entryGroupDisplayName(group) }}</span>
                    <n-tag size="small" round :bordered="false" type="info">{{ group.listen.listen_port }}</n-tag>
                    <n-tag size="small" round :bordered="false" :type="group.listen.https_enabled ? 'success' : 'default'">
                      {{ group.listen.https_enabled ? 'HTTPS' : 'HTTP' }}
                    </n-tag>
                    <n-tag v-if="group.listen.listen_ipv4" size="small" round :bordered="false">IPv4</n-tag>
                    <n-tag v-if="group.listen.listen_ipv6" size="small" round :bordered="false">IPv6</n-tag>
                    <span class="proxy-entry-card__count">{{ group.rules.length }} 条规则</span>
                    <span v-if="group.enabledCount > 0" class="proxy-entry-card__running">{{ group.enabledCount }} 已启用</span>
                  </div>
                  <div v-if="group.portHasMixedEntries" class="proxy-entry-card__warn">
                    同端口存在不同监听配置，建议统一
                  </div>
                  <div v-if="collapsed && group.rules.length > 0" class="proxy-entry-card__preview">
                    {{ preview }}
                  </div>
                </div>
                <div class="proxy-entry-card__stats">
                  <span class="proxy-entry-card__stat" :title="'近期访问 IP 数（各规则相加） ' + traffic.connections">
                    <n-icon :component="PeopleOutline" />
                    {{ traffic.connections }}
                  </span>
                  <span class="proxy-entry-card__stat" title="当前上传">
                    <n-icon :component="ArrowUpOutline" />
                    {{ formatRateIdle(traffic.uploadRate) }}
                  </span>
                  <span class="proxy-entry-card__stat" title="当前下载">
                    <n-icon :component="ArrowDownOutline" />
                    {{ formatRateIdle(traffic.downloadRate) }}
                  </span>
                </div>
                <div class="proxy-entry-card__actions" @click.stop>
                  <n-space :size="4" align="center">
                    <n-tooltip trigger="hover">
                      <template #trigger>
                        <n-button
                          size="small"
                          quaternary
                          circle
                          class="proxy-entry-card__action-btn proxy-entry-card__action-btn--accent"
                          @click="emit('create')"
                        >
                          <template #icon><n-icon :component="AddOutline" :size="16" /></template>
                        </n-button>
                      </template>
                      新增规则
                    </n-tooltip>
                    <n-tooltip trigger="hover">
                      <template #trigger>
                        <n-button
                          size="small"
                          quaternary
                          circle
                          class="proxy-entry-card__action-btn"
                          @click="emit('discover')"
                        >
                          <template #icon><n-icon :component="GlobeOutline" :size="16" /></template>
                        </n-button>
                      </template>
                      发现内网服务
                    </n-tooltip>
                    <n-tooltip trigger="hover">
                      <template #trigger>
                        <n-button
                          size="small"
                          quaternary
                          circle
                          class="proxy-entry-card__action-btn"
                          :disabled="!group.entryId"
                          @click="emit('edit')"
                        >
                          <template #icon><n-icon :component="CreateOutline" :size="16" /></template>
                        </n-button>
                      </template>
                      编辑
                    </n-tooltip>
                    <n-tooltip trigger="hover">
                      <template #trigger>
                        <n-button
                          size="small"
                          quaternary
                          circle
                          class="proxy-entry-card__action-btn"
                          :disabled="!group.entryId"
                          @click="emit('duplicate')"
                        >
                          <template #icon><n-icon :component="CopyOutline" :size="16" /></template>
                        </n-button>
                      </template>
                      复制
                    </n-tooltip>
                    <n-tooltip trigger="hover">
                      <template #trigger>
                        <n-button
                          size="small"
                          quaternary
                          circle
                          type="error"
                          class="proxy-entry-card__action-btn proxy-entry-card__action-btn--danger"
                          :disabled="!group.entryId"
                          @click="emit('delete')"
                        >
                          <template #icon><n-icon :component="TrashOutline" :size="16" /></template>
                        </n-button>
                      </template>
                      删除
                    </n-tooltip>
                  </n-space>
                </div>
              </header>
  <slot />
</section>
</template>
<script setup lang="ts">
import {NButton,NIcon,NTag,NSpace,NTooltip} from 'naive-ui'
import {AddOutline,ArrowDownOutline,ArrowUpOutline,ChevronDownOutline,ChevronForwardOutline,ReorderThreeOutline,PeopleOutline,GlobeOutline,CreateOutline,CopyOutline,TrashOutline} from '@vicons/ionicons5'
import {entryGroupDisplayName,type ProxyEntryGroup} from '../../utils/proxyEntries'
import {formatRateIdle} from '../../utils/format'
defineProps<{group:ProxyEntryGroup;collapsed:boolean;sortable:boolean;preview:string;traffic:{connections:number;uploadRate:number;downloadRate:number}}>()
const emit=defineEmits<{toggle:[];create:[];discover:[];edit:[];duplicate:[];delete:[]}>()
</script>
<style scoped>
.proxy-entry-stack--sortable .proxy-entry-drag-handle {
  cursor: grab;
}

.proxy-entry-stack--sortable .proxy-entry-drag-handle:active {
  cursor: grabbing;
}

.proxy-entry-drag-handle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  color: var(--fonu-text-secondary);
  flex-shrink: 0;
}

.proxy-entry-card__toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--fonu-text-secondary);
  cursor: pointer;
  flex-shrink: 0;
}

.proxy-entry-card__head {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-2);
  padding: 12px 14px;
  background: var(--fonu-bg-subtle, rgba(0, 0, 0, 0.02));
  border-bottom: 1px solid transparent;
}

.proxy-entry-card__intro {
  flex: 1;
  min-width: 0;
  cursor: pointer;
}

.proxy-entry-card__head:focus-within .proxy-entry-card__intro:focus-visible {
  outline: 2px solid rgba(16, 185, 129, 0.45);
  outline-offset: 2px;
  border-radius: 6px;
}

.proxy-entry-card {
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius-lg, 12px);
  background: var(--fonu-surface, #fff);
  overflow: hidden;
}

.proxy-entry-card--collapsed .proxy-entry-card__head {
  border-bottom-color: transparent;
}

.proxy-entry-card:not(.proxy-entry-card--collapsed) .proxy-entry-card__head {
  border-bottom-color: var(--fonu-border);
}

.proxy-entry-card__title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.proxy-entry-card__name {
  font-size: 15px;
  font-weight: 700;
  color: var(--fonu-text);
  line-height: 1.3;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proxy-entry-card__preview {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proxy-entry-card__count,
.proxy-entry-card__running {
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.proxy-entry-card__running {
  color: #059669;
}

.proxy-entry-card__warn {
  margin-top: 4px;
  font-size: 11px;
  color: #d97706;
  line-height: 1.3;
}

.proxy-entry-card__stats {
  display: none;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.proxy-entry-card__stat {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-family: var(--fonu-mono);
  color: var(--fonu-text-secondary);
  white-space: nowrap;
}

.proxy-entry-card__actions {
  flex-shrink: 0;
}

.proxy-entry-card__actions :deep(.n-space) {
  flex-wrap: nowrap;
}

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn) {
  width: 32px;
  height: 32px;
  color: var(--fonu-text-secondary);
  transition: color 0.15s, background 0.15s;
}

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn:hover) {
  color: var(--fonu-text);
  background: rgba(15, 23, 42, 0.06);
}

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn--accent) {
  color: #059669;
}

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn--accent:hover) {
  color: #047857;
  background: rgba(16, 185, 129, 0.12);
}

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn--danger) {
  color: #ef4444;
}

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn--danger:hover) {
  color: #dc2626;
  background: rgba(239, 68, 68, 0.1);
}

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn.n-button--disabled) {
  opacity: 0.35;
}

.proxy-entry-card__body {
  padding: 0;
}

.proxy-entry-card__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-5) var(--fonu-space-4);
}

.proxy-entry-card { border-radius:var(--fonu-radius); }

.proxy-entry-card__head { background:var(--fonu-bg-muted); padding:10px 12px; }

.proxy-entry-card__running { color:var(--fonu-success); }

.proxy-entry-card__warn { color:var(--fonu-warning); }

.proxy-entry-card__head:focus-within .proxy-entry-card__intro:focus-visible { outline-color:var(--fonu-brand); }

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn--accent) { color:var(--fonu-brand); }

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn--accent:hover) { color:var(--fonu-brand-hover); background:var(--fonu-brand-soft); }
@media(min-width:960px){.proxy-entry-card__stats{display:flex}}
@media(max-width:760px){.proxy-entry-card__head{flex-wrap:wrap}.proxy-entry-card__intro{flex:1 1 150px}.proxy-entry-card__actions{max-width:100%}}
</style>
