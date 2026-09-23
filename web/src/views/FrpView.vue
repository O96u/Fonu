<template>
  <PageHeader
    title="内网穿透"
    description="通过 VPS + FRP，让你即使没有公网 IP，也能随时访问 NAS 上的服务。"
  >
    <template #actions>
      <n-button quaternary @click="showHelp = true">
        <template #icon><n-icon :component="HelpCircleOutline" /></template>
        使用说明
      </n-button>
    </template>
  </PageHeader>

  <LoadError v-if="pageError" :message="pageError" @retry="load" />

  <n-spin v-else :show="loading">
    <div class="frp-page">
      <div class="top-grid">
        <FonuCard class="status-card" flush>
          <div class="status-card__body">
            <div class="status-hero" :class="statusHeroClass">
              <div class="status-hero__main">
                <div class="status-hero__icon-wrap">
                  <n-icon :component="statusHeroIcon" />
                </div>
                <div>
                  <div class="status-hero__title">{{ connectionTitle }}</div>
                  <div class="status-hero__sub">{{ connectionSubtitle }}</div>
                </div>
              </div>
              <div v-if="frpForm.enabled" class="status-hero__aside">
                <span
                  class="status-hero__badge"
                  :class="{ 'status-hero__badge--on': frpStatus.connected }"
                >
                  <span class="status-hero__badge-dot" />
                  {{ frpStatus.connected ? "运行中" : "未连接" }}
                </span>
                <div v-if="frpStatus.connected_at" class="status-hero__time">
                  自 {{ connectedAtLabel }} 连接
                </div>
              </div>
            </div>

            <div class="status-flow-panel">
              <div class="status-flow">
                <div class="status-flow__step">
                  <n-icon :component="PersonOutline" />
                  <span>外网用户</span>
                </div>
                <span class="status-flow__arrow">→</span>
                <div class="status-flow__step status-flow__step--accent">
                  <n-icon :component="CloudOutline" />
                  <span>frps</span>
                </div>
                <span class="status-flow__arrow">→</span>
                <div class="status-flow__step">
                  <n-icon :component="DesktopOutline" />
                  <span>frpc</span>
                </div>
                <span class="status-flow__arrow">→</span>
                <div class="status-flow__step">
                  <n-icon :component="ServerOutline" />
                  <span>Fonu Nginx</span>
                </div>
              </div>
            </div>

            <div class="status-metrics status-metrics--row">
              <div class="status-metric">
                <div class="status-metric__label">服务器地址</div>
                <code class="status-metric__value">{{
                  displayServerEndpoint
                }}</code>
              </div>
              <div class="status-metric">
                <div class="status-metric__label">连接时间</div>
                <div class="status-metric__value">{{ connectedAtLabel }}</div>
              </div>
              <div class="status-metric">
                <div class="status-metric__label">运行时长</div>
                <div class="status-metric__value">{{ uptimeLabel }}</div>
              </div>
              <div class="status-metric">
                <div class="status-metric__label">frpc 版本</div>
                <div class="status-metric__value">
                  {{ frpStatus.client_version || "—" }}
                </div>
              </div>
            </div>

            <div class="status-overview">
              <div class="status-overview__head">
                <span>已配置服务</span>
              </div>

              <template v-if="frpForm.enabled">
                <div v-if="hasConfiguredRoutes" class="status-overview__groups">
                  <div v-if="domainList.length" class="status-overview__group">
                    <div class="status-overview__group-label">Web 域名</div>
                    <div ref="domainChipsRef" class="status-overview__chips">
                      <span
                        v-for="domain in domainList.slice(
                          0,
                          domainVisibleCount,
                        )"
                        :key="domain"
                        class="status-overview__chip"
                      >
                        {{ domain }}
                      </span>
                      <button
                        v-if="domainHasMore"
                        type="button"
                        class="status-overview__more"
                        @click="routeTab = 'web'"
                      >
                        查看更多
                      </button>
                    </div>
                  </div>
                  <div v-if="tcpProxies.length" class="status-overview__group">
                    <div class="status-overview__group-label">TCP 隧道</div>
                    <div ref="tcpChipsRef" class="status-overview__chips">
                      <span
                        v-for="item in tcpProxies.slice(0, tcpVisibleCount)"
                        :key="item.id"
                        class="status-overview__chip status-overview__chip--tcp"
                        :class="{ 'status-overview__chip--off': !item.enabled }"
                      >
                        {{ tcpOverviewLabel(item) }}
                      </span>
                      <button
                        v-if="tcpHasMore"
                        type="button"
                        class="status-overview__more"
                        @click="routeTab = 'tcp'"
                      >
                        查看更多
                      </button>
                    </div>
                  </div>
                </div>
                <div v-else class="status-overview__empty">
                  尚未配置 Web 域名或 TCP 隧道
                </div>
              </template>

              <div v-else class="status-overview__empty">
                启用内网穿透后，这里将显示已配置的 Web 域名和 TCP 隧道
              </div>
            </div>

            <div class="status-footer">
              <div class="status-tags">
                <span
                  class="status-tag"
                  :class="{ 'status-tag--on': frpForm.enabled }"
                >
                  {{ frpForm.enabled ? "✓ 穿透已启用" : "穿透未启用" }}
                </span>
                <span
                  class="status-tag"
                  :class="{ 'status-tag--on': frpForm.tls_enabled }"
                >
                  {{ frpForm.tls_enabled ? "🛡️ TLS 已开启" : "TLS 未开启" }}
                </span>
              </div>
              <n-button
                v-if="frpForm.enabled"
                quaternary
                size="small"
                :loading="restarting"
                @click="restartFrpc"
              >
                <template #icon
                  ><n-icon :component="RefreshOutline"
                /></template>
                重启 FRP 服务
              </n-button>
            </div>
          </div>
        </FonuCard>

        <FonuCard
          class="settings-card"
          subtitle="配置与 VPS 上的 frps 连接信息，保存后将自动重连。"
        >
          <template #title>
            <n-icon :component="SettingsOutline" class="card-title-icon" />
            <span>连接设置</span>
          </template>

          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">启用内网穿透</div>
            </div>
            <n-switch v-model:value="frpForm.enabled" />
          </div>

          <div class="settings-fields">
            <div class="settings-field-row">
              <div class="settings-field settings-field--grow">
                <div class="settings-field__label">
                  服务器地址 <span class="required">*</span>
                </div>
                <n-input
                  v-model:value="frpForm.server_addr"
                  placeholder="127.0.0.1 或 frp.example.com"
                  :disabled="!frpForm.enabled"
                />
                <p class="field-hint field-hint--inline">
                  VPS 的公网 IP 或域名
                </p>
              </div>
              <div class="settings-field settings-field--port">
                <div class="settings-field__label">
                  端口 <span class="required">*</span>
                </div>
                <n-input-number
                  v-model:value="frpForm.server_port"
                  :min="1"
                  :max="65535"
                  class="settings-field__input-full"
                  :disabled="!frpForm.enabled"
                />
              </div>
            </div>
            <div class="settings-field">
              <div class="settings-field__label frp-secret-label">
                <span>Token <span class="required">*</span></span>
                <n-tag
                  v-if="frpForm.has_auth_token"
                  size="small"
                  type="success"
                  :bordered="false"
                >
                  {{ CONFIGURED_SECRET_TAG }}
                </n-tag>
              </div>
              <n-input
                v-model:value="frpForm.auth_token"
                type="password"
                show-password-on="click"
                :disabled="!frpForm.enabled"
                :placeholder="
                  frpForm.has_auth_token
                    ? CONFIGURED_SECRET_PLACEHOLDER
                    : '与 frps 配置一致'
                "
              />
              <p class="field-hint field-hint--inline">
                与 frps 配置的 token 保持一致
              </p>
            </div>
            <div class="form-switch-row">
              <div class="form-switch-row__text">
                <div class="form-switch-row__label">启用 TLS</div>
                <div class="form-switch-row__hint">
                  使用 TLS 连接到 frps，建议开启
                </div>
              </div>
              <n-switch
                v-model:value="frpForm.tls_enabled"
                :disabled="!frpForm.enabled"
              />
            </div>
          </div>

          <div class="save-row">
            <n-button type="primary" :loading="saving" @click="() => save()">
              <template #icon><n-icon :component="LinkOutline" /></template>
              {{ frpForm.enabled ? "保存并连接" : "保存并关闭" }}
            </n-button>
          </div>
        </FonuCard>
      </div>

      <FonuCard flush class="routes-card">
        <n-tabs
          v-model:value="routeTab"
          type="line"
          animated
          class="routes-tabs"
        >
          <n-tab-pane name="web">
            <template #tab>
              <span class="routes-tab">
                <n-icon :component="GlobeOutline" />
                Web 穿透
              </span>
            </template>

            <div class="routes-panel">
              <div class="routes-panel__head">
                <p class="section-desc">
                  将当前已启用的反向代理域名同步到 FRP Web 网关，所有域名将通过
                  Fonu Nginx 进行分流。
                </p>
                <div class="routes-panel__actions">
                  <n-button
                    size="small"
                    :disabled="!frpForm.enabled"
                    @click="showDomains = true"
                  >
                    <template #icon
                      ><n-icon :component="CreateOutline"
                    /></template>
                    管理域名
                  </n-button>
                  <n-button
                    size="small"
                    :loading="syncing"
                    @click="syncDomains"
                  >
                    <template #icon
                      ><n-icon :component="RefreshOutline"
                    /></template>
                    同步域名
                  </n-button>
                </div>
              </div>

              <div class="gateway-grid">
                <div class="gateway-item">
                  <span
                    class="gateway-dot"
                    :class="{ 'gateway-dot--on': httpGatewayOn }"
                  />
                  <div>
                    <div class="gateway-item__title">HTTP 网关</div>
                    <div class="gateway-item__state">
                      {{ httpGatewayOn ? "已启用" : "未启用" }}
                    </div>
                    <div class="gateway-item__hint">
                      转发到 Fonu Nginx (HTTP)
                    </div>
                  </div>
                </div>
                <div class="gateway-item">
                  <span
                    class="gateway-dot"
                    :class="{ 'gateway-dot--on': httpsGatewayOn }"
                  />
                  <div>
                    <div class="gateway-item__title">HTTPS 网关</div>
                    <div class="gateway-item__state">
                      {{ httpsGatewayOn ? "已启用" : "未启用" }}
                    </div>
                    <div class="gateway-item__hint">
                      转发到 Fonu Nginx (HTTPS)
                    </div>
                  </div>
                </div>
                <button
                  type="button"
                  class="gateway-item gateway-item--action"
                  @click="showDomains = true"
                >
                  <n-icon :component="LinkOutline" class="gateway-item__icon" />
                  <div class="gateway-item__body">
                    <div class="gateway-item__title">已同步域名</div>
                    <div class="gateway-item__state gateway-item__state--count">
                      {{ domainCount }}
                    </div>
                    <div class="gateway-item__hint">点击查看或编辑</div>
                  </div>
                  <n-icon
                    :component="ChevronForwardOutline"
                    class="gateway-item__chevron"
                  />
                </button>
              </div>
            </div>
          </n-tab-pane>

          <n-tab-pane name="tcp">
            <template #tab>
              <span class="routes-tab">
                <n-icon :component="SwapHorizontalOutline" />
                TCP 穿透
              </span>
            </template>

            <div class="routes-panel">
              <div class="routes-panel__head">
                <p class="section-desc">
                  将 VPS 上的远程端口直接转发到内网 TCP 服务（如
                  SSH、数据库、RDP）。需在 VPS 防火墙放行对应远程端口。
                </p>
                <n-button
                  size="small"
                  :disabled="!frpForm.enabled"
                  @click="openTcpModal()"
                >
                  添加隧道
                </n-button>
              </div>

              <n-data-table
                v-if="tcpProxies.length"
                :columns="tcpColumns"
                :data="tcpProxies"
                :bordered="false"
                :row-key="(row: FRPTCPProxy) => row.id"
                size="small"
                class="route-table"
              />
              <EmptyState
                v-else
                title="暂无 TCP 隧道"
                description="添加隧道后，外网可通过 VPS 远程端口访问内网 TCP 服务。"
              >
                <template #action>
                  <n-button
                    size="small"
                    :disabled="!frpForm.enabled"
                    @click="openTcpModal()"
                  >
                    添加隧道
                  </n-button>
                </template>
              </EmptyState>
            </div>
          </n-tab-pane>

          <n-tab-pane name="advanced">
            <template #tab>
              <span class="routes-tab">
                <n-icon :component="OptionsOutline" />
                高级设置
              </span>
            </template>

            <div class="routes-panel">
              <p class="section-desc">
                在 VPS 部署 frps 后，将下方配置保存为
                <code>frps.toml</code> 并启动。配置会随上方连接设置实时更新。
              </p>

              <div class="settings-field">
                <div class="settings-field__label">
                  VPS 服务端配置 (frps.toml)
                </div>
                <n-input
                  type="textarea"
                  :rows="12"
                  readonly
                  :value="frpsConfigPreview"
                  class="frps-config-input"
                />
                <p class="field-hint field-hint--inline">
                  网关端口与 Fonu Nginx 一致（HTTP {{ nginxHttpPort }} / HTTPS
                  {{ nginxHttpsPort }}）。VPS 防火墙需放行
                  {{ frpForm.server_port || 7000 }}、{{ nginxHttpPort }}、{{
                    nginxHttpsPort
                  }}<template v-if="tcpRemotePorts.length">
                    ，以及 TCP 远程端口
                    {{ tcpRemotePorts.join("、") }}</template
                  >。
                </p>
                <div class="frps-config-actions">
                  <n-button secondary @click="copyFrpsConfig"
                    >复制配置</n-button
                  >
                  <n-button secondary @click="downloadFrpsConfig">
                    <template #icon
                      ><n-icon :component="DownloadOutline"
                    /></template>
                    下载配置
                  </n-button>
                </div>
              </div>
            </div>
          </n-tab-pane>

          <n-tab-pane name="logs">
            <template #tab>
              <span class="routes-tab">
                <n-icon :component="DocumentTextOutline" />
                运行日志
              </span>
            </template>

            <div class="routes-panel">
              <div class="routes-panel__head">
                <p class="section-desc">
                  frpc 客户端运行日志，用于排查连接与隧道问题。
                </p>
                <div class="routes-panel__actions">
                  <n-button
                    size="small"
                    quaternary
                    :loading="loadingLogs"
                    @click="fetchLogs"
                  >
                    <template #icon
                      ><n-icon :component="RefreshOutline"
                    /></template>
                    刷新
                  </n-button>
                  <n-button
                    size="small"
                    quaternary
                    @click="logFullscreen = true"
                  >
                    <template #icon
                      ><n-icon :component="ExpandOutline"
                    /></template>
                    全屏
                  </n-button>
                </div>
              </div>
              <n-spin :show="loadingLogs">
                <pre ref="logPanelRef" class="log-view log-view--panel">{{
                  logText || "暂无日志"
                }}</pre>
              </n-spin>
            </div>
          </n-tab-pane>
        </n-tabs>
      </FonuCard>
    </div>
  </n-spin>

  <Teleport to="body">
    <div v-if="logFullscreen" class="frp-log-fullscreen">
      <div class="frp-log-fullscreen__head">
        <span class="frp-log-fullscreen__title">frpc 运行日志</span>
        <div class="frp-log-fullscreen__actions">
          <n-button
            size="small"
            quaternary
            :loading="loadingLogs"
            @click="fetchLogs"
          >
            <template #icon><n-icon :component="RefreshOutline" /></template>
            刷新
          </n-button>
          <n-button size="small" quaternary @click="logFullscreen = false">
            <template #icon><n-icon :component="ContractOutline" /></template>
            退出全屏
          </n-button>
        </div>
      </div>
      <pre ref="logFullscreenRef" class="log-view log-view--fullscreen">{{
        logText || "暂无日志"
      }}</pre>
    </div>
  </Teleport>

  <n-modal
    v-model:show="showHelp"
    preset="card"
    title="内网穿透使用说明"
    style="width: min(640px, 92vw)"
  >
    <div class="help-content">
      <p>
        1. 在 VPS 部署 frps，放行控制端口（默认 7000）、Web
        网关端口（HTTP/HTTPS）及 TCP 远程端口。
      </p>
      <p>2. 在本页填写 frps 地址、端口与 Token，保存并连接。</p>
      <p>3. Web 服务：点击「同步域名」，从反代规则导入域名到 FRP Web 网关。</p>
      <p>
        4. TCP 服务：在「TCP 穿透」添加隧道，例如 VPS:6000 →
        127.0.0.1:22（SSH）。
      </p>
      <p>
        5. 在
        <router-link :to="{ name: 'ddns' }">DDNS</router-link> 将域名解析到 VPS
        公网 IP（Web 穿透）。
      </p>
      <p>6. HTTPS 证书推荐继续使用 DNS-01，由 Fonu Nginx 终结 TLS。</p>
    </div>
  </n-modal>

  <n-modal
    v-model:show="showDomains"
    preset="card"
    title="穿透域名"
    style="width: min(520px, 92vw)"
  >
    <p class="field-hint field-hint--inline">
      以下域名将通过 FRP Web 网关暴露，并由 Fonu Nginx
      按域名分流到反代规则中的上游服务。
    </p>
    <n-input
      v-model:value="frpDomainsText"
      type="textarea"
      :rows="8"
      placeholder="每行一个域名，支持 *.example.com"
      :disabled="!frpForm.enabled"
    />
    <p class="field-hint field-hint--inline">
      可手动编辑，或点击「同步域名」从已启用的反向代理规则导入。
    </p>
    <div class="modal-actions">
      <n-button @click="showDomains = false">取消</n-button>
      <n-button
        type="primary"
        :loading="saving"
        :disabled="!frpForm.enabled"
        @click="saveDomains"
      >
        保存
      </n-button>
    </div>
  </n-modal>

  <n-modal
    v-model:show="showTcpModal"
    preset="card"
    :title="tcpModalTitle"
    style="width: min(520px, 92vw)"
  >
    <div class="settings-fields">
      <div class="settings-field">
        <div class="settings-field__label">
          名称 <span class="required">*</span>
        </div>
        <n-input
          v-model:value="tcpForm.name"
          placeholder="如 ssh、mysql"
          :disabled="!frpForm.enabled"
        />
        <p class="field-hint field-hint--inline">
          仅字母、数字、下划线和连字符
        </p>
      </div>
      <div class="settings-field">
        <div class="settings-field__label">本地地址</div>
        <n-input
          v-model:value="tcpForm.local_ip"
          placeholder="127.0.0.1"
          :disabled="!frpForm.enabled"
        />
      </div>
      <div class="settings-field">
        <div class="settings-field__label">
          本地端口 <span class="required">*</span>
        </div>
        <n-input-number
          v-model:value="tcpForm.local_port"
          :min="1"
          :max="65535"
          class="settings-field__input-full"
          :disabled="!frpForm.enabled"
        />
      </div>
      <div class="settings-field">
        <div class="settings-field__label">
          远程端口 <span class="required">*</span>
        </div>
        <n-input-number
          v-model:value="tcpForm.remote_port"
          :min="0"
          :max="65535"
          class="settings-field__input-full"
          :disabled="!frpForm.enabled"
        />
        <p class="field-hint field-hint--inline">
          填 0 表示自动分配未占用的随机端口（1024–65535），或手动指定 1–65535 的固定端口
        </p>
      </div>
      <div class="form-switch-row">
        <div class="form-switch-row__text">
          <div class="form-switch-row__label">启用隧道</div>
        </div>
        <n-switch
          v-model:value="tcpForm.enabled"
          :disabled="!frpForm.enabled"
        />
      </div>
    </div>
    <div class="modal-actions">
      <n-button @click="showTcpModal = false">取消</n-button>
      <n-button
        type="primary"
        :loading="saving"
        :disabled="!frpForm.enabled"
        @click="saveTcpModal"
      >
        保存隧道
      </n-button>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import {
  computed,
  h,
  nextTick,
  onMounted,
  onUnmounted,
  reactive,
  ref,
  watch,
  type Ref,
} from "vue";
import {
  NButton,
  NDataTable,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NSpin,
  NSwitch,
  NTabPane,
  NTabs,
  NTag,
  useMessage,
  type DataTableColumns,
} from "naive-ui";
import {
  CONFIGURED_SECRET_PLACEHOLDER,
  CONFIGURED_SECRET_TAG,
} from "../constants/secretField";
import {
  AlertCircleOutline,
  CheckmarkCircleOutline,
  ChevronForwardOutline,
  CloudOutline,
  ContractOutline,
  CreateOutline,
  DesktopOutline,
  DocumentTextOutline,
  DownloadOutline,
  ExpandOutline,
  GlobeOutline,
  HelpCircleOutline,
  LinkOutline,
  OptionsOutline,
  PauseCircleOutline,
  PersonOutline,
  RefreshOutline,
  ServerOutline,
  SettingsOutline,
  SwapHorizontalOutline,
} from "@vicons/ionicons5";
import { api } from "../api/client";
import type { FRPTCPProxy, FRPStatus } from "../api/types";
import EmptyState from "../components/EmptyState.vue";
import FonuCard from "../components/FonuCard.vue";
import LoadError from "../components/LoadError.vue";
import PageHeader from "../components/PageHeader.vue";
import StatusBadge from "../components/StatusBadge.vue";
import { formatDate } from "../utils/format";
import { copyToClipboard } from "../utils/clipboard";
import { createId } from "../utils/id";

