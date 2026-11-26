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
      <FonuCard flush class="workspace-overview dashboard-hero" data-testid="frp-overview">
        <div class="hero-grid">
          <section class="hero-service" :class="statusHeroClass">
            <div class="hero-service-head">
              <div class="overview-icon"><n-icon :component="statusHeroIcon" /></div>
              <div class="hero-service-title"><div class="hero-title-line"><strong>FRP 服务</strong><StatusBadge :kind="frpStatus.connected ? 'success' : frpStatus.enabled ? 'warning' : 'disabled'" :text="connectionTitle" /></div><small>{{ connectionSubtitle }}</small></div>
            </div>
            <dl class="hero-details">
              <div><dt><n-icon :component="CloudOutline" />服务器地址</dt><dd><code :title="displayServerEndpoint">{{ displayServerEndpoint }}</code><n-button v-if="displayServerEndpoint !== '—'" text size="tiny" aria-label="复制服务器地址" @click="copyText(displayServerEndpoint, '已复制服务器地址')"><n-icon :component="CopyOutline" /></n-button></dd></div>
              <div><dt><n-icon :component="DesktopOutline" />客户端版本</dt><dd>{{ frpStatus.client_version || '—' }}</dd></div>
              <div><dt><n-icon :component="TimeOutline" />运行时长</dt><dd :title="connectedAtLabel">{{ uptimeLabel }}</dd></div>
            </dl>
            <div class="hero-actions">
              <n-button v-if="frpStatus.enabled" type="primary" size="small" :loading="restarting" :disabled="busy" data-testid="frp-restart" @click="restartFrpc"><template #icon><n-icon :component="RefreshOutline" /></template>重启服务</n-button>
              <n-button v-if="frpStatus.enabled" size="small" :disabled="busy" @click="confirmStopService"><template #icon><n-icon :component="PauseCircleOutline" /></template>停止服务</n-button>
              <n-button v-else type="primary" size="small" @click="routeTab = 'connection'">配置并启用</n-button>
              <n-button size="small" @click="routeTab = 'connection'"><template #icon><n-icon :component="SettingsOutline" /></template>连接设置</n-button>
            </div>
          </section>
          <div class="hero-right">
            <div class="hero-kpis">
              <button class="hero-kpi" type="button" @click="routeTab = 'web'"><span><n-icon :component="GlobeOutline" />Web 穿透</span><strong>{{ domainCount }}</strong><small>配置域名</small></button>
              <button class="hero-kpi" type="button" @click="routeTab = 'tcp'"><span><n-icon :component="SwapHorizontalOutline" />端口穿透</span><strong>{{ tcpProxies.length }}</strong><small>配置隧道</small></button>
              <button class="hero-kpi" type="button" @click="routeTab = 'tcp'"><span><n-icon :component="LinkOutline" />隧道注册</span><strong>{{ registeredTunnelCount }}<em> / {{ enabledTunnelCount }}</em></strong><small>已注册 / 已启用</small></button>
              <button class="hero-kpi" type="button" @click="routeTab = 'logs'"><span><n-icon :component="TimeOutline" />运行时长</span><strong class="hero-uptime">{{ uptimeLabel }}</strong><small>{{ frpStatus.process_running ? '进程运行中' : '进程未运行' }}</small></button>
            </div>
            <div class="hero-topology" :class="{ 'topology-connected': frpStatus.connected }">
              <div class="topology-node"><n-icon :component="PersonOutline" /><strong>外网访问</strong><small>Internet</small></div>
              <span class="topology-arrow">→</span>
              <div class="topology-node" :class="{ 'node-active': frpStatus.connected }"><n-icon :component="CloudOutline" /><strong>VPS / frps</strong><small>{{ frpStatus.connected ? '已登录' : '待连接' }}</small></div>
              <span class="topology-arrow">→</span>
              <div class="topology-node" :class="{ 'node-active': frpStatus.process_running }"><n-icon :component="DesktopOutline" /><strong>本机 / frpc</strong><small>{{ frpStatus.process_running ? '运行中' : '未运行' }}</small></div>
              <span class="topology-arrow">→</span>
              <div class="topology-node"><n-icon :component="ServerOutline" /><strong>内网服务</strong><small>NAS / 应用</small></div>
            </div>
          </div>
        </div>
      </FonuCard>

      <FonuCard flush class="routes-card">
        <n-tabs
          v-model:value="routeTab"
          type="line"
          animated
          class="routes-tabs"
        >
          <n-tab-pane name="overview">
            <template #tab><span class="routes-tab" data-testid="frp-tab-overview"><n-icon :component="HomeOutline" />概览</span></template>
            <div class="dashboard-content" data-testid="frp-dashboard">
              <section class="dashboard-panel dash-web">
                <header class="dash-heading"><h3><n-icon :component="GlobeOutline" />Web 穿透</h3><n-button text type="primary" @click="routeTab = 'web'">管理域名 <n-icon :component="ChevronForwardOutline" /></n-button></header>
                <div class="dash-counts"><div><strong>{{ domainCount }}</strong><small>已配置域名</small></div><div><strong :class="{ 'good-text': httpGatewayOn || httpsGatewayOn }">{{ Number(httpGatewayOn) + Number(httpsGatewayOn) }}</strong><small>网关已注册</small></div><div><strong>{{ 2 - Number(httpGatewayOn) - Number(httpsGatewayOn) }}</strong><small>网关未注册</small></div></div>
                <div class="dash-preview-list"><div v-if="!domainList.length" class="dash-empty">暂无穿透域名，可同步反代或手动添加。</div>
                <div v-for="domain in domainList.slice(0, 3)" :key="domain" class="dash-service-row">
                  <div class="dash-service-icon"><n-icon :component="GlobeOutline" /></div><div class="dash-service-name"><strong :title="domain">{{ domain }}</strong><small>Web · Fonu Nginx</small></div>
                  <StatusBadge :kind="!frpStatus.enabled ? 'disabled' : httpGatewayOn && httpsGatewayOn ? 'success' : 'unknown'" :text="!frpStatus.enabled ? '未启用' : httpGatewayOn && httpsGatewayOn ? '网关已注册' : '待确认'" />
                  <n-button v-if="!domain.startsWith('*.')" text tag="a" :href="'https://' + domain" target="_blank" rel="noopener noreferrer" aria-label="打开域名"><n-icon :component="OpenOutline" /></n-button>
                  <n-button text :disabled="busy" aria-label="编辑域名" @click="openDomainEditor(domain)"><n-icon :component="CreateOutline" /></n-button>
                </div>
                <div v-for="n in Math.max(0, 3 - domainList.length)" :key="'web-slot-'+n" class="dash-preview-placeholder" aria-hidden="true"></div></div>
                <footer class="dash-footer"><n-button size="small" type="primary" secondary :disabled="busy" @click="openDomainEditor()"><template #icon><n-icon :component="AddOutline" /></template>新增域名</n-button></footer>
              </section>
              <section class="dashboard-panel dash-tcp">
                <header class="dash-heading"><h3><n-icon :component="SwapHorizontalOutline" />端口穿透</h3><n-button text type="primary" @click="routeTab = 'tcp'">管理隧道 <n-icon :component="ChevronForwardOutline" /></n-button></header>
                <div class="dash-counts"><div><strong>{{ tcpProxies.length }}</strong><small>已配置隧道</small></div><div><strong class="good-text">{{ registeredTunnelCount }}</strong><small>已注册</small></div><div><strong :class="{ 'warn-text': failedTunnelCount }">{{ failedTunnelCount }}</strong><small>注册失败</small></div></div>
                <div class="dash-preview-list"><div v-if="!tcpProxies.length" class="dash-empty">暂无端口隧道，可添加 TCP / UDP 服务。</div>
                <div v-for="row in tcpProxies.slice(0, 3)" :key="row.id" class="dash-service-row">
                  <div class="dash-service-icon"><n-icon :component="SwapHorizontalOutline" /></div><div class="dash-service-name"><strong>{{ row.name }}</strong><small>{{ tunnelProtocolOf(row).toUpperCase() }} · {{ tcpMappingLabel(row) }}</small></div>
                  <StatusBadge :kind="tcpStatusBadge(row).kind" :text="tcpStatusBadge(row).text" />
                  <n-switch size="small" :value="row.enabled" :disabled="busy" @update:value="value => toggleTcpEnabled(row.id, value)" />
                  <n-button text :disabled="busy" aria-label="编辑隧道" @click="openTcpModal(row)"><n-icon :component="CreateOutline" /></n-button>
                </div>
                <div v-for="n in Math.max(0, 3 - tcpProxies.length)" :key="'tcp-slot-'+n" class="dash-preview-placeholder" aria-hidden="true"></div></div>
                <footer class="dash-footer"><n-button size="small" type="primary" secondary :disabled="busy" data-testid="frp-add-tunnel" @click="openTcpModal()"><template #icon><n-icon :component="AddOutline" /></template>添加隧道</n-button></footer>
              </section>
              <aside class="dash-side">
                <section class="dashboard-panel"><header class="dash-heading"><h3><n-icon :component="FlashOutline" />快捷操作</h3></header>
                  <div class="dash-shortcuts">
                    <button type="button" @click="routeTab = 'connection'"><n-icon :component="SettingsOutline" />连接设置</button>
                    <button type="button" @click="routeTab = 'diagnostics'"><n-icon :component="PulseOutline" />诊断检测</button>
                    <button type="button" @click="routeTab = 'advanced'"><n-icon :component="OptionsOutline" />配置预览</button>
                    <button type="button" :disabled="busy" @click="syncDomains"><n-icon :component="RefreshOutline" />同步域名</button>
                    <button type="button" @click="downloadFrpsConfig"><n-icon :component="DownloadOutline" />导出配置</button>
                    <button type="button" @click="routeTab = 'logs'"><n-icon :component="DocumentTextOutline" />运行日志</button>
                  </div>
                </section>
                <section class="dashboard-panel"><header class="dash-heading"><h3><n-icon :component="PulseOutline" />运行状态</h3></header>
                  <div class="dash-health"><span>frpc 进程</span><StatusBadge :kind="frpStatus.process_running ? 'success' : 'disabled'" :text="frpStatus.process_running ? '运行中' : '未运行'" /></div>
                  <div class="dash-health"><span>服务器登录</span><StatusBadge :kind="frpStatus.connected ? 'success' : 'unknown'" :text="frpStatus.connected ? '已登录' : '未连接'" /></div>
                  <div class="dash-health"><span>HTTP 网关</span><StatusBadge :kind="httpGatewayOn ? 'success' : frpStatus.http_gateway_state === 'failed' ? 'error' : 'unknown'" :text="gatewayLabel('http')" /></div>
                  <div class="dash-health"><span>HTTPS 网关</span><StatusBadge :kind="httpsGatewayOn ? 'success' : frpStatus.https_gateway_state === 'failed' ? 'error' : 'unknown'" :text="gatewayLabel('https')" /></div>
                  <div class="dash-health"><span>TLS 配置</span><StatusBadge :kind="savedConfig?.tls_enabled ? 'success' : 'disabled'" :text="savedConfig?.tls_enabled ? '已开启' : '未开启'" /></div>
                </section>
              </aside>
              <section class="dashboard-panel dash-logs">
                <header class="dash-heading"><h3><n-icon :component="DocumentTextOutline" />最近日志</h3><n-button text type="primary" @click="routeTab = 'logs'">查看全部 <n-icon :component="ChevronForwardOutline" /></n-button></header>
                <pre class="fonu-log-console dash-log-preview">{{ logLines.slice(-5).join("\n") || (loadingLogs ? '正在加载日志…' : '暂无运行日志') }}</pre>
              </section>
            </div>
          </n-tab-pane>
          <n-tab-pane name="connection">
            <template #tab>
              <span class="routes-tab" data-testid="frp-tab-connection"><n-icon :component="SettingsOutline" />连接设置</span>
            </template>
            <div class="routes-panel connection-panel" data-testid="frp-connection-panel">
              <section class="connection-form">
                <div class="workspace-section-head">
                  <div><h3>服务器连接</h3><p>配置 VPS 上的 frps，保存后自动应用。</p></div>
                  <span v-if="connectionDirty" class="draft-label">未保存</span>
                </div>
                <div class="form-switch-row">
                  <div class="form-switch-row__text">
                    <div class="form-switch-row__label">启用内网穿透</div>
                    <div class="form-switch-row__hint">关闭时保留配置和隧道，可提前编辑。</div>
                  </div>
                  <n-switch v-model:value="frpForm.enabled" :disabled="busy" />
                </div>
                <div class="settings-fields">
                  <div class="settings-field-row">
                    <div class="settings-field settings-field--grow">
                      <div class="settings-field__label">服务器地址 <span class="required">*</span></div>
                      <n-input v-model:value="frpForm.server_addr" placeholder="127.0.0.1 或 frp.example.com" :disabled="busy" />
                      <p class="field-hint field-hint--inline">VPS 的 IP 或域名</p>
                    </div>
                    <div class="settings-field settings-field--port">
                      <div class="settings-field__label">端口 <span class="required">*</span></div>
                      <n-input-number v-model:value="frpForm.server_port" :min="1" :max="65535" class="settings-field__input-full" :disabled="busy" />
                    </div>
                  </div>
                  <div class="settings-field">
                    <div class="settings-field__label frp-secret-label">
                      <span>认证 Token <span class="required">*</span></span>
                      <n-tag v-if="frpForm.has_auth_token" size="small" type="success" :bordered="false">{{ CONFIGURED_SECRET_TAG }}</n-tag>
                    </div>
                    <n-input v-model:value="frpForm.auth_token" type="password" show-password-on="click" :disabled="busy" :placeholder="frpForm.has_auth_token ? CONFIGURED_SECRET_PLACEHOLDER : '与 frps 配置一致'" />
                    <p class="field-hint field-hint--inline">留空保留已保存的 Token。</p>
                  </div>
                  <div class="form-switch-row">
                    <div class="form-switch-row__text">
                      <div class="form-switch-row__label">TLS 加密连接</div>
                      <div class="form-switch-row__hint">建议开启，保护 frpc 与 frps 间的传输。</div>
                    </div>
                    <n-switch v-model:value="frpForm.tls_enabled" :disabled="busy" />
                  </div>
                </div>
                <div class="save-row">
                  <span class="save-hint">{{ connectionDirty ? '修改尚未应用' : '配置与运行状态独立显示' }}</span>
                  <n-button type="primary" :loading="saving" :disabled="busy" @click="() => save()">
                    <template #icon><n-icon :component="LinkOutline" /></template>
                    {{ frpForm.enabled ? "保存并连接" : savedConfig?.enabled ? "保存并关闭" : "保存配置" }}
                  </n-button>
                </div>
              </section>
              <aside class="connection-aside">
                <div class="aside-section">
                  <h3>连接链路</h3>
                  <div class="connection-path">
                    <div><n-icon :component="PersonOutline" /><span>外网用户</span></div>
                    <div><n-icon :component="CloudOutline" /><span>VPS · frps</span></div>
                    <div><n-icon :component="DesktopOutline" /><span>本机 · frpc</span></div>
                    <div><n-icon :component="ServerOutline" /><span>Fonu Nginx / 本地服务</span></div>
                  </div>
                  <p>Web 域名由 Nginx 分流，TCP/UDP 隧道直接转发至指定服务。</p>
                </div>
                <div class="aside-section aside-links">
                  <h3>快捷操作</h3>
                  <button type="button" @click="routeTab = 'advanced'"><n-icon :component="OptionsOutline" /><span>服务端配置<small>查看 / 下载 frps.toml</small></span><n-icon :component="ChevronForwardOutline" /></button>
                  <button type="button" @click="routeTab = 'logs'"><n-icon :component="DocumentTextOutline" /><span>运行诊断<small>检查连接、登录和网关</small></span><n-icon :component="ChevronForwardOutline" /></button>
                </div>
              </aside>
            </div>
          </n-tab-pane>
          <n-tab-pane name="web">
            <template #tab>
              <span class="routes-tab" data-testid="frp-tab-web">
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
                    data-testid="frp-sync-domains"
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
                <div class="gateway-item gateway-item--action">
                  <n-icon :component="LinkOutline" class="gateway-item__icon" />
                  <div class="gateway-item__body">
                    <div class="gateway-item__title">已配置域名</div>
                    <div class="gateway-item__state gateway-item__state--count">
                      {{ domainCount }}
                    </div>
                    <div class="gateway-item__hint">在下方列表管理</div>
                  </div>
                </div>
              </div>
            </div>
              <FrpDomainWorkspace :domains="domainList" :busy="busy" :enabled="frpStatus.enabled" :http-state="frpStatus.http_gateway_state" :https-state="frpStatus.https_gateway_state" @edit="openDomainEditor" @remove="removeDomain" @copy="domain => copyText(domain, '已复制域名')" />
          </n-tab-pane>

          <n-tab-pane name="tcp">
            <template #tab>
              <span class="routes-tab" data-testid="frp-tab-tcp">
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
              <span class="routes-tab" data-testid="frp-tab-advanced">
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
              <span class="routes-tab" data-testid="frp-tab-logs">
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
                    @click="fetchLogs()"
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
              <div class="frp-toolbar log-toolbar"><n-input v-model:value="logSearch" clearable placeholder="搜索日志" /><n-select v-model:value="logLevel" :options="[{label:'全部级别',value:'all'},{label:'警告',value:'warn'},{label:'错误',value:'error'}]" /><n-checkbox v-model:checked="logAuto">自动刷新</n-checkbox><n-checkbox v-model:checked="followLogs">跟随滚动</n-checkbox><n-button size="small" @click="copyText(logText, '已复制日志')">复制</n-button></div>
              <n-spin :show="loadingLogs">
                <pre ref="logPanelRef" class="fonu-log-console log-view log-view--panel">{{
                  logText || "暂无日志"
                }}</pre>
              </n-spin>
            </div>
          </n-tab-pane>
          <n-tab-pane name="diagnostics">
            <template #tab><span class="routes-tab" data-testid="frp-tab-diagnostics"><n-icon :component="PulseOutline" />诊断工具</span></template>
            <div class="routes-panel"><FrpDiagnostics /></div>
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
            @click="fetchLogs()"
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
      <pre ref="logFullscreenRef" class="fonu-log-console log-view log-view--fullscreen">{{
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
    <div class="modal-actions"><n-button @click="showSync = false">取消</n-button><n-button type="primary" :loading="saving" :disabled="!selectedDomains.length" data-testid="frp-import-selected" @click="applySelectedDomains">导入 {{ selectedDomains.length }} 个域名</n-button></div>
  </n-modal>

  <n-modal v-model:show="showTcpModal" :mask-closable="!busy" :close-on-esc="!busy" transform-origin="center">
    <div class="tunnel-editor" role="dialog" aria-modal="true" :aria-label="tcpModalTitle" data-testid="frp-tunnel-editor">
      <header class="tunnel-editor-head">
        <div class="tunnel-editor-heading"><div class="tunnel-editor-icon"><n-icon :component="SwapHorizontalOutline" /></div><div><h3>{{ tcpModalTitle }}</h3><p>将 VPS 端口转发到本地 TCP / UDP 服务</p></div></div>
        <n-button text :disabled="busy" aria-label="关闭弹窗" @click="showTcpModal = false"><n-icon :component="CloseOutline" size="22" /></n-button>
      </header>
      <div class="tunnel-editor-body">
        <div v-if="!tcpEditingId" class="tunnel-presets"><span>常用服务</span><n-button v-for="preset in tunnelPresets" :key="preset.name" size="tiny" :disabled="busy" @click="applyTunnelPreset(preset)">{{ preset.label }}</n-button></div>
        <div class="tunnel-form-grid">
          <div class="settings-field"><div class="settings-field__label">协议 <span class="required">*</span></div><n-select v-model:value="tcpForm.protocol" :options="tunnelProtocolOptions" :disabled="busy" /></div>
          <div class="settings-field"><div class="settings-field__label">名称 <span class="required">*</span></div><n-input v-model:value="tcpForm.name" placeholder="如 ssh、dns" :maxlength="32" :disabled="busy" /><small class="tunnel-hint">字母、数字、下划线或连字符</small></div>
          <div class="settings-field"><div class="settings-field__label">本地地址</div><n-input v-model:value="tcpForm.local_ip" placeholder="127.0.0.1" :disabled="busy" /></div>
          <div class="settings-field"><div class="settings-field__label">本地端口 <span class="required">*</span></div><n-input-number v-model:value="tcpForm.local_port" :min="1" :max="65535" class="settings-field__input-full" :disabled="busy" /></div>
          <div class="settings-field"><div class="settings-field__label">远程端口 <span class="required">*</span></div><n-input-number v-model:value="tcpForm.remote_port" :min="0" :max="65535" class="settings-field__input-full" :disabled="busy" /><small class="tunnel-hint">0 自动分配，或指定 1–65535</small></div>
          <div class="settings-field"><div class="settings-field__label">隧道状态</div><div class="tunnel-enable"><span>{{ tcpForm.enabled ? '已启用' : '已停用' }}</span><n-switch v-model:value="tcpForm.enabled" :disabled="busy" /></div></div>
        </div>
        <div class="tunnel-port-preview"><div><small>VPS 远程端口</small><strong>{{ tcpForm.remote_port || '自动分配' }}</strong></div><n-icon :component="SwapHorizontalOutline" /><div><small>本地服务</small><strong>{{ tcpForm.local_ip || '127.0.0.1' }}:{{ tcpForm.local_port || '—' }}</strong></div></div>
        <p class="tunnel-hint tunnel-note">自动端口从 Fonu 未使用的 1024–65535 中分配，不保证 VPS 端口空闲；请在 VPS 防火墙放行相应协议和端口。</p>
      </div>
      <footer class="tunnel-editor-footer">
        <span class="tunnel-hint">{{ tcpEditingId ? '修改后重新应用隧道' : '保存后应用隧道配置' }}</span>
        <div><n-button :disabled="busy" @click="showTcpModal = false">取消</n-button><n-button v-if="!tcpEditingId" :disabled="busy" @click="saveTcpModal(true)">保存并继续</n-button><n-button type="primary" :loading="saving" :disabled="busy" data-testid="frp-save-tunnel" @click="saveTcpModal()">保存隧道</n-button></div>
      </footer>
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
  AddOutline,
  CopyOutline,
  CloseOutline,
  FlashOutline,
  HomeOutline,
  OpenOutline,
  PulseOutline,
  TimeOutline,
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
const routeTab = ref<"overview" | "connection" | "web" | "tcp" | "advanced" | "logs" | "diagnostics">("overview");
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
const domainList = computed(() => parseDomains());
const busy = computed(() => saving.value || syncing.value || restarting.value);
const connectionDirty = computed(() => !!savedConfig.value && (frpForm.enabled !== savedConfig.value.enabled || frpForm.server_addr.trim() !== savedConfig.value.server_addr || frpForm.server_port !== savedConfig.value.server_port || frpForm.tls_enabled !== savedConfig.value.tls_enabled || !!frpForm.auth_token.trim()));
const filteredTunnels = computed(() => tcpProxies.value.filter(row => (protocolFilter.value === "all" || tunnelProtocolOf(row) === protocolFilter.value) && [row.name, row.local_ip, row.local_port, row.remote_port].join(" ").toLowerCase().includes(tunnelSearch.value.trim().toLowerCase())));
const domainCount = computed(() => domainList.value.length);
const enabledTunnelCount = computed(() => tcpProxies.value.filter(item => item.enabled).length);
const registeredTunnelCount = computed(() => frpStatus.value.enabled && frpStatus.value.connected ? tcpProxies.value.filter(item => item.enabled && frpStatus.value.tunnel_states?.[item.id] === "registered").length : 0);
const failedTunnelCount = computed(() => frpStatus.value.enabled && frpStatus.value.connected ? tcpProxies.value.filter(item => item.enabled && frpStatus.value.tunnel_states?.[item.id] === "failed").length : 0);
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
function gatewayLabel(protocol: "http" | "https") {
  if (!frpStatus.value.enabled) return "未启用";
  const state = protocol === "http" ? frpStatus.value.http_gateway_state : frpStatus.value.https_gateway_state;
  const configured = protocol === "http" ? frpStatus.value.http_gateway_enabled : frpStatus.value.https_gateway_enabled;
  return state === "registered" ? "已注册" : state === "failed" ? "注册失败" : configured ? "待确认" : "未配置";
}
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
  if (!frpStatus.value.enabled) return "服务已关闭，仍可编辑域名和隧道。";
  if (frpStatus.value.connected) {
    return "客户端已登录 VPS，隧道状态独立确认。";
  }
  return "检查服务器配置，或前往运行日志排查。";
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

const tunnelPresets = [
  { label: "SSH", name: "ssh", protocol: "tcp" as const, port: 22 },
  { label: "RDP", name: "rdp", protocol: "tcp" as const, port: 3389 },
  { label: "MySQL", name: "mysql", protocol: "tcp" as const, port: 3306 },
  { label: "DNS", name: "dns", protocol: "udp" as const, port: 53 },
];
function applyTunnelPreset(preset: typeof tunnelPresets[number]) {
  if (busy.value || tcpEditingId.value) return;
  tcpForm.protocol = preset.protocol;
  tcpForm.local_port = preset.port;
  if (!tcpForm.name.trim() || tunnelPresets.some(item => item.name === tcpForm.name)) {
    let name = preset.name, i = 2;
    while (tcpProxies.value.some(item => item.name.toLowerCase() === name)) name = preset.name + "-" + i++;
    tcpForm.name = name;
  }
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
    tcpForm.remote_port == null ||
    !Number.isInteger(tcpForm.remote_port) ||
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

async function saveTcpModal(continueAdding = false) {
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
    if (continueAdding && !tcpEditingId.value) resetTcpForm();
    else showTcpModal.value = false;
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
    if (routeTab.value === "overview") void fetchLogs(true);
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
  serviceEnabled?: boolean;
}): Promise<boolean> {
  if (saving.value || syncing.value || restarting.value) return false;
  saving.value = true;
  try {
    const conn = opts?.routesOnly ? savedConfig.value : frpForm;
    if (!conn) return false;
    const domains = parseDomains();
    const tcpList = tcpProxies.value;
    const result = await api.saveFRP({
      enabled: opts?.serviceEnabled ?? conn.enabled,
      server_addr: normalizeServerAddr(conn.server_addr),
      server_port: conn.server_port || 7000,
      auth_token: opts?.routesOnly ? undefined : tokenForSave(),
      tls_enabled: conn.tls_enabled,
      custom_domains: domains,
      tcp_proxies: tcpList,
    });
    savedConfig.value = result.config;
    if (opts?.serviceEnabled !== undefined) frpForm.enabled = result.config.enabled;
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
function confirmStopService() {
  if (busy.value || !savedConfig.value?.enabled) return;
  dialog.warning({
    title: "停止 FRP 服务",
    content: "所有穿透服务将断开，已保存的连接、域名和隧道配置将保留。未保存的其他连接修改不会提交。",
    positiveText: "停止服务",
    negativeText: "取消",
    onPositiveClick: async () => { await save({ routesOnly: true, serviceEnabled: false, successText: "已停止 FRP 服务" }); },
  });
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

let logsController: AbortController | undefined;
async function fetchLogs(quiet = false) {
  if (disposed || loadingLogs.value) return;
  const controller = new AbortController();
  logsController = controller;
  loadingLogs.value = true;
  try {
    const lines = await api.getFRPLogs(300, controller.signal);
    if (!disposed && !controller.signal.aborted) logLines.value = lines;
  } catch (error) {
    if (!disposed && !controller.signal.aborted && !quiet) message.error(error instanceof Error ? error.message : "读取日志失败");
  } finally {
    logsController = undefined;
    loadingLogs.value = false;
    if (!disposed && followLogs.value) scrollLogsToBottom();
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
  if (tab === "logs" || tab === "overview") void fetchLogs(tab === "overview");
});

watch(logFullscreen, (open) => {
  if (open) scrollLogsToBottom();
});

let statusTimer: ReturnType<typeof setInterval> | undefined;
let statusRequestRunning = false;
let statusController: AbortController | undefined;
let disposed = false;
async function refreshStatus() {
  if (disposed || statusRequestRunning || loading.value || saving.value || restarting.value || syncing.value || document.hidden) return;
  statusRequestRunning = true;
  const controller = new AbortController();
  statusController = controller;
  try {
    const status = await api.getFRPStatus(controller.signal);
    if (!disposed && !controller.signal.aborted && !saving.value && !restarting.value && !syncing.value) frpStatus.value = status;
  } catch {
    // Keep the last known status on a transient network error.
  } finally {
    statusController = undefined;
    statusRequestRunning = false;
  }
}
onMounted(() => {
  void load();
  statusTimer = setInterval(() => {
    void refreshStatus();
    if (!document.hidden && routeTab.value === "overview") void fetchLogs(true);
    else if (logAuto.value && (routeTab.value === "logs" || logFullscreen.value) && !document.hidden) void fetchLogs();
  }, 10000);
});

onUnmounted(() => {
  disposed = true;
  statusController?.abort();
  logsController?.abort();
  if (statusTimer !== undefined) clearInterval(statusTimer);
});
</script>

<style scoped>
.frp-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  container-type: inline-size;
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


.save-row {
  display: flex;
  justify-content: flex-end;
  margin-top: auto;
  padding-top: var(--fonu-space-4);
}

.gateway-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--fonu-space-4);
  align-items: stretch;
}

.gateway-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-height: 0;
  padding: 12px;
  border: 1px solid var(--fonu-border);
  border-radius: 12px;
  background: var(--fonu-bg);
  text-align: left;
}

.gateway-item--action {
  align-items: center;
  font: inherit;
  color: inherit;
  width: 100%;
}

.gateway-item__body {
  flex: 1;
  min-width: 0;
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
  background: var(--fonu-bg-muted);
  color: var(--fonu-text);
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
  background: var(--fonu-bg-muted);
}

.frp-log-fullscreen__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  background: var(--fonu-surface);
  border-bottom: 1px solid var(--fonu-border);
  flex-shrink: 0;
}

.frp-log-fullscreen__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
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

.frp-toolbar { display:flex; align-items:center; flex-wrap:wrap; gap:8px; margin:12px 0; }
.frp-toolbar > .n-input { flex:1; min-width:160px; }
.frp-toolbar > .n-select { width:124px; }
.frp-card-mode { display:none; }
.layout-cards .frp-card-mode { display:block; }
.layout-cards .frp-table-mode { display:none; }
.frp-sync-list { display:flex; flex-direction:column; gap:10px; max-height:45vh; overflow:auto; overflow-wrap:anywhere; }
@media (max-width: 1000px) { .layout-auto .frp-card-mode { display:block; } .layout-auto .frp-table-mode { display:none; } }
@media (max-width: 540px) { .frp-toolbar > .n-input { flex-basis:100%; } .frp-toolbar > .n-select { flex:1; width:auto; min-width:100px; } }

/* Compact FRP workspace: size by actual FPK content area, not monitor width. */
.workspace-overview :deep(.fonu-card__body) { padding: 0; }
.overview-head { display:flex; align-items:center; justify-content:space-between; gap:12px; padding:16px; flex-wrap:wrap; }
.overview-status { display:flex; align-items:center; gap:12px; flex:1; min-width: min(100%, 240px); }
.overview-icon { width:44px; height:44px; flex-shrink:0; display:grid; place-items:center; border-radius:var(--fonu-radius); background:var(--fonu-bg-muted); color:var(--fonu-text-secondary); font-size:24px; }
.status-hero--on .overview-icon { color:var(--fonu-success); background:color-mix(in srgb,var(--fonu-success) 10%,var(--fonu-surface)); }
.status-hero--warn .overview-icon { color:#b7791f; background:color-mix(in srgb,#d97706 10%,var(--fonu-surface)); }
.overview-copy { min-width:0; }
.overview-kicker { color:var(--fonu-text-secondary); font-size:11px; line-height:1.4; }
.overview-title { font-size:20px; font-weight:700; line-height:1.35; color:var(--fonu-text); overflow-wrap:anywhere; }
.overview-sub { color:var(--fonu-text-secondary); font-size:12px; overflow-wrap:anywhere; }
.overview-actions { display:flex; align-items:center; justify-content:flex-end; flex-wrap:wrap; gap:6px; }
.overview-tag { font-size:11px; padding:3px 7px; color:var(--fonu-text-secondary); border:1px solid var(--fonu-border); border-radius:6px; white-space:nowrap; }
.overview-tag--on { color:var(--fonu-success); border-color:color-mix(in srgb,var(--fonu-success) 25%,var(--fonu-border)); }
.overview-grid { display:grid; grid-template-columns:1.5fr repeat(3,minmax(0,1fr)); border-top:1px solid var(--fonu-border); }
.overview-cell { min-width:0; padding:12px 16px; display:grid; align-content:start; gap:4px; border:0; border-right:1px solid var(--fonu-border); color:var(--fonu-text); background:transparent; font:inherit; text-align:left; }
.overview-cell:last-child { border-right:0; }
.overview-link { cursor:pointer; }
.overview-link:hover { background:var(--fonu-bg-muted); }
.overview-link:focus-visible, .aside-links button:focus-visible { outline:2px solid var(--fonu-success); outline-offset:-2px; }
.overview-label { display:flex; align-items:center; gap:6px; font-size:12px; color:var(--fonu-text-secondary); }
.overview-label .n-icon { flex-shrink:0; font-size:15px; }
.overview-chevron { margin-left:auto; }
.overview-value { display:block; font-size:15px; line-height:1.5; font-weight:650; overflow-wrap:anywhere; }
.endpoint-value { font-family:var(--fonu-mono); font-size:14px; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.overview-number { font-size:23px; line-height:1.25; font-variant-numeric:tabular-nums; }
.overview-cell small { color:var(--fonu-text-secondary); font-size:11px; overflow-wrap:anywhere; }
.routes-panel { min-width:0; padding:16px; }
.routes-tabs :deep(.n-tabs-nav) { padding-inline:16px; }
.routes-tabs :deep(.n-tabs-tab) { padding-block:12px; }
.routes-panel__head { align-items:center; gap:12px; margin-bottom:12px; flex-wrap:wrap; }
.routes-panel__head .section-desc { min-width:min(100%,220px); }
.routes-panel__actions { flex-wrap:wrap; }
.connection-panel { display:grid; grid-template-columns:minmax(0,1.7fr) minmax(230px,1fr); gap:20px; align-items:start; }
.connection-form { min-width:0; display:grid; gap:12px; }
.workspace-section-head { display:flex; align-items:center; justify-content:space-between; gap:8px; }
.workspace-section-head h3,.aside-section h3 { margin:0; font-size:15px; font-weight:650; color:var(--fonu-text); }
.workspace-section-head p,.aside-section p { margin:4px 0 0; font-size:12px; color:var(--fonu-text-secondary); line-height:1.55; }
.draft-label { flex-shrink:0; font-size:12px; color:#b7791f; }
.connection-form .settings-fields { margin-top:0; gap:12px; }
.connection-form .field-hint--inline { margin:4px 0 0; }
.connection-form .settings-field__label { margin-bottom:6px; font-size:13px; }
.form-switch-row { padding:10px 12px; gap:12px; }
.form-switch-row :deep(.n-switch) { flex-shrink:0; }
.save-row { padding-top:12px; border-top:1px solid var(--fonu-border); align-items:center; justify-content:space-between; gap:8px; }
.save-hint { color:var(--fonu-text-secondary); font-size:12px; }
.connection-aside { display:grid; gap:12px; min-width:0; border-left:1px solid var(--fonu-border); padding-left:20px; }
.aside-section { display:grid; gap:10px; min-width:0; }
.connection-path { display:grid; gap:0; }
.connection-path > div { display:flex; align-items:center; gap:8px; min-height:32px; color:var(--fonu-text-secondary); font-size:12px; position:relative; }
.connection-path > div:not(:last-child)::after { content:""; position:absolute; left:13px; top:29px; width:1px; height:6px; background:var(--fonu-border); }
.connection-path .n-icon { display:grid; place-items:center; width:28px; height:28px; border:1px solid var(--fonu-border); border-radius:7px; flex-shrink:0; background:var(--fonu-bg-muted); font-size:16px; }
.aside-links { border-top:1px solid var(--fonu-border); padding-top:12px; }
.aside-links button { display:flex; align-items:center; gap:9px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); padding:9px; text-align:left; background:var(--fonu-surface); color:var(--fonu-text); font:inherit; cursor:pointer; }
.aside-links button:hover { background:var(--fonu-bg-muted); }
.aside-links button > span { flex:1; min-width:0; font-size:13px; }
.aside-links small { display:block; font-size:11px; color:var(--fonu-text-secondary); }
.aside-links .n-icon { flex-shrink:0; color:var(--fonu-text-secondary); }
.gateway-grid { gap:10px; }
.gateway-item { gap:8px; }
.gateway-item__state--count { font-size:20px; }
.frp-tunnel-workspace { min-width:0; }
@container (max-width: 760px) {
  .overview-grid { grid-template-columns:repeat(2,minmax(0,1fr)); }
  .overview-cell:nth-child(2) { border-right:0; }
  .overview-cell:nth-child(-n+2) { border-bottom:1px solid var(--fonu-border); }
  .connection-panel { grid-template-columns:minmax(0,1fr); gap:16px; }
  .connection-aside { border-left:0; border-top:1px solid var(--fonu-border); padding:14px 0 0; grid-template-columns:repeat(2,minmax(0,1fr)); }
  .aside-links { border-top:0; padding-top:0; }
  .gateway-grid { grid-template-columns:repeat(2,minmax(0,1fr)); }
  .layout-auto .frp-card-mode { display:block; }
  .layout-auto .frp-table-mode { display:none; }
}
@container (max-width: 450px) {
  .overview-head { padding:12px; }
  .overview-cell { padding:10px 12px; }
  .overview-actions { justify-content:flex-start; }
  .routes-panel { padding:12px; }
  .routes-tabs :deep(.n-tabs-nav) { padding-inline:12px; }
  .settings-field-row { flex-direction:column; align-items:stretch; }
  .settings-field--port { width:100%; }
  .connection-aside { grid-template-columns:minmax(0,1fr); }
  .gateway-grid { grid-template-columns:minmax(0,1fr); }
  .save-row { flex-wrap:wrap; }
}

.dashboard-hero :deep(.fonu-card__body) { padding:10px; }
.hero-grid { display:grid; grid-template-columns:minmax(0,1fr) minmax(0,2.1fr); gap:12px; }
.hero-service { min-width:0; display:grid; gap:10px; align-content:start; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); padding:12px; background:var(--fonu-bg-muted); }
.hero-service.status-hero--on { background:linear-gradient(145deg,color-mix(in srgb,var(--fonu-success) 9%,var(--fonu-surface)),var(--fonu-surface) 65%); }
.hero-service-head { display:flex; gap:10px; align-items:center; }
.hero-service-title { min-width:0; flex:1; }
.hero-title-line { display:flex; align-items:center; flex-wrap:wrap; gap:8px; }
.hero-title-line strong { font-size:18px; color:var(--fonu-text); }
.hero-service-title small { display:block; font-size:12px; line-height:1.5; color:var(--fonu-text-secondary); }
.hero-details { margin:0; display:grid; gap:0; min-width:0; }
.hero-details > div { display:grid; grid-template-columns:minmax(80px,0.8fr) minmax(0,1.4fr); align-items:center; gap:8px; min-height:30px; border-bottom:1px solid var(--fonu-border); }
.hero-details dt { display:flex; align-items:center; gap:5px; font-size:12px; color:var(--fonu-text-secondary); }
.hero-details dd { margin:0; min-width:0; display:flex; align-items:center; gap:6px; font-size:12px; font-weight:600; color:var(--fonu-text); }
.hero-details dd code { min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:12px; }
.hero-actions { display:flex; flex-wrap:wrap; gap:6px; }
.hero-actions > .n-button { flex:1 1 auto; }
.hero-right { display:grid; min-width:0; gap:10px; grid-template-rows:auto 1fr; }
.hero-kpis { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:8px; }
.hero-kpi { display:grid; gap:3px; min-width:0; padding:10px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); color:var(--fonu-text); background:var(--fonu-surface); text-align:left; font:inherit; cursor:pointer; }
.hero-kpi:hover { background:var(--fonu-bg-muted); }
.hero-kpi span { display:flex; align-items:center; gap:5px; font-size:12px; color:var(--fonu-text-secondary); }
.hero-kpi .n-icon { font-size:17px; color:var(--fonu-success); flex-shrink:0; }
.hero-kpi strong { font-size:23px; line-height:1.25; overflow-wrap:anywhere; font-variant-numeric:tabular-nums; }
.hero-kpi strong em { font-size:12px; font-style:normal; color:var(--fonu-text-secondary); }
.hero-kpi .hero-uptime { font-size:16px; }
.hero-kpi small { color:var(--fonu-text-secondary); font-size:11px; }
.hero-topology { min-width:0; display:grid; grid-template-columns:minmax(0,1fr) auto minmax(0,1fr) auto minmax(0,1fr) auto minmax(0,1fr); align-items:center; gap:7px; padding:12px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); background:var(--fonu-bg-muted); }
.topology-node { min-width:0; display:grid; justify-items:center; text-align:center; gap:4px; padding:9px 4px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); background:var(--fonu-surface); }
.topology-node .n-icon { color:var(--fonu-text-secondary); font-size:22px; }
.topology-node strong { font-size:12px; overflow-wrap:anywhere; }
.topology-node small { color:var(--fonu-text-secondary); font-size:11px; }
.node-active { border-color:color-mix(in srgb,var(--fonu-success) 45%,var(--fonu-border)); background:color-mix(in srgb,var(--fonu-success) 5%,var(--fonu-surface)); }
.node-active .n-icon,.topology-connected .topology-arrow { color:var(--fonu-success); }
.topology-arrow { color:var(--fonu-text-secondary); font-size:16px; }
.dashboard-content { display:grid; grid-template-columns:minmax(0,1fr) minmax(0,1fr) minmax(210px,0.75fr); gap:12px; padding:12px; align-items:start; }
.dashboard-panel { min-width:0; display:grid; gap:10px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); padding:12px; background:var(--fonu-surface); }
.dash-heading { display:flex; justify-content:space-between; align-items:center; gap:8px; flex-wrap:wrap; }
.dash-heading h3 { display:flex; align-items:center; gap:7px; margin:0; font-size:15px; font-weight:650; color:var(--fonu-text); }
.dash-heading h3 .n-icon { color:var(--fonu-success); font-size:20px; }
.dash-heading > .n-button { font-size:12px; }
.dash-counts { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:8px; border-bottom:1px solid var(--fonu-border); padding-bottom:10px; }
.dash-counts > div { display:grid; gap:2px; }
.dash-counts strong { font-size:22px; color:var(--fonu-text); line-height:1.3; }
.dash-counts small { font-size:11px; color:var(--fonu-text-secondary); }
.dash-counts .good-text { color:var(--fonu-success); }
.dash-counts .warn-text { color:#b7791f; }
.dash-service-row { display:flex; gap:7px; align-items:center; min-width:0; border-bottom:1px solid var(--fonu-border); padding:4px 0 9px; flex-wrap:wrap; }
.dash-service-icon { display:grid; place-items:center; width:32px; height:32px; background:var(--fonu-bg-muted); color:var(--fonu-text-secondary); border-radius:8px; flex-shrink:0; font-size:18px; }
.dash-service-name { flex:1 1 105px; min-width:0; display:grid; gap:2px; }
.dash-service-name strong { font-size:13px; overflow-wrap:anywhere; }
.dash-service-name small { color:var(--fonu-text-secondary); font-size:11px; overflow-wrap:anywhere; }
.dash-service-row :deep(.n-switch) { flex-shrink:0; }
.dash-empty { padding:18px 4px; color:var(--fonu-text-secondary); text-align:center; font-size:12px; }
.dash-footer { display:flex; justify-content:center; padding-top:2px; }
.dash-side { display:grid; gap:12px; grid-column:3; grid-row:1 / span 2; min-width:0; }
.dash-shortcuts { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:7px; }
.dash-shortcuts button { min-width:0; display:grid; place-items:center; gap:5px; padding:10px 3px; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); background:var(--fonu-bg-muted); color:var(--fonu-text); font:inherit; font-size:11px; cursor:pointer; }
.dash-shortcuts button:hover { border-color:var(--fonu-success); }
.dash-shortcuts button:disabled { opacity:.55; cursor:not-allowed; }
.dash-shortcuts .n-icon { font-size:21px; color:var(--fonu-text-secondary); }
.dash-health { display:flex; justify-content:space-between; gap:8px; align-items:center; font-size:12px; padding:5px 0; border-bottom:1px solid var(--fonu-border); }
.dash-health:last-child { border:0; }
.dash-logs { grid-column:1 / span 2; }
.dash-log-preview { margin:0; min-width:0; white-space:pre-wrap; overflow-wrap:anywhere; max-height:150px; overflow:auto; background:var(--fonu-bg-muted); color:var(--fonu-text); padding:12px; border-radius:8px; font-size:12px; line-height:1.7; }
@container (max-width: 1100px) {
  .hero-grid { grid-template-columns:minmax(0,1fr) minmax(0,1.6fr); }
  .hero-kpis { grid-template-columns:repeat(2,minmax(0,1fr)); }
  .dashboard-content { grid-template-columns:repeat(2,minmax(0,1fr)); }
  .dash-side { grid-row:3; grid-column:1/-1; grid-template-columns:repeat(2,minmax(0,1fr)); }
}
@container (max-width: 760px) {
  .hero-grid { grid-template-columns:minmax(0,1fr); }
  .hero-service { gap:8px; }
  .hero-kpis { grid-template-columns:repeat(4,minmax(0,1fr)); }
  .hero-topology { padding:8px; }
}
@container (max-width: 540px) {
  .hero-kpis { grid-template-columns:repeat(2,minmax(0,1fr)); }
  .hero-topology { grid-template-columns:repeat(2,minmax(0,1fr)); }
  .topology-arrow { display:none; }
  .dashboard-content,.dash-side { grid-template-columns:minmax(0,1fr); }
  .dash-side,.dash-logs { grid-column:auto; grid-row:auto; }
  .dashboard-panel { padding:10px; }
}

