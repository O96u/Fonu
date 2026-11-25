<template>
  <n-layout has-sider class="app-layout" :class="{ 'app-layout--compact': collapsed }">
    <n-layout-sider
      class="app-sider"
      bordered
      collapse-mode="width"
      :collapsed-width="56"
      :width="208"
      :collapsed="collapsed"
       show-trigger="bar"
      @collapse="setNavigation(true)"
      @expand="setNavigation(false)"
    >
      <router-link
        v-if="!collapsed"
        class="brand"
        :class="{ 'brand--collapsed': collapsed }"
        :to="{ name: 'dashboard' }"
      >
        <img class="brand__logo" :src="logoImg" alt="Fonu" />
        <div v-if="!collapsed" class="brand__text">
          <div class="brand__name">Fonu</div>
          <div class="brand__tagline">让 NAS 访问更简单</div>
        </div>
      </router-link>

      <nav class="side-nav" aria-label="主导航"><button v-for="item in menuDefs" :key="item.key" type="button" :title="item.label" :aria-label="item.label" :aria-current="activeKey === item.key ? 'page' : undefined" @click="handleMenu(item.key)"><n-icon :component="item.icon"/><span v-if="!collapsed">{{ item.label }}</span></button></nav>

      <div v-if="!collapsed" class="sider-footer">
        <div class="sider-footer__row">
          <div class="sider-footer__status">
            <span class="sider-footer__icon">
              <n-icon :component="Ellipse" class="sider-footer__status-icon" />
            </span>
            <span>运行中</span>
          </div>
          <router-link
            class="sider-footer__version-chip"
            :class="{ 'sider-footer__version-chip--update': hasUpdate }"
            :to="{ name: 'settings', query: { tab: 'version' } }"
            :title="hasUpdate && latestVersion ? `有新版本 v${latestVersion}，打开更新日志` : `当前版本 v${appVersion}`"
          >
            <span class="sider-footer__version">v{{ appVersion }}</span>
            <span v-if="hasUpdate" class="sider-footer__update-dot" aria-label="有新版本" />
          </router-link>
        </div>

        <a
          class="sider-footer__github"
          :href="GITHUB_REPO_URL"
          target="_blank"
          rel="noopener noreferrer"
        >
          <span class="sider-footer__icon">
            <n-icon :component="LogoGithub" />
          </span>
          <span>{{ GITHUB_REPO }}</span>
        </a>
      </div>
    </n-layout-sider>

    <n-layout class="app-main">
      <header class="topbar">
        <div class="topbar__context"><span class="topbar__page">{{ route.name === 'dashboard' ? '概览' : route.meta.title }}</span></div>
        <div class="topbar__right">
          <n-button v-if="embedded" tag="a" :href="standaloneUrl" target="_blank" rel="noopener noreferrer" quaternary circle title="独立网页打开" aria-label="独立网页打开"><template #icon><n-icon :component="OpenOutline"/></template></n-button>
          <n-popover trigger="click" placement="bottom-end" :show-arrow="false">
            <template #trigger><n-button quaternary circle aria-label="主题颜色" title="主题颜色" class="theme-trigger"><template #icon><n-icon :component="ColorPaletteOutline" /></template></n-button></template>
            <div class="appearance-panel"><strong>外观设置</strong><p>背景风格与强调色独立选择 · 自动保存</p><div class="background-options" role="group" aria-label="背景风格"><button v-for="style in backgroundOptions" :key="style.value" type="button" :aria-pressed="backgroundStyle === style.value" @click="setBackgroundStyle(style.value)">{{ style.label }}</button></div><p>主题颜色</p>
              <div class="appearance-grid"><button v-for="preset in themePresets" :key="preset.value" type="button" :aria-pressed="themeAccent === preset.value" @click="setThemeAccent(preset.value)" :style="{ '--preview-accent': previewTheme(preset.value, isDark, backgroundStyle).brandText, '--preview-bg': previewTheme(preset.value, isDark, backgroundStyle).bg, '--preview-chrome': previewTheme(preset.value, isDark, backgroundStyle).chrome, '--preview-surface': previewTheme(preset.value, isDark, backgroundStyle).surface, '--preview-border': previewTheme(preset.value, isDark, backgroundStyle).border }">
                <span class="appearance-preview"><i/><span><b/><em/><em/></span></span>
                <span class="appearance-label"><i/>{{ preset.label }}<span v-if="themeAccent === preset.value" class="appearance-check">✓</span></span>
              </button></div>
            </div>
          </n-popover>
          <n-dropdown trigger="click" :options="modeOptions" @select="setThemeMode"><n-button quaternary circle aria-label="主题模式" title="主题模式"><template #icon><n-icon :component="isDark ? MoonOutline : SunnyOutline" /></template></n-button></n-dropdown>
          <n-popover trigger="click" placement="bottom-end" @update:show="loadNotifications"><template #trigger><n-button quaternary circle aria-label="通知" title="通知"><template #icon><n-icon :component="NotificationsOutline"/></template></n-button></template><div class="notification-pop"><strong>状态提醒</strong><p v-if="notificationLoading">正在获取状态…</p><p v-else-if="notificationError">状态获取失败，请重新打开。</p><template v-else><router-link v-for="n in notificationRows" :key="n.title" :to="{name:n.route}"><strong>{{ n.title }}</strong><small>{{ n.detail }}</small></router-link><p v-if="!notificationRows.length">暂无待处理事项</p></template><router-link :to="{name:'settings',query:{tab:'notify'}}">通知渠道设置</router-link></div></n-popover>
          <n-dropdown :options="userMenuOptions" @select="handleUserMenu">
            <n-button quaternary class="user-btn">
              <span class="user-avatar">{{ displayUsername.charAt(0).toUpperCase() }}</span>
              <span class="user-name">{{ displayUsername }}</span>
              <n-icon
                :component="ChevronDownOutline"
                class="user-btn__chevron"
              />
            </n-button>
          </n-dropdown>
        </div>
      </header>

      <n-modal
        v-model:show="usernameModalOpen"
        preset="card"
        title="修改用户名"
        class="user-account-modal"
        :style="{ width: 'min(420px, calc(100vw - 32px))' }"
        :mask-closable="!changingUsername"
      >
        <n-form
          label-placement="top"
          :show-feedback="false"
          class="user-account-form"
        >
          <n-form-item label="新用户名">
            <n-input
              v-model:value="usernameForm.new_username"
              autocomplete="username"
            />
          </n-form-item>
          <n-form-item label="当前密码">
            <n-input
              v-model:value="usernameForm.password"
              type="password"
              show-password-on="click"
              autocomplete="current-password"
              placeholder="验证身份"
            />
          </n-form-item>
        </n-form>
        <template #footer>
          <n-space justify="end">
            <n-button
              :disabled="changingUsername"
              @click="usernameModalOpen = false"
              >取消</n-button
            >
            <n-button
              type="primary"
              :loading="changingUsername"
              :disabled="!usernameChanged"
              @click="submitUsername"
            >
              保存
            </n-button>
          </n-space>
        </template>
      </n-modal>

      <n-modal
        v-model:show="passwordModalOpen"
        preset="card"
        title="修改密码"
        class="user-account-modal"
        :style="{ width: 'min(420px, calc(100vw - 32px))' }"
        :mask-closable="!changingPassword"
      >
        <n-form
          label-placement="top"
          :show-feedback="false"
          class="user-account-form"
        >
          <n-form-item label="当前密码">
            <n-input
              v-model:value="passwordForm.old_password"
              type="password"
              show-password-on="click"
              autocomplete="current-password"
            />
          </n-form-item>
          <n-form-item label="新密码">
            <n-input
              v-model:value="passwordForm.new_password"
              type="password"
              show-password-on="click"
              autocomplete="new-password"
            />
          </n-form-item>
          <n-form-item label="确认新密码">
            <n-input
              v-model:value="passwordForm.confirm_password"
              type="password"
              show-password-on="click"
              autocomplete="new-password"
            />
          </n-form-item>
        </n-form>
        <template #footer>
          <n-space justify="end">
            <n-button
              :disabled="changingPassword"
              @click="passwordModalOpen = false"
              >取消</n-button
            >
            <n-button
              type="primary"
              :loading="changingPassword"
              @click="submitPassword"
              >保存</n-button
            >
          </n-space>
        </template>
      </n-modal>

      <n-layout-content class="content">
        <div class="page-shell page-body">
          <router-view v-slot="{ Component }">
            <keep-alive include="DashboardView">
              <component :is="Component" />
            </keep-alive>
          </router-view>
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NButton,
  NDropdown,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NLayout,
  NLayoutContent,
  NLayoutSider,
  NModal,
  NPopover,
  NSpace,
  useMessage,
  type DropdownOption,
} from "naive-ui";
import {
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
  ColorPaletteOutline,
  NotificationsOutline,
  SunnyOutline,
  SwapHorizontalOutline,
} from "@vicons/ionicons5";
import { api, asList } from "../api/client";
import { routeAuthStatus, invalidateRouteAuth } from "../router";
import { GITHUB_REPO, GITHUB_REPO_URL } from "../constants/app";
import { useTheme, themePresets, previewTheme } from "../composables/useTheme";
import { fetchLatestRelease, isNewerVersion } from "../utils/version";
import logoImg from "../assets/brand/logo.png";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { isDark, themeMode, themeAccent, setThemeMode, setThemeAccent, backgroundStyle, setBackgroundStyle } = useTheme();
const modeOptions = computed<DropdownOption[]>(() => [{key:'system',label:'跟随系统'},{key:'light',label:'浅色'},{key:'dark',label:'深色'}].map(p => ({...p,label:(themeMode.value === p.key ? '✓ ' : '') + p.label})));