const message = useMessage();
const loading = ref(false);
const saving = ref(false);
const restarting = ref(false);
const syncing = ref(false);
const loadingLogs = ref(false);
const pageError = ref("");
const showHelp = ref(false);
const showDomains = ref(false);
const showTcpModal = ref(false);
const routeTab = ref<"web" | "tcp" | "advanced" | "logs">("web");
const logFullscreen = ref(false);
const tcpEditingId = ref<string | null>(null);

const frpForm = reactive({
  enabled: false,
  server_addr: "",
  server_port: 7000,
  auth_token: "",
  has_auth_token: false,
  tls_enabled: false,
});
const frpDomainsText = ref("");
const tcpProxies = ref<FRPTCPProxy[]>([]);
const tcpForm = reactive({
  name: "",
  local_ip: "127.0.0.1",
  local_port: 22,
  remote_port: 0,
  enabled: true,
});
const frpsConfigSaved = ref("");
const nginxHttpPort = ref(80);
const nginxHttpsPort = ref(443);
const frpStatus = ref<FRPStatus>({
  enabled: false,
  connected: false,
  message: "未启用",
});
const logLines = ref<string[]>([]);
const logPanelRef = ref<HTMLElement | null>(null);
const logFullscreenRef = ref<HTMLElement | null>(null);
const domainChipsRef = ref<HTMLElement | null>(null);
const tcpChipsRef = ref<HTMLElement | null>(null);
const domainVisibleCount = ref(0);
const domainHasMore = ref(false);
const tcpVisibleCount = ref(0);
const tcpHasMore = ref(false);

