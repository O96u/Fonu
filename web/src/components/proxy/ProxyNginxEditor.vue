<template>
<div class="proxy-modal__pane proxy-modal__pane--nginx">
            <template v-if="!editing">
              <n-empty description="请先保存规则后再配置 Nginx">
                <template #extra>
                  <n-button type="primary" @click="emit('basic')">去填写基础配置</n-button>
                </template>
              </n-empty>
            </template>
            <template v-else>
              <n-alert v-if="ruleDirty" type="warning" :bordered="false">基础配置、安全策略或监听尚未保存。请先保存规则，再发布 Nginx 文本。</n-alert>
              <n-alert v-if="nginxEditMode === 'custom'" type="info" :bordered="false" class="nginx-pane-alert">
                手动模式下，修改基础/安全设置后需先保存规则，再在此处同步 Nginx 文本。
              </n-alert>
              <div class="log-panel-head">
                <div class="nginx-mode-toggle">
                  <n-button
                    size="small"
                    quaternary
                    :type="nginxEditMode === 'auto' ? 'primary' : 'default'"
                    @click="emit('mode','auto')"
                  >
                    自动生成
                  </n-button>
                  <n-button
                    size="small"
                    quaternary
                    :type="nginxEditMode === 'custom' ? 'primary' : 'default'"
                    @click="emit('mode','custom')"
                  >
                    手动编辑
                  </n-button>
                </div>
                <div class="log-panel-actions">
                  <n-button
                    v-if="nginxEditMode === 'custom'"
                    size="small"
                    quaternary
                    type="primary"
                    :loading="nginxSaving"
                    :disabled="!nginxDirty||ruleDirty||nginxLoading"
                    @click="emit('save')"
                  >
                    保存 Nginx
                  </n-button>
                  <n-button
                    v-if="nginxEditMode === 'custom' && nginxBackups.length > 0"
                    size="small"
                    quaternary
                    :loading="nginxSaving"
                    :disabled="ruleDirty||nginxLoading"
                    @click="emit('rollback')"
                  >
                    回滚
                  </n-button>
                  <n-button
                    v-if="nginxEditMode === 'custom' && nginxServerMode === 'custom'"
                    size="small"
                    quaternary
                    :loading="nginxSaving"
                    :disabled="ruleDirty||nginxLoading"
                    @click="emit('reset')"
                  >
                    恢复自动生成
                  </n-button>
                </div>
              </div>
              <n-spin :show="nginxLoading" class="nginx-editor-spin nginx-editor-spin--modal">
                <NginxCodeEditor
                  :model-value="nginxDraft"
                  embedded
                  :readonly="nginxEditMode === 'auto'||nginxLoading||nginxSaving"
                  placeholder="server { ... }"
                  @update:model-value="emit('update:nginxDraft', $event)"
                />
              </n-spin>
            </template>
          </div>
</template>
<script setup lang="ts">
import {defineAsyncComponent} from 'vue'
import {NEmpty,NButton,NAlert,NSpin} from 'naive-ui'
const NginxCodeEditor=defineAsyncComponent(()=>import('../NginxCodeEditor.vue'))
defineProps<{
 editing:boolean
 nginxEditMode:'auto'|'custom'
 nginxServerMode:'auto'|'custom'
 nginxDraft:string
 ruleDirty:boolean
 nginxDirty:boolean
 nginxSaving:boolean
 nginxLoading:boolean
 nginxBackups:{name:string;created_at:string}[]
}>()
const emit=defineEmits<{basic:[];mode:[value:'auto'|'custom'];save:[];rollback:[];reset:[];'update:nginxDraft':[value:string]}>()
</script>
<style scoped>
.nginx-pane-alert {
  margin-bottom: var(--fonu-space-2);
  flex-shrink: 0;
}

.nginx-mode-toggle {
  display: inline-flex;
  gap: 4px;
}

.nginx-editor-spin--modal {
  min-height: min(420px, 50vh);
}

.nginx-editor-spin {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.nginx-editor-spin :deep(.n-spin-container),
.nginx-editor-spin :deep(.n-spin-content) {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.log-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
  flex-shrink: 0;
}

.log-panel-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.log-panel-actions :deep(.n-button) {
  height: 22px;
  padding: 0 8px;
  font-size: 12px;
}

.proxy-modal__pane {
  width: 100%;
  box-sizing: border-box;
  padding: var(--fonu-space-4) var(--fonu-space-5) 0;
}

.proxy-modal__pane--nginx {
  display: flex;
  flex-direction: column;
  min-height: min(480px, 55vh);
}

.proxy-modal__pane--nginx .nginx-editor-spin {
  flex: 1;
  min-height: 0;
}

.proxy-modal__pane :deep(.n-form) {
  width: 100%;
}

.proxy-modal__pane :deep(.n-form-item .n-form-item-blank) {
  display: block;
  width: 100%;
}
.proxy-modal__pane{padding:0!important;min-width:0}.proxy-modal__pane--nginx{min-height:0;gap:8px}.log-panel-head{flex-wrap:wrap}.nginx-editor-spin--modal{height:min(430px,48dvh);min-height:200px}
</style>