function isFpkEmbed(): boolean {
  try {
    return window.self !== window.top;
  } catch {
    return true;
  }
}

const backgroundOptions = [{value:'neutral',label:'经典中性'},{value:'tinted',label:'柔和主题'},{value:'colorful',label:'全局主题'}] as const;
const embedded = isFpkEmbed();
const standaloneUrl = window.location.href;
const viewportWidth = ref(window.innerWidth);
const compactViewport = computed(() => viewportWidth.value < (embedded ? 1320 : 1000));
function readNavigation(): boolean | null { try { const v = localStorage.getItem('fonu-nav-collapsed'); return v === null ? null : v === 'true' } catch { return null } }
const manualCollapsed = ref<boolean | null>(readNavigation());
const collapsed = computed(() => manualCollapsed.value ?? compactViewport.value);
function setNavigation(value: boolean) { manualCollapsed.value = value; try { localStorage.setItem('fonu-nav-collapsed', String(value)) } catch { /* storage unavailable */ } }
function updateViewport() { viewportWidth.value = window.innerWidth; }

const appVersion = ref(__APP_VERSION__);
const adminUsername = ref("");
const usernameModalOpen = ref(false);
const passwordModalOpen = ref(false);
const changingUsername = ref(false);
const changingPassword = ref(false);
const usernameForm = reactive({ new_username: "", password: "" });
const passwordForm = reactive({
  old_password: "",
  new_password: "",
  confirm_password: "",
});
let updateTimer: number | undefined;
let layoutActive = true;
const latestVersion = ref("");
const hasUpdate = ref(false);


