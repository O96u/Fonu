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
      <router-link class="brand" :class="{ 'brand--collapsed': collapsed }" :to="{ name: 'dashboard' }">
        <img class="brand__logo" :src="logoImg" alt="Fonu" />
        <div v-if="!collapsed" class="brand__text">
          <div class="brand__name">Fonu</div>
          <div class="brand__tagline">让 NAS 访问更简单</div>
        </div>
      </router-link>

      <n-menu
        class="app-menu"
        :value="activeKey"
        :collapsed="collapsed"
        :collapsed-width="72"
        :options="menuOptions"
        @update:value="handleMenu"
      />

      <div v-if="!collapsed" class="sider-footer">
        <div class="sider-footer__version-row">
          <div class="sider-footer__title">
            <span class="sider-footer__brand">Fonu</span>
            <span class="sider-footer__version">v{{ appVersion }}</span>
          </div>
          <a
            v-if="hasUpdate"
            class="sider-footer__update"
            :href="latestReleaseUrl"
            target="_blank"
            rel="noopener noreferrer"
          >
            <n-icon :component="ArrowUpOutline" />
            有新版本
            <n-icon :component="OpenOutline" />
          </a>
        </div>

        <div class="sider-footer__status">
          <n-icon :component="Ellipse" class="sider-footer__status-icon" />
          <span>运行中</span>
        </div>

        <a
          class="sider-footer__github"
          :href="GITHUB_REPO_URL"
          target="_blank"
          rel="noopener noreferrer"
        >
          <n-icon :component="LogoGithub" />
          <span>{{ GITHUB_REPO }}</span>
        </a>
      </div>
    </n-layout-sider>

    <n-layout class="app-main">
      <header class="topbar">
        <span class="topbar__datetime">{{ datetime }}</span>
        <div class="topbar__right">
          <n-button quaternary circle @click="toggleTheme">
            <template #icon>
              <n-icon :component="isDark ? MoonOutline : SunnyOutline" />
            </template>
          </n-button>
          <n-dropdown :options="userMenuOptions" @select="handleUserMenu">
            <n-button quaternary class="user-btn">
              <n-icon :component="PersonOutline" class="user-btn__icon" />
              <span class="user-name">管理员</span>
              <n-icon :component="ChevronDownOutline" class="user-btn__chevron" />
            </n-button>
          </n-dropdown>
        </div>
      </header>

      <n-layout-content class="content">
        <div class="page-shell page-body">
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
  NButton,
  NDropdown,
  NIcon,
  NLayout,
  NLayoutContent,
  NLayoutSider,
  NMenu,
  useMessage,
  type MenuOption,
} from 'naive-ui'
import {
  ArrowUpOutline,
  ChevronDownOutline,
  CloudOutline,
  CogOutline,
  DocumentTextOutline,
  Ellipse,
  GitNetworkOutline,
  GridOutline,
  LockClosedOutline,
  LogoGithub,
  MoonOutline,
  OpenOutline,
  PersonOutline,
  SunnyOutline,
  SwapHorizontalOutline,
} from '@vicons/ionicons5'
import { api } from '../api/client'
import { GITHUB_REPO, GITHUB_REPO_URL } from '../constants/app'
import { useTheme } from '../composables/useTheme'
import { fetchLatestRelease, isNewerVersion } from '../utils/version'
import logoImg from '../assets/brand/logo.png'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { isDark, toggleTheme } = useTheme()
const collapsed = ref(false)
const datetime = ref('')
const appVersion = ref(__APP_VERSION__)
const latestVersion = ref('')
const latestReleaseUrl = ref('')
const hasUpdate = ref(false)
let clockTimer: number | undefined

async function checkForUpdate() {
  const release = await fetchLatestRelease()
  if (!release) {
    return
  }

  latestVersion.value = release.version
  latestReleaseUrl.value = release.url
  hasUpdate.value = isNewerVersion(release.version, appVersion.value)
}

async function loadVersion() {
  try {
    const { version } = await api.getVersion()
    if (version) {
      appVersion.value = version
    }
  } catch {
    // keep build-time fallback
  }

  await checkForUpdate()
}

