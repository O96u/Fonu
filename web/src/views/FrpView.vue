<template>
  <PageHeader
    title="内网穿透"
    description="通过 VPS + FRP，让你即使没有公网 IP，也能随时访问 NAS 上的服务。"
  >
    <template #actions>
      <n-button quaternary @click="routeTab = 'connection'">连接设置</n-button>
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
        <FonuCard class="status-card" title="连接状态" flush>
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
              <div v-if="frpStatus.enabled" class="status-hero__aside">
                <span
                  class="status-hero__badge"
                  :class="{ 'status-hero__badge--on': frpStatus.connected }"
                >
                  <span class="status-hero__badge-dot" />
                  {{ frpStatus.connected ? "已连接" : frpStatus.process_running ? "连接待确认" : "未运行" }}
                </span>
                <div v-if="frpStatus.connected_at" class="status-hero__time">
                  进程启动于 {{ connectedAtLabel }}
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

            <div class="status-metrics">
              <div class="status-metric status-metric--endpoint">
                <div class="status-metric__label">服务器地址</div>
                <code
                  class="status-metric__value"
                  :title="displayServerEndpoint"
                  >{{ displayServerEndpoint }}</code
                >
              </div>
              <div class="status-metric">
                <div class="status-metric__label">frpc 版本</div>
                <div class="status-metric__value">
                  {{ frpStatus.client_version || "—" }}
                </div>
              </div>
              <div class="status-metric">
                <div class="status-metric__label">进程启动时间</div>
                <div class="status-metric__value">{{ connectedAtLabel }}</div>
              </div>
              <div class="status-metric">
                <div class="status-metric__label">运行时长</div>
                <div class="status-metric__value">{{ uptimeLabel }}</div>
              </div>
            </div>

            <div class="status-overview">
              <div class="status-overview__head">
                <span>已配置服务</span>
              </div>

              <template v-if="frpStatus.enabled">
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
                    <div class="status-overview__group-label">端口隧道</div>
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
                  尚未配置 Web 域名或端口隧道
                </div>
              </template>

              <div v-else class="status-overview__empty">
                启用内网穿透后，这里将显示已配置的 Web 域名和端口隧道
              </div>
            </div>

            <div class="status-footer">
              <div class="status-tags">
                <span
                  class="status-tag"
                  :class="{ 'status-tag--on': frpStatus.enabled }"
                >
                  {{ frpStatus.enabled ? "穿透已启用" : "穿透未启用" }}
                </span>
                <span
                  class="status-tag"
                  :class="{ 'status-tag--on': savedConfig?.tls_enabled }"
                >
                  {{ savedConfig?.tls_enabled ? "TLS 已开启" : "TLS 未开启" }}
                </span>
              </div>
              <n-button
                v-if="frpStatus.enabled"
                quaternary
                size="small"
                :loading="restarting"
                :disabled="busy"
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


      </div>

      <FonuCard flush class="routes-card">
        <n-tabs
          v-model:value="routeTab"
          type="line"
          animated
          class="routes-tabs"
        >
          <n-tab-pane name="connection">
            <template #tab>
              <span class="routes-tab"><n-icon :component="SettingsOutline" />连接设置</span>
            </template>
            <div class="routes-panel connection-panel">
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
            <n-switch v-model:value="frpForm.enabled" :disabled="busy" />
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

          <p v-if="connectionDirty" class="field-hint">连接设置有未保存修改；隧道和域名操作不会自动提交这些修改。</p>
          <div class="save-row">
            <n-button type="primary" :loading="saving" @click="() => save()">
              <template #icon><n-icon :component="LinkOutline" /></template>
              {{ frpForm.enabled ? "保存并连接" : "保存并关闭" }}
            </n-button>
          </div>
        </FonuCard>
            </div>
          </n-tab-pane>
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
                    :disabled="busy"
                    @click="openDomainEditor()"
                  >
                    <template #icon
                      ><n-icon :component="CreateOutline"
                    /></template>
                    新增域名
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
                      {{ httpGatewayOn ? "已注册" : frpStatus.http_gateway_state === "failed" ? "注册失败" : frpStatus.http_gateway_enabled ? "待确认" : "未配置" }}
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
                      {{ httpsGatewayOn ? "已注册" : frpStatus.https_gateway_state === "failed" ? "注册失败" : frpStatus.https_gateway_enabled ? "待确认" : "未配置" }}
                    </div>
                    <div class="gateway-item__hint">
                      转发到 Fonu Nginx (HTTPS)
                    </div>
                  </div>
                </div>
                <button
                  type="button"
                  class="gateway-item gateway-item--action"
                  @click="openDomainEditor()"
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
              <FrpDomainWorkspace :domains="domainList" :busy="busy" :http-state="frpStatus.http_gateway_state" :https-state="frpStatus.https_gateway_state" @edit="openDomainEditor" @remove="removeDomain" @copy="domain => copyText(domain, '已复制域名')" />
          </n-tab-pane>

          <n-tab-pane name="tcp">
            <template #tab>
              <span class="routes-tab">
                <n-icon :component="SwapHorizontalOutline" />
                端口穿透
              </span>
            </template>

            <div class="routes-panel">
              <div class="routes-panel__head">
                <p class="section-desc">
                  将 VPS 上的远程端口转发到内网服务。可选 TCP（如
                  SSH、数据库）或 UDP（如 DNS、游戏/VoIP）。需在 VPS
                  防火墙对对应远程端口放行 TCP 与/或 UDP。
                </p>
                <n-button
                  size="small"
                  :disabled="busy"
                  @click="openTcpModal()"
                >
                  添加隧道
                </n-button>
              </div>

              <div class="frp-toolbar">
                <n-input v-model:value="tunnelSearch" clearable placeholder="搜索名称、地址、端口" />
                <n-select v-model:value="protocolFilter" :options="[{label:'全部协议',value:'all'}, {label:'TCP',value:'tcp'}, {label:'UDP',value:'udp'}]" />
                <n-select v-model:value="tunnelLayout" :options="[{label:'自动布局',value:'auto'}, {label:'列表',value:'list'}, {label:'卡片',value:'cards'}]" />
              </div>
              <div :class="['frp-tunnel-workspace', 'layout-' + tunnelLayout]" v-if="filteredTunnels.length">
                <div class="frp-table-mode"><n-data-table :columns="tcpColumns" :data="filteredTunnels" :bordered="false" :row-key="(row: FRPTCPProxy) => row.id" size="small" :scroll-x="800" /></div>
                <div class="frp-card-mode"><FrpTunnelCards :rows="filteredTunnels" :busy="busy" :status="frpStatus" :server-host="frpServerHost" @edit="openTcpModal" @duplicate="duplicateTunnel" @remove="confirmRemoveTcpProxy" @toggle="toggleTcpEnabled" @copy="row => copyText(tcpExternalAccess(row), '已复制访问地址')" /></div>
              </div>
              <EmptyState v-else :title="tcpProxies.length ? '没有匹配的隧道' : '暂无端口隧道'" description="可以在服务停用时预先配置 TCP/UDP 隧道。" />
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
                <code>frps.toml</code> 并启动。预览隐藏 Token；复制和下载将导出已保存的配置（含 Token）。
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
                  {{ savedConfig?.server_port || 7000 }}、{{ nginxHttpPort }}、{{
                    nginxHttpsPort
                  }}<template v-if="tcpRemotePorts.length">
                    ，以及隧道远程端口
                    {{ tcpRemotePorts.join("、") }}（TCP/UDP
                    均需放行）</template
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
              <FrpDiagnostics />
              <div class="frp-toolbar log-toolbar"><n-input v-model:value="logSearch" clearable placeholder="搜索日志" /><n-select v-model:value="logLevel" :options="[{label:'全部级别',value:'all'},{label:'警告',value:'warn'},{label:'错误',value:'error'}]" /><n-checkbox v-model:checked="logAuto">自动刷新</n-checkbox><n-checkbox v-model:checked="followLogs">跟随滚动</n-checkbox><n-button size="small" @click="copyText(logText, '已复制日志')">复制</n-button></div>
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
        网关端口（HTTP/HTTPS）及隧道远程端口（TCP/UDP）。
      </p>
      <p>2. 在本页填写 frps 地址、端口与 Token，保存并连接。</p>
      <p>3. Web 服务：点击「同步域名」，从反代规则导入域名到 FRP Web 网关。</p>
      <p>
        4. 端口穿透：在「端口穿透」添加隧道并选择 TCP 或 UDP，例如 VPS:6000 →
        127.0.0.1:22（SSH），或 UDP 53 → 127.0.0.1:53（DNS）。
      </p>
      <p>
        5. 在
        <router-link :to="{ name: 'ddns' }">DDNS</router-link> 将域名解析到 VPS
        公网 IP（Web 穿透）。
      </p>
      <p>6. HTTPS 证书推荐继续使用 DNS-01，由 Fonu Nginx 终结 TLS。</p>
    </div>
  </n-modal>

  <n-modal v-model:show="showDomains" preset="card" :title="domainEditing ? '编辑穿透域名' : '新增穿透域名'" style="width: min(520px, 92vw)">
    <n-input v-model:value="domainDraft" type="textarea" :rows="5" placeholder="每行一个域名，支持 *.example.com" />
    <p class="field-hint">仅配置 FRP 网关，不会创建反代规则或修改 DNS。</p>
    <div class="modal-actions"><n-button @click="showDomains = false">取消</n-button><n-button type="primary" :loading="saving" @click="saveDomains">保存</n-button></div>
  </n-modal>
  <n-modal v-model:show="showSync" preset="card" title="选择反代域名" style="width: min(560px, 92vw)">
    <p class="field-hint">仅追加选中的域名，不会删除现有手动配置。</p>
    <n-checkbox-group v-model:value="selectedDomains"><div class="frp-sync-list">
      <n-checkbox v-for="domain in syncCandidates" :key="domain" :value="domain" :disabled="domainList.some(d => d.toLowerCase() === domain.toLowerCase())">{{ domain }}<span v-if="domainList.some(d => d.toLowerCase() === domain.toLowerCase())"> · 已添加</span></n-checkbox>
      <span v-if="!syncCandidates.length">暂无可导入的反代域名</span>
    </div></n-checkbox-group>
    <div class="modal-actions"><n-button @click="showSync = false">取消</n-button><n-button type="primary" :loading="saving" :disabled="!selectedDomains.length" @click="applySelectedDomains">导入 {{ selectedDomains.length }} 个域名</n-button></div>
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
          协议 <span class="required">*</span>
        </div>
        <n-select
          v-model:value="tcpForm.protocol"
          :options="tunnelProtocolOptions"
          :disabled="busy"
        />
      </div>
      <div class="settings-field">
        <div class="settings-field__label">
          名称 <span class="required">*</span>
        </div>
        <n-input
          v-model:value="tcpForm.name"
          placeholder="如 ssh、dns"
          :disabled="busy"
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
          :disabled="busy"
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
          :disabled="busy"
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
          :disabled="busy"
        />
        <p class="field-hint field-hint--inline">
          填 0 表示从 Fonu 未使用的端口中随机分配（1024–65535），不保证 VPS 端口空闲；也可手动指定 1–65535
          的固定端口
        </p>
      </div>
      <div class="form-switch-row">
        <div class="form-switch-row__text">
          <div class="form-switch-row__label">启用隧道</div>
        </div>
        <n-switch
          v-model:value="tcpForm.enabled"
          :disabled="busy"
        />
      </div>
    </div>
    <div class="modal-actions">
      <n-button @click="showTcpModal = false">取消</n-button>
      <n-button
        type="primary"
        :loading="saving"
        :disabled="busy"
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
  NCheckbox,
  NCheckboxGroup,
  NDataTable,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpin,
  NSwitch,
  NTabPane,
  NTabs,
  NTag,
  useDialog,
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
import type { FRPConfig, FRPTCPProxy, FRPStatus } from "../api/types";
import EmptyState from "../components/EmptyState.vue";
import FrpDomainWorkspace from "../components/frp/FrpDomainWorkspace.vue";
import FrpTunnelCards from "../components/frp/FrpTunnelCards.vue";
import FrpDiagnostics from "../components/frp/FrpDiagnostics.vue";
import FonuCard from "../components/FonuCard.vue";
import LoadError from "../components/LoadError.vue";
import PageHeader from "../components/PageHeader.vue";
import StatusBadge from "../components/StatusBadge.vue";
import { formatDate } from "../utils/format";
import { copyToClipboard } from "../utils/clipboard";
import { createId } from "../utils/id";

