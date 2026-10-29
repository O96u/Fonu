<template>
  <n-popover
    v-model:show="open"
    trigger="manual"
    placement="bottom-start"
    :width="300"
    :show-arrow="false"
    class="dir-path-popover"
    @clickoutside="open = false"
  >
    <template #trigger>
      <n-input
        :value="value"
        :placeholder="placeholder"
        :size="size"
        @update:value="onInput"
        @click="openPicker"
      >
        <template #suffix>
          <n-button quaternary circle :size="size" title="浏览目录" @click.stop="openPicker">
            <template #icon>
              <n-icon :component="FolderOpenOutline" />
            </template>
          </n-button>
        </template>
      </n-input>
    </template>

    <div class="dir-picker">
      <button type="button" class="dir-picker__current" @click="selectPath(browsePath)">
        <n-icon :component="FolderOutline" class="dir-picker__current-icon" />
        <span class="dir-picker__current-text">{{ browsePath || '/' }}</span>
      </button>

      <n-spin v-if="loading" size="small" class="dir-picker__spin" />

      <div v-else class="dir-picker__list">
        <button
          v-for="entry in entries"
          :key="entry.path"
          type="button"
          class="dir-picker__item"
          @click="enter(entry.path)"
        >
          <n-icon :component="FolderOutline" class="dir-picker__item-icon" />
          <span>{{ entry.name }}</span>
        </button>
        <div v-if="entries.length === 0" class="dir-picker__empty">没有子文件夹</div>
      </div>

      <div class="dir-picker__footer">
        <n-button size="tiny" quaternary :disabled="!parentPath" @click="enter(parentPath)">上一级</n-button>
        <n-button size="tiny" type="primary" @click="selectPath(browsePath)">选择此目录</n-button>
      </div>
    </div>
  </n-popover>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { FolderOpenOutline, FolderOutline } from '@vicons/ionicons5'
import { NButton, NIcon, NInput, NPopover, NSpin, useMessage } from 'naive-ui'
import { api } from '../api/client'
import type { FsDirEntry } from '../api/types'

const props = withDefaults(
  defineProps<{
    value: string
    placeholder?: string
    size?: 'small' | 'medium' | 'large'
  }>(),
  {
    placeholder: '绝对路径',
    size: 'small',
  },
)

const emit = defineEmits<{
  'update:value': [value: string]
}>()

const message = useMessage()
const open = ref(false)
const loading = ref(false)
const browsePath = ref('')
const parentPath = ref('')
const entries = ref<FsDirEntry[]>([])

function onInput(v: string) {
  emit('update:value', v)
}

function seedPath() {
  const trimmed = props.value.trim()
  return trimmed || ''
}

async function load(path?: string) {
  loading.value = true
  try {
    const res = await api.listFsDirectories(path)
    browsePath.value = res.path
    parentPath.value = res.parent
    entries.value = res.entries
  } catch (error) {
    message.error(error instanceof Error ? error.message : '读取目录失败')
    open.value = false
  } finally {
    loading.value = false
  }
}

async function openPicker() {
  open.value = true
  await load(seedPath())
}

async function enter(path: string) {
  if (!path) return
  await load(path)
}

function selectPath(path: string) {
  if (!path) return
  emit('update:value', path)
  open.value = false
}
</script>

<style scoped>
.dir-picker {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 320px;
}

.dir-picker__current {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 8px;
  border: none;
  border-radius: 8px;
  background: var(--fonu-bg-subtle, rgba(0, 0, 0, 0.03));
  cursor: pointer;
  text-align: left;
  font-size: 13px;
  color: var(--fonu-text);
}

.dir-picker__current:hover {
  background: var(--fonu-bg-muted, rgba(0, 0, 0, 0.06));
}

.dir-picker__current-icon {
  flex-shrink: 0;
  color: #8b5cf6;
}

.dir-picker__current-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dir-picker__spin {
  padding: 16px 0;
  display: flex;
  justify-content: center;
}

.dir-picker__list {
  overflow-y: auto;
  max-height: 220px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.dir-picker__item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  text-align: left;
  font-size: 13px;
  color: var(--fonu-text);
}

.dir-picker__item:hover {
  background: var(--fonu-bg-subtle, rgba(0, 0, 0, 0.04));
}

.dir-picker__item-icon {
  flex-shrink: 0;
  color: #8b5cf6;
}

.dir-picker__empty {
  padding: 12px 8px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  text-align: center;
}

.dir-picker__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-top: 4px;
  border-top: 1px solid var(--fonu-border);
}
</style>
