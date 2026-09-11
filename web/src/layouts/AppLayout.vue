<template>
  <n-layout has-sider class="app-layout">
    <n-layout-sider
      class="app-sider"
      bordered
      collapse-mode="width"
      :collapsed-width="72"
      :width="240"
      :collapsed="collapsed"
      show-trigger="bar"
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="brand" :class="{ 'brand--collapsed': collapsed }">
        <div class="brand__logo">F</div>
        <div v-if="!collapsed" class="brand__text">
          <div class="brand__name">Fonu</div>
          <div class="brand__tagline">Simple Access for Your NAS</div>
        </div>
      </div>

      <n-menu
        class="app-menu"
        :value="activeKey"
        :collapsed="collapsed"
        :collapsed-width="72"
        :options="menuOptions"
        @update:value="handleMenu"
      />

      <div v-if="!collapsed" class="sider-footer">
        <div class="sider-footer__version">Fonu v0.1.0</div>
        <div class="sider-footer__status">
          <span class="status-dot" />
          <span>运行中</span>
        </div>
      </div>
    </n-layout-sider>

    <n-layout class="app-main">
      <header class="topbar">
        <div class="topbar__left">
          <span class="topbar__datetime">{{ datetime }}</span>
        </div>
        <div class="topbar__right">
          <n-tag size="small" :bordered="false" class="demo-tag">演示模式</n-tag>
          <n-button quaternary circle @click="toggleTheme">
            <template #icon>
              <n-icon :component="isDark ? MoonOutline : SunnyOutline" />
            </template>
          </n-button>
          <n-dropdown :options="userMenuOptions" @select="handleUserMenu">
            <n-button quaternary class="user-btn">
              <n-avatar round :size="32" class="user-avatar">{{ avatarText }}</n-avatar>
              <span class="user-name">管理员</span>
              <n-icon :component="ChevronDownOutline" />
            </n-button>
          </n-dropdown>
        </div>
      </header>

      <n-layout-content class="content">
        <div class="page-content">
          <router-view />
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NAvatar,
  NButton,
  NDropdown,
  NIcon,
  NLayout,
  NLayoutContent,
  NLayoutSider,
  NMenu,
  NTag,
  useMessage,
  type MenuOption,
} from 'naive-ui'
import {
  ChevronDownOutline,
  CloudOutline,
  CogOutline,
  DocumentTextOutline,
  GridOutline,
  LockClosedOutline,
  MoonOutline,
  SunnyOutline,
  SwapHorizontalOutline,
} from '@vicons/ionicons5'
import { api } from '../api/client'
import { useTheme } from '../composables/useTheme'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { isDark, toggleTheme } = useTheme()
const collapsed = ref(false)
const datetime = ref('')
let clockTimer: number | undefined

const menuDefs = [
  { label: '仪表盘', key: 'dashboard', icon: GridOutline },
  { label: '反向代理', key: 'proxies', icon: SwapHorizontalOutline },
  { label: 'DDNS', key: 'ddns', icon: CloudOutline },
  { label: '证书', key: 'certificates', icon: LockClosedOutline },
  { label: '日志', key: 'logs', icon: DocumentTextOutline },
  { label: '设置', key: 'settings', icon: CogOutline },
]

const menuOptions = computed<MenuOption[]>(() =>
  menuDefs.map((item) => ({
    label: item.label,
    key: item.key,
    icon: () => h(NIcon, null, { default: () => h(item.icon) }),
  })),
)

const activeKey = computed(() => {
  const name = route.name
  return typeof name === 'string' ? name : 'dashboard'
})

const avatarText = '管'

const userMenuOptions = [{ label: '退出登录', key: 'logout' }]

function updateClock() {
  const now = new Date()
  datetime.value = now.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function handleMenu(key: string) {
  router.push({ name: key })
}

async function handleUserMenu(key: string) {
  if (key === 'logout') {
    await api.logout()
    message.success('已退出登录')
    router.push({ name: 'login' })
  }
}

onMounted(() => {
  updateClock()
  clockTimer = window.setInterval(updateClock, 30_000)
})

onUnmounted(() => {
  if (clockTimer) clearInterval(clockTimer)
})
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
  background: var(--fonu-bg);
}

.app-sider {
  background: var(--fonu-surface) !important;
  border-right: 1px solid var(--fonu-border) !important;
}

.app-sider :deep(.n-layout-sider-scroll-container) {
  display: flex;
  flex-direction: column;
}

.brand {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-5) var(--fonu-space-4);
  border-bottom: 1px solid var(--fonu-border);
}

.brand--collapsed {
  justify-content: center;
  padding: var(--fonu-space-4) 0;
}

.brand__logo {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--fonu-brand);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  flex-shrink: 0;
}

.brand__name {
  font-size: 18px;
  font-weight: 700;
  color: var(--fonu-text);
  line-height: 1.2;
}

.brand__tagline {
  font-size: 11px;
  color: var(--fonu-text-muted);
  margin-top: 2px;
  line-height: 1.3;
}

.app-menu {
  flex: 1;
  padding: var(--fonu-space-3);
}

.app-menu :deep(.n-menu-item) {
  margin-bottom: 2px;
}

.app-menu :deep(.n-menu-item-content) {
  border-radius: var(--fonu-radius-sm) !important;
  padding-left: 14px !important;
}

.app-menu :deep(.n-menu-item-content--selected) {
  background: var(--fonu-brand) !important;
  color: #fff !important;
  font-weight: 500;
}

.app-menu :deep(.n-menu-item-content--selected .n-icon) {
  color: #fff !important;
}

.app-menu :deep(.n-menu-item-content--selected::before) {
  display: none !important;
}

.sider-footer {
  padding: var(--fonu-space-4);
  border-top: 1px solid var(--fonu-border);
}

.sider-footer__version {
  font-size: 12px;
  color: var(--fonu-text-muted);
  margin-bottom: var(--fonu-space-2);
}

.sider-footer__status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--fonu-success);
  font-weight: 500;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--fonu-success);
}

.app-main {
  background: var(--fonu-bg);
}

.topbar {
  height: var(--fonu-topbar-height);
  padding: 0 var(--fonu-space-5);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--fonu-surface);
  border-bottom: 1px solid var(--fonu-border);
}

.topbar__datetime {
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.topbar__right {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-2);
}

.demo-tag {
  background: rgba(59, 130, 246, 0.1) !important;
  color: var(--fonu-info) !important;
}

.user-btn {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-2);
}

.user-avatar {
  background: var(--fonu-brand-soft) !important;
  color: var(--fonu-brand) !important;
  font-size: 13px;
  font-weight: 600;
}

.user-name {
  font-size: 14px;
  color: var(--fonu-text);
}

.content {
  padding: var(--fonu-space-5);
}

@media (max-width: 768px) {
  .user-name,
  .topbar__datetime {
    display: none;
  }
}
</style>