async function checkForUpdate() {
  const release = await fetchLatestRelease();
  if (!release) {
    return;
  }

  latestVersion.value = release.version;
  hasUpdate.value = isNewerVersion(release.version, appVersion.value);
}

async function loadVersion() {
  try {
    const { version } = await api.getVersion();
    if (version && /^\d+\.\d+/.test(version.replace(/^v/i, "").trim())) {
      appVersion.value = version;
    }
  } catch {
    // keep build-time fallback
  }

  // Release metadata is nonessential during startup.
  if (layoutActive) updateTimer = window.setTimeout(() => { void checkForUpdate(); }, 2500);
}

const menuDefs = [
  { label: "仪表盘", key: "dashboard", icon: GridOutline },
  { label: "DDNS", key: "ddns", icon: CloudOutline },
  { label: "证书管理", key: "certificates", icon: LockClosedOutline },
  { label: "反向代理", key: "proxies", icon: SwapHorizontalOutline },
  { label: "内网穿透", key: "frp", icon: GitNetworkOutline },
  { label: "日志中心", key: "logs", icon: DocumentTextOutline },
  { label: "系统设置", key: "settings", icon: CogOutline },
];


const notificationRows = ref<{title:string;detail:string;route:string}[]>([]);
const notificationLoading = ref(false);
const notificationError = ref(false);
async function loadNotifications(show:boolean) {
  if (!show || notificationLoading.value) return;
  notificationLoading.value=true; notificationError.value=false;
  try { const [status,certs]=await Promise.all([api.getStatus(),api.listCertificates()]);
    notificationRows.value=asList(certs).filter(c=>c.days_left<=30).map(c=>({title:c.domain,detail:c.days_left<0?'证书已过期':'证书剩余 '+c.days_left+' 天',route:'certificates'}));
    if(status.nginx_status==='stopped') notificationRows.value.push({title:'Nginx 已停止',detail:'检查反向代理',route:'proxies'});
  } catch {notificationError.value=true;} finally {notificationLoading.value=false;}
}
const activeKey = computed(() => {
  const name = route.name;
  return typeof name === "string" ? name : "dashboard";
});