const message = useMessage();
const dialog = useDialog();
const loading = ref(false);
const saving = ref(false);
const restarting = ref(false);
const syncing = ref(false);
const loadingLogs = ref(false);
const pageError = ref("");
const showHelp = ref(false);
const showDomains = ref(false);
const domainDraft = ref("");
const domainEditing = ref<string | null>(null);
const showSync = ref(false);
const syncCandidates = ref<string[]>([]);
const selectedDomains = ref<string[]>([]);
const savedConfig = ref<FRPConfig | null>(null);
const tunnelLayout = ref("auto");
const tunnelSearch = ref("");
const protocolFilter = ref("all");
const logSearch = ref("");
const logLevel = ref("all");
const followLogs = ref(true);
const logAuto = ref(false);
const showTcpModal = ref(false);
const routeTab = ref<"connection" | "web" | "tcp" | "advanced" | "logs">("web");
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
const tunnelProtocolOptions = [
  { label: "TCP", value: "tcp" as const },
  { label: "UDP", value: "udp" as const },
];
const tcpForm = reactive({
  protocol: "tcp" as "tcp" | "udp",
  name: "",
  local_ip: "127.0.0.1",
  local_port: 22,
  remote_port: 0,
  enabled: true,
});
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
const busy = computed(() => saving.value || syncing.value || restarting.value);
const connectionDirty = computed(() => !!savedConfig.value && (frpForm.enabled !== savedConfig.value.enabled || frpForm.server_addr.trim() !== savedConfig.value.server_addr || frpForm.server_port !== savedConfig.value.server_port || frpForm.tls_enabled !== savedConfig.value.tls_enabled || !!frpForm.auth_token.trim()));
const filteredTunnels = computed(() => tcpProxies.value.filter(row => (protocolFilter.value === "all" || tunnelProtocolOf(row) === protocolFilter.value) && [row.name, row.local_ip, row.local_port, row.remote_port].join(" ").toLowerCase().includes(tunnelSearch.value.trim().toLowerCase())));
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
  tcpEditingId.value ? "编辑端口隧道" : "添加端口隧道",
);
const logText = computed(() => logLines.value.filter(line => line.toLowerCase().includes(logSearch.value.trim().toLowerCase()) && (logLevel.value === "all" || (logLevel.value === "error" ? /error|\[E\]|failed/i.test(line) : /warn|\[W\]/i.test(line)))).join("\n"));
const httpGatewayOn = computed(
  () => frpStatus.value.enabled && frpStatus.value.http_gateway_state === "registered",
);
const httpsGatewayOn = computed(
  () => frpStatus.value.enabled && frpStatus.value.https_gateway_state === "registered",
);
const statusHeroClass = computed(() => {
  if (!frpStatus.value.enabled) return "status-hero--off";
  if (frpStatus.value.connected) return "status-hero--on";
  return "status-hero--warn";
});