let overviewChipObserver: ResizeObserver | null = null;

const domainList = computed(() => parseDomains());
const domainCount = computed(() => domainList.value.length);
const hasConfiguredRoutes = computed(
  () => domainList.value.length > 0 || tcpProxies.value.length > 0,
);
const tcpRemotePorts = computed(() =>
  [
    ...new Set(
      tcpProxies.value
        .filter((item) => item.enabled && item.remote_port > 0)
        .map((item) => item.remote_port),
    ),
  ].sort((a, b) => a - b),
);
const tcpModalTitle = computed(() =>
  tcpEditingId.value ? "编辑 TCP 隧道" : "添加 TCP 隧道",
);
const logText = computed(() => logLines.value.join("\n"));
const httpGatewayOn = computed(
  () => frpForm.enabled && (frpStatus.value.http_gateway_enabled ?? false),
);
const httpsGatewayOn = computed(
  () => frpForm.enabled && (frpStatus.value.https_gateway_enabled ?? false),
);
const statusHeroClass = computed(() => {
  if (!frpForm.enabled) return "status-hero--off";
  if (frpStatus.value.connected) return "status-hero--on";
  return "status-hero--warn";
});

const statusHeroIcon = computed(() => {
  if (!frpForm.enabled) return PauseCircleOutline;
  if (frpStatus.value.connected) return CheckmarkCircleOutline;
  return AlertCircleOutline;
});