const displayUsername = computed(() => adminUsername.value.trim() || "管理员");

const usernameChanged = computed(
  () =>
    usernameForm.new_username.trim() !== "" &&
    usernameForm.new_username.trim() !== adminUsername.value,
);

const userMenuOptions: DropdownOption[] = [
  { label: "外观设置", key: "appearance" },
  { label: "系统设置", key: "settings" },
  { label: "版本信息", key: "version" },
  { type: "divider", key: "settings-divider" },
  { label: "修改用户名", key: "username" },
  { label: "修改密码", key: "password" },
  { type: "divider", key: "account-divider" },
  { label: "退出登录", key: "logout" },
];

function handleMenu(key: string) {
  router.push({ name: key });
}

async function loadAuthProfile() {
  try {
    const status = await routeAuthStatus();
    if (status.username) {
      adminUsername.value = status.username;
      usernameForm.new_username = status.username;
    }
  } catch {
    // ignore
  }
}

function openUsernameModal() {
  usernameForm.new_username = adminUsername.value;
  usernameForm.password = "";
  usernameModalOpen.value = true;
}

function openPasswordModal() {
  passwordForm.old_password = "";
  passwordForm.new_password = "";
  passwordForm.confirm_password = "";
  passwordModalOpen.value = true;
}

async function submitUsername() {
  const newUsername = usernameForm.new_username.trim();
  if (!newUsername) {
    message.warning("请填写登录用户名");
    return;
  }
  if (!usernameForm.password) {
    message.warning("修改用户名需填写当前密码");
    return;
  }
  changingUsername.value = true;
  try {
    await api.changeUsername(usernameForm.password, newUsername);
    adminUsername.value = newUsername;
    usernameForm.new_username = newUsername;
    usernameForm.password = "";
    usernameModalOpen.value = false;
    message.success("用户名已更新");
  } catch (error) {
    message.error(error instanceof Error ? error.message : "更新用户名失败");
  } finally {
    changingUsername.value = false;
  }
}