const statusHeroIcon = computed(() => {
  if (!frpStatus.value.enabled) return PauseCircleOutline;
  if (frpStatus.value.connected) return CheckmarkCircleOutline;
  return AlertCircleOutline;
});

const displayServerEndpoint = computed(() => {
  if (frpStatus.value.server_endpoint) return frpStatus.value.server_endpoint;
  const addr = savedConfig.value?.server_addr?.trim() || "";
  if (!addr) return "—";
  const port = savedConfig.value?.server_port || 7000;
  return `${addr.includes(":") && !addr.startsWith("[") ? "[" + addr + "]" : addr}:${port}`;
});

const frpServerHost = computed(() => {
  const host = savedConfig.value?.server_addr || "";
  return host.includes(":") && !host.startsWith("[") ? "[" + host + "]" : host;
});

const frpsConfigPreview = computed(() =>
  buildFrpsConfig({
    bindPort: savedConfig.value?.server_port || 7000,
    httpPort: nginxHttpPort.value,
    httpsPort: nginxHttpsPort.value,
    token: resolveFrpsTokenPreview(),
    tcpRemotePorts: tcpRemotePorts.value,
  }),
);

function tcpStatusBadge(row: FRPTCPProxy) {
  if (!row.enabled) return { kind: "disabled" as const, text: "已停用" };
  if (!frpStatus.value.enabled) return { kind: "disabled" as const, text: "服务未启用" };
  const state = frpStatus.value.tunnel_states?.[row.id];
  if (frpStatus.value.connected && state === "registered") return { kind: "success" as const, text: "已注册" };
  if (frpStatus.value.connected && state === "failed") return { kind: "error" as const, text: "注册失败" };
  if (frpStatus.value.connected) return { kind: "warning" as const, text: "待确认" };
  return { kind: "warning" as const, text: "待连接" };
}