const displayServerEndpoint = computed(() => {
  if (frpStatus.value.server_endpoint) return frpStatus.value.server_endpoint;
  const addr = frpForm.server_addr.trim();
  if (!addr) return "—";
  const port = frpForm.server_port > 0 ? frpForm.server_port : 7000;
  return `${addr}:${port}`;
});

const frpServerHost = computed(() => {
  const endpoint = displayServerEndpoint.value;
  if (endpoint === "—") return "";
  return endpoint.replace(/:\d+$/, "");
});

const frpsConfigPreview = computed(() =>
  buildFrpsConfig({
    bindPort: frpForm.server_port > 0 ? frpForm.server_port : 7000,
    httpPort: nginxHttpPort.value,
    httpsPort: nginxHttpsPort.value,
    token: resolveFrpsTokenPreview(),
    tcpRemotePorts: tcpRemotePorts.value,
  }),
);

function tcpStatusBadge(row: FRPTCPProxy) {
  if (!row.enabled) return { kind: "disabled" as const, text: "已停用" };
  if (!frpForm.enabled) return { kind: "disabled" as const, text: "未启用" };
  if (frpStatus.value.connected)
    return { kind: "success" as const, text: "运行中" };
  return { kind: "warning" as const, text: "待连接" };
}

function tcpOverviewLabel(row: FRPTCPProxy) {
  const port = row.remote_port > 0 ? `:${row.remote_port}` : "";
  return `${row.name}${port}`;
}

function tcpMappingLabel(row: FRPTCPProxy) {
  const remote = row.remote_port > 0 ? String(row.remote_port) : "随机";
  return `${remote} → ${row.local_ip || "127.0.0.1"}:${row.local_port}`;
}

function tcpExternalAccess(row: FRPTCPProxy) {
  const host = frpServerHost.value || "VPS_IP";
  if (row.remote_port <= 0) return "保存后分配";
  return `${host}:${row.remote_port}`;
}

