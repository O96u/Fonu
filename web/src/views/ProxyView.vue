<template>
  <PageHeader
    title="反向代理"
    description="通过 Nginx 反向代理，让内网服务可以通过域名安全访问"
  >
    <template #actions>
      <n-button type="primary" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        新建规则
      </n-button>
    </template>
  </PageHeader>

  <LoadError v-if="loadError" :message="loadError" @retry="load" />

  <template v-else>
    <div class="stats-row">
      <div class="stat-card">
        <n-icon :component="LayersOutline" class="stat-card__icon stat-card__icon--blue" />
        <div class="stat-card__value">{{ rules.length }}</div>
        <div class="stat-card__label">服务总数</div>
        <div class="stat-card__sub">{{ enabledCount }} 运行 · {{ disabledCount }} 停止</div>
      </div>
      <div class="stat-card">
        <n-icon :component="CloudUploadOutline" class="stat-card__icon stat-card__icon--green" />
        <div class="stat-card__value stat-card__value--sm">{{ formatBytes(trafficTotals.upload) }}</div>
        <div class="stat-card__label">总上传</div>
        <div class="stat-card__sub">累计流量</div>
      </div>
      <div class="stat-card">
        <n-icon :component="CloudDownloadOutline" class="stat-card__icon stat-card__icon--blue" />
        <div class="stat-card__value stat-card__value--sm">{{ formatBytes(trafficTotals.download) }}</div>
        <div class="stat-card__label">总下载</div>
        <div class="stat-card__sub">累计流量</div>
      </div>
      <div class="stat-card">
        <n-icon :component="ArrowUpOutline" class="stat-card__icon stat-card__icon--amber" />
        <div class="stat-card__value stat-card__value--sm">{{ formatRate(trafficTotals.uploadRate) }}</div>
        <div class="stat-card__label">当前上传</div>
        <div class="stat-card__sub">实时速率</div>
      </div>
      <div class="stat-card">
        <n-icon :component="ArrowDownOutline" class="stat-card__icon stat-card__icon--green" />
        <div class="stat-card__value stat-card__value--sm">{{ formatRate(trafficTotals.downloadRate) }}</div>
        <div class="stat-card__label">当前下载</div>
        <div class="stat-card__sub">实时速率</div>
      </div>
      <div class="stat-card">
        <n-icon :component="PeopleOutline" class="stat-card__icon stat-card__icon--purple" />
        <div class="stat-card__value">{{ trafficTotals.connections }}</div>
        <div class="stat-card__label">当前连接</div>
        <div class="stat-card__sub">全部规则合计</div>
      </div>
    </div>

    <div class="proxy-layout">
      <FonuCard flush class="proxy-panel">
        <div class="proxy-toolbar">
          <n-input
            v-model:value="search"
            clearable
            size="small"
            placeholder="搜索名称、域名或目标地址..."
            class="proxy-toolbar__search"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-select
            v-model:value="statusFilter"
            size="small"
            :options="statusOptions"
            clearable
            placeholder="全部状态"
            class="proxy-toolbar__filter"
          />
          <n-select
            v-model:value="httpsFilter"
            size="small"
            :options="httpsOptions"
            clearable
            placeholder="全部协议"
            class="proxy-toolbar__filter"
          />
          <div class="proxy-toolbar__spacer" />
          <n-button size="small" quaternary :loading="loading" @click="refreshAll">
            <template #icon><n-icon :component="RefreshOutline" /></template>
          </n-button>
        </div>

        <div v-if="loading && rules.length === 0" class="proxy-loading">
          <n-spin size="medium" />
        </div>

        <template v-else-if="showEntryGroups && entryGroups.length > 0">
          <div
            ref="entryStackRef"
            class="proxy-entry-stack"
            :class="{ 'proxy-entry-stack--sortable': canReorderEntries }"
          >
            <section
              v-for="group in entryGroups"
              :key="group.key"
              class="proxy-entry-card"
              :class="{ 'proxy-entry-card--collapsed': isEntryCollapsed(group.key) }"
            >
              <header class="proxy-entry-card__head">
                <span
                  v-if="canReorderEntries"
                  class="proxy-entry-drag-handle"
                  title="拖动排序"
                  @click.stop
                >
                  <n-icon :component="ReorderThreeOutline" />
                </span>
                <button
                  type="button"
                  class="proxy-entry-card__toggle"
                  @click="toggleEntryCollapsed(group.key)"
                >
                  <n-icon :component="isEntryCollapsed(group.key) ? ChevronForwardOutline : ChevronDownOutline" />
                </button>
                <div
                  class="proxy-entry-card__intro"
                  role="button"
                  tabindex="0"
                  @click="toggleEntryCollapsed(group.key)"
                  @keydown.enter.prevent="toggleEntryCollapsed(group.key)"
                  @keydown.space.prevent="toggleEntryCollapsed(group.key)"
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
                    <span v-if="group.enabledCount > 0" class="proxy-entry-card__running">{{ group.enabledCount }} 运行</span>
                  </div>
                  <div v-if="group.portHasMixedEntries" class="proxy-entry-card__warn">
                    同端口存在不同监听配置，建议统一
                  </div>
                  <div v-if="isEntryCollapsed(group.key) && group.rules.length > 0" class="proxy-entry-card__preview">
                    {{ entryGroupRuleNames(group, ruleName) }}
                  </div>
                </div>
                <div class="proxy-entry-card__stats">
                  <span class="proxy-entry-card__stat" :title="'当前连接 ' + entryGroupTraffic(group).connections">
                    <n-icon :component="PeopleOutline" />
                    {{ entryGroupTraffic(group).connections }}
                  </span>
                  <span class="proxy-entry-card__stat" title="当前上传">
                    <n-icon :component="ArrowUpOutline" />
                    {{ formatRate(entryGroupTraffic(group).uploadRate) }}
                  </span>
                  <span class="proxy-entry-card__stat" title="当前下载">
                    <n-icon :component="ArrowDownOutline" />
                    {{ formatRate(entryGroupTraffic(group).downloadRate) }}
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
                          @click="openCreateWithEntry(group.listen, group.entryId)"
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
                          @click="openDiscoveryModal(group)"
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
                          @click="openEntryEdit(group)"
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
                          @click="duplicateEntry(group)"
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
                          class="proxy-entry-card__action-btn proxy-entry-card__action-btn--danger"
                          :disabled="!group.entryId"
                          @click="confirmDeleteEntry(group)"
                        >
                          <template #icon><n-icon :component="TrashOutline" :size="16" /></template>
                        </n-button>
                      </template>
                      删除
                    </n-tooltip>
                  </n-space>
                </div>
              </header>
              <div v-show="!isEntryCollapsed(group.key)" class="proxy-entry-card__body" :data-entry-key="group.key">
                <n-data-table
                  v-if="group.rules.length > 0"
                  class="proxy-table proxy-table--nested"
                  :class="{ 'proxy-table--sortable': canReorderInGroups && group.rules.length > 1 }"
                  :columns="entryGroupColumnsFor(group)"
                  :data="group.rules"
                  :bordered="false"
                  size="small"
                  :scroll-x="canReorderInGroups ? 1220 : 1180"
                  :row-key="(r: ProxyRule) => r.id"
                  :row-props="rowProps"
                />
                <div v-else class="proxy-entry-card__empty">
                  <span class="text-muted">此入口下还没有规则</span>
                  <n-button size="small" quaternary type="primary" @click="openCreateWithEntry(group.listen, group.entryId)">
                    在此入口下新增
                  </n-button>
                </div>
              </div>
            </section>
          </div>
        </template>

        <template v-else-if="tableRules.length > 0">
          <div ref="tableWrapRef" class="proxy-table-wrap">
            <n-data-table
              class="proxy-table"
              :class="{ 'proxy-table--sortable': canReorderGlobally }"
              :columns="columns"
              :data="tableRules"
              :bordered="false"
              size="small"
              :scroll-x="canReorderGlobally ? 1460 : 1420"
              :row-key="(r: ProxyRule) => r.id"
              :row-props="rowProps"
            />
          </div>
        </template>

        <EmptyState
          v-else-if="rules.length === 0"
          title="还没有反向代理规则"
          description="同一内网服务多个域名：一条规则多行域名即可。同一端口多个不同服务：先建入口，再在同一入口下新增规则。"
        >
          <template #action>
            <n-button type="primary" @click="openCreate">新建规则</n-button>
          </template>
        </EmptyState>

        <EmptyState
          v-else
          title="没有匹配的规则"
          description="试试调整搜索关键词或筛选条件。"
        />
      </FonuCard>
    </div>
  </template>

  <n-modal v-model:show="showDetailPanel" :mask-closable="true" transform-origin="center">
    <div v-if="selectedRule" class="proxy-detail-modal">
      <div class="proxy-detail-modal__header">
        <div class="proxy-detail-modal__intro">
          <h3 class="proxy-detail-modal__title">{{ ruleName(selectedRule) }}</h3>
          <StatusBadge
            :value="selectedRule.enabled ? 'ok' : 'disabled'"
            :text="selectedRule.enabled ? '运行中' : '已停止'"
          />
          <span class="proxy-detail-modal__conn" :class="{ 'is-active': (selectedTraffic?.connections ?? 0) > 0 }">
            <n-icon :component="PeopleOutline" />
            {{ selectedTraffic?.connections ?? 0 }} 连接
          </span>
        </div>
        <n-space :size="4" align="center">
          <n-button size="small" quaternary @click="openDuplicate(selectedRule)">复制</n-button>
          <n-button size="small" quaternary type="primary" @click="openEdit(selectedRule)">编辑</n-button>
          <n-button size="small" quaternary type="error" @click="confirmDelete(selectedRule)">删除</n-button>
          <n-button size="small" quaternary @click="closeDetail">
            <template #icon><n-icon :component="CloseOutline" /></template>
          </n-button>
        </n-space>
      </div>

      <div class="proxy-detail__tabbar">
        <button
          type="button"
          class="proxy-detail__tab"
          :class="{ 'proxy-detail__tab--active': detailTab === 'overview' }"
          @click="switchDetailTab('overview')"
        >
          概览
        </button>
        <button
          type="button"
          class="proxy-detail__tab"
          :class="{ 'proxy-detail__tab--active': detailTab === 'logs' }"
          @click="switchDetailTab('logs')"
        >
          日志
        </button>
        <button
          type="button"
          class="proxy-detail__tab"
          :class="{ 'proxy-detail__tab--active': detailTab === 'nginx' }"
          @click="switchDetailTab('nginx')"
        >
          Nginx
        </button>
      </div>

      <div class="proxy-detail__scroll">
        <div v-show="detailTab === 'overview'" class="proxy-detail__pane proxy-detail__pane--overview">
            <div class="overview-strip">
              <div class="overview-strip__item">
                <span class="overview-strip__label">域名</span>
                <span class="overview-strip__value" :title="ruleHosts(selectedRule).join('、')">
                  {{ ruleHosts(selectedRule).join('、') }}
                </span>
              </div>
              <div class="overview-strip__item">
                <span class="overview-strip__label">监听</span>
                <span class="overview-strip__value">{{ listenLabel(selectedRule) }}</span>
              </div>
              <div class="overview-strip__item overview-strip__item--wide">
                <span class="overview-strip__label">目标</span>
                <span class="overview-strip__value mono" :title="selectedRule.upstream">{{ selectedRule.upstream }}</span>
              </div>
              <div class="overview-strip__item">
                <span class="overview-strip__label">协议</span>
                <span class="overview-strip__value">
                  <n-tag size="small" :type="selectedRule.https_enabled ? 'success' : 'default'" :bordered="false" round>
                    {{ selectedRule.https_enabled ? 'HTTPS' : 'HTTP' }}
                  </n-tag>
                </span>
              </div>
            </div>

            <div class="overview-security">
              <span class="overview-security__label">安全策略</span>
              <div v-if="selectedSecurityFeatures.length" class="overview-security__tags">
                <n-tag
                  v-for="tag in selectedSecurityFeatures"
                  :key="tag"
                  size="small"
                  round
                  :bordered="false"
                >
                  {{ tag }}
                </n-tag>
              </div>
              <span v-else class="overview-security__empty">未启用</span>
            </div>

            <div class="overview-main">
              <section class="overview-card overview-card--chart">
                <div class="overview-card__head">
                  <h4>实时流量 <span class="overview-card__sub">最近 5 分钟</span></h4>
                  <span class="live-badge"><span class="live-badge__dot" />实时</span>
                </div>
                <div class="traffic-live-legend traffic-live-legend--compact">
                  <span class="traffic-live-legend__item traffic-live-legend__item--up">
                    <span class="traffic-live-legend__dot" />
                    上传 {{ formatRate(selectedTraffic?.upload_rate ?? 0) }}
                  </span>
                  <span class="traffic-live-legend__item traffic-live-legend__item--down">
                    <span class="traffic-live-legend__dot" />
                    下载 {{ formatRate(selectedTraffic?.download_rate ?? 0) }}
                  </span>
                </div>
                <MiniTrafficChart
                  class="overview-chart"
                  :labels="trafficChart.labels"
                  :upload="trafficChart.upload"
                  :download="trafficChart.download"
                />
              </section>

              <aside class="overview-side">
                <section class="overview-metrics">
                  <div class="overview-metric">
                    <div class="overview-metric__label">总上传</div>
                    <div class="overview-metric__value">{{ formatBytes(selectedTraffic?.upload_total ?? 0) }}</div>
                  </div>
                  <div class="overview-metric">
                    <div class="overview-metric__label">总下载</div>
                    <div class="overview-metric__value">{{ formatBytes(selectedTraffic?.download_total ?? 0) }}</div>
                  </div>
                  <div class="overview-metric">
                    <div class="overview-metric__label">当前上传</div>
                    <div class="overview-metric__value">{{ formatRate(selectedTraffic?.upload_rate ?? 0) }}</div>
                  </div>
                  <div class="overview-metric">
                    <div class="overview-metric__label">当前下载</div>
                    <div class="overview-metric__value">{{ formatRate(selectedTraffic?.download_rate ?? 0) }}</div>
                  </div>
                </section>

                <section class="overview-card overview-card--clients">
                  <div class="overview-card__head">
                    <h4>
                      最近访问
                      <span class="overview-card__sub">65 秒内 · {{ clientRows.length }} 个</span>
                    </h4>
                    <n-button size="tiny" quaternary :loading="clientsLoading" @click="loadClients">刷新</n-button>
                  </div>
                  <div class="overview-client-list">
                    <template v-if="clientRows.length > 0">
                      <div v-for="row in clientRows" :key="row.ip" class="overview-client-row">
                        <span class="mono">{{ row.ip }}</span>
                        <span class="overview-client-row__time">{{ formatRelativeTime(row.last_seen) }}</span>
                      </div>
                    </template>
                    <p v-else class="overview-empty">暂无访问记录</p>
                  </div>
                </section>
              </aside>
            </div>
        </div>

        <div v-show="detailTab === 'logs'" class="proxy-detail__pane proxy-detail__pane--logs">
              <div class="log-panel-head">
                <span class="text-muted">实时访问日志</span>
                <div class="log-panel-actions">
                  <n-button size="tiny" quaternary @click="openLogFullscreen">全屏</n-button>
                  <n-button size="tiny" quaternary @click="clearLogLines">清空</n-button>
                </div>
              </div>
              <ProxyAccessLogBox
                ref="logBox"
                :lines="logLines"
                embedded
                @scroll="onLogBoxScroll"
              />
        </div>

        <div v-show="detailTab === 'nginx'" class="proxy-detail__pane proxy-detail__pane--nginx">
            <n-alert v-if="!nginxEnabled" type="warning" :bordered="false" class="nginx-pane-alert">
              规则已停用，以下配置不会写入 Nginx。
            </n-alert>
            <n-alert
              v-if="nginxServerMode === 'custom'"
              type="info"
              :bordered="false"
              class="nginx-pane-alert"
            >
              当前为手动编辑配置。如需修改，请使用右上角「编辑」。
            </n-alert>
            <n-spin :show="nginxLoading" class="nginx-editor-spin">
              <NginxCodeEditor
                :model-value="detailNginxText"
                embedded
                readonly
                placeholder="server { ... }"
              />
            </n-spin>
        </div>
      </div>
    </div>
  </n-modal>

  <Teleport to="body">
    <div v-if="logFullscreen && selectedRule" class="proxy-log-fullscreen">
      <div class="log-panel-head proxy-log-fullscreen__head">
        <div class="proxy-log-fullscreen__title">
          <span class="text-muted">实时访问日志</span>
          <span class="proxy-log-fullscreen__rule">{{ ruleName(selectedRule) }}</span>
        </div>
        <div class="log-panel-actions">
          <n-button size="tiny" quaternary @click="clearLogLines">清空</n-button>
          <n-button size="tiny" quaternary @click="logFullscreen = false">退出全屏</n-button>
        </div>
      </div>
      <ProxyAccessLogBox
        ref="logBoxFullscreen"
        :lines="logLines"
        fullscreen
        @scroll="onLogBoxScroll"
      />
    </div>
  </Teleport>

  <n-modal v-model:show="showModal" :mask-closable="false" transform-origin="center">
    <div class="proxy-modal">
      <div class="proxy-modal__form">
        <div class="proxy-modal__header">
          <h3 class="modal-title">{{ editing ? '编辑规则' : '新增规则' }}</h3>
          <n-button size="small" quaternary @click="closeModal">
            <template #icon><n-icon :component="CloseOutline" /></template>
          </n-button>
        </div>

        <div class="proxy-modal__tabbar">
          <button
            type="button"
            class="proxy-modal__tab"
            :class="{ 'proxy-modal__tab--active': formTab === 'basic' }"
            @click="switchFormTab('basic')"
          >
            基础配置
          </button>
          <button
            type="button"
            class="proxy-modal__tab"
            :class="{ 'proxy-modal__tab--active': formTab === 'security' }"
            @click="switchFormTab('security')"
          >
            <span class="proxy-modal__tab-label">
              安全设置
              <n-tag v-if="activeSecurityFeatures.length" size="tiny" round :bordered="false" type="success">
                {{ activeSecurityFeatures.length }}
              </n-tag>
            </span>
          </button>
          <button
            type="button"
            class="proxy-modal__tab"
            :class="{ 'proxy-modal__tab--active': formTab === 'nginx' }"
            @click="switchFormTab('nginx')"
          >
            Nginx
          </button>
        </div>

        <div class="proxy-modal__scroll">
          <n-form v-show="formTab === 'basic'" label-placement="top" class="proxy-modal__pane">
              <n-form-item label="名称">
                <n-input
                  v-model:value="form.name"
                  maxlength="100"
                  show-count
                  placeholder="选填，用于在列表中识别该规则"
                />
              </n-form-item>

              <n-form-item required>
                <template #label>
                  <span class="form-label">
                    前端域名
                    <n-tooltip trigger="hover">
                      <template #trigger>
                        <n-icon :component="HelpCircleOutline" class="form-label__help" />
                      </template>
                      每行一个域名；如需单独端口可写 example.com:6893
                    </n-tooltip>
                  </span>
                </template>
                <div class="field-stack">
                  <n-input
                    v-model:value="form.hostsText"
                    type="textarea"
                    :rows="3"
                    placeholder="s.example.com&#10;api.example.com&#10;example.com:6893"
                  />
                  <p class="field-hint">
                    多个域名指向<strong>同一</strong>内网服务时，在此每行填一个域名即可，无需新建多条规则
                  </p>
                </div>
              </n-form-item>

              <div v-if="!bindEntryListen" class="listen-row">
                <div class="listen-col listen-col--port">
                  <div class="listen-col__label">监听端口 <span class="required-mark">*</span></div>
                  <div class="listen-col__control">
                    <n-input-number v-model:value="form.listen_port" :min="1" :max="65535" class="port-input" />
                  </div>
                </div>
                <div class="listen-col listen-col--protocol">
                  <div class="listen-col__label">监听协议</div>
                  <div class="listen-col__control">
                    <div class="listen-types">
                      <n-checkbox v-model:checked="form.listen_ipv4">IPv4</n-checkbox>
                      <n-checkbox v-model:checked="form.listen_ipv6">IPv6</n-checkbox>
                    </div>
                  </div>
                </div>
              </div>
              <div v-else class="listen-row listen-row--readonly">
                <div class="listen-col listen-col--port">
                  <div class="listen-col__label">监听入口</div>
                  <div class="listen-col__control">
                    <n-tag size="small" round :bordered="false">
                      {{ entryGroupDisplayName({ key: '', name: '', listen: form, rules: [], enabledCount: 0, portHasMixedEntries: false }) }}
                    </n-tag>
                  </div>
                </div>
              </div>

              <n-form-item label="目标地址" required>
                <div class="field-stack">
                  <n-input v-model:value="form.upstream" placeholder="例如：http://192.168.1.100:5173" />
                  <p class="field-hint">支持 http://、https://，也可以是 IP 地址或内网域名</p>
                </div>
              </n-form-item>

              <div v-if="!bindEntryListen" class="form-switch-list form-switch-list--compact">
                <div class="form-switch-row">
                  <div class="form-switch-row__text">
                    <div class="form-switch-row__label">启用 HTTPS</div>
                    <div class="form-switch-row__hint">为前端域名启用 HTTPS 访问</div>
                  </div>
                  <n-switch v-model:value="form.https_enabled" />
                </div>
                <div class="form-switch-row">
                  <div class="form-switch-row__text">
                    <div class="form-switch-row__label">HTTP 跳转 HTTPS</div>
                    <div class="form-switch-row__hint">将 HTTP 请求自动跳转为 HTTPS</div>
                  </div>
                  <n-switch v-model:value="form.http_redirect" :disabled="!form.https_enabled" />
                </div>
                <div class="form-switch-row">
                  <div class="form-switch-row__text">
                    <div class="form-switch-row__label">启用规则</div>
                    <div class="form-switch-row__hint">保存后立即开始转发请求</div>
                  </div>
                  <n-switch v-model:value="form.enabled" />
                </div>
              </div>
              <div v-else-if="editing" class="form-switch-list form-switch-list--compact">
                <div class="form-switch-row">
                  <div class="form-switch-row__text">
                    <div class="form-switch-row__label">启用规则</div>
                    <div class="form-switch-row__hint">保存后立即开始转发请求</div>
                  </div>
                  <n-switch v-model:value="form.enabled" />
                </div>
              </div>
            </n-form>

          <div v-show="formTab === 'security'" class="proxy-modal__pane proxy-modal__pane--security">
              <div class="security-section">
              <div class="security-header">
                <div class="security-header__row">
                  <div class="security-header__status">
                    <span class="security-header__status-label">已启用</span>
                    <div v-if="activeSecurityFeatures.length" class="security-header__tags">
                      <n-tag
                        v-for="tag in activeSecurityFeatures"
                        :key="tag"
                        size="small"
                        round
                        :bordered="false"
                      >
                        {{ tag }}
                      </n-tag>
                    </div>
                    <span v-else class="security-header__empty">暂无</span>
                  </div>
                  <n-button type="primary" size="tiny" class="security-header__preset" @click="applySecurityPreset">
                    <template #icon><n-icon :component="FlashOutline" :size="14" /></template>
                    一键推荐
                  </n-button>
                </div>
                <p class="security-header__note">
                  IP 策略依赖「设置 → 信任代理」；白名单 IP 可豁免「仅中国大陆」限制。
                </p>
              </div>

              <n-collapse v-model:expanded-names="securityExpanded" class="security-collapse">
                <n-collapse-item title="IP 访问控制" name="ip">
                  <div class="security-panel">
                    <div class="security-option">
                      <div class="security-option__text">
                        <div class="security-option__label">仅中国大陆 IP</div>
                        <div class="security-option__hint">
                          需先在设置页更新中国 IP 段；内网（10/8、172.16/12、192.168/16 等）默认放行
                        </div>
                      </div>
                      <n-switch v-model:value="form.china_only" size="small" />
                    </div>

                    <div class="security-option">
                      <div class="security-option__text">
                        <div class="security-option__label">黑名单模式</div>
                        <div class="security-option__hint">启用后拒绝列表中的 IP 访问</div>
                      </div>
                      <n-switch v-model:value="form.ip_blacklist_mode" size="small" />
                    </div>
                    <div v-if="form.ip_blacklist_mode" class="security-panel__fields">
                      <n-form-item label="IP 黑名单">
                        <n-input
                          v-model:value="form.ip_blacklist_text"
                          type="textarea"
                          :rows="2"
                          placeholder="每行一个 IP 或 CIDR"
                        />
                      </n-form-item>
                    </div>

                    <div class="security-option">
                      <div class="security-option__text">
                        <div class="security-option__label">白名单模式</div>
                        <div class="security-option__hint">启用后仅允许白名单 IP 访问</div>
                      </div>
                      <n-switch v-model:value="form.ip_whitelist_mode" size="small" />
                    </div>
                    <div v-if="form.ip_whitelist_mode" class="security-panel__fields">
                      <n-form-item label="IP 白名单">
                        <n-input
                          v-model:value="form.ip_whitelist_text"
                          type="textarea"
                          :rows="2"
                          placeholder="每行一个 IP 或 CIDR"
                        />
                      </n-form-item>
                    </div>
                  </div>
                </n-collapse-item>

                <n-collapse-item title="认证" name="auth">
                  <div class="security-panel">
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">Basic Auth</div>
                      <div class="security-option__hint">浏览器弹窗认证</div>
                    </div>
                    <n-switch v-model:value="form.basic_auth_enabled" size="small" />
                  </div>
                  <div v-if="form.basic_auth_enabled" class="security-fields-grid">
                    <n-form-item label="用户名">
                      <n-input v-model:value="form.basic_auth_username" placeholder="用户名" />
                    </n-form-item>
                    <n-form-item>
                      <template #label>
                        <span class="proxy-secret-label">
                          密码
                          <n-tag
                            v-if="editing?.security?.basic_auth?.has_password"
                            size="small"
                            type="success"
                            :bordered="false"
                          >
                            {{ CONFIGURED_SECRET_TAG }}
                          </n-tag>
                        </span>
                      </template>
                      <n-input
                        v-model:value="form.basic_auth_password"
                        type="password"
                        show-password-on="click"
                        :placeholder="
                          editing?.security?.basic_auth?.has_password
                            ? CONFIGURED_SECRET_PLACEHOLDER
                            : '至少 8 位'
                        "
                      />
                    </n-form-item>
                  </div>
                  </div>
                </n-collapse-item>

                <n-collapse-item title="流量控制" name="traffic">
                  <div class="security-panel">
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">请求限流</div>
                      <div class="security-option__hint">超出速率返回 429</div>
                    </div>
                    <n-switch v-model:value="form.rate_limit_enabled" size="small" />
                  </div>
                  <div v-if="form.rate_limit_enabled" class="security-fields-grid">
                    <n-form-item label="每秒请求数">
                      <n-input-number v-model:value="form.rate_limit_rate" :min="1" :max="10000" class="w-full" />
                    </n-form-item>
                    <n-form-item label="突发上限">
                      <n-input-number v-model:value="form.rate_limit_burst" :min="1" :max="100000" class="w-full" />
                    </n-form-item>
                  </div>

                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">连接数限制</div>
                      <div class="security-option__hint">每 IP 最大并发连接</div>
                    </div>
                    <n-switch v-model:value="form.conn_limit_enabled" size="small" />
                  </div>
                  <div v-if="form.conn_limit_enabled" class="security-panel__fields">
                    <n-form-item label="最大连接数">
                      <n-input-number v-model:value="form.conn_limit_max" :min="1" :max="10000" class="conn-limit-input" />
                    </n-form-item>
                  </div>
                  </div>
                </n-collapse-item>

                <n-collapse-item title="高级" name="advanced">
                  <div class="security-panel">
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">忽略后端 TLS 证书</div>
                      <div class="security-option__hint">上游为自签 https:// 时使用</div>
                    </div>
                    <n-switch v-model:value="form.proxy_ssl_verify_off" size="small" />
                  </div>
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">使用目标 Host 头</div>
                      <div class="security-option__hint">转发时使用上游地址作为 Host</div>
                    </div>
                    <n-switch v-model:value="form.proxy_host_upstream" size="small" />
                  </div>
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">仅 TLS 1.3</div>
                    </div>
                    <n-switch v-model:value="form.tls_min_13_only" size="small" :disabled="!form.https_enabled" />
                  </div>
                  <div class="security-option">
                    <div class="security-option__text">
                      <div class="security-option__label">安全响应头</div>
                      <div class="security-option__hint">HSTS、X-Frame-Options 等</div>
                    </div>
                    <n-switch v-model:value="form.security_headers" size="small" :disabled="!form.https_enabled" />
                  </div>
                  </div>
                </n-collapse-item>
              </n-collapse>
              </div>
          </div>

          <div v-show="formTab === 'nginx'" class="proxy-modal__pane proxy-modal__pane--nginx">
            <template v-if="!editing">
              <n-empty description="请先保存规则后再配置 Nginx">
                <template #extra>
                  <n-button type="primary" @click="switchFormTab('basic')">去填写基础配置</n-button>
                </template>
              </n-empty>
            </template>
            <template v-else>
              <n-alert v-if="nginxEditMode === 'custom'" type="info" :bordered="false" class="nginx-pane-alert">
                手动模式下，修改基础/安全设置后需先保存规则，再在此处同步 Nginx 文本。
              </n-alert>
              <div class="log-panel-head">
                <div class="nginx-mode-toggle">
                  <n-button
                    size="tiny"
                    quaternary
                    :type="nginxEditMode === 'auto' ? 'primary' : 'default'"
                    @click="setNginxEditMode('auto')"
                  >
                    自动生成
                  </n-button>
                  <n-button
                    size="tiny"
                    quaternary
                    :type="nginxEditMode === 'custom' ? 'primary' : 'default'"
                    @click="setNginxEditMode('custom')"
                  >
                    手动编辑
                  </n-button>
                </div>
                <div class="log-panel-actions">
                  <n-button
                    v-if="nginxEditMode === 'custom'"
                    size="tiny"
                    quaternary
                    type="primary"
                    :loading="nginxSaving"
                    :disabled="!nginxDirty"
                    @click="saveRuleNginx"
                  >
                    保存 Nginx
                  </n-button>
                  <n-button
                    v-if="nginxEditMode === 'custom' && nginxBackups.length > 0"
                    size="tiny"
                    quaternary
                    :loading="nginxSaving"
                    @click="rollbackRuleNginx()"
                  >
                    回滚
                  </n-button>
                  <n-button
                    v-if="nginxEditMode === 'custom' && nginxServerMode === 'custom'"
                    size="tiny"
                    quaternary
                    :loading="nginxSaving"
                    @click="resetRuleNginxAuto"
                  >
                    恢复自动生成
                  </n-button>
                </div>
              </div>
              <n-spin :show="nginxLoading" class="nginx-editor-spin nginx-editor-spin--modal">
                <NginxCodeEditor
                  v-model="nginxDraft"
                  embedded
                  :readonly="nginxEditMode === 'auto'"
                  placeholder="server { ... }"
                  @update:model-value="onNginxDraftInput"
                />
              </n-spin>
            </template>
          </div>
        </div>

        <div class="modal-footer">
          <n-button @click="closeModal">取消</n-button>
          <n-button v-if="formTab !== 'nginx'" type="primary" :loading="saving" @click="save">
            {{ editing ? '保存' : '创建' }}
          </n-button>
        </div>
      </div>

      <div class="proxy-modal__help">
        <template v-if="formTab === 'basic'">
          <h4>配置说明</h4>
          <ol>
            <li>
              <strong>入口分组</strong>：同一端口、不同内网服务应放在同一入口下分别建规则；同一服务多域名写在一条规则里。
            </li>
            <li>
              <strong>前端域名</strong>：支持多个域名，每行一个；单独端口可写
              <code>example.com:6893</code>。
            </li>
            <li>
              <strong>目标地址</strong>：内网服务地址，支持 <code>http://</code>、<code>https://</code> 或 IP。
            </li>
            <li>
              <strong>HTTPS</strong>：需在「证书」页为域名申请或上传证书。
            </li>
          </ol>
          <div class="proxy-modal__tip">
            <n-icon :component="InformationCircleOutline" class="proxy-modal__tip-icon" />
            <span>请确保域名已解析到本机，且内网服务可访问。</span>
          </div>
        </template>
        <template v-else-if="formTab === 'nginx'">
          <h4>Nginx 说明</h4>
          <ol>
            <li><strong>自动生成</strong>：根据基础配置与安全设置生成，并附带中文注释。</li>
            <li><strong>手动编辑</strong>：保存前会自动备份，支持回滚。</li>
            <li>修改基础/安全项后请先点「保存」，再回到此页刷新预览。</li>
          </ol>
          <div class="proxy-modal__tip">
            <n-icon :component="InformationCircleOutline" class="proxy-modal__tip-icon" />
            <span>详情页的 Nginx 页签仅展示当前生效配置。</span>
          </div>
          <div class="proxy-modal__tip proxy-modal__tip--warn">
            <n-icon :component="WarningOutline" class="proxy-modal__tip-icon" />
            <span>小白勿碰。若保存后 Nginx 启动失败，请点「恢复自动生成」或「回滚」还原配置。</span>
          </div>
        </template>
        <template v-else>
          <h4>安全说明</h4>
          <ol>
            <li>
              <strong>IP 策略</strong>：经 CDN 访问时，请在「设置」配置信任代理，否则限流与 IP 规则可能不准。
            </li>
            <li>
              <strong>仅中国大陆</strong>：需先在设置页更新中国 IP 段；内网 IP 默认放行，额外豁免 IP 可写在白名单里（无需开启白名单模式）。
            </li>
            <li>
              <strong>一键推荐</strong>：HTTPS + 限流 + 安全响应头，适合公网暴露场景。
            </li>
          </ol>
          <div class="proxy-modal__tip">
            <n-icon :component="InformationCircleOutline" class="proxy-modal__tip-icon" />
            <span>安全策略保存后会自动重载 Nginx 配置。</span>
          </div>
        </template>
      </div>
    </div>
  </n-modal>

  <n-modal v-model:show="showEntryModal" :mask-closable="false" transform-origin="center">
    <div class="proxy-entry-edit-modal">
      <div class="proxy-entry-edit-modal__header">
        <h3 class="modal-title">编辑入口</h3>
        <n-button size="small" quaternary @click="showEntryModal = false">
          <template #icon><n-icon :component="CloseOutline" /></template>
        </n-button>
      </div>
      <n-form label-placement="top" class="proxy-entry-edit-modal__body">
        <n-form-item label="名称">
          <n-input v-model:value="entryForm.name" maxlength="100" show-count placeholder="选填，便于识别该入口" />
        </n-form-item>
        <div class="listen-row">
          <div class="listen-col listen-col--port">
            <div class="listen-col__label">监听端口 <span class="required-mark">*</span></div>
            <div class="listen-col__control">
              <n-input-number v-model:value="entryForm.listen_port" :min="1" :max="65535" class="port-input" />
            </div>
          </div>
          <div class="listen-col listen-col--protocol">
            <div class="listen-col__label">监听协议</div>
            <div class="listen-col__control">
              <div class="listen-types">
                <n-checkbox v-model:checked="entryForm.listen_ipv4">IPv4</n-checkbox>
                <n-checkbox v-model:checked="entryForm.listen_ipv6">IPv6</n-checkbox>
              </div>
            </div>
          </div>
        </div>
        <div class="form-switch-list form-switch-list--compact">
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">启用 HTTPS</div>
            </div>
            <n-switch v-model:value="entryForm.https_enabled" />
          </div>
          <div class="form-switch-row">
            <div class="form-switch-row__text">
              <div class="form-switch-row__label">HTTP 跳转 HTTPS</div>
            </div>
            <n-switch v-model:value="entryForm.http_redirect" :disabled="!entryForm.https_enabled" />
          </div>
        </div>
        <p class="field-hint">修改后将同步应用到该入口下的全部规则。</p>
      </n-form>
      <div class="modal-footer">
        <n-button @click="showEntryModal = false">取消</n-button>
        <n-button type="primary" :loading="savingEntry" @click="saveEntry">保存</n-button>
      </div>
    </div>
  </n-modal>

  <n-modal v-model:show="showDiscoveryModal" :mask-closable="false" transform-origin="center">
    <div class="discovery-modal">
      <div class="discovery-modal__header">
        <div>
          <h3 class="modal-title">发现内网服务</h3>
          <p class="discovery-modal__subtitle">
            {{ discoveryTarget ? '扫描常见端口并识别 HTTP 服务，勾选后填写前端域名添加到当前入口' : '扫描常见端口并识别 HTTP 服务，勾选后批量创建反向代理规则' }}
          </p>
        </div>
        <n-button size="small" quaternary @click="showDiscoveryModal = false">
          <template #icon><n-icon :component="CloseOutline" /></template>
        </n-button>
      </div>

      <div class="discovery-modal__toolbar">
        <n-input
          v-model:value="discoveryHost"
          size="small"
          placeholder="扫描地址，如 127.0.0.1"
          class="discovery-modal__host"
          :disabled="discoveryLoading"
          @keydown.enter.prevent="runDiscoveryScan"
        />
        <n-button size="small" type="primary" :loading="discoveryLoading" @click="runDiscoveryScan">
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
              <span>选择</span>
              <span>服务</span>
              <span>规则名称</span>
              <span>前端域名</span>
            </div>
            <div
              v-for="row in discoveryRows"
              :key="row.id"
              class="discovery-item"
            >
              <n-checkbox v-model:checked="row.selected" :disabled="discoveryLoading" />
              <div class="discovery-item__info">
                <div class="discovery-item__title mono">{{ row.service.upstream }}</div>
                <div class="discovery-item__meta">{{ row.service.name }}</div>
                <div v-if="row.service.suggestion" class="discovery-item__hint">{{ row.service.suggestion }}</div>
              </div>
              <n-input
                v-model:value="row.ruleName"
                size="small"
                maxlength="100"
                placeholder="规则名称"
                :disabled="!row.selected || discoveryLoading"
              />
              <n-input
                v-model:value="row.hostname"
                size="small"
                placeholder="例如 nas.example.com"
                :disabled="!row.selected || discoveryLoading"
              />
            </div>
          </div>
        </div>
      </div>

      <div class="modal-footer discovery-modal__footer">
        <n-button @click="showDiscoveryModal = false" :disabled="discoveryLoading">取消</n-button>
        <n-button :loading="discoveryLoading" @click="runDiscoveryScan">
          {{ discoveryLoading ? '扫描中…' : '重新扫描' }}
        </n-button>
        <n-button
          type="primary"
          :loading="discoveryAdding"
          :disabled="!hasSelectedDiscoveryRows || discoveryLoading"
          @click="submitDiscovery"
        >
          {{ discoveryTarget ? '添加为子规则' : '批量创建规则' }}
        </n-button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onMounted, onUnmounted, reactive, ref, watch, type Component, type VNode } from 'vue'