function tunnelProtocolOf(row: FRPTCPProxy): "tcp" | "udp" {
  return row.protocol === "udp" ? "udp" : "tcp";
}

function tcpOverviewLabel(row: FRPTCPProxy) {
  const port = row.remote_port > 0 ? `:${row.remote_port}` : "";
  const proto = tunnelProtocolOf(row).toUpperCase();
  return `${proto} ${row.name}${port}`;
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
    title: "协议",
    key: "protocol",
    width: 72,
    render: (row) => {
      const proto = tunnelProtocolOf(row);
      return h(
        NTag,
        {
          size: "small",
          bordered: false,
          type: proto === "udp" ? "info" : "default",
        },
        { default: () => proto.toUpperCase() },
      );
    },
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
        disabled: busy.value,
        size: "small",
        onUpdateValue: (value: boolean) => toggleTcpEnabled(row.id, value),
      }),
  },
  {
    title: "操作",
    key: "actions",
    width: 210,
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
            disabled: busy.value,
            onClick: () => openTcpModal(row),
          },
          { default: () => "编辑" },
        ),
        h(NButton, { size: "small", quaternary: true, disabled: busy.value, onClick: () => duplicateTunnel(row) }, { default: () => "克隆" }),
        h(
          NButton,
          {
            size: "small",
            quaternary: true,
            disabled: busy.value,
            type: "error",
            onClick: () => confirmRemoveTcpProxy(row),
          },
          { default: () => "删除" },
        ),
      ]),
  },
];