const menuDefs = [
  { label: '仪表盘', key: 'dashboard', icon: GridOutline },
  { label: 'DDNS', key: 'ddns', icon: CloudOutline },
  { label: '证书管理', key: 'certificates', icon: LockClosedOutline },
  { label: '反向代理', key: 'proxies', icon: SwapHorizontalOutline },
  { label: '内网穿透', key: 'frp', icon: GitNetworkOutline },
  { label: '日志中心', key: 'logs', icon: DocumentTextOutline },
  { label: '系统设置', key: 'settings', icon: CogOutline },
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

const userMenuOptions = [{ label: '退出登录', key: 'logout' }]

function updateClock() {
  const now = new Date()
  const date = now.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
  const time = now.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  const weekday = now.toLocaleDateString('zh-CN', { weekday: 'long' })
  datetime.value = `${date} ${time} · ${weekday}`
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
  loadVersion()
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
  height: 100vh;
  background: var(--fonu-bg);
}

.app-layout :deep(.n-layout-scroll-container) {
  min-height: 100%;
}

.app-sider {
  background: var(--fonu-surface) !important;
  border-right: 1px solid var(--fonu-border) !important;
  min-height: 100vh;
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
  color: inherit;
  text-decoration: none;
  cursor: pointer;
  transition: background 0.15s ease;
}

.brand:hover {
  background: var(--fonu-bg-muted);
}

.brand--collapsed {
  justify-content: center;
  padding: var(--fonu-space-4) 0;
}

.brand__logo {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  object-fit: cover;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.2);
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
  background: var(--fonu-brand-soft) !important;
  font-weight: 600;
}

.app-menu :deep(.n-menu-item-content--selected .n-menu-item-content-header),
.app-menu :deep(.n-menu-item-content--selected .n-menu-item-content-header *) {
  color: var(--fonu-brand-text) !important;
}

.app-menu :deep(.n-menu-item-content--selected .n-icon),
.app-menu :deep(.n-menu-item-content--selected .n-menu-item-content__icon) {
  color: var(--fonu-brand) !important;
}

.app-menu :deep(.n-menu-item-content--selected::before) {
  display: none !important;
}

.sider-footer {
  padding: var(--fonu-space-4);
  border-top: 1px solid var(--fonu-border);
}

.sider-footer__version-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: var(--fonu-space-4);
}

.sider-footer__title {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.sider-footer__brand {
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text);
}

.sider-footer__version {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.sider-footer__update {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  font-weight: 500;
  color: var(--fonu-brand);
  text-decoration: none;
  white-space: nowrap;
  transition: opacity 0.15s;
}

.sider-footer__update:hover {
  opacity: 0.8;
}

.sider-footer__update :deep(.n-icon) {
  font-size: 12px;
}

.sider-footer__status {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: var(--fonu-space-4);
  font-size: 12px;
  color: var(--fonu-success);
  font-weight: 500;
}

.sider-footer__status-icon {
  font-size: 8px;
  color: var(--fonu-success);
}

.sider-footer__github {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--fonu-text-secondary);
  text-decoration: none;
  transition: color 0.15s;
}

.sider-footer__github:hover {
  color: var(--fonu-text);
}

.sider-footer__github :deep(.n-icon) {
  font-size: 15px;
}

.app-main {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--fonu-bg);
}

.topbar {
  flex-shrink: 0;
  height: var(--fonu-topbar-height);
  padding: 0 var(--fonu-space-5);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-4);
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
  gap: var(--fonu-space-4);
}

.user-btn {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  padding: 0 var(--fonu-space-2);
}

.user-btn__icon,
.user-btn__chevron {
  font-size: 18px;
  color: var(--fonu-text-secondary);
}

.user-name {
  font-size: 14px;
  color: var(--fonu-text);
}

.content {
  flex: 1;
  padding: var(--fonu-space-5);
  overflow: auto;
}

.page-body {
  min-height: 100%;
}

@media (max-width: 768px) {
  .topbar,
  .content {
    padding-inline: var(--fonu-space-4);
  }

  .user-name,
  .topbar__datetime {
    display: none;
  }
}
</style>