async function submitPassword() {
  if (!passwordForm.old_password || !passwordForm.new_password) {
    message.warning("请填写当前密码和新密码");
    return;
  }
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    message.warning("两次输入的新密码不一致");
    return;
  }
  changingPassword.value = true;
  try {
    await api.changePassword(
      passwordForm.old_password,
      passwordForm.new_password,
    );
    passwordForm.old_password = "";
    passwordForm.new_password = "";
    passwordForm.confirm_password = "";
    passwordModalOpen.value = false;
    invalidateRouteAuth();
    message.success("密码已更新");
  } catch (error) {
    message.error(error instanceof Error ? error.message : "更新密码失败");
  } finally {
    changingPassword.value = false;
  }
}

async function handleUserMenu(key: string) {
  if (['appearance','settings','version'].includes(key)) {
    await router.push({ name: 'settings', query: { tab: key === 'version' ? 'version' : 'general' } });
    return;
  }
  if (key === "username") {
    openUsernameModal();
    return;
  }
  if (key === "password") {
    openPasswordModal();
    return;
  }
  if (key === "logout") {
    await api.logout();
    invalidateRouteAuth();
    message.success("已退出登录");
    router.push({ name: "login" });
  }
}

onMounted(() => {
  loadVersion();
  loadAuthProfile();
  window.addEventListener("resize", updateViewport, { passive: true });
});

onUnmounted(() => {
  layoutActive = false;
  if (updateTimer !== undefined) window.clearTimeout(updateTimer);
  window.removeEventListener("resize", updateViewport);
});
</script>

<style scoped>
.app-layout {
  height: 100%;
  min-height: 0;
  overflow: hidden;
  background: var(--fonu-bg);
}

.app-layout :deep(.n-layout-scroll-container) {
  min-height: 100%;
}

.app-sider {
  background: var(--fonu-bg-muted) !important;
  border-right: 1px solid var(--fonu-border) !important;
  height: 100%;
  min-height: 0;
}

.app-sider :deep(.n-layout-sider-scroll-container) {
  display: flex;
  flex-direction: column;
  overflow-x: hidden;
}

.brand {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-5) var(--fonu-space-4);
  overflow: hidden;
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
  justify-content: flex-start;
  padding: var(--fonu-space-4) 0 var(--fonu-space-4) var(--fonu-space-4);
}

.brand__logo {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  object-fit: cover;
  flex-shrink: 0;
  box-shadow: none;
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
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 16px 14px;
  border-top: 1px solid var(--fonu-border);
}

.sider-footer__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 22px;
}

.sider-footer__version-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--fonu-text-muted) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--fonu-border) 80%, transparent);
  text-decoration: none;
  flex-shrink: 0;
  transition:
    background 0.15s,
    border-color 0.15s;
}

.sider-footer__version-chip--update {
  cursor: pointer;
  border-color: color-mix(in srgb, var(--fonu-error) 35%, var(--fonu-border));
  background: color-mix(in srgb, var(--fonu-error) 8%, transparent);
}

.sider-footer__version-chip--update:hover {
  background: color-mix(in srgb, var(--fonu-error) 14%, transparent);
  border-color: color-mix(in srgb, var(--fonu-error) 45%, var(--fonu-border));
}

.sider-footer__version {
  font-size: 11px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  color: var(--fonu-text-secondary);
  line-height: 1;
}

.sider-footer__version-chip--update .sider-footer__version {
  color: var(--fonu-text);
}

.sider-footer__update-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--fonu-error);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--fonu-error) 28%, transparent);
  flex-shrink: 0;
}

.sider-footer__status,
.sider-footer__github {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 22px;
  font-size: 12px;
  line-height: 1;
}

.sider-footer__status {
  color: var(--fonu-success);
  font-weight: 500;
}