import Sortable from 'sortablejs'
import {
  NAlert,
  NButton,
  NCheckbox,
  NCollapse,
  NCollapseItem,
  NDataTable,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NPopover,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NTag,
  NTooltip,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import {
  AddOutline,
  ArrowDownOutline,
  ArrowUpOutline,
  ChevronDownOutline,
  ChevronForwardOutline,
  CloseOutline,
  CopyOutline,
  CreateOutline,
  OpenOutline,
  ReorderThreeOutline,
  CloudDownloadOutline,
  CloudUploadOutline,
  FlashOutline,
  GlobeOutline,
  HelpCircleOutline,
  InformationCircleOutline,
  LayersOutline,
  PeopleOutline,
  RefreshOutline,
  SearchOutline,
  TrashOutline,
  WarningOutline,
} from '@vicons/ionicons5'
import { api, asList } from '../api/client'
import type { DiscoveredService, ProxyClientConn, ProxyEntry, ProxyRule, ProxySavePayload, ProxyTraffic } from '../api/types'
import {
  applyEntryListen,
  buildProxyEntryGroups,
  entryGroupDisplayName,
  entryGroupRuleNames,
  type ProxyEntryGroup,
  type ProxyEntryListen,
} from '../utils/proxyEntries'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import ProxyAccessLogBox from '../components/ProxyAccessLogBox.vue'
import NginxCodeEditor from '../components/NginxCodeEditor.vue'
import LoadError from '../components/LoadError.vue'
import MiniTrafficChart from '../components/MiniTrafficChart.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { CONFIGURED_SECRET_PLACEHOLDER, CONFIGURED_SECRET_TAG } from '../constants/secretField'
import { formatBytes, formatRate, formatRelativeTime } from '../utils/format'
import { renderTableRowActions } from '../utils/tableActions'

const message = useMessage()
const dialog = useDialog()
const showDiscoveryModal = ref(false)
const discoveryHost = ref('127.0.0.1')
const discoveryScanHost = ref('127.0.0.1')
const discoveryLoading = ref(false)
const discoveryAdding = ref(false)
const discoveryTarget = ref<ProxyEntryGroup | null>(null)

interface DiscoveryRow {
  id: string
  selected: boolean
  service: DiscoveredService
  ruleName: string
  hostname: string
}

const discoveryRows = ref<DiscoveryRow[]>([])

const discoverySummary = computed(() => {
  if (discoveryLoading.value || discoveryRows.value.length === 0) return ''
  return `扫描完成，发现 ${discoveryRows.value.length} 个 HTTP 服务`
})

const hasSelectedDiscoveryRows = computed(() =>
  discoveryRows.value.some((row) => row.selected && row.hostname.trim()),
)

const showEntryModal = ref(false)
const savingEntry = ref(false)
const editingEntryId = ref<number | null>(null)
const entryForm = reactive({
  name: '',
  listen_port: 443,
  listen_ipv4: true,
  listen_ipv6: false,
  https_enabled: true,
  http_redirect: true,
})
const rules = ref<ProxyRule[]>([])
const entries = ref<ProxyEntry[]>([])
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const showModal = ref(false)
const formTab = ref<'basic' | 'security' | 'nginx'>('basic')
const securityExpanded = ref<string[]>(['ip'])
const editing = ref<ProxyRule | null>(null)
const search = ref('')
const statusFilter = ref<string | null>(null)
const httpsFilter = ref<string | null>(null)
const selectedRuleId = ref<number | null>(null)
const showDetailPanel = ref(false)
const detailTab = ref<'overview' | 'logs' | 'nginx'>('overview')
const nginxLoading = ref(false)
const nginxSaving = ref(false)
const nginxEditMode = ref<'auto' | 'custom'>('auto')
const nginxServerMode = ref<'auto' | 'custom'>('auto')
const nginxGenerated = ref('')
const nginxDraft = ref('')
const nginxDirty = ref(false)
const nginxEnabled = ref(true)
const nginxBackups = ref<{ name: string; created_at: string }[]>([])
let nginxRefreshTimer: ReturnType<typeof setTimeout> | null = null
const logLines = ref<string[]>([])
const logBox = ref<InstanceType<typeof ProxyAccessLogBox> | null>(null)
const logBoxFullscreen = ref<InstanceType<typeof ProxyAccessLogBox> | null>(null)
const logFullscreen = ref(false)
const logStickToBottom = ref(true)
const LOG_SCROLL_BOTTOM_THRESHOLD = 24
let logEventSource: EventSource | null = null
const trafficByRule = ref<Record<number, ProxyTraffic>>({})
const clientRows = ref<ProxyClientConn[]>([])
const clientsLoading = ref(false)
type RatePoint = { at: number; upload: number; download: number }
const rateHistory = ref<RatePoint[]>([])
const RATE_HISTORY_MS = 5 * 60 * 1000
let trafficTimer: ReturnType<typeof setInterval> | null = null
let clientsTimer: ReturnType<typeof setInterval> | null = null

const nginxHttpPort = ref(80)
const nginxHttpsPort = ref(443)

function defaultListenPort(httpsEnabled: boolean) {
  return httpsEnabled ? nginxHttpsPort.value : nginxHttpPort.value
}

const statusOptions = [
  { label: '运行中', value: 'enabled' },
  { label: '已停止', value: 'disabled' },
]
const httpsOptions = [
  { label: 'HTTPS', value: 'on' },
  { label: 'HTTP', value: 'off' },
]

const defaultSecurityForm = () => ({
  ip_blacklist_text: '',
  ip_blacklist_mode: false,
  ip_whitelist_text: '',
  ip_whitelist_mode: false,
  china_only: false,
  basic_auth_enabled: false,
  basic_auth_username: '',
  basic_auth_password: '',
  rate_limit_enabled: false,
  rate_limit_rate: 10,
  rate_limit_burst: 20,
  conn_limit_enabled: false,
  conn_limit_max: 20,
  proxy_ssl_verify_off: false,
  proxy_host_upstream: false,
  tls_min_13_only: false,
  security_headers: false,
})

const form = reactive({
  listen_port: 80,
  listen_ipv4: true,
  listen_ipv6: false,
  hostsText: '',
  upstream: '',
  https_enabled: true,
  http_redirect: true,
  enabled: true,
  name: '',
  ...defaultSecurityForm(),
})

const tableWrapRef = ref<HTMLElement | null>(null)
const entryStackRef = ref<HTMLElement | null>(null)
const collapsedEntryKeys = ref<Set<string>>(new Set())
const formEntryId = ref<number | null>(null)
const bindEntryListen = computed(() => formEntryId.value != null)
const reordering = ref(false)
const togglingRuleId = ref<number | null>(null)

const enabledCount = computed(() => rules.value.filter((r) => r.enabled).length)
const disabledCount = computed(() => rules.value.length - enabledCount.value)

const trafficTotals = computed(() => {
  let upload = 0
  let download = 0
  let uploadRate = 0
  let downloadRate = 0
  let connections = 0
  for (const stats of Object.values(trafficByRule.value)) {
    upload += stats.upload_total
    download += stats.download_total
    uploadRate += stats.upload_rate
    downloadRate += stats.download_rate
    connections += stats.connections
  }
  return { upload, download, uploadRate, downloadRate, connections }
})

const showEntryGroups = computed(
  () => !search.value.trim() && !statusFilter.value && !httpsFilter.value && rules.value.length > 0,
)

const canReorder = computed(
  () => !search.value.trim() && !statusFilter.value && !httpsFilter.value && rules.value.length > 1,
)

const canReorderGlobally = computed(() => canReorder.value && !showEntryGroups.value)
const canReorderInGroups = computed(() => canReorder.value && showEntryGroups.value)
const canReorderEntries = computed(
  () =>
    canReorderInGroups.value &&
    entries.value.length > 0 &&
    entryGroups.value.length > 1 &&
    entryGroups.value.every((group) => group.entryId != null) &&
    entryGroups.value.length === entries.value.length,
)

const filteredRules = computed(() =>
  rules.value.filter((rule) => {
    const q = search.value.toLowerCase()
    const hostText = ruleHosts(rule).join(' ').toLowerCase()
    const name = (rule.name ?? '').toLowerCase()
    if (q && !hostText.includes(q) && !rule.upstream.toLowerCase().includes(q) && !name.includes(q)) return false
    if (statusFilter.value === 'enabled' && !rule.enabled) return false
    if (statusFilter.value === 'disabled' && rule.enabled) return false
    if (httpsFilter.value === 'on' && !rule.https_enabled) return false
    if (httpsFilter.value === 'off' && rule.https_enabled) return false
    return true
  }),
)

const entryGroups = computed(() => buildProxyEntryGroups(rules.value, entries.value))

function isEntryCollapsed(key: string): boolean {
  return collapsedEntryKeys.value.has(key)
}

function toggleEntryCollapsed(key: string) {
  const next = new Set(collapsedEntryKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsedEntryKeys.value = next
}

function entryGroupTraffic(group: ProxyEntryGroup) {
  let upload = 0
  let download = 0
  let uploadRate = 0
  let downloadRate = 0
  let connections = 0
  for (const rule of group.rules) {
    const stats = trafficByRule.value[rule.id]
    if (!stats) continue
    upload += stats.upload_total
    download += stats.download_total
    uploadRate += stats.upload_rate
    downloadRate += stats.download_rate
    connections += stats.connections
  }
  return { upload, download, uploadRate, downloadRate, connections }
}

const tableRules = computed(() => (canReorderGlobally.value ? rules.value : filteredRules.value))

const activeSecurityFeatures = computed(() => {
  const tags: string[] = []
  if (form.ip_blacklist_mode) tags.push('黑名单')
  if (form.ip_whitelist_mode) tags.push('白名单')
  if (form.china_only) tags.push('大陆 IP')
  if (form.basic_auth_enabled) tags.push('Auth')
  if (form.rate_limit_enabled) tags.push('限流')
  if (form.conn_limit_enabled) tags.push('连接限制')
  if (form.proxy_ssl_verify_off) tags.push('跳过 TLS 校验')
  if (form.proxy_host_upstream) tags.push('目标 Host')
  if (form.tls_min_13_only) tags.push('TLS 1.3')
  if (form.security_headers) tags.push('响应头')
  return tags
})

function syncSecurityExpanded() {
  const expanded = new Set<string>()
  if (
    form.ip_blacklist_mode ||
    form.ip_whitelist_mode ||
    form.china_only
  ) {
    expanded.add('ip')
  }
  if (form.basic_auth_enabled) expanded.add('auth')
  if (form.rate_limit_enabled || form.conn_limit_enabled) expanded.add('traffic')
  if (
    form.proxy_ssl_verify_off ||
    form.proxy_host_upstream ||
    form.tls_min_13_only ||
    form.security_headers
  ) {
    expanded.add('advanced')
  }
  securityExpanded.value = expanded.size > 0 ? [...expanded] : ['ip']
}

const selectedRule = computed(() => rules.value.find((r) => r.id === selectedRuleId.value) ?? null)
const selectedTraffic = computed(() =>
  selectedRule.value ? trafficByRule.value[selectedRule.value.id] : undefined,
)
const detailNginxText = computed(() =>
  nginxServerMode.value === 'custom' ? nginxDraft.value : nginxGenerated.value,
)

function securityFeatureLabels(rule: ProxyRule): string[] {
  const sec = rule.security ?? {}
  const tags: string[] = []
  if ((sec.ip_blacklist?.length ?? 0) > 0) tags.push('黑名单')
  if (sec.ip_whitelist_mode) tags.push('白名单')
  if (sec.china_only) tags.push('大陆 IP')
  if (sec.basic_auth?.enabled) tags.push('Auth')
  if (sec.rate_limit?.enabled) tags.push('限流')
  if (sec.conn_limit?.enabled) tags.push('连接限制')
  if (sec.proxy_ssl_verify_off) tags.push('跳过 TLS 校验')
  if (sec.proxy_host_upstream) tags.push('目标 Host')
  if (sec.tls_min_13_only) tags.push('TLS 1.3')
  if (sec.security_headers) tags.push('响应头')
  return tags
}

const selectedSecurityFeatures = computed(() =>
  selectedRule.value ? securityFeatureLabels(selectedRule.value) : [],
)

const trafficChart = computed(() => {
  const now = Date.now()
  const labels: string[] = []
  const upload: number[] = []
  const download: number[] = []
  for (let i = 5; i >= 0; i--) {
    const bucketEnd = now - i * 60 * 1000
    const bucketStart = bucketEnd - 60 * 1000
    const points = rateHistory.value.filter((p) => p.at > bucketStart && p.at <= bucketEnd)
    const avg = (key: 'upload' | 'download') =>
      points.length ? points.reduce((sum, p) => sum + p[key], 0) / points.length : 0
    upload.push(avg('upload'))
    download.push(avg('download'))
    const d = new Date(bucketEnd)
    labels.push(
      `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`,
    )
  }
  return { labels, upload, download }
})

function openCreateWithEntry(listen: ProxyEntryListen, entryId?: number) {
  editing.value = null
  formTab.value = 'basic'
  resetForm()
  applyEntryListen(form, listen)
  formEntryId.value = entryId ?? null
  securityExpanded.value = ['ip']
  showModal.value = true
}


function openEntryEdit(group: ProxyEntryGroup) {
  if (!group.entryId) return
  editingEntryId.value = group.entryId
  Object.assign(entryForm, {
    name: group.name,
    listen_port: group.listen.listen_port,
    listen_ipv4: group.listen.listen_ipv4,
    listen_ipv6: group.listen.listen_ipv6,
    https_enabled: group.listen.https_enabled,
    http_redirect: group.listen.http_redirect,
  })
  showEntryModal.value = true
}

async function saveEntry() {
  if (!editingEntryId.value) return
  if (!entryForm.listen_ipv4 && !entryForm.listen_ipv6) {
    message.error('至少需要启用 IPv4 或 IPv6 监听')
    return
  }
  savingEntry.value = true
  try {
    await api.updateProxyEntry(editingEntryId.value, {
      name: entryForm.name.trim(),
      listen_port: entryForm.listen_port,
      listen_ipv4: entryForm.listen_ipv4,
      listen_ipv6: entryForm.listen_ipv6,
      https_enabled: entryForm.https_enabled,
      http_redirect: entryForm.http_redirect,
    })
    showEntryModal.value = false
    message.success('入口已保存')
    await load()
  } catch (error) {
    const msg = error instanceof Error ? error.message : '保存失败'
    if (msg.startsWith('入口已保存')) {
      message.warning(msg)
      showEntryModal.value = false
      await load()
    } else {
      message.error(msg)
    }
  } finally {
    savingEntry.value = false
  }
}

function duplicateEntryHosts(rule: ProxyRule): string[] {
  return ruleHosts(rule).map((host) => {
    const idx = host.lastIndexOf(':')
    if (idx > 0 && /^\d+$/.test(host.slice(idx + 1))) {
      return `${host.slice(0, idx)}-copy${host.slice(idx)}`
    }
    return `${host}-copy`
  })
}

async function duplicateEntry(group: ProxyEntryGroup) {
  if (!group.entryId) return
  try {
    const entry = await api.createProxyEntry({
      name: duplicateLabel(group.name || entryGroupDisplayName(group)),
      listen_port: group.listen.listen_port,
      listen_ipv4: group.listen.listen_ipv4,
      listen_ipv6: group.listen.listen_ipv6,
      https_enabled: group.listen.https_enabled,
      http_redirect: group.listen.http_redirect,
    })
    for (const rule of group.rules) {
      await api.createProxy({
        entry_id: entry.id,
        upstream: rule.upstream,
        hosts: duplicateEntryHosts(rule),
        enabled: rule.enabled,
        name: duplicateName(rule),
        security: rule.security,
      })
    }
    message.success('入口已复制，域名已追加 -copy 后缀')
    await load()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '复制失败')
  }
}

function confirmDeleteEntry(group: ProxyEntryGroup) {
  if (!group.entryId) return
  const title = entryGroupDisplayName(group)
  dialog.warning({
    title: `确定删除入口「${title}」？`,
    content: `将删除该入口下的 ${group.rules.length} 条规则，相关域名将停止反向代理。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: () =>
      api
        .deleteProxyEntry(group.entryId!)
        .then(async () => {
          message.success('入口已删除')
          await load()
        })
        .catch(async (error: unknown) => {
          const msg = error instanceof Error ? error.message : '删除失败'
          if (msg.startsWith('入口已删除')) {
            message.warning(msg)
            await load()
          } else {
            message.error(msg)
            return false
          }
        }),
  })
}

function suggestDiscoveryHostname(service: DiscoveredService): string {
  const slug = (service.platform || service.name)
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return slug ? `${slug}.example.com` : ''
}

function buildDiscoveryRows(items: DiscoveredService[]): DiscoveryRow[] {
  return items
    .filter((service) => service.detected)
    .map((service, idx) => ({
      id: `${service.platform}-${service.port}-${idx}`,
      selected: true,
      service,
      ruleName: service.name,
      hostname: suggestDiscoveryHostname(service),
    }))
}

function openDiscoveryModal(group?: ProxyEntryGroup) {
  discoveryTarget.value = group ?? null
  discoveryHost.value = '127.0.0.1'
  discoveryRows.value = []
  showDiscoveryModal.value = true
  void runDiscoveryScan()
}

async function runDiscoveryScan() {
  if (discoveryLoading.value) return

  const host = discoveryHost.value.trim() || '127.0.0.1'
  discoveryScanHost.value = host
  discoveryLoading.value = true
  const loadingMsg = message.loading(`正在扫描 ${host} …`, { duration: 0 })

  try {
    const items = asList(await api.scanDiscovery(host))
    discoveryRows.value = buildDiscoveryRows(items)
    loadingMsg.destroy()
    if (discoveryRows.value.length === 0) {
      message.info(`扫描完成：${host} 未发现可添加的 HTTP 服务`)
    } else {
      message.success(`扫描完成：发现 ${discoveryRows.value.length} 个 HTTP 服务`)
    }
  } catch (error) {
    loadingMsg.destroy()
    message.error(error instanceof Error ? error.message : '扫描失败')
  } finally {
    discoveryLoading.value = false
  }
}

async function submitDiscovery() {
  const selected = discoveryRows.value.filter(
    (row) => row.selected && row.hostname.trim(),
  )
  if (selected.length === 0) {
    message.warning('请至少选择一项并填写前端域名')
    return
  }

  discoveryAdding.value = true
  try {
    const group = discoveryTarget.value
    for (const row of selected) {
      const payload: ProxySavePayload = {
        upstream: row.service.upstream,
        hosts: [row.hostname.trim()],
        name: row.ruleName.trim() || row.service.name,
        enabled: true,
      }
      if (group?.entryId) {
        payload.entry_id = group.entryId
      } else if (group) {
        payload.listen_port = group.listen.listen_port
        payload.listen_ipv4 = group.listen.listen_ipv4
        payload.listen_ipv6 = group.listen.listen_ipv6
        payload.https_enabled = group.listen.https_enabled
        payload.http_redirect = group.listen.http_redirect
      }
      await api.createProxy(payload)
    }
    message.success(`已添加 ${selected.length} 条规则`)
    showDiscoveryModal.value = false
    await load()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '添加失败')
  } finally {
    discoveryAdding.value = false
  }
}

watch(entryGroups, (groups) => {
  if (!showEntryGroups.value) return
  const validKeys = new Set(groups.map((group) => group.key))
  const next = new Set([...collapsedEntryKeys.value].filter((key) => validKeys.has(key)))
  collapsedEntryKeys.value = next
})

function clearRateHistory() {
  rateHistory.value = []
}

function recordRateSample(ruleId: number) {
  const stats = trafficByRule.value[ruleId]
  if (!stats) return
  const now = Date.now()
  rateHistory.value.push({
    at: now,
    upload: stats.upload_rate,
    download: stats.download_rate,
  })
  const cutoff = now - RATE_HISTORY_MS
  rateHistory.value = rateHistory.value.filter((p) => p.at >= cutoff)
}

watch(filteredRules, (list) => {
  if (list.length === 0) {
    selectedRuleId.value = null
    showDetailPanel.value = false
    return
  }
  if (selectedRuleId.value && !list.some((r) => r.id === selectedRuleId.value)) {
    selectedRuleId.value = null
    showDetailPanel.value = false
  }
})

function openDetail(rule: ProxyRule, tab: 'overview' | 'logs' | 'nginx' = 'overview') {
  selectedRuleId.value = rule.id
  detailTab.value = tab
  showDetailPanel.value = true
  clearRateHistory()
  recordRateSample(rule.id)
  if (tab === 'nginx') {
    void loadRuleNginxPreview()
  }
}

function confirmDiscardNginxDraft(): Promise<boolean> {
  if (!nginxDirty.value || nginxEditMode.value !== 'custom') {
    return Promise.resolve(true)
  }
  return new Promise((resolve) => {
    dialog.warning({
      title: '未保存的 Nginx 配置',
      content: '当前手动编辑尚未保存，确定放弃更改吗？',
      positiveText: '放弃更改',
      negativeText: '继续编辑',
      onPositiveClick: () => resolve(true),
      onNegativeClick: () => resolve(false),
      onClose: () => resolve(false),
    })
  })
}

async function switchDetailTab(tab: 'overview' | 'logs' | 'nginx') {
  if (tab === detailTab.value) return
  detailTab.value = tab
  if (tab === 'nginx') {
    await loadRuleNginxPreview()
  }
}

async function switchFormTab(tab: 'basic' | 'security' | 'nginx') {
  if (tab === formTab.value) return
  if (formTab.value === 'nginx') {
    const ok = await confirmDiscardNginxDraft()
    if (!ok) return
  }
  formTab.value = tab
  if (tab === 'nginx' && editing.value) {
    nginxDirty.value = false
    await loadRuleNginxForEdit(editing.value.id)
  }
}

function closeModal() {
  if (formTab.value === 'nginx' && nginxDirty.value && nginxEditMode.value === 'custom') {
    void confirmDiscardNginxDraft().then((ok) => {
      if (ok) {
        formEntryId.value = null
        showModal.value = false
      }
    })
    return
  }
  formEntryId.value = null
  showModal.value = false
}

async function loadRuleNginxPreview() {
  const rule = selectedRule.value
  if (!rule) return
  nginxLoading.value = true
  try {
    const view = await api.getProxyNginx(rule.id)
    nginxServerMode.value = view.mode
    nginxGenerated.value = view.generated
    nginxEnabled.value = view.enabled
    nginxDraft.value = view.mode === 'custom' ? view.content : view.generated
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载 Nginx 配置失败')
  } finally {
    nginxLoading.value = false
  }
}

async function loadRuleNginxForEdit(ruleId: number) {
  nginxLoading.value = true
  try {
    const view = await api.getProxyNginx(ruleId)
    nginxServerMode.value = view.mode
    nginxEditMode.value = view.mode
    nginxGenerated.value = view.generated
    nginxEnabled.value = view.enabled
    nginxBackups.value = view.backups ?? []
    if (view.mode === 'auto') {
      nginxDraft.value = view.generated
      nginxDirty.value = false
    } else if (!nginxDirty.value) {
      nginxDraft.value = view.content
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载 Nginx 配置失败')
  } finally {
    nginxLoading.value = false
  }
}

function scheduleNginxRefresh() {
  if (detailTab.value !== 'nginx' || showDetailPanel.value === false) return
  if (nginxRefreshTimer) clearTimeout(nginxRefreshTimer)
  nginxRefreshTimer = setTimeout(() => {
    void loadRuleNginxPreview()
  }, 300)
}

function setNginxEditMode(mode: 'auto' | 'custom') {
  if (mode === nginxEditMode.value) return
  if (mode === 'custom' && nginxEditMode.value === 'auto') {
    nginxDraft.value = nginxGenerated.value
    nginxDirty.value = nginxServerMode.value !== 'custom'
  }
  if (mode === 'auto') {
    nginxDraft.value = nginxGenerated.value
    nginxDirty.value = false
  }
  nginxEditMode.value = mode
}

function onNginxDraftInput() {
  if (nginxEditMode.value === 'custom') {
    nginxDirty.value = true
  }
}

async function saveRuleNginx() {
  const ruleId = editing.value?.id
  if (!ruleId) return
  nginxSaving.value = true
  try {
    const view = await api.saveProxyNginx(ruleId, {
      mode: 'custom',
      content: nginxDraft.value,
    })
    nginxServerMode.value = view.mode
    nginxEditMode.value = view.mode
    nginxGenerated.value = view.generated
    nginxDraft.value = view.content
    nginxBackups.value = view.backups ?? []
    nginxDirty.value = false
    message.success('Nginx 配置已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    nginxSaving.value = false
  }
}

async function rollbackRuleNginx(backup?: string) {
  const ruleId = editing.value?.id
  if (!ruleId) return
  nginxSaving.value = true
  try {
    const view = await api.rollbackProxyNginx(ruleId, backup)
    nginxServerMode.value = view.mode
    nginxEditMode.value = view.mode
    nginxGenerated.value = view.generated
    nginxDraft.value = view.content
    nginxBackups.value = view.backups ?? []
    nginxDirty.value = false
    message.success('已回滚到上一版本')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '回滚失败')
  } finally {
    nginxSaving.value = false
  }
}

function resetRuleNginxAuto() {
  dialog.warning({
    title: '恢复自动生成',
    content: '将切回自动生成模式，手动保存的配置文件仍保留在备份中。',
    positiveText: '确认',
    negativeText: '取消',
    onPositiveClick: async () => {
      const ruleId = editing.value?.id
      if (!ruleId) return
      nginxSaving.value = true
      try {
        const view = await api.saveProxyNginx(ruleId, { mode: 'auto' })
        nginxServerMode.value = view.mode
        nginxEditMode.value = view.mode
        nginxGenerated.value = view.generated
        nginxDraft.value = view.generated
        nginxBackups.value = view.backups ?? []
        nginxDirty.value = false
        message.success('已恢复自动生成')
      } catch (error) {
        message.error(error instanceof Error ? error.message : '操作失败')
      } finally {
        nginxSaving.value = false
      }
    },
  })
}

function closeDetail() {
  logFullscreen.value = false
  showDetailPanel.value = false
  detailTab.value = 'overview'
  clearRateHistory()
}

watch(selectedRule, () => scheduleNginxRefresh(), { deep: true })

watch([showDetailPanel, selectedRuleId, detailTab], async ([visible, id, tab], [, , prevTab]) => {
  if (visible && id && tab === 'nginx' && prevTab !== 'nginx') {
    await loadRuleNginxPreview()
  }
})

watch(selectedRuleId, (id, prev) => {
  if (id !== prev) clearRateHistory()
})

function ruleHosts(rule: ProxyRule): string[] {
  const hosts = asList(rule.hosts)
  if (hosts.length > 0) {
    return hosts.map((host) => {
      if (host.listen_port && host.listen_port !== rule.listen_port) {
        return `${host.hostname}:${host.listen_port}`
      }
      return host.hostname
    })
  }
  return rule.domain ? [rule.domain] : []
}

function primaryHost(rule: ProxyRule): string {
  const hosts = ruleHosts(rule)
  return hosts[0] ?? `规则 #${rule.id}`
}

function ruleName(rule: ProxyRule): string {
  const name = rule.name?.trim()
  return name || primaryHost(rule)
}

function duplicateName(rule: ProxyRule): string {
  const base = rule.name?.trim() || primaryHost(rule)
  return duplicateLabel(base)
}

function duplicateLabel(base: string): string {
  const suffix = '-复制'
  const maxBase = 100 - suffix.length
  const trimmedBase = base.length > maxBase ? base.slice(0, maxBase) : base
  return `${trimmedBase}${suffix}`
}

function hostsToText(rule: ProxyRule): string {
  return ruleHosts(rule).join('\n')
}

function ruleTitle(rule: ProxyRule): string {
  return ruleName(rule)
}

function listenLabel(rule: ProxyRule): string {
  const stacks = []
  if (rule.listen_ipv4) stacks.push('IPv4')
  if (rule.listen_ipv6) stacks.push('IPv6')
  const stack = stacks.length > 0 ? stacks.join('/') : '-'
  return `${rule.listen_port} (${stack})`
}

function hostAccessUrl(rule: ProxyRule, hostPart: string): string {
  const scheme = rule.https_enabled ? 'https' : 'http'
  const hasExplicitPort = hostPart.startsWith('[')
    ? /]:\d+$/.test(hostPart)
    : /^[^:[\]]+:\d+$/.test(hostPart)
  if (hasExplicitPort) {
    return `${scheme}://${hostPart}`
  }

  const defaultPort = rule.https_enabled ? nginxHttpsPort.value : nginxHttpPort.value
  if (rule.listen_port !== defaultPort) {
    return `${scheme}://${hostPart}:${rule.listen_port}`
  }
  return `${scheme}://${hostPart}`
}

function hostAccessUrlWithoutListenPort(rule: ProxyRule, hostPart: string): string {
  const scheme = rule.https_enabled ? 'https' : 'http'
  const hasExplicitPort = hostPart.startsWith('[')
    ? /]:\d+$/.test(hostPart)
    : /^[^:[\]]+:\d+$/.test(hostPart)
  if (hasExplicitPort) {
    return `${scheme}://${hostPart}`
  }
  return `${scheme}://${hostPart}`
}

async function copyAccessUrl(url: string) {
  try {
    await navigator.clipboard.writeText(url)
    message.success('链接已复制')
  } catch {
    message.error('复制失败')
  }
}

function renderLinkAction(icon: Component, title: string, onClick: () => void): VNode {
  return h(
    'button',
    {
      type: 'button',
      class: 'domain-cell__link-action',
      title,
      onClick: (e: Event) => {
        e.stopPropagation()
        onClick()
      },
    },
    [h(NIcon, { component: icon, size: 14 })],
  )
}

function renderAccessLinkRow(href: string, displayText?: string): VNode {
  const label = displayText ?? href
  return h(
    'div',
    { class: 'domain-cell__link-row', onClick: (e: Event) => e.stopPropagation() },
    [
      h(
        'a',
        {
          class: 'domain-cell__link',
          href,
          target: '_blank',
          rel: 'noopener noreferrer',
          title: href,
          onClick: (e: Event) => e.stopPropagation(),
        },
        label,
      ),
      renderLinkAction(CopyOutline, '复制链接', () => copyAccessUrl(href)),
      renderLinkAction(OpenOutline, '新窗口打开', () => window.open(href, '_blank', 'noopener,noreferrer')),
    ],
  )
}

interface DomainLinkOptions {
  hideListenPort?: boolean
  showWhenSingleHost?: boolean
}

function accessLinkPairs(rule: ProxyRule, options: DomainLinkOptions = {}) {
  return ruleHosts(rule).map((host) => ({
    href: hostAccessUrl(rule, host),
    display: options.hideListenPort ? hostAccessUrlWithoutListenPort(rule, host) : hostAccessUrl(rule, host),
  }))
}

function renderDomainAccessLinks(rule: ProxyRule, options: DomainLinkOptions = {}): VNode | null {
  const hosts = ruleHosts(rule)
  if (hosts.length === 0) return null

  const showLinks = options.showWhenSingleHost || hosts.length > 1 || !!rule.name?.trim()
  if (!showLinks) return null

  const pairs = accessLinkPairs(rule, options)
  const maxInline = 2
  const inline = pairs.slice(0, maxInline)
  const rest = pairs.slice(maxInline)

  const children: VNode[] = inline.map((pair) => renderAccessLinkRow(pair.href, pair.display))
  if (rest.length > 0) {
    children.push(
      h(
        NPopover,
        { trigger: 'click', placement: 'bottom-start', showArrow: false },
        {
          trigger: () =>
            h(
              'button',
              {
                type: 'button',
                class: 'domain-cell__more',
                onClick: (e: Event) => e.stopPropagation(),
              },
              `还有 ${rest.length} 个域名`,
            ),
          default: () =>
            h(
              'div',
              { class: 'domain-cell__popover-links' },
              rest.map((pair) => renderAccessLinkRow(pair.href, pair.display)),
            ),
        },
      ),
    )
  }

  return h('div', { class: 'domain-cell__links' }, children)
}

function renderEntryGroupRuleNameCell(rule: ProxyRule, group: ProxyEntryGroup): VNode {
  const label = ruleName(rule)
  const headerName = entryGroupDisplayName(group)
  const singleRuleWithHeaderName = group.rules.length === 1 && !!label && label === headerName

  const children: VNode[] = []
  if (label && !singleRuleWithHeaderName) {
    children.push(h('div', { class: 'domain-cell__main' }, label))
  }
  const links = renderDomainAccessLinks(rule, {
    hideListenPort: true,
    showWhenSingleHost: singleRuleWithHeaderName || !label,
  })
  if (links) children.push(links)
  return h('div', { class: 'domain-cell' }, children)
}

function parseHostsText(text: string): string[] {
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

function parseHostname(line: string): string {
  const raw = line.trim()
  if (!raw) return ''
  if (raw.includes(':') && !raw.includes(']') && raw.split(':').length === 2) {
    return raw.split(':')[0].toLowerCase()
  }
  return raw.toLowerCase()
}

function effectiveListenPort(line: string, ruleListenPort: number): number {
  const raw = line.trim()
  if (raw.includes(':') && !raw.includes(']') && raw.split(':').length === 2) {
    const port = Number.parseInt(raw.split(':')[1], 10)
    if (port > 0) return port
  }
  return ruleListenPort
}

function hostBindingKey(hostname: string, port: number): string {
  return `${hostname.toLowerCase()}:${port}`
}

function findHostConflict(
  hosts: string[],
  ruleListenPort: number,
  excludeRuleId?: number,
): { host: string; port: number; rule: ProxyRule } | null {
  const wanted = new Set(
    hosts.map((line) => hostBindingKey(parseHostname(line), effectiveListenPort(line, ruleListenPort))).filter((k) => !k.startsWith(':')),
  )
  if (wanted.size === 0) return null
  for (const rule of rules.value) {
    if (excludeRuleId && rule.id === excludeRuleId) continue
    for (const host of asList(rule.hosts)) {
      const hostname = host.hostname.toLowerCase()
      const port = host.listen_port ?? rule.listen_port
      if (wanted.has(hostBindingKey(hostname, port))) {
        return { host: hostname, port, rule }
      }
    }
  }
  return null
}

function renderProtocol(row: ProxyRule): VNode {
  const tags: VNode[] = []
  if (row.https_enabled) {
    tags.push(h(NTag, { size: 'small', type: 'info', bordered: false, round: true }, () => 'HTTPS'))
  } else {
    tags.push(h(NTag, { size: 'small', type: 'success', bordered: false, round: true }, () => 'HTTP'))
  }
  return h('div', { class: 'proto-tags' }, tags)
}

function rowProps(row: ProxyRule) {
  return {
    class: showDetailPanel.value && selectedRuleId.value === row.id ? 'proxy-row--active' : '',
  }
}

const columns = computed<DataTableColumns<ProxyRule>>(() => {
  const cols: DataTableColumns<ProxyRule> = []

  if (canReorderGlobally.value) {
    cols.push({
      title: '',
      key: 'sort',
      width: 40,
      render: () =>
        h('span', { class: 'proxy-drag-handle', title: '拖动排序' }, [
          h(NIcon, { component: ReorderThreeOutline, size: 16 }),
        ]),
    })
  }

  cols.push({
    title: '名称',
    key: 'name',
    minWidth: 260,
    render: (row) =>
      h('div', { class: 'domain-cell' }, [
        h('div', { class: 'domain-cell__main' }, ruleName(row)),
        renderDomainAccessLinks(row),
      ]),
  })

  cols.push(
  {
    title: '监听端口',
    key: 'listen_port',
    width: 128,
    render: (row) => h('span', { class: 'mono text-secondary' }, listenLabel(row)),
  },
  {
    title: '目标地址',
    key: 'upstream',
    minWidth: 180,
    render: (row) => h('span', { class: 'mono text-secondary' }, row.upstream),
  },
  {
    title: '协议',
    key: 'https_enabled',
    width: 88,
    render: (row) => renderProtocol(row),
  },
  {
    title: '安全',
    key: 'security',
    width: 120,
    render: (row) => {
      const tags = securityTags(row)
      if (tags.length === 0) return h('span', { class: 'text-muted' }, '—')
      return h(
        'div',
        { class: 'proxy-security-tags' },
        tags.map((tag) => h(NTag, { size: 'small', bordered: false, round: true }, { default: () => tag })),
      )
    },
  },
  {
    title: '状态',
    key: 'enabled',
    width: 108,
    render: (row) =>
      h(
        'div',
        {
          class: 'proxy-enable-cell',
          onClick: (e: Event) => e.stopPropagation(),
        },
        [
          h(
            NSwitch,
            {
              value: row.enabled,
              size: 'small',
              loading: togglingRuleId.value === row.id,
              onUpdateValue: (enabled: boolean) => toggleRuleEnabled(row, enabled),
            },
            {
              checked: () => '启用',
              unchecked: () => '停用',
            },
          ),
        ],
      ),
  },
  {
    title: '当前连接',
    key: 'connections',
    width: 88,
    render: (row) => h('span', { class: 'mono' }, String(trafficByRule.value[row.id]?.connections ?? 0)),
  },
  {
    title: '当前上传',
    key: 'upload_rate',
    width: 96,
    render: (row) => h('span', { class: 'mono text-secondary' }, formatRate(trafficByRule.value[row.id]?.upload_rate ?? 0)),
  },
  {
    title: '当前下载',
    key: 'download_rate',
    width: 96,
    render: (row) => h('span', { class: 'mono text-secondary' }, formatRate(trafficByRule.value[row.id]?.download_rate ?? 0)),
  },
  {
    title: '总上传',
    key: 'upload_total',
    width: 96,
    render: (row) => h('span', { class: 'mono' }, formatBytes(trafficByRule.value[row.id]?.upload_total ?? 0)),
  },
  {
    title: '总下载',
    key: 'download_total',
    width: 96,
    render: (row) => h('span', { class: 'mono' }, formatBytes(trafficByRule.value[row.id]?.download_total ?? 0)),
  },
    {
      title: '操作',
      key: 'actions',
      width: 208,
      fixed: 'right',
      render: (row) =>
        renderTableRowActions([
          { label: '复制', onClick: () => openDuplicate(row) },
          { label: '详情', onClick: () => openDetail(row) },
          { label: '编辑', type: 'primary', onClick: () => openEdit(row) },
          { label: '删除', type: 'error', onClick: () => confirmDelete(row) },
        ]),
    },
  )

  return cols
})

function entryGroupColumnsFor(group: ProxyEntryGroup): DataTableColumns<ProxyRule> {
  const cols = columns.value
    .filter((col) => {
      if (!('key' in col)) return true
      const key = String(col.key ?? '')
      return key !== 'listen_port' && key !== 'https_enabled'
    })
    .map((col) => {
      if ('key' in col && col.key === 'name') {
        return {
          ...col,
          render: (row: ProxyRule) => renderEntryGroupRuleNameCell(row, group),
        }
      }
      return col
    })
  if (!canReorderInGroups.value) return cols

  return [
    {
      title: '',
      key: 'sort',
      width: 40,
      render: () =>
        h('span', { class: 'proxy-drag-handle', title: '拖动排序' }, [
          h(NIcon, { component: ReorderThreeOutline, size: 16 }),
        ]),
    },
    ...cols,
  ]
}

function isLogAtBottom(el: HTMLElement): boolean {
  return el.scrollHeight - el.scrollTop - el.clientHeight <= LOG_SCROLL_BOTTOM_THRESHOLD
}

function onLogBoxScroll(el: HTMLElement) {
  logStickToBottom.value = isLogAtBottom(el)
}

function scrollLogToBottom(force = false) {
  if (!force && !logStickToBottom.value) return
  requestAnimationFrame(() => {
    logBox.value?.scrollToBottom()
    if (logFullscreen.value) logBoxFullscreen.value?.scrollToBottom()
  })
}

async function openLogFullscreen() {
  logFullscreen.value = true
  await nextTick()
  scrollLogToBottom(true)
}

function onLogFullscreenKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') logFullscreen.value = false
}

function stopLogStream() {
  logEventSource?.close()
  logEventSource = null
}

function startLogStream() {
  const rule = selectedRule.value
  if (!rule || logEventSource) return
  logEventSource = new EventSource(`/api/proxies/${rule.id}/logs/stream?tail=100`, {
    withCredentials: true,
  })
  logEventSource.addEventListener('log', (event) => {
    logLines.value.push(event.data)
    if (logLines.value.length > 500) logLines.value = logLines.value.slice(-400)
    scrollLogToBottom()
  })
  logEventSource.onerror = () => {
    message.warning('日志连接中断')
    stopLogStream()
  }
}

function clearLogLines() {
  logLines.value = []
  logStickToBottom.value = true
}

watch([showDetailPanel, selectedRuleId, detailTab], async ([visible, id, tab], [wasVisible, wasId]) => {
  stopLogStream()
  if (!visible || !id || tab !== 'logs') return
  if (!wasVisible || id !== wasId) {
    logLines.value = []
    logStickToBottom.value = true
  }
  startLogStream()
  await nextTick()
  scrollLogToBottom(true)
})

watch(
  () => logLines.value.length,
  async () => {
    if (detailTab.value !== 'logs' || !logStickToBottom.value) return
    await nextTick()
    scrollLogToBottom()
  },
)

watch(logFullscreen, (open) => {
  if (open) {
    document.addEventListener('keydown', onLogFullscreenKeydown)
    document.body.style.overflow = 'hidden'
    return
  }
  document.removeEventListener('keydown', onLogFullscreenKeydown)
  document.body.style.overflow = ''
})

async function refreshTraffic() {
  try {
    const rows = asList(await api.getProxyTraffic())
    const next: Record<number, ProxyTraffic> = {}
    for (const row of rows) {
      next[row.rule_id] = row
    }
    trafficByRule.value = next
    if (showDetailPanel.value && selectedRuleId.value && detailTab.value === 'overview') {
      recordRateSample(selectedRuleId.value)
    }
  } catch {
    // ignore polling errors
  }
}

async function refreshAll() {
  await Promise.all([load(), refreshTraffic()])
}

function startTrafficPoll() {
  stopTrafficPoll()
  trafficTimer = setInterval(refreshTraffic, 2000)
}

function stopTrafficPoll() {
  if (trafficTimer) {
    clearInterval(trafficTimer)
    trafficTimer = null
  }
}

async function loadClients() {
  const rule = selectedRule.value
  if (!rule) return
  clientsLoading.value = true
  try {
    clientRows.value = asList(await api.getProxyClients(rule.id))
  } catch (error) {
    message.error(error instanceof Error ? error.message : '读取连接失败')
  } finally {
    clientsLoading.value = false
  }
}

function startClientsPoll() {
  stopClientsPoll()
  void loadClients()
  clientsTimer = setInterval(loadClients, 5000)
}

function stopClientsPoll() {
  if (clientsTimer) {
    clearInterval(clientsTimer)
    clientsTimer = null
  }
}

watch([showDetailPanel, selectedRuleId, detailTab], ([visible, id, tab]) => {
  if (!visible || !id || tab !== 'overview') {
    stopClientsPoll()
    return
  }
  startClientsPoll()
})

async function loadDefaults() {
  try {
    const { nginx_http_port, nginx_https_port } = await api.getVersion()
    if (nginx_http_port > 0) nginxHttpPort.value = nginx_http_port
    if (nginx_https_port > 0) nginxHttpsPort.value = nginx_https_port
  } catch {
    // keep local fallbacks
  }
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const [ruleRows, entryRows] = await Promise.all([
      api.listProxies(),
      api.listProxyEntries().catch(() => [] as ProxyEntry[]),
    ])
    rules.value = asList(ruleRows)
    entries.value = asList(entryRows)
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.listen_port = defaultListenPort(form.https_enabled)
  form.listen_ipv4 = true
  form.listen_ipv6 = false
  form.hostsText = ''
  form.upstream = ''
  form.https_enabled = true
  form.http_redirect = true
  form.enabled = true
  form.name = ''
  Object.assign(form, defaultSecurityForm())
}

function loadSecurityToForm(rule?: ProxyRule | null) {
  const sec = rule?.security ?? {}
  form.ip_blacklist_text = (sec.ip_blacklist ?? []).join('\n')
  form.ip_blacklist_mode = (sec.ip_blacklist?.length ?? 0) > 0
  form.ip_whitelist_text = (sec.ip_whitelist ?? []).join('\n')
  form.ip_whitelist_mode = sec.ip_whitelist_mode ?? false
  form.china_only = sec.china_only ?? false
  form.basic_auth_enabled = sec.basic_auth?.enabled ?? false
  form.basic_auth_username = sec.basic_auth?.username ?? ''
  form.basic_auth_password = ''
  form.rate_limit_enabled = sec.rate_limit?.enabled ?? false
  form.rate_limit_rate = sec.rate_limit?.rate ?? 10
  form.rate_limit_burst = sec.rate_limit?.burst ?? 20
  form.conn_limit_enabled = sec.conn_limit?.enabled ?? false
  form.conn_limit_max = sec.conn_limit?.max ?? 20
  form.proxy_ssl_verify_off = sec.proxy_ssl_verify_off ?? false
  form.proxy_host_upstream = sec.proxy_host_upstream ?? false
  form.tls_min_13_only = sec.tls_min_13_only ?? false
  form.security_headers = sec.security_headers ?? false
}

function buildSecurityPayload() {
  const payload: ProxySavePayload['security'] = {
    ip_blacklist_text: form.ip_blacklist_mode ? form.ip_blacklist_text : '',
    ip_whitelist_text: form.ip_whitelist_text,
    ip_whitelist_mode: form.ip_whitelist_mode,
    china_only: form.china_only,
    proxy_ssl_verify_off: form.proxy_ssl_verify_off,
    proxy_host_upstream: form.proxy_host_upstream,
    tls_min_13_only: form.tls_min_13_only,
    security_headers: form.security_headers,
    basic_auth: {
      enabled: form.basic_auth_enabled,
      username: form.basic_auth_username.trim(),
    },
    rate_limit: form.rate_limit_enabled
      ? { enabled: true, rate: form.rate_limit_rate, burst: form.rate_limit_burst }
      : { enabled: false },
    conn_limit: form.conn_limit_enabled
      ? { enabled: true, max: form.conn_limit_max }
      : { enabled: false },
  }
  if (form.basic_auth_password.trim()) {
    payload.basic_auth!.password = form.basic_auth_password
  }
  return payload
}

function applySecurityPreset() {
  form.https_enabled = true
  form.http_redirect = true
  form.rate_limit_enabled = true
  form.rate_limit_rate = 10
  form.rate_limit_burst = 20
  form.security_headers = true
  formTab.value = 'security'
  securityExpanded.value = ['traffic', 'advanced']
  message.success('已填入推荐配置：HTTPS、限流与安全响应头')
}

function securityTags(rule: ProxyRule): string[] {
  const sec = rule.security ?? {}
  const tags: string[] = []
  if (sec.basic_auth?.enabled) tags.push('Auth')
  if ((sec.ip_blacklist?.length ?? 0) > 0 || sec.ip_whitelist_mode) tags.push('IP')
  if (sec.china_only) tags.push('CN')
  if (sec.rate_limit?.enabled || sec.conn_limit?.enabled) tags.push('Limit')
  return tags
}

function buildPayload(): ProxySavePayload {
  const hosts = parseHostsText(form.hostsText)
  if (hosts.length === 0) {
    throw new Error('至少需要一个前端域名')
  }
  if (!form.listen_ipv4 && !form.listen_ipv6) {
    throw new Error('至少需要启用 IPv4 或 IPv6 监听')
  }
  return {
    upstream: form.upstream,
    ...(formEntryId.value
      ? { entry_id: formEntryId.value }
      : {
          listen_port: form.listen_port,
          listen_ipv4: form.listen_ipv4,
          listen_ipv6: form.listen_ipv6,
          https_enabled: form.https_enabled,
          http_redirect: form.http_redirect,
        }),
    hosts,
    enabled: form.enabled,
    name: form.name.trim(),
    security: buildSecurityPayload(),
  }
}

function openCreate() {
  editing.value = null
  formTab.value = 'basic'
  resetForm()
  formEntryId.value = null
  securityExpanded.value = ['ip']
  showModal.value = true
}

async function toggleRuleEnabled(row: ProxyRule, enabled: boolean) {
  if (togglingRuleId.value === row.id || row.enabled === enabled) return
  togglingRuleId.value = row.id
  const prev = row.enabled
  rules.value = rules.value.map((rule) => (rule.id === row.id ? { ...rule, enabled } : rule))
  try {
    const updated = await api.updateProxy(row.id, { enabled })
    rules.value = rules.value.map((rule) => (rule.id === updated.id ? updated : rule))
    message.success(enabled ? '已启用' : '已停用')
  } catch (error) {
    rules.value = rules.value.map((rule) => (rule.id === row.id ? { ...rule, enabled: prev } : rule))
    const msg = error instanceof Error ? error.message : '更新失败'
    if (msg.startsWith('规则已保存')) {
      message.warning(msg)
      await load()
    } else {
      message.error(msg)
    }
  } finally {
    togglingRuleId.value = null
  }
}

function findEntryGroupForRule(rule: ProxyRule): ProxyEntryGroup | undefined {
  return entryGroups.value.find((group) => group.rules.some((item) => item.id === rule.id))
}

function openDuplicate(rule: ProxyRule) {
  closeDetail()
  editing.value = null
  const parentGroup = findEntryGroupForRule(rule)
  formEntryId.value = rule.entry_id ?? parentGroup?.entryId ?? null

  if (parentGroup && formEntryId.value) {
    applyEntryListen(form, parentGroup.listen)
  } else {
    form.listen_port = rule.listen_port || defaultListenPort(rule.https_enabled)
    form.listen_ipv4 = rule.listen_ipv4 ?? true
    form.listen_ipv6 = rule.listen_ipv6 ?? false
    form.https_enabled = rule.https_enabled
    form.http_redirect = rule.http_redirect
  }

  Object.assign(form, {
    hostsText: duplicateEntryHosts(rule).join('\n'),
    upstream: rule.upstream,
    enabled: rule.enabled,
    name: duplicateName(rule),
  })
  loadSecurityToForm(rule)
  syncSecurityExpanded()
  formTab.value = 'basic'
  showModal.value = true
  message.info('已填入复制内容，域名已追加 -copy 后缀')
}

function openEdit(rule: ProxyRule, tab: 'basic' | 'security' | 'nginx' = 'basic') {
  closeDetail()
  editing.value = rule
  selectedRuleId.value = rule.id
  const parentGroup = findEntryGroupForRule(rule)
  formEntryId.value = rule.entry_id ?? parentGroup?.entryId ?? null

  if (parentGroup && formEntryId.value) {
    applyEntryListen(form, parentGroup.listen)
  } else {
    form.listen_port = rule.listen_port || defaultListenPort(rule.https_enabled)
    form.listen_ipv4 = rule.listen_ipv4 ?? true
    form.listen_ipv6 = rule.listen_ipv6 ?? false
    form.https_enabled = rule.https_enabled
    form.http_redirect = rule.http_redirect
  }

  Object.assign(form, {
    hostsText: hostsToText(rule),
    upstream: rule.upstream,
    enabled: rule.enabled,
    name: rule.name ?? '',
  })
  loadSecurityToForm(rule)
  syncSecurityExpanded()
  if (tab === 'nginx') {
    formTab.value = 'nginx'
  } else if (tab === 'security' || activeSecurityFeatures.value.length > 0) {
    formTab.value = 'security'
  } else {
    formTab.value = 'basic'
  }
  showModal.value = true
  if (tab === 'nginx') {
    nginxDirty.value = false
    void loadRuleNginxForEdit(rule.id)
  }
}

let rowSortables: Sortable[] = []
let entryStackSortable: Sortable | null = null

function destroyRowSortable() {
  for (const instance of rowSortables) instance.destroy()
  rowSortables = []
}

async function persistRuleOrder(next: ProxyRule[]) {
  rules.value = next
  reordering.value = true
  try {
    await api.reorderProxies(next.map((rule) => rule.id))
    message.success('排序已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '排序保存失败')
    await load()
  } finally {
    reordering.value = false
  }
}

function mergeGroupOrder(groupKey: string, groupRules: ProxyRule[]): ProxyRule[] {
  const next: ProxyRule[] = []
  for (const group of entryGroups.value) {
    if (group.key === groupKey) next.push(...groupRules)
    else next.push(...group.rules)
  }
  return next
}

function attachSortable(tbody: HTMLElement, onEnd: (oldIndex: number, newIndex: number) => void) {
  rowSortables.push(
    Sortable.create(tbody, {
      handle: '.proxy-drag-handle',
      animation: 150,
      draggable: '.n-data-table-tr',
      onEnd: async (evt) => {
        if (evt.oldIndex == null || evt.newIndex == null || evt.oldIndex === evt.newIndex || reordering.value) {
          return
        }
        onEnd(evt.oldIndex, evt.newIndex)
      },
    }),
  )
}

function destroyEntryStackSortable() {
  entryStackSortable?.destroy()
  entryStackSortable = null
}

async function persistEntryOrder(reorderedGroups: ProxyEntryGroup[]) {
  const ids = reorderedGroups.map((group) => group.entryId).filter((id): id is number => id != null)
  if (ids.length !== entries.value.length) return

  const byId = new Map(entries.value.map((entry) => [entry.id, entry]))
  entries.value = ids.map((id) => byId.get(id)!).filter(Boolean)
  reordering.value = true
  try {
    await api.reorderProxyEntries(ids)
    message.success('入口排序已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '入口排序保存失败')
    await load()
  } finally {
    reordering.value = false
  }
}

async function setupEntryStackSortable() {
  destroyEntryStackSortable()
  if (!canReorderEntries.value) return
  await nextTick()
  const stack = entryStackRef.value
  if (!stack) return

  entryStackSortable = Sortable.create(stack, {
    handle: '.proxy-entry-drag-handle',
    animation: 150,
    draggable: '.proxy-entry-card',
    onEnd: (evt) => {
      if (evt.oldIndex == null || evt.newIndex == null || evt.oldIndex === evt.newIndex || reordering.value) {
        return
      }
      const groups = [...entryGroups.value]
      const [moved] = groups.splice(evt.oldIndex, 1)
      groups.splice(evt.newIndex, 0, moved)
      void persistEntryOrder(groups)
    },
  })
}

async function setupRowSortable() {
  destroyRowSortable()
  destroyEntryStackSortable()
  if (!canReorder.value) return
  await nextTick()

  if (canReorderGlobally.value) {
    const tbody = tableWrapRef.value?.querySelector('.n-data-table-tbody') as HTMLElement | null
    if (!tbody) return
    attachSortable(tbody, (oldIndex, newIndex) => {
      const next = [...rules.value]
      const [moved] = next.splice(oldIndex, 1)
      next.splice(newIndex, 0, moved)
      void persistRuleOrder(next)
    })
    return
  }

  if (!canReorderInGroups.value) return

  for (const group of entryGroups.value) {
    if (group.rules.length < 2) continue
    const body = document.querySelector(
      `.proxy-entry-card__body[data-entry-key="${CSS.escape(group.key)}"] .n-data-table-tbody`,
    ) as HTMLElement | null
    if (!body) continue
    const groupKey = group.key
    attachSortable(body, (oldIndex, newIndex) => {
      const nextGroupRules = [...(entryGroups.value.find((item) => item.key === groupKey)?.rules ?? [])]
      const [moved] = nextGroupRules.splice(oldIndex, 1)
      nextGroupRules.splice(newIndex, 0, moved)
      void persistRuleOrder(mergeGroupOrder(groupKey, nextGroupRules))
    })
  }

  await setupEntryStackSortable()
}

watch([canReorder, canReorderGlobally, canReorderInGroups, canReorderEntries, () => rules.value.length, entryGroups, collapsedEntryKeys], () => {
  void setupRowSortable()
})

watch(showModal, (open) => {
  if (!open) {
    editing.value = null
    formEntryId.value = null
  }
})

async function save() {
  saving.value = true
  try {
    const payload = buildPayload()
    const listenPort = payload.listen_port ?? form.listen_port
    const conflict = findHostConflict(payload.hosts, listenPort, editing.value?.id)
    if (conflict) {
      const owner = primaryHost(conflict.rule)
      message.error(`域名 ${conflict.host}:${conflict.port} 已被规则「${owner}」使用，请编辑现有规则或更换域名/端口`)
      return
    }
    if (editing.value) {
      await api.updateProxy(editing.value.id, payload)
      message.success('规则已保存')
    } else {
      await api.createProxy(payload)
      message.success('规则已创建')
    }
    showModal.value = false
    await load()
  } catch (error) {
    const msg = error instanceof Error ? error.message : '保存失败'
    if (msg.startsWith('规则已保存')) {
      message.warning(msg)
      showModal.value = false
      await load()
    } else {
      message.error(msg)
    }
  } finally {
    saving.value = false
  }
}

function confirmDelete(rule: ProxyRule) {
  dialog.warning({
    title: `确定删除 ${ruleTitle(rule)}？`,
    content: '删除后相关域名将停止反向代理。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: () =>
      api
        .deleteProxy(rule.id)
        .then(async () => {
          if (selectedRuleId.value === rule.id) {
            selectedRuleId.value = null
            showDetailPanel.value = false
          }
          rules.value = rules.value.filter((r) => r.id !== rule.id)
          message.success('规则已删除')
          await load()
        })
        .catch(async (error: unknown) => {
          const msg = error instanceof Error ? error.message : '删除失败'
          if (msg.startsWith('规则已删除')) {
            message.warning(msg)
            await load()
          } else {
            message.error(msg)
            return false
          }
        }),
  })
}

watch(
  () => form.https_enabled,
  (httpsEnabled) => {
    if (!showModal.value || editing.value) return
    form.listen_port = defaultListenPort(httpsEnabled)
  },
)

onMounted(async () => {
  await loadDefaults()
  await load()
  await refreshTraffic()
  startTrafficPoll()
})
onUnmounted(() => {
  destroyRowSortable()
  destroyEntryStackSortable()
  stopLogStream()
  stopTrafficPoll()
  stopClientsPoll()
  document.removeEventListener('keydown', onLogFullscreenKeydown)
  document.body.style.overflow = ''
})
</script>

<style scoped>
.stats-row {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.stat-card {
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  padding: var(--fonu-space-4);
  min-height: 118px;
}

.stat-card__icon { font-size: 18px; margin-bottom: 4px; }
.stat-card__icon--blue { color: #3b82f6; }
.stat-card__icon--green { color: #10b981; }
.stat-card__icon--amber { color: #f59e0b; }
.stat-card__icon--purple { color: #8b5cf6; }

.stat-card__value {
  margin-top: 4px;
  font-size: 28px;
  font-weight: 700;
  line-height: 1.1;
  color: var(--fonu-text);
}

.stat-card__value--sm { font-size: 22px; }

.stat-card__label {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
}

.stat-card__sub {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.proxy-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--fonu-space-4);
  align-items: start;
}

.proxy-panel {
  min-width: 0;
}

.proxy-panel :deep(.fonu-card__body) {
  padding-left: 0;
  padding-right: 0;
  padding-bottom: 0;
}

.proxy-toolbar {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-4);
  border-bottom: 1px solid var(--fonu-border);
  flex-wrap: wrap;
}

.proxy-toolbar__search {
  width: 240px;
  max-width: 100%;
  flex-shrink: 0;
}

.proxy-toolbar__filter {
  width: 132px;
  flex-shrink: 0;
}

.proxy-toolbar__spacer {
  flex: 1;
  min-width: 0;
}

.proxy-loading {
  display: flex;
  justify-content: center;
  padding: var(--fonu-space-6) 0;
}

.proxy-entry-stack {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-4);
}

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
  outline: none;
}

.proxy-entry-card__head:focus-within .proxy-entry-card__intro:focus-visible {
  outline: none;
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

.proxy-entry-card__head:focus-within .proxy-entry-card__intro:focus-visible {
  outline: 2px solid rgba(16, 185, 129, 0.45);
  outline-offset: 2px;
  border-radius: 6px;
}

.proxy-entry-card__chevron {
  display: none;
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

@media (min-width: 960px) {
  .proxy-entry-card__stats {
    display: flex;
  }
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

.discovery-modal {
  width: min(760px, 94vw);
  max-height: 82vh;
  display: flex;
  flex-direction: column;
  padding: var(--fonu-space-4);
  background: var(--fonu-surface, #fff);
  border-radius: var(--fonu-radius-lg, 12px);
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
  min-height: 180px;
  overflow: hidden;
}

.discovery-modal__loading {
  display: flex;
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
  min-height: 120px;
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
  overflow: auto;
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

.proxy-entry-edit-modal {
  width: min(480px, 92vw);
  padding: var(--fonu-space-4);
  background: var(--fonu-surface, #fff);
  border-radius: var(--fonu-radius-lg, 12px);
}

.proxy-entry-edit-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--fonu-space-4);
}

.proxy-entry-edit-modal__body {
  margin-bottom: var(--fonu-space-4);
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

.proxy-table--nested :deep(.n-data-table-th) {
  background: transparent;
}

.listen-row--readonly {
  padding: 8px 0 4px;
}

.listen-row--readonly .listen-col__label {
  margin-bottom: 6px;
}

.proxy-drag-handle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  color: var(--fonu-text-secondary);
  cursor: grab;
}

.proxy-table--sortable :deep(.proxy-drag-handle:active) {
  cursor: grabbing;
}

.proxy-table-wrap {
  width: 100%;
  overflow-x: auto;
}

.proxy-table { width: 100%; }

.proxy-table :deep(.proxy-row--active td) {
  background: rgba(16, 185, 129, 0.06);
}

.proxy-enable-cell {
  display: inline-flex;
  align-items: center;
}

.proxy-table :deep(.domain-cell__main) {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.proxy-table :deep(.domain-cell__links) {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 4px;
}

.proxy-table :deep(.domain-cell__link-row) {
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
}

.proxy-table :deep(.domain-cell__link) {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  font-family: var(--fonu-mono);
  color: #2563eb;
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proxy-table :deep(.domain-cell__link:hover) {
  color: #1d4ed8;
  text-decoration: underline;
  text-decoration-skip-ink: none;
  text-underline-offset: 2px;
}

.proxy-table :deep(.domain-cell__link-action) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  background: none;
  color: #2563eb;
  cursor: pointer;
  border-radius: 4px;
}

.proxy-table :deep(.domain-cell__link-action:hover) {
  color: #1d4ed8;
  background: rgba(37, 99, 235, 0.08);
}

.proxy-table :deep(.domain-cell__more) {
  align-self: flex-start;
  margin-top: 2px;
  padding: 0;
  border: none;
  background: none;
  font: inherit;
  font-size: 12px;
  color: #2563eb;
  cursor: pointer;
}

.proxy-table :deep(.domain-cell__more:hover) {
  color: #1d4ed8;
  text-decoration: underline;
  text-decoration-skip-ink: none;
  text-underline-offset: 2px;
}

.domain-cell__popover-links {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 240px;
  max-width: min(420px, 80vw);
  padding: 4px 0;
}

.domain-cell__popover-links :deep(.domain-cell__link-row) {
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
  padding: 2px 4px;
  border-radius: 6px;
}

.domain-cell__popover-links :deep(.domain-cell__link-row:hover) {
  background: var(--fonu-bg);
}

.domain-cell__popover-links :deep(.domain-cell__link) {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  font-family: var(--fonu-mono);
  color: #2563eb;
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.domain-cell__popover-links :deep(.domain-cell__link:hover) {
  color: #1d4ed8;
  text-decoration: underline;
  text-decoration-skip-ink: none;
  text-underline-offset: 2px;
}

.domain-cell__popover-links :deep(.domain-cell__link-action) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  background: none;
  color: #2563eb;
  cursor: pointer;
  border-radius: 4px;
}

.domain-cell__popover-links :deep(.domain-cell__link-action:hover) {
  color: #1d4ed8;
  background: rgba(37, 99, 235, 0.08);
}

.proxy-table :deep(.proto-tags) {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.proxy-detail-modal {
  width: min(920px, 96vw);
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: var(--fonu-surface);
  border-radius: var(--fonu-radius);
  overflow: hidden;
  box-shadow: var(--fonu-shadow-md);
}

.proxy-detail-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  padding: var(--fonu-space-5) var(--fonu-space-5) 0;
  flex-shrink: 0;
}

.proxy-detail-modal__intro {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-3);
  min-width: 0;
}

.proxy-detail-modal__title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.proxy-detail-modal__conn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  color: var(--fonu-text-muted);
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
}

.proxy-detail-modal__conn.is-active {
  color: var(--fonu-brand-text);
  background: var(--fonu-brand-soft);
  border-color: rgba(16, 185, 129, 0.25);
}

.proxy-detail__tabbar {
  display: flex;
  gap: var(--fonu-space-5);
  padding: var(--fonu-space-3) var(--fonu-space-5) 0;
  border-bottom: 1px solid var(--fonu-border);
  flex-shrink: 0;
}

.proxy-detail__tab {
  margin: 0;
  padding: 8px 2px 10px;
  border: none;
  background: none;
  font: inherit;
  font-size: 14px;
  color: var(--fonu-text-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}

.proxy-detail__tab:hover {
  color: var(--fonu-text);
}

.proxy-detail__tab--active {
  color: var(--fonu-brand-text);
  font-weight: 600;
  border-bottom-color: var(--fonu-brand);
}

.proxy-detail__scroll {
  flex: 1;
  min-height: min(480px, 60vh);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.proxy-detail__pane {
  width: 100%;
  box-sizing: border-box;
  padding: var(--fonu-space-4) var(--fonu-space-5) var(--fonu-space-5);
}

.proxy-detail__pane--overview {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
}

.overview-strip {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 0.8fr) minmax(0, 1.4fr) auto;
  gap: 12px 16px;
  padding: 12px 14px;
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  flex-shrink: 0;
}

.overview-security {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 12px;
  padding: 10px 14px;
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  flex-shrink: 0;
}

.overview-security__label {
  font-size: 11px;
  color: var(--fonu-text-muted);
  letter-spacing: 0.02em;
  flex-shrink: 0;
}

.overview-security__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.overview-security__empty {
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.overview-strip__item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.overview-strip__label {
  font-size: 11px;
  color: var(--fonu-text-muted);
  letter-spacing: 0.02em;
}

.overview-strip__value {
  font-size: 13px;
  font-weight: 500;
  color: var(--fonu-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-main {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(0, 1fr);
  gap: var(--fonu-space-4);
  flex: 1 1 0;
  min-height: 0;
  align-items: stretch;
}

.overview-side {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
}

.overview-card {
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  background: var(--fonu-surface);
  padding: 14px 16px;
}

.overview-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  margin-bottom: 10px;
}

.overview-card__head h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.overview-card__sub {
  margin-left: 6px;
  font-size: 12px;
  font-weight: 400;
  color: var(--fonu-text-muted);
}

.overview-card--chart {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.overview-chart {
  flex: 1;
  min-height: 180px;
}

.overview-metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-template-rows: 1fr 1fr;
  gap: 10px;
  flex-shrink: 0;
}

.overview-metric {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
}

.overview-metric__label {
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.overview-metric__value {
  margin-top: 4px;
  font-size: 18px;
  font-weight: 600;
  line-height: 1.2;
  color: var(--fonu-text);
}

.overview-card--clients {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  margin: 0;
  padding-bottom: 12px;
}

.overview-client-list {
  flex: 1 1 0;
  min-height: 72px;
  overflow-y: auto;
  margin: 0 -2px;
  padding: 0 2px;
  scrollbar-width: thin;
  scrollbar-color: rgba(148, 163, 184, 0.35) transparent;
}

.overview-client-list:hover {
  scrollbar-color: rgba(148, 163, 184, 0.55) var(--fonu-bg-muted);
}

.overview-client-list::-webkit-scrollbar {
  width: 6px;
}

.overview-client-list::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.25);
  border-radius: 4px;
}

.overview-client-list:hover::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.5);
}

.overview-client-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: 7px 10px;
  border-radius: 8px;
  font-size: 12px;
}

.overview-client-row + .overview-client-row {
  margin-top: 2px;
}

.overview-client-row:hover {
  background: var(--fonu-bg);
}

.overview-client-row__time {
  flex-shrink: 0;
  color: var(--fonu-text-muted);
  font-size: 11px;
}

.overview-empty {
  margin: 0;
  height: 100%;
  min-height: 72px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.traffic-live-legend--compact {
  margin-bottom: 6px;
}

.proxy-detail__pane--logs,
.proxy-detail__pane--nginx {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding-top: var(--fonu-space-3);
}

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

.live-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: #10b981;
  font-weight: 500;
}

.live-badge__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
  animation: live-pulse 1.5s ease-in-out infinite;
}

@keyframes live-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.traffic-live-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.traffic-live-legend__item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.traffic-live-legend__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.traffic-live-legend__item--up .traffic-live-legend__dot {
  background: #10b981;
}

.traffic-live-legend__item--down .traffic-live-legend__dot {
  background: #3b82f6;
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

.proxy-log-fullscreen {
  position: fixed;
  inset: 0;
  z-index: 10000;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 20px 20px;
  background: #0b1220;
}

.proxy-log-fullscreen__head {
  margin-bottom: 8px;
}

.proxy-log-fullscreen__title {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.proxy-log-fullscreen__rule {
  color: #94a3b8;
  font-size: 13px;
  font-weight: 400;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.listen-types {
  display: flex;
  align-items: center;
  gap: var(--fonu-space-5);
}

.proxy-modal {
  display: flex;
  width: 900px;
  max-width: 95vw;
  max-height: 90vh;
  background: var(--fonu-surface);
  border-radius: var(--fonu-radius);
  overflow: hidden;
  box-shadow: var(--fonu-shadow-md);
}

.proxy-modal__form {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  max-height: 90vh;
}

.proxy-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-5) var(--fonu-space-5) 0;
}

.proxy-modal__tabbar {
  display: flex;
  gap: var(--fonu-space-5);
  padding: var(--fonu-space-3) var(--fonu-space-5) 0;
  border-bottom: 1px solid var(--fonu-border);
  flex-shrink: 0;
}

.proxy-modal__tab {
  margin: 0;
  padding: 8px 2px 10px;
  border: none;
  background: none;
  font: inherit;
  font-size: 14px;
  color: var(--fonu-text-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}

.proxy-modal__tab:hover {
  color: var(--fonu-text);
}

.proxy-modal__tab--active {
  color: var(--fonu-brand-text);
  font-weight: 600;
  border-bottom-color: var(--fonu-brand);
}

.proxy-modal__tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.proxy-modal__scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
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

.security-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.security-header {
  padding: 14px 16px 16px;
  border-radius: 10px;
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
}

.security-header__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
}

.security-header__preset {
  flex-shrink: 0;
  font-weight: 500;
}

.security-header__status {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.security-header__status-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
  flex-shrink: 0;
}

.security-header__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.security-header__empty {
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.security-header__note {
  margin: 14px 0 0;
  padding-top: 14px;
  border-top: 1px solid var(--fonu-border);
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.6;
}

.security-collapse {
  width: 100%;
  border: 1px solid var(--fonu-border);
  border-radius: 10px;
  overflow: hidden;
  background: var(--fonu-surface);
}

.security-collapse :deep(.n-collapse-item) {
  margin: 0 !important;
  border: none !important;
  border-radius: 0 !important;
}

.security-collapse :deep(.n-collapse-item + .n-collapse-item) {
  border-top: 1px solid var(--fonu-border) !important;
}

/* naive-ui 首个 collapse-item 默认 padding-top: 0，导致首项偏矮 */
.security-collapse :deep(.n-collapse-item__header) {
  padding: 12px 14px !important;
  min-height: 44px;
  box-sizing: border-box;
  font-weight: 600;
  background: var(--fonu-bg);
}

.security-collapse :deep(.n-collapse-item:first-child > .n-collapse-item__header) {
  padding-top: 12px !important;
}

.security-collapse :deep(.n-collapse-item__content-inner) {
  padding: 0 14px 14px;
}

.security-panel {
  padding-top: 2px;
}

.security-panel__fields {
  padding: 8px 0 16px;
  margin-bottom: 4px;
  border-bottom: 1px solid var(--fonu-border);
}

.security-panel__fields :deep(.n-form-item) {
  margin-bottom: 0;
}

.security-panel__fields :deep(.n-form-item-label) {
  padding-bottom: 6px;
}

.security-panel__fields:last-child {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 6px;
}

.security-panel .security-fields-grid {
  padding: 8px 0 16px;
  margin-bottom: 4px;
  border-bottom: 1px solid var(--fonu-border);
}

.security-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: 12px 0;
}

.security-panel .security-option + .security-option,
.security-panel__fields + .security-option,
.security-fields-grid + .security-option {
  border-top: 1px solid var(--fonu-border);
}

.security-option__label {
  font-size: 14px;
  font-weight: 500;
  color: var(--fonu-text);
}

.security-option__hint {
  margin-top: 2px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.45;
}

.security-fields-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--fonu-space-3);
  padding: 4px 0 8px;
}

.conn-limit-input {
  width: 160px;
}

.w-full {
  width: 100%;
}

.form-switch-list--compact {
  margin-bottom: 0;
}

.proxy-security-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.proxy-modal__help {
  width: 300px;
  flex-shrink: 0;
  padding: var(--fonu-space-5);
  background: #f8fafc;
  border-left: 1px solid var(--fonu-border);
  font-size: 13px;
  color: var(--fonu-text-secondary);
  overflow: auto;
}

.proxy-modal__help h4 {
  margin: 0 0 var(--fonu-space-3);
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.proxy-modal__help ol {
  margin: 0;
  padding-left: 18px;
  line-height: 1.75;
}

.proxy-modal__help ul {
  margin: 6px 0 0;
  padding-left: 18px;
}

.proxy-modal__help li + li {
  margin-top: 10px;
}

.proxy-modal__help code {
  font-size: 12px;
  background: var(--fonu-surface);
  padding: 1px 4px;
  border-radius: 4px;
}

.proxy-modal__tip {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: var(--fonu-space-5);
  padding: 12px;
  border-radius: 8px;
  background: #eff6ff;
  color: #1d4ed8;
  font-size: 12px;
  line-height: 1.6;
}

.proxy-modal__tip-icon {
  flex-shrink: 0;
  margin-top: 1px;
  font-size: 16px;
}

.proxy-modal__tip--warn {
  background: #fffbeb;
  color: #b45309;
}

.form-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.form-label__help {
  font-size: 14px;
  color: var(--fonu-text-muted);
  cursor: help;
}

.field-stack {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
}

.proxy-secret-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.field-hint {
  margin: 0;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

.proxy-modal__pane :deep(.n-form-item .n-form-item-blank) {
  display: block;
  width: 100%;
}

.listen-row {
  display: flex;
  align-items: flex-start;
  gap: 24px;
  margin-bottom: var(--fonu-space-5);
}

.listen-col {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.listen-col--port {
  width: 168px;
  flex-shrink: 0;
}

.listen-col--protocol {
  flex-shrink: 0;
}

.listen-col__label {
  font-size: 14px;
  line-height: 1.25;
  color: var(--fonu-text);
}

.required-mark {
  color: #d03050;
}

.listen-col__control {
  min-height: 34px;
  display: flex;
  align-items: center;
}

.port-input {
  width: 100%;
}

.form-switch-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-bottom: var(--fonu-space-4);
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
  font-weight: 600;
  color: var(--fonu-text);
}

.form-switch-row__hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-muted);
  line-height: 1.5;
}

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

.clients-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 10px;
}

.clients-list--compact .clients-row {
  padding: 6px 8px;
}

.clients-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--fonu-bg);
}

.mono { font-family: var(--fonu-mono); }
.text-muted { color: var(--fonu-text-muted); }
.text-secondary { color: var(--fonu-text-secondary); }

@media (max-width: 1199px) {
  .stats-row { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}

@media (max-width: 767px) {
  .stats-row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .proxy-toolbar__filter { width: 100%; }
  .proxy-detail-modal__header {
    flex-direction: column;
    align-items: flex-start;
  }
  .overview-strip {
    grid-template-columns: 1fr 1fr;
  }
  .overview-main {
    grid-template-columns: 1fr;
  }
  .overview-card--clients {
    min-height: 100px;
  }
}

@media (max-width: 640px) {
  .proxy-modal {
    flex-direction: column;
    width: 100%;
  }
  .proxy-modal__help {
    width: 100%;
    border-left: none;
    border-top: 1px solid var(--fonu-border);
  }
  .listen-row {
    flex-direction: column;
    align-items: stretch;
    gap: var(--fonu-space-4);
  }

  .listen-col--port {
    width: 100%;
  }
}
</style>