async function copyText(text: string, successMsg: string) {
  if (await copyToClipboard(text)) {
    message.success(successMsg);
  } else {
    message.error("复制失败");
  }
}

const tcpColumns: DataTableColumns<FRPTCPProxy> = [
  {
    title: "名称",
    key: "name",
    minWidth: 88,
    render: (row) => h("span", { class: "route-name" }, row.name),
  },
  {
    title: "端口映射",
    key: "mapping",
    minWidth: 180,
    render: (row) =>
      h("code", { class: "route-mapping" }, tcpMappingLabel(row)),
  },
  {
    title: "外网访问",
    key: "access",
    minWidth: 160,
    render: (row) =>
      h("code", { class: "route-access" }, tcpExternalAccess(row)),
  },
  {
    title: "状态",
    key: "status",
    width: 96,
    render: (row) => {
      const badge = tcpStatusBadge(row);
      return h(StatusBadge, { kind: badge.kind, text: badge.text });
    },
  },
  {
    title: "启用",
    key: "enabled",
    width: 72,
    render: (row) =>
      h(NSwitch, {
        value: row.enabled,
        disabled: !frpForm.enabled,
        size: "small",
        onUpdateValue: (value: boolean) => toggleTcpEnabled(row.id, value),
      }),
  },
  {
    title: "操作",
    key: "actions",
    width: 168,
    render: (row) =>
      h("div", { class: "route-actions" }, [
        h(
          NButton,
          {
            size: "small",
            quaternary: true,
            onClick: () => copyText(tcpExternalAccess(row), "已复制访问地址"),
          },
          { default: () => "复制" },
        ),
        h(
          NButton,
          {
            size: "small",
            quaternary: true,
            onClick: () => openTcpModal(row),
          },
          { default: () => "编辑" },
        ),
        h(
          NButton,
          {
            size: "small",
            quaternary: true,
            type: "error",
            onClick: () => removeTcpProxy(row.id),
          },
          { default: () => "删除" },
        ),
      ]),
  },
];

const connectionTitle = computed(() => {
  if (!frpForm.enabled) return "未启用";
  if (frpStatus.value.connected) return "已连接";
  return frpStatus.value.message || "未连接";
});

const connectionSubtitle = computed(() => {
  if (!frpForm.enabled) return "内网穿透未开启";
  if (frpStatus.value.connected) {
    return "当前内网穿透服务运行正常，已连接到 VPS，所有规则已生效。";
  }
  return "请检查服务器配置，或前往运行日志 Tab 排查";
});

const connectedAtLabel = computed(() => {
  if (!frpStatus.value.connected_at) return "—";
  return formatDate(frpStatus.value.connected_at);
});

const uptimeLabel = computed(() =>
  formatUptime(frpStatus.value.uptime_seconds ?? 0),
);

function formatUptime(seconds: number) {
  if (!seconds || seconds <= 0) return "—";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days} 天 ${hours} 小时`;
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`;
  return `${minutes} 分钟`;
}

function buildFrpsConfig(opts: {
  bindPort: number;
  httpPort: number;
  httpsPort: number;
  token: string;
  tcpRemotePorts: number[];
}) {
  const token = opts.token.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  const allowPorts =
    opts.tcpRemotePorts.length > 0
      ? `\n# TCP 隧道远程端口，需在 VPS 防火墙放行\nallowPorts = [${opts.tcpRemotePorts.map((port) => `"${port}"`).join(", ")}]\n`
      : "";
  return `bindAddr = "0.0.0.0"
bindPort = ${opts.bindPort}

vhostHTTPPort = ${opts.httpPort}
vhostHTTPSPort = ${opts.httpsPort}
${allowPorts}
[auth]
method = "token"
token = "${token}"
`;
}

function resolveFrpsTokenPreview() {
  const typed = frpForm.auth_token.trim();
  if (typed) return typed;
  if (frpForm.has_auth_token && frpsConfigSaved.value) {
    const match = frpsConfigSaved.value.match(/token\s*=\s*"([^"]*)"/);
    if (match?.[1]) return match[1];
  }
  return "请填写 Token 后保存";
}

function applyFrpsMeta(data: {
  frps_config?: string;
  nginx_http_port?: number;
  nginx_https_port?: number;
}) {
  frpsConfigSaved.value = data.frps_config ?? "";
  if (data.nginx_http_port && data.nginx_http_port > 0)
    nginxHttpPort.value = data.nginx_http_port;
  if (data.nginx_https_port && data.nginx_https_port > 0)
    nginxHttpsPort.value = data.nginx_https_port;
}

function tokenForSave() {
  const token = frpForm.auth_token.trim();
  if (!token) return undefined;
  return token;
}

function applyData(data: Awaited<ReturnType<typeof api.getFRP>>) {
  frpForm.enabled = data.enabled;
  frpForm.server_addr = data.server_addr ?? "";
  frpForm.server_port = data.server_port > 0 ? data.server_port : 7000;
  frpForm.has_auth_token = data.has_auth_token;
  frpForm.auth_token = "";
  frpForm.tls_enabled = data.tls_enabled;
  frpDomainsText.value = (data.custom_domains ?? []).join("\n");
  tcpProxies.value = (data.tcp_proxies ?? []).map((item) => ({
    ...item,
    local_ip: item.local_ip || "127.0.0.1",
  }));
  applyFrpsMeta(data);
  frpStatus.value = data.status;
}

function resetTcpForm() {
  tcpForm.name = "";
  tcpForm.local_ip = "127.0.0.1";
  tcpForm.local_port = 22;
  tcpForm.remote_port = 0;
  tcpForm.enabled = true;
}

function openTcpModal(row?: FRPTCPProxy) {
  if (row) {
    tcpEditingId.value = row.id;
    tcpForm.name = row.name;
    tcpForm.local_ip = row.local_ip || "127.0.0.1";
    tcpForm.local_port = row.local_port;
    tcpForm.remote_port = row.remote_port;
    tcpForm.enabled = row.enabled;
  } else {
    tcpEditingId.value = null;
    resetTcpForm();
  }
  showTcpModal.value = true;
}