const connectionTitle = computed(() => {
  if (!frpStatus.value.enabled) return "未启用";
  if (frpStatus.value.connected) return "已连接";
  return frpStatus.value.message || "未连接";
});

const connectionSubtitle = computed(() => {
  if (!frpStatus.value.enabled) return "内网穿透未开启";
  if (frpStatus.value.connected) {
    return "客户端已连接到 VPS；各隧道是否生效仍需单独确认。";
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
      ? `\n# TCP/UDP 隧道远程端口，需在 VPS 防火墙放行（TCP 与 UDP 均需放行对应端口）\nallowPorts = [${opts.tcpRemotePorts.map((port) => `{ single = ${port} }`).join(", ")}]\n`
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
  return savedConfig.value?.has_auth_token
    ? "********"
    : "请填写 Token 后保存";
}

function applyFrpsMeta(data: {
  frps_config?: string;
  nginx_http_port?: number;
  nginx_https_port?: number;
}) {
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
  savedConfig.value = { enabled: data.enabled, server_addr: data.server_addr, server_port: data.server_port, auth_token: "", has_auth_token: data.has_auth_token, tls_enabled: data.tls_enabled, custom_domains: data.custom_domains ?? [], tcp_proxies: data.tcp_proxies ?? [] };
  frpForm.enabled = data.enabled;
  frpForm.server_addr = data.server_addr ?? "";
  frpForm.server_port = data.server_port > 0 ? data.server_port : 7000;
  frpForm.has_auth_token = data.has_auth_token;
  frpForm.auth_token = "";
  frpForm.tls_enabled = data.tls_enabled;
  frpDomainsText.value = (data.custom_domains ?? []).join("\n");
  tcpProxies.value = (data.tcp_proxies ?? []).map((item) => ({
    ...item,
    protocol: item.protocol === "udp" ? "udp" : "tcp",
    local_ip: item.local_ip || "127.0.0.1",
  }));
  applyFrpsMeta(data);
  frpStatus.value = data.status;
}

function resetTcpForm() {
  tcpForm.protocol = "tcp";
  tcpForm.name = "";
  tcpForm.local_ip = "127.0.0.1";
  tcpForm.local_port = 22;
  tcpForm.remote_port = 0;
  tcpForm.enabled = true;
}

function duplicateTunnel(row: FRPTCPProxy) {
  openTcpModal(row);
  tcpEditingId.value = null;
  const base = row.name.slice(0, 24);
  let n = 1;
  while (tcpProxies.value.some(item => item.name.toLowerCase() === (base + "-copy" + n).toLowerCase())) n++;
  tcpForm.name = base + "-copy" + n;
  tcpForm.remote_port = 0;
}

function openTcpModal(row?: FRPTCPProxy) {
  if (row) {
    tcpEditingId.value = row.id;
    tcpForm.protocol = tunnelProtocolOf(row);
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
  if (name.length > 32 || !/^[a-zA-Z0-9_-]+$/.test(name)) {
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
  if (busy.value || !validateTcpForm()) return;
  const payload: FRPTCPProxy = {
    id: tcpEditingId.value ?? createId(),
    name: tcpForm.name.trim(),
    protocol: tcpForm.protocol,
    local_ip: tcpForm.local_ip.trim() || "127.0.0.1",
    local_port: tcpForm.local_port,
    remote_port: tcpForm.remote_port,
    enabled: tcpForm.enabled,
  };
  const previous = tcpProxies.value;
  if (tcpEditingId.value) {
    tcpProxies.value = tcpProxies.value.map((item) =>
      item.id === payload.id ? payload : item,
    );
  } else {
    tcpProxies.value = [...tcpProxies.value, payload];
  }
  if (await save({ quiet: true, routesOnly: true, successText: "隧道已保存" })) {
    showTcpModal.value = false;
  } else {
    tcpProxies.value = previous;
  }
}

function confirmRemoveTcpProxy(row: FRPTCPProxy) {
  dialog.error({
    title: "删除隧道",
    content: `确定删除隧道「${row.name}」吗？对应的端口转发将停止。`,
    positiveText: "删除",
    negativeText: "取消",
    positiveButtonProps: { type: "error" },
    onPositiveClick: async () => {
      await removeTcpProxy(row.id);
    },
  });
}

async function removeTcpProxy(id: string) {
  if (busy.value) return;
  const previous = tcpProxies.value;
  tcpProxies.value = tcpProxies.value.filter((item) => item.id !== id);
  if (!await save({ quiet: true, routesOnly: true, successText: "隧道已删除" })) tcpProxies.value = previous;
}

async function toggleTcpEnabled(id: string, enabled: boolean) {
  if (busy.value) return;
  const previous = tcpProxies.value;
  tcpProxies.value = tcpProxies.value.map((item) =>
    item.id === id ? { ...item, enabled } : item,
  );
  if (!await save({ quiet: true, routesOnly: true })) tcpProxies.value = previous;
}

async function load() {
  loading.value = true;
  pageError.value = "";
  try {
    applyData(await api.getFRP());
    if (!savedConfig.value?.server_addr) routeTab.value = "connection";
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
  routesOnly?: boolean;
  successText?: string;
}): Promise<boolean> {
  if (saving.value || syncing.value || restarting.value) return false;
  saving.value = true;
  try {
    const conn = opts?.routesOnly ? savedConfig.value : frpForm;
    if (!conn) return false;
    const domains = parseDomains();
    const tcpList = tcpProxies.value;
    const result = await api.saveFRP({
      enabled: conn.enabled,
      server_addr: normalizeServerAddr(conn.server_addr),
      server_port: conn.server_port || 7000,
      auth_token: opts?.routesOnly ? undefined : tokenForSave(),
      tls_enabled: conn.tls_enabled,
      custom_domains: domains,
      tcp_proxies: tcpList,
    });
    savedConfig.value = result.config;
    if (!opts?.routesOnly) {
      frpForm.enabled = result.config.enabled;
      frpForm.server_addr = result.config.server_addr;
      frpForm.server_port = result.config.server_port;
      frpForm.tls_enabled = result.config.tls_enabled;
      frpForm.has_auth_token = result.config.has_auth_token;
      frpForm.auth_token = "";
    }
    frpDomainsText.value = (result.config.custom_domains ?? []).join("\n");
    tcpProxies.value = (result.config.tcp_proxies ?? []).map((item) => ({
      ...item,
      protocol: item.protocol === "udp" ? "udp" : "tcp",
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

function openDomainEditor(domain?: string) {
  domainEditing.value = domain ?? null;
  domainDraft.value = domain ?? "";
  showDomains.value = true;
}
async function persistDomains(domains: string[]) {
  if (busy.value) return false;
  const old = frpDomainsText.value;
  frpDomainsText.value = [...new Set(domains.map(d => d.trim().toLowerCase()).filter(Boolean))].join("\n");
  if (await save({ routesOnly: true, successText: "域名已保存" })) return true;
  frpDomainsText.value = old;
  return false;
}
async function saveDomains() {
  const additions = domainDraft.value.split(/\s+/).map(d => d.trim().toLowerCase()).filter(Boolean);
  if (!additions.length || additions.some(d => !/^(?:\*\.)?(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/.test(d))) {
    message.warning("请输入有效域名（支持 *.example.com）"); return;
  }
  if (await persistDomains([...domainList.value.filter(d => d !== domainEditing.value), ...additions])) showDomains.value = false;
}
function removeDomain(domain: string) {
  dialog.warning({ title: "移除穿透域名", content: "确定移除 " + domain + "？不会删除关联的反代规则。", positiveText: "移除", negativeText: "取消", onPositiveClick: async () => { await persistDomains(domainList.value.filter(d => d !== domain)); } });
}
async function restartFrpc() {
  if (busy.value || !frpStatus.value.enabled) return;
  restarting.value = true;
  try {
    const result = await api.restartFRP();
    frpStatus.value = result.status;
    message.success(result.message);
  } catch (error) {
    message.error(error instanceof Error ? error.message : "重启失败");
  } finally { restarting.value = false; }
}
async function syncDomains() {
  if (busy.value) return;
  syncing.value = true;
  try {
    const result = await api.getFRPDomainCandidates();
    syncCandidates.value = result.domains ?? [];
    const existing = new Set(domainList.value.map(d => d.toLowerCase()));
    selectedDomains.value = syncCandidates.value.filter(d => !existing.has(d.toLowerCase()));
    showSync.value = true;
  } catch (error) { message.error(error instanceof Error ? error.message : "读取反代域名失败"); }
  finally { syncing.value = false; }
}
async function applySelectedDomains() {
  if (await persistDomains([...domainList.value, ...selectedDomains.value])) showSync.value = false;
}

function scrollLogsToBottom() {
  nextTick(() => {
    for (const el of [logPanelRef.value, logFullscreenRef.value]) {
      if (el) el.scrollTop = el.scrollHeight;
    }
  });
}

async function fetchLogs() {
  if (loadingLogs.value) return;
  loadingLogs.value = true;
  try {
    logLines.value = await api.getFRPLogs(300);
  } catch (error) {
    message.error(error instanceof Error ? error.message : "读取日志失败");
  } finally {
    loadingLogs.value = false;
    if (followLogs.value) scrollLogsToBottom();
  }
}

async function resolveFrpsConfigText() {
  return (await api.exportFRPSConfig()).config;
}

function ensureFrpsConfigReady() {
  if (connectionDirty.value || !savedConfig.value?.has_auth_token) {
    message.warning("请先保存当前 Token");
    return false;
  }
  return true;
}

async function copyFrpsConfig() {
  if (!ensureFrpsConfigReady()) return;
  try {
    if (await copyToClipboard(await resolveFrpsConfigText())) {
      message.success("已复制 frps 配置，请妥善保管 Token");
    } else {
      message.error("复制失败");
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : "导出失败");
  }
}

async function downloadFrpsConfig() {
  if (!ensureFrpsConfigReady()) return;
  try {
    const blob = new Blob([await resolveFrpsConfigText()], {
      type: "text/plain;charset=utf-8",
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "frps.toml";
    link.click();
    URL.revokeObjectURL(url);
    message.success("已下载 frps.toml，请妥善保管 Token");
  } catch (error) {
    message.error(error instanceof Error ? error.message : "导出失败");
  }
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
  for (let count = Math.min(total, 8); count >= 0; count--) {
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

let statusTimer: ReturnType<typeof setInterval> | undefined;
let statusRequestRunning = false;
let disposed = false;
async function refreshStatus() {
  if (disposed || statusRequestRunning || loading.value || saving.value || restarting.value || syncing.value || document.hidden) return;
  statusRequestRunning = true;
  try {
    const status = await api.getFRPStatus();
    if (!disposed && !saving.value && !restarting.value && !syncing.value) frpStatus.value = status;
  } catch {
    // Keep the last known status on a transient network error.
  } finally {
    statusRequestRunning = false;
  }
}
onMounted(() => {
  void load();
  statusTimer = setInterval(() => {
    void refreshStatus();
    if (logAuto.value && (routeTab.value === "logs" || logFullscreen.value) && !document.hidden) void fetchLogs();
  }, 10000);
});

onUnmounted(() => {
  disposed = true;
  if (statusTimer !== undefined) clearInterval(statusTimer);
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
  gap: 6px 10px;
}

.status-flow__step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-width: 64px;
  max-width: 100%;
  padding: 8px 10px;
  border-radius: 10px;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  font-size: 12px;
  color: var(--fonu-text-secondary);
  white-space: nowrap;
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
  gap: 10px;
}

.status-metric {
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--fonu-border);
  background: var(--fonu-bg);
  min-width: 0;
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
  line-height: 1.35;
  word-break: normal;
  overflow-wrap: break-word;
}

.status-metric--endpoint .status-metric__value {
  display: block;
  font-family: var(--fonu-mono);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
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

.settings-field--grow :deep(.n-input) {
  width: 100%;
}

.settings-field--port {
  width: 148px;
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

@media (max-width: 1280px) {
  .top-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 1080px) {
  .gateway-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .settings-field-row {
    flex-direction: column;
    align-items: stretch;
  }

  .settings-field--port {
    width: 100%;
  }
}
@media (max-width: 760px) {
  .frp-page { min-width: 0; }
}
.top-grid { gap:12px; align-items:start; }
.top-grid > .settings-card { height:auto; }
.status-card :deep(.fonu-card__header) { padding-bottom:12px; }
.status-hero { border-radius:var(--fonu-radius); flex-wrap:wrap; }
.status-hero--on { border-color:color-mix(in srgb,var(--fonu-success) 35%,var(--fonu-border)); background:color-mix(in srgb,var(--fonu-success) 6%,var(--fonu-bg)); }
.status-hero--on .status-hero__icon-wrap,.status-hero__badge--on { color:var(--fonu-success); }
.status-hero__badge--on { background:color-mix(in srgb,var(--fonu-success) 10%,var(--fonu-surface)); }
.status-hero__badge--on .status-hero__badge-dot { background:var(--fonu-success); box-shadow:none; }
.status-flow-panel { background:var(--fonu-bg-muted); border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); }
.status-metrics { gap:1px; background:var(--fonu-border); border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); overflow:hidden; }
.status-metric { border:0; border-radius:0; background:var(--fonu-surface); min-width:0; }
@media(max-width:1000px) { .top-grid { grid-template-columns:minmax(0,1fr); } }
@media(max-width:640px) { .status-hero__aside { align-items:flex-start; } .status-flow { flex-wrap:wrap; } }
.top-grid { grid-template-columns: minmax(0, 1fr); }
.top-grid > .status-card { height: auto; }
.connection-panel .settings-card { max-width: 780px; }
.frp-domain-rows { display: grid; gap: 0; padding-top: 0; }
.frp-domain-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; min-width: 0; padding: 10px 0; border-top: 1px solid var(--fonu-border); }
.frp-domain-name { display: flex; align-items: center; gap: 8px; min-width: 0; }
.frp-domain-name .n-icon { flex-shrink: 0; }
.frp-domain-name span { overflow-wrap: anywhere; min-width: 0; }
@media (max-width: 800px) { .route-table { overflow-x: auto; } }
.frp-toolbar { display:flex; align-items:center; flex-wrap:wrap; gap:8px; margin:12px 0; }
.frp-toolbar > .n-input { flex:1; min-width:160px; }
.frp-toolbar > .n-select { width:124px; }
.frp-card-mode { display:none; }
.layout-cards .frp-card-mode { display:block; }
.layout-cards .frp-table-mode { display:none; }
.frp-sync-list { display:flex; flex-direction:column; gap:10px; max-height:45vh; overflow:auto; overflow-wrap:anywhere; }
@media (max-width: 1000px) { .layout-auto .frp-card-mode { display:block; } .layout-auto .frp-table-mode { display:none; } }
@media (max-width: 540px) { .frp-toolbar > .n-input { flex-basis:100%; } .frp-toolbar > .n-select { flex:1; width:auto; min-width:100px; } }
</style>