/* Same row footprint and footer alignment even when service counts differ. */
.dash-web,.dash-tcp { align-self:stretch; grid-template-rows:auto auto 1fr auto; }
.dash-preview-list { display:flex; flex-direction:column; gap:0; min-width:0; position:relative; }
.dash-preview-list .dash-service-row { box-sizing:border-box; min-height:62px; padding:8px 0; }
.dash-preview-placeholder { min-height:62px; box-sizing:border-box; border-bottom:1px solid var(--fonu-border); opacity:.45; }
.dash-preview-list .dash-empty { position:absolute; inset:0; display:grid; place-items:center; padding:12px; background:var(--fonu-surface); }
.dash-footer { align-self:end; }
.tunnel-editor { width:min(640px,calc(100vw - 24px)); max-height:calc(100vh - 24px); max-height:calc(100dvh - 24px); display:flex; flex-direction:column; min-height:0; overflow:hidden; background:var(--fonu-surface); color:var(--fonu-text); border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); box-shadow:0 16px 50px #0003; }
.tunnel-editor-head { display:flex; align-items:center; justify-content:space-between; gap:12px; padding:14px 20px; border-bottom:1px solid var(--fonu-border); flex-shrink:0; }
.tunnel-editor-heading { display:flex; align-items:center; gap:10px; min-width:0; }
.tunnel-editor-icon { display:grid; place-items:center; width:36px; height:36px; flex-shrink:0; background:var(--fonu-brand-soft); color:var(--fonu-brand-text); border-radius:9px; font-size:21px; }
.tunnel-editor-head h3 { font-size:17px; margin:0; }
.tunnel-editor-head p { font-size:12px; color:var(--fonu-text-secondary); margin:3px 0 0; }
.tunnel-editor-body { padding:16px 20px; overflow:auto; overscroll-behavior:contain; min-height:0; flex:1 1 auto; display:flex; flex-direction:column; gap:12px; }
.tunnel-presets { display:flex; align-items:center; flex-wrap:wrap; gap:6px; font-size:12px; color:var(--fonu-text-secondary); }
.tunnel-form-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:12px 16px; }
.tunnel-form-grid .settings-field { min-width:0; }
.tunnel-form-grid .settings-field__label { font-size:13px; margin-bottom:6px; }
.tunnel-hint { font-size:11px; color:var(--fonu-text-secondary); line-height:1.5; display:block; margin-top:4px; }
.tunnel-enable { height:34px; display:flex; align-items:center; justify-content:space-between; gap:8px; padding:0 10px; border:1px solid var(--fonu-border); border-radius:8px; background:var(--fonu-bg-muted); font-size:12px; }
.tunnel-port-preview { display:grid; grid-template-columns:minmax(0,1fr) auto minmax(0,1.5fr); gap:10px; align-items:center; background:var(--fonu-bg-muted); border:1px solid var(--fonu-border); border-radius:8px; padding:9px 12px; }
.tunnel-port-preview > div { display:grid; gap:3px; min-width:0; }
.tunnel-port-preview small { color:var(--fonu-text-secondary); font-size:11px; }
.tunnel-port-preview strong { font-size:13px; overflow-wrap:anywhere; }
.tunnel-note { margin:0; }
.tunnel-editor-footer { flex-shrink:0; display:flex; align-items:center; justify-content:space-between; flex-wrap:wrap; gap:8px; padding:12px 20px; border-top:1px solid var(--fonu-border); }
.tunnel-editor-footer > div { display:flex; flex-wrap:wrap; gap:8px; margin-left:auto; }
@media(max-width:480px) { .tunnel-editor-head,.tunnel-editor-footer { padding:10px 12px; } .tunnel-editor-body { padding:12px; } .tunnel-form-grid { grid-template-columns:minmax(0,1fr); gap:10px; } .tunnel-editor-footer > .tunnel-hint { display:none; } }
</style>