.sider-footer__icon {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.sider-footer__status-icon {
  font-size: 8px;
  color: var(--fonu-success);
}

.sider-footer__github {
  color: var(--fonu-text-secondary);
  text-decoration: none;
  transition: color 0.15s;
}

.sider-footer__github:hover {
  color: var(--fonu-text);
}

.sider-footer__github :deep(.n-icon) {
  font-size: 14px;
}

.app-main {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
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
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-account-form :deep(.n-form-item) {
  margin-bottom: var(--fonu-space-4);
}

.user-account-form :deep(.n-form-item:last-child) {
  margin-bottom: 0;
}

.content {
  flex: 1;
  min-height: 0;
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
  .topbar__workspace,
  .topbar__separator {
    display: none;
  }
}
.topbar__context { display: flex; align-items: center; gap: 8px; min-width: 0; font-size: 13px; }
.topbar__workspace, .topbar__separator { color: var(--fonu-text-muted); }
.topbar__page { color: var(--fonu-text); font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.app-main { min-width: 0; }
.app-layout--compact .app-menu { padding: 8px 0; }
.app-layout--compact .content { padding: 16px; }
.app-layout--compact .topbar { padding-inline: 16px; }
@media (prefers-reduced-motion: reduce) { .app-layout :deep(*) { scroll-behavior: auto; } }
.topbar__right { gap:4px; }
.user-avatar { display:inline-grid; place-items:center; width:26px; height:26px; border-radius:50%; background:var(--fonu-brand-soft); color:var(--fonu-brand); font-weight:600; }
.user-btn { margin-left:8px; }
.app-layout--compact .user-name,.app-layout--compact .user-btn__chevron { display:none; }
.app-main :deep(.n-layout-scroll-container) { min-height:0; }
.app-main > :deep(.n-layout-scroll-container) { display:flex; flex-direction:column; height:100%; overflow:hidden; }
.content > :deep(.n-layout-scroll-container) { min-height:0; }
.app-layout--compact .content { padding:12px; }
.app-sider { background:var(--fonu-chrome)!important; }
.app-layout--compact .app-menu { padding:12px 4px; }
.app-menu :deep(.n-menu-item-content) { border-radius:10px!important; }
.topbar__right { gap:6px; }
.topbar__right > :deep(.n-button) { border-radius:9px; }
.appearance-panel { width:min(310px,calc(100vw - 48px)); padding:4px; }
.appearance-panel p { margin:4px 0 12px; color:var(--fonu-text-muted); font-size:12px; }
.appearance-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:8px; }
.appearance-grid button { min-width:0; padding:7px; border:1px solid var(--fonu-border); border-radius:9px; background:var(--fonu-surface); color:var(--fonu-text); text-align:left; cursor:pointer; }
.appearance-grid button[aria-pressed=true] { border-color:var(--preview-accent); box-shadow:0 0 0 1px var(--preview-accent); }
.appearance-preview { display:flex; gap:5px; height:42px; padding:5px; border-radius:5px; background:var(--fonu-bg); }
.appearance-preview > i { width:20%; background:var(--preview-accent); border-radius:3px; opacity:.85; }
.appearance-preview > span { flex:1; display:grid; gap:4px; grid-template-columns:1fr 1fr; }
.appearance-preview b { grid-column:1/-1; background:var(--preview-accent); border-radius:3px; opacity:.75; }
.appearance-preview em { background:var(--fonu-surface); border:1px solid var(--fonu-border); border-radius:3px; }
.appearance-label { display:flex; align-items:center; gap:5px; margin-top:5px; font-size:12px; }
.appearance-label i { width:9px;height:9px;border-radius:50%;background:var(--preview-accent); }
.appearance-check { margin-left:auto; color:var(--preview-accent);font-weight:bold; }
@media(max-width:600px) { .app-layout:not(.app-layout--compact) .app-sider { position:absolute; z-index:20; box-shadow:var(--fonu-shadow-md); } .app-layout:not(.app-layout--compact) .app-main { margin-left:56px; } }
.rail-toggle { flex-shrink:0; display:grid; place-items:center; width:40px; height:40px; margin:4px auto 0; border:0; border-radius:9px; background:transparent; color:var(--fonu-text-secondary); font-size:20px; cursor:pointer; }
.rail-toggle:hover { background:var(--fonu-brand-soft); color:var(--fonu-brand-text); }
.app-layout--compact .app-menu { padding:4px 0; }
.app-layout--compact .app-menu :deep(.n-menu-item) { margin:4px 0; }
.app-layout--compact .app-menu :deep(.n-menu-item-content) { width:44px; height:42px; box-sizing:border-box; margin-inline:auto; padding:0!important; justify-content:center; }
.app-layout--compact .app-menu :deep(.n-menu-item-content__icon) { margin:0!important; width:20px!important; min-width:20px; flex:0 0 20px; }
.app-layout--compact .app-menu :deep(.n-menu-item-content-header) { display:none; }
.app-layout--compact .app-menu :deep(.n-menu-item-content::before) { inset:0; }
.rail-toggle:focus-visible { outline:2px solid var(--fonu-brand); outline-offset:2px; }
.side-nav{flex:1;display:flex;flex-direction:column;gap:4px;padding:8px;min-height:0;overflow-y:auto;overflow-x:hidden}
.side-nav button,.nav-toggle{display:flex;align-items:center;gap:12px;min-height:40px;padding:0 12px;border:0;border-radius:8px;background:transparent;color:var(--fonu-text-secondary);cursor:pointer;white-space:nowrap;text-align:left}
.side-nav button[aria-current=page]{background:var(--fonu-brand-soft);color:var(--fonu-brand-text);font-weight:600}
.side-nav button:hover,.nav-toggle:hover{background:var(--fonu-brand-soft);color:var(--fonu-brand-text)}
.side-nav button :deep(.n-icon),.nav-toggle :deep(.n-icon){font-size:20px;flex:0 0 20px;display:grid;place-items:center}
.app-layout--compact .side-nav{padding:8px 0;align-items:center}
.app-layout--compact .side-nav button,.app-layout--compact .nav-toggle{width:42px;padding:0;justify-content:center}
.nav-toggle{flex-shrink:0;margin:6px 8px 10px;justify-content:center;border:1px solid var(--fonu-border)}
.app-layout--compact .nav-toggle{margin:6px auto 10px}
.notification-pop{width:min(300px,calc(100vw - 48px));padding:4px;display:flex;flex-direction:column;gap:10px}
.notification-pop a{display:flex;flex-direction:column;color:var(--fonu-brand-text);text-decoration:none;padding:8px;border-top:1px solid var(--fonu-border)}
.notification-pop small,.notification-pop p{color:var(--fonu-text-secondary);margin:4px 0;font-size:12px}
.topbar { background:var(--fonu-chrome); }
.app-sider :deep(.n-layout-sider-scroll-container) { background:var(--fonu-chrome); }
.appearance-preview { background:var(--preview-bg); border:1px solid var(--preview-border); }
.appearance-preview > i { background:var(--preview-chrome); border:1px solid var(--preview-border); border-left:3px solid var(--preview-accent); opacity:1; }
.appearance-preview b { background:var(--preview-accent); }
.appearance-preview em { background:var(--preview-surface); border-color:var(--preview-border); }
.background-options{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:5px;margin-bottom:12px}
.background-options button{padding:7px 3px;border:1px solid var(--fonu-border);border-radius:7px;background:var(--fonu-surface);color:var(--fonu-text-secondary);font-size:12px;cursor:pointer}
.background-options button[aria-pressed=true]{background:var(--fonu-brand-soft);color:var(--fonu-brand-text);border-color:var(--fonu-brand);font-weight:600}
</style>