function validateTcpForm() {
  const name = tcpForm.name.trim();
  if (!name) {
    message.warning("请填写隧道名称");
    return false;
  }
  if (!/^[a-zA-Z0-9_-]+$/.test(name)) {
    message.warning("名称仅允许字母、数字、下划线和连字符");
    return false;
  }
  if (
    !tcpForm.local_port ||
    tcpForm.local_port < 1 ||
    tcpForm.local_port > 65535
  ) {
    message.warning("本地端口无效");
    return false;
  }
  if (
    tcpForm.remote_port < 0 ||
    (tcpForm.remote_port > 0 &&
      (tcpForm.remote_port < 1 || tcpForm.remote_port > 65535))
  ) {
    message.warning("远程端口请填 0（自动分配）或 1–65535");
    return false;
  }
  const duplicate =
    tcpForm.remote_port > 0 &&
    tcpProxies.value.some(
      (item) =>
        item.id !== tcpEditingId.value &&
        item.remote_port === tcpForm.remote_port,
    );
  if (duplicate) {
    message.warning("远程端口已被其他隧道占用");
    return false;
  }
  const nameDuplicate = tcpProxies.value.some(
    (item) =>
      item.id !== tcpEditingId.value &&
      item.name.toLowerCase() === name.toLowerCase(),
  );
  if (nameDuplicate) {
    message.warning("隧道名称重复");
    return false;
  }
  return true;
}

async function saveTcpModal() {
  if (!validateTcpForm()) return;
  if (!frpForm.enabled) {
    message.warning("请先启用内网穿透");
    return;
  }
  if (!frpForm.server_addr.trim()) {
    message.warning("请先填写并保存连接设置");
    return;
  }
  if (!frpForm.has_auth_token && !frpForm.auth_token.trim()) {
    message.warning("请先填写 Token 并保存连接设置");
    return;
  }
  const payload: FRPTCPProxy = {
    id: tcpEditingId.value ?? createId(),
    name: tcpForm.name.trim(),
    local_ip: tcpForm.local_ip.trim() || "127.0.0.1",
    local_port: tcpForm.local_port,
    remote_port: tcpForm.remote_port,
    enabled: tcpForm.enabled,
  };
  if (tcpEditingId.value) {
    tcpProxies.value = tcpProxies.value.map((item) =>
      item.id === payload.id ? payload : item,
    );
  } else {
    tcpProxies.value = [...tcpProxies.value, payload];
  }
  if (await save({ quiet: true, successText: "TCP 隧道已保存" })) {
    showTcpModal.value = false;
  }
}

async function removeTcpProxy(id: string) {
  tcpProxies.value = tcpProxies.value.filter((item) => item.id !== id);
  if (!canPersistRoutes()) return;
  await save({ quiet: true, successText: "TCP 隧道已删除" });
}

async function toggleTcpEnabled(id: string, enabled: boolean) {
  tcpProxies.value = tcpProxies.value.map((item) =>
    item.id === id ? { ...item, enabled } : item,
  );
  if (!canPersistRoutes()) return;
  await save({ quiet: true });
}

function canPersistRoutes() {
  return (
    frpForm.enabled &&
    frpForm.server_addr.trim() !== "" &&
    (frpForm.has_auth_token || frpForm.auth_token.trim() !== "")
  );
}

async function load() {
  loading.value = true;
  pageError.value = "";
  try {
    applyData(await api.getFRP());
  } catch (error) {
    pageError.value =
      error instanceof Error ? error.message : "请检查 Fonu 服务是否正常运行";
  } finally {
    loading.value = false;
  }
}

function normalizeServerAddr(addr: string) {
  return addr
    .trim()
    .replace(/^https?:\/\//i, "")
    .replace(/[/?#].*$/, "")
    .trim();
}

function parseDomains() {
  return frpDomainsText.value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
}

async function save(opts?: {
  quiet?: boolean;
  successText?: string;
}): Promise<boolean> {
  saving.value = true;
  try {
    frpForm.server_addr = normalizeServerAddr(frpForm.server_addr);
    const domains = parseDomains();
    const tcpList = tcpProxies.value;
    const result = await api.saveFRP({
      enabled: frpForm.enabled,
      server_addr: frpForm.server_addr,
      server_port: frpForm.server_port || 7000,
      auth_token: tokenForSave(),
      tls_enabled: frpForm.tls_enabled,
      custom_domains: domains,
      tcp_proxies: tcpList,
    });
    frpForm.has_auth_token = result.config.has_auth_token;
    frpForm.auth_token = "";
    frpDomainsText.value = (result.config.custom_domains ?? []).join("\n");
    tcpProxies.value = (result.config.tcp_proxies ?? []).map((item) => ({
      ...item,
      local_ip: item.local_ip || "127.0.0.1",
    }));
    applyFrpsMeta(result);
    frpStatus.value = result.status;
    if (!opts?.quiet) {
      message.success(
        opts?.successText ||
          result.message ||
          (frpForm.enabled ? "已保存并连接" : "已关闭内网穿透"),
      );
    } else if (opts.successText) {
      message.success(opts.successText);
    }
    return true;
  } catch (error) {
    message.error(error instanceof Error ? error.message : "保存失败");
    return false;
  } finally {
    saving.value = false;
  }
}

async function saveDomains() {
  if (await save()) {
    showDomains.value = false;
  }
}

async function restartFrpc() {
  if (!frpForm.enabled) {
    message.warning("请先启用内网穿透");
    return;
  }
  restarting.value = true;
  try {
    const result = await api.saveFRP({
      enabled: frpForm.enabled,
      server_addr: frpForm.server_addr,
      server_port: frpForm.server_port || 7000,
      auth_token: tokenForSave(),
      tls_enabled: frpForm.tls_enabled,
      custom_domains: parseDomains(),
      tcp_proxies: tcpProxies.value,
    });
    frpForm.has_auth_token = result.config.has_auth_token;
    frpDomainsText.value = (result.config.custom_domains ?? []).join("\n");
    tcpProxies.value = (result.config.tcp_proxies ?? []).map((item) => ({
      ...item,
      local_ip: item.local_ip || "127.0.0.1",
    }));
    applyFrpsMeta(result);
    frpStatus.value = result.status;
    message.success(result.message || "frpc 已重启");
  } catch (error) {
    message.error(error instanceof Error ? error.message : "重启失败");
  } finally {
    restarting.value = false;
  }
}

async function syncDomains() {
  syncing.value = true;
  try {
    const result = await api.syncFRPDomains();
    frpDomainsText.value = (result.domains ?? []).join("\n");
    applyFrpsMeta(result);
    frpStatus.value = result.status;
    message.success(result.message || `已同步 ${result.domains.length} 个域名`);
  } catch (error) {
    message.error(error instanceof Error ? error.message : "同步失败");
  } finally {
    syncing.value = false;
  }
}

function scrollLogsToBottom() {
  nextTick(() => {
    for (const el of [logPanelRef.value, logFullscreenRef.value]) {
      if (el) el.scrollTop = el.scrollHeight;
    }
  });
}

async function fetchLogs() {
  loadingLogs.value = true;
  try {
    logLines.value = await api.getFRPLogs(300);
  } catch (error) {
    message.error(error instanceof Error ? error.message : "读取日志失败");
  } finally {
    loadingLogs.value = false;
    scrollLogsToBottom();
  }
}

function resolveFrpsConfigText() {
  const typed = frpForm.auth_token.trim();
  if (!typed && frpForm.has_auth_token && frpsConfigSaved.value) {
    return frpsConfigSaved.value;
  }
  return frpsConfigPreview.value;
}

function ensureFrpsConfigReady() {
  const typed = frpForm.auth_token.trim();
  if (!typed && !frpForm.has_auth_token) {
    message.warning("请先填写 Token 并保存");
    return false;
  }
  return true;
}

async function copyFrpsConfig() {
  if (!ensureFrpsConfigReady()) return;
  if (await copyToClipboard(resolveFrpsConfigText())) {
    message.success("已复制 frps 配置");
  } else {
    message.error("复制失败");
  }
}

function downloadFrpsConfig() {
  if (!ensureFrpsConfigReady()) return;
  const blob = new Blob([resolveFrpsConfigText()], {
    type: "text/plain;charset=utf-8",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = "frps.toml";
  link.click();
  URL.revokeObjectURL(url);
  message.success("已下载 frps.toml");
}

watch(routeTab, (tab) => {
  if (tab === "logs") fetchLogs();
});

watch(logFullscreen, (open) => {
  if (open) scrollLogsToBottom();
});

async function fitOverviewChipLine(
  container: HTMLElement | null,
  total: number,
  visible: Ref<number>,
  hasMore: Ref<boolean>,
) {
  if (!container || total === 0) {
    visible.value = 0;
    hasMore.value = false;
    return;
  }
  for (let count = total; count >= 0; count--) {
    visible.value = count;
    hasMore.value = count < total;
    await nextTick();
    if (container.scrollWidth <= container.clientWidth) return;
  }
  visible.value = 0;
  hasMore.value = total > 0;
}

async function remeasureOverviewChips() {
  await fitOverviewChipLine(
    domainChipsRef.value,
    domainList.value.length,
    domainVisibleCount,
    domainHasMore,
  );
  await fitOverviewChipLine(
    tcpChipsRef.value,
    tcpProxies.value.length,
    tcpVisibleCount,
    tcpHasMore,
  );
}

function observeOverviewChipLines() {
  overviewChipObserver?.disconnect();
  if (!overviewChipObserver) {
    overviewChipObserver = new ResizeObserver(() => {
      void remeasureOverviewChips();
    });
  }
  for (const el of [domainChipsRef.value, tcpChipsRef.value]) {
    if (el) overviewChipObserver.observe(el);
  }
}

watch(
  [domainList, tcpProxies, domainChipsRef, tcpChipsRef, loading],
  () => {
    observeOverviewChipLines();
    void remeasureOverviewChips();
  },
  { deep: true },
);

onMounted(() => {
  void load();
});

onUnmounted(() => {
  overviewChipObserver?.disconnect();
  overviewChipObserver = null;
});
</script>

<style scoped>
.frp-page {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-5);
}

.top-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.15fr);
  gap: var(--fonu-space-5);
  align-items: stretch;
}

.top-grid > .status-card,
.top-grid > .settings-card {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.top-grid > .status-card :deep(.fonu-card__body),
.top-grid > .settings-card :deep(.fonu-card__body) {
  display: flex;
  flex-direction: column;
  flex: 1;
}

.card-title-icon {
  font-size: 18px;
  margin-right: 8px;
  color: var(--fonu-text-secondary);
  vertical-align: -3px;
}

.card-title-icon--blue {
  color: #3b82f6;
}

.card-title-icon--green {
  color: #10b981;
}

.route-table {
  margin-top: var(--fonu-space-3);
}

.route-table :deep(.n-data-table-th) {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.route-name {
  font-weight: 600;
  color: var(--fonu-text);
}

.route-access,
.route-mapping {
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.route-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
}

.status-card__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1;
  min-height: 0;
  padding: var(--fonu-space-4) var(--fonu-space-5);
}

.status-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--fonu-border);
  background: var(--fonu-bg);
}

.status-hero--on {
  border-color: color-mix(in srgb, #10b981 35%, var(--fonu-border));
  background: color-mix(in srgb, #10b981 6%, var(--fonu-bg));
}

.status-hero--warn {
  border-color: color-mix(in srgb, #f59e0b 35%, var(--fonu-border));
  background: color-mix(in srgb, #f59e0b 6%, var(--fonu-bg));
}

.status-hero--off {
  border-color: var(--fonu-border);
  background: color-mix(in srgb, #94a3b8 6%, var(--fonu-bg));
}

.status-hero__main {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.status-hero__icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: var(--fonu-surface);
  color: var(--fonu-text-secondary);
  font-size: 22px;
  flex-shrink: 0;
}

.status-hero--on .status-hero__icon-wrap {
  color: #10b981;
}

.status-hero--warn .status-hero__icon-wrap {
  color: #f59e0b;
}

.status-hero__title {
  font-size: 20px;
  font-weight: 700;
  color: var(--fonu-text);
  line-height: 1.2;
}

.status-hero__sub {
  margin-top: 4px;
  font-size: 13px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

.status-hero__aside {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex-shrink: 0;
}

.status-hero__badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  color: var(--fonu-text-muted);
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
}

.status-hero__badge--on {
  color: #047857;
  background: color-mix(in srgb, #10b981 10%, var(--fonu-surface));
  border-color: color-mix(in srgb, #10b981 25%, var(--fonu-border));
}

.status-hero__badge-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #d1d5db;
}

.status-hero__badge--on .status-hero__badge-dot {
  background: #10b981;
  box-shadow: 0 0 0 3px color-mix(in srgb, #10b981 20%, transparent);
}

.status-hero__time {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.routes-card :deep(.fonu-card__body) {
  padding: 0;
}

.routes-tabs :deep(.n-tabs-nav) {
  padding: 0 var(--fonu-space-5);
}

.routes-tabs :deep(.n-tabs-tab) {
  padding: 14px 4px;
  font-weight: 500;
}

.routes-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.routes-panel {
  padding: var(--fonu-space-4) var(--fonu-space-5) var(--fonu-space-5);
}

.routes-panel__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.routes-panel__head .section-desc {
  margin: 0;
  flex: 1;
}

.routes-panel__actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.status-flow-panel {
  padding: 14px 12px;
  border-radius: 10px;
  background: color-mix(in srgb, #3b82f6 4%, var(--fonu-bg));
  border: 1px dashed color-mix(in srgb, #3b82f6 22%, var(--fonu-border));
}

.status-flow {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px;
}

.status-flow__step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-width: 76px;
  padding: 8px 10px;
  border-radius: 10px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.status-flow__step .n-icon {
  font-size: 18px;
  color: var(--fonu-text-muted);
}

.status-flow__step--accent {
  color: #2563eb;
  border-color: color-mix(in srgb, #3b82f6 30%, var(--fonu-border));
}

.status-flow__step--accent .n-icon {
  color: #3b82f6;
}

.status-flow__arrow {
  color: var(--fonu-text-muted);
  font-size: 14px;
  flex-shrink: 0;
}

.status-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.status-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.status-tag {
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
}

.status-tag--on {
  color: #047857;
  background: color-mix(in srgb, #10b981 8%, var(--fonu-bg));
  border-color: color-mix(in srgb, #10b981 25%, var(--fonu-border));
}

.status-metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.status-metrics--row {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.status-metric {
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--fonu-border);
  background: var(--fonu-bg);
}

.status-metric__label {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.status-metric__value {
  margin-top: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
  word-break: break-all;
}

.status-overview {
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 1;
  min-height: 0;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid var(--fonu-border);
  background: var(--fonu-bg);
}

.status-overview__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
}

.status-overview__groups {
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 1;
  min-height: 0;
}

.status-overview__group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.status-overview__group-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--fonu-text-muted);
}

.status-overview__chips {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 6px;
  overflow: hidden;
  min-height: 28px;
}

.status-overview__chip {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  max-width: 100%;
  padding: 3px 10px;
  border-radius: 8px;
  font-size: 12px;
  white-space: nowrap;
  color: #1d4ed8;
  background: color-mix(in srgb, #3b82f6 8%, var(--fonu-bg));
  border: 1px solid color-mix(in srgb, #3b82f6 18%, var(--fonu-border));
}

.status-overview__more {
  flex-shrink: 0;
  padding: 3px 8px;
  border: none;
  background: none;
  font: inherit;
  font-size: 12px;
  color: #2563eb;
  cursor: pointer;
  white-space: nowrap;
}

.status-overview__more:hover {
  color: #1d4ed8;
}

.status-overview__chip--tcp {
  color: #047857;
  background: color-mix(in srgb, #10b981 8%, var(--fonu-bg));
  border-color: color-mix(in srgb, #10b981 18%, var(--fonu-border));
}

.status-overview__chip--off {
  opacity: 0.55;
}

.status-overview__empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 4px;
  padding: 8px 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--fonu-text-muted);
  text-align: center;
}

.section-desc {
  margin: 0 0 var(--fonu-space-4);
  font-size: 13px;
  line-height: 1.6;
  color: var(--fonu-text-muted);
}

.field-hint--inline {
  margin: 8px 0 var(--fonu-space-4);
  font-size: 12px;
  line-height: 1.6;
  color: var(--fonu-text-muted);
}

.settings-fields {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: var(--fonu-space-4);
}

.settings-field-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.settings-field--grow {
  flex: 1;
  min-width: 0;
}

.settings-field--port {
  width: 112px;
  flex-shrink: 0;
}

.settings-field__label {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.frp-secret-label {
  display: flex;
  align-items: center;
  gap: 8px;
}

.settings-field__input-full {
  width: 100%;
}

.required {
  color: #ef4444;
}

.form-switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  padding: 12px 14px;
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  background: var(--fonu-bg);
}

.form-switch-row__label {
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.form-switch-row__hint {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.settings-card .settings-fields {
  flex: 1;
}

.save-row {
  display: flex;
  justify-content: flex-end;
  margin-top: auto;
  padding-top: var(--fonu-space-4);
}

.gateway-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(180px, 1fr));
  gap: var(--fonu-space-4);
  align-items: stretch;
}

.gateway-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-height: 104px;
  padding: 16px;
  border: 1px solid var(--fonu-border);
  border-radius: 12px;
  background: var(--fonu-bg);
  text-align: left;
}

.gateway-item--action {
  align-items: center;
  cursor: pointer;
  transition: border-color 0.15s;
  font: inherit;
  color: inherit;
  width: 100%;
}

.gateway-item__body {
  flex: 1;
  min-width: 0;
}

.gateway-item--action:hover {
  border-color: color-mix(in srgb, #3b82f6 35%, var(--fonu-border));
}

.gateway-dot {
  width: 8px;
  height: 8px;
  margin-top: 6px;
  border-radius: 50%;
  background: #cbd5e1;
  flex-shrink: 0;
}

.gateway-dot--on {
  background: #10b981;
}

.gateway-item__icon {
  font-size: 18px;
  color: #3b82f6;
}

.gateway-item__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.gateway-item__state {
  margin-top: 4px;
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.gateway-item__state--count {
  font-size: 24px;
  font-weight: 700;
  color: var(--fonu-text);
}

.gateway-item__hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.gateway-item__chevron {
  margin-left: auto;
  color: var(--fonu-text-muted);
}

.frps-config-input {
  margin-top: 8px;
}

.frps-config-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
}

.advanced-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.advanced-item {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--fonu-border);
  border-radius: 12px;
  background: var(--fonu-bg);
  cursor: pointer;
  text-align: left;
  font: inherit;
  color: inherit;
  transition: border-color 0.15s;
}

.advanced-item:hover {
  border-color: color-mix(in srgb, #10b981 30%, var(--fonu-border));
}

.advanced-item__icon {
  font-size: 18px;
  color: var(--fonu-text-secondary);
}

.advanced-item__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.advanced-item__hint {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

.advanced-item__chevron {
  margin-left: auto;
  color: var(--fonu-text-muted);
}

.help-content p {
  margin: 0 0 10px;
  line-height: 1.7;
  color: var(--fonu-text-secondary);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: var(--fonu-space-4);
}

.log-view {
  overflow: auto;
  margin: 0;
  padding: 14px;
  border-radius: 10px;
  background: #0f172a;
  color: #e2e8f0;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.log-view--panel {
  min-height: 180px;
  max-height: min(32vh, 320px);
}

.log-view--fullscreen {
  flex: 1;
  min-height: 0;
  max-height: none;
  margin: 0;
  border-radius: 0;
}

.frp-log-fullscreen {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  flex-direction: column;
  background: #0f172a;
}

.frp-log-fullscreen__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  background: #1e293b;
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
  flex-shrink: 0;
}

.frp-log-fullscreen__title {
  font-size: 14px;
  font-weight: 600;
  color: #e2e8f0;
}

.frp-log-fullscreen__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.domain-list {
  margin: 0;
  padding-left: 18px;
  line-height: 1.8;
}

.empty-domains {
  color: var(--fonu-text-muted);
  font-size: 14px;
}

@media (max-width: 1080px) {
  .top-grid,
  .gateway-grid {
    grid-template-columns: 1fr;
  }

  .status-metrics--row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .settings-field-row {
    flex-direction: column;
  }

  .settings-field--port {
    width: 100%;
  }
}
</style>
