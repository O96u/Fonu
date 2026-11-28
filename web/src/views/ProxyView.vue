<template>
  <header class="service-page-header">
    <div class="service-page-title"><h1>反向代理</h1></div>
    <div class="service-page-actions">
      <n-button @click="openDiscoveryModal()"><template #icon><n-icon :component="GlobeOutline"/></template>发现服务</n-button>
      <n-button @click="router.push({name:'settings',query:{tab:'advanced'}})">全局设置</n-button>
      <n-button v-if="presentation==='service'" type="primary" @click="openQuickCreate"><template #icon><n-icon :component="AddOutline" /></template>新增代理</n-button>
      <n-button v-else type="primary" @click="openEntryCreate"><template #icon><n-icon :component="AddOutline" /></template>新建入口</n-button>
    </div>
  </header>
  <NginxApplyBanner compact/>

  <LoadError v-if="loadError" :message="loadError" @retry="load" />

  <template v-else>
    <ProxyStats :total-count="rules.length" :enabled-count="enabledCount" :disabled-count="disabledCount" :traffic-totals="trafficTotals" :sampled-at="trafficSampledAt" :error="trafficError" />

    <n-alert v-if="businessError && presentation==='service'" type="warning" class="business-error">{{businessError}} <n-button size="tiny" @click="loadBusiness">重试分组</n-button></n-alert>
    <ProxyServiceWorkspace v-model:presentation="presentation" :entry-count="entryGroups.length" :rules="rules" :entries="entries" :traffic="trafficByRule" :business="business" :loading="loading" :group-busy="businessBusy || !!businessError" :reordering="reordering" :toggling-id="togglingRuleId" :rule-name="ruleName" :rule-hosts="ruleHosts" :security-tags="securityTags" :access-url="hostAccessUrl" @refresh="refreshAll" @create="openQuickCreate" @detail="openDetail" @edit="openEdit" @duplicate="openDuplicate" @delete="confirmDelete" @toggle="toggleRuleEnabled"  :change-business="changeBusiness" @select-group="activeBusinessGroup=$event" @copy-link="copyAccessUrl" @reorder="persistRuleOrder">
      <template #port-content>
        <div ref="layoutRef" class="proxy-layout port-workspace">
          <div class="port-workspace__note"><span>按监听端口及协议分组 · 入口独立于业务分组</span><span>删除入口会删除其关联规则</span></div>
          <FonuCard flush class="proxy-panel" title="监听入口">
            <template #header><span class="proxy-result-count">{{ filteredRules.length }} / {{ rules.length }} 条规则 · {{ visibleEntryGroups.length }} / {{ entryGroups.length }} 个入口</span></template>
        <ProxyToolbar
          v-model:search="search"
          v-model:status-filter="statusFilter"
          v-model:https-filter="httpsFilter"
          :status-options="statusOptions"
          :https-options="httpsOptions"
          :loading="loading"
          view-mode="grouped"
          grouped
          hide-view-switch
          :can-expand="visibleEntryGroups.length > 0"
          :all-expanded="allEntriesExpanded"
          @toggle-all="toggleAllEntries"
          @clear="clearFilters"
          @refresh="refreshAll"
        />

        <div v-if="loading && rules.length === 0" class="proxy-loading">
          <n-spin size="medium" />
        </div>

        <template v-else-if="visibleEntryGroups.length > 0">
          <div
            ref="entryStackRef"
            class="proxy-entry-stack"
            :class="{ 'proxy-entry-stack--sortable': canReorderEntries }"
          >
            <ProxyEntryGroupCard
              v-for="group in visibleEntryGroups"
              :key="group.key"
              :group="group"
              :collapsed="isEntryCollapsed(group.key)"
              :sortable="canReorderEntries"
              :preview="entryGroupRuleNames(group,ruleName)"
              :traffic="entryGroupTraffic(group)"
              @toggle="toggleEntryCollapsed(group.key)"
              @create="openCreateWithEntry(group.listen,group.entryId)"
              @discover="openDiscoveryModal(group)"
              @edit="openEntryEdit(group)"
              @duplicate="duplicateEntry(group)"
              @delete="confirmDeleteEntry(group)"
            >
              <div v-if="!isEntryCollapsed(group.key)" class="proxy-entry-card__body" :data-entry-key="group.key">
                <ProxyRuleTable
                  v-if="group.rules.length > 0"
                  nested
                  :sortable="canReorderInGroups && group.rules.length > 1"
                  :columns="entryGroupColumnsFor(group)"
                  :compact="compactTable"
                  :data="group.rules"
                  :row-props="rowProps"
                />
                <div v-else class="proxy-entry-card__empty">
                  <span class="text-muted">此入口下还没有规则</span>
                  <n-button size="small" quaternary type="primary" @click="openCreateWithEntry(group.listen, group.entryId)">
                    在此入口下新增
                  </n-button>
                </div>
              </div>
            </ProxyEntryGroupCard>
          </div>
        </template>
        <EmptyState
          v-else-if="rules.length === 0 && entries.length === 0"
          title="还没有反向代理规则"
          description="先新建监听入口，或直接新增规则。一个入口可以管理多个内网服务。"
        ><template #action><n-button type="primary" @click="openEntryCreate">新建入口</n-button></template></EmptyState>
        <EmptyState
          v-else
          title="没有匹配的规则"
          description="试试调整搜索关键词或筛选条件。"
        ><template #action><n-button @click="clearFilters">清除筛选</n-button></template></EmptyState>
          </FonuCard>
        </div>
      </template>
    </ProxyServiceWorkspace>
  </template>

  <n-drawer :show="showDetailPanel" width="min(740px, 100vw)" placement="right" :mask-closable="true" @update:show="value=>{if(!value)closeDetail()}">
    <div v-if="selectedRule" class="proxy-detail-modal">
      <div class="proxy-detail-modal__header">
        <div class="proxy-detail-modal__intro">
          <ServiceFavicon :rule-id="selectedRule.id"/>
          <div class="detail-title"><h3 class="proxy-detail-modal__title">{{ ruleName(selectedRule) }}</h3><small>{{primaryHost(selectedRule)}} · 规则 #{{selectedRule.id}}</small></div>
          <StatusBadge
            :value="selectedRule.enabled ? 'ok' : 'disabled'"
            :text="selectedRule.enabled ? '已启用' : '已停用'"
          />
          <span class="proxy-detail-modal__conn" :class="{ 'is-active': (selectedTraffic?.connections ?? 0) > 0 }">
            <n-icon :component="PeopleOutline" />
            {{ selectedTraffic?.connections ?? 0 }} 近期访问 IP
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
        <n-tabs :value="detailTab" role="tablist" aria-label="代理详情" type="line" animated @update:value="value=>switchDetailTab(value as 'overview' | 'diagnostics' | 'logs' | 'nginx')">
          <n-tab name="overview" role="tab" :aria-selected="detailTab==='overview'" tabindex="0" @keydown.enter.prevent="switchDetailTab('overview')" @keydown.space.prevent="switchDetailTab('overview')">概览</n-tab>
          <n-tab name="diagnostics" role="tab" :aria-selected="detailTab==='diagnostics'" tabindex="0" @keydown.enter.prevent="switchDetailTab('diagnostics')" @keydown.space.prevent="switchDetailTab('diagnostics')">运行诊断</n-tab>
          <n-tab name="logs" role="tab" :aria-selected="detailTab==='logs'" tabindex="0" @keydown.enter.prevent="switchDetailTab('logs')" @keydown.space.prevent="switchDetailTab('logs')">日志</n-tab>
          <n-tab name="nginx" role="tab" :aria-selected="detailTab==='nginx'" tabindex="0" @keydown.enter.prevent="switchDetailTab('nginx')" @keydown.space.prevent="switchDetailTab('nginx')">Nginx</n-tab>
        </n-tabs>
      </div>

      <div class="proxy-detail__scroll">
        <ProxyOverviewPanel
          v-if="detailTab === 'overview'"
          :selected-rule="selectedRule"
          :hosts="ruleHosts(selectedRule)"
          :listen="listenLabel(selectedRule)"
          :selected-security-features="selectedSecurityFeatures"
          :selected-traffic="selectedTraffic"
          :traffic-chart="trafficChart"
          :client-rows="clientRows"
          :clients-loading="clientsLoading"
          :clients-error="clientsError"
          :traffic-error="trafficError"
          :sampled-at="trafficSampledAt"
          @refresh-clients="loadClients"
        />

        <ProxyDiagnosticsPanel v-if="showDetailPanel&&detailTab==='diagnostics'" :key="selectedRule.id" :rule="selectedRule"/>
        <div v-if="detailTab === 'logs'" class="proxy-detail__pane proxy-detail__pane--logs">
              <div class="log-panel-head">
                <span class="text-muted">实时访问日志 · {{ logStreamState === 'open' ? '已连接' : logStreamState === 'retrying' ? '重连中' : logStreamState === 'paused' ? '已暂停' : '连接中' }}</span>
                <div class="log-panel-actions">
                  <n-button size="small" quaternary @click="toggleLogPause">{{logPaused?'继续显示':'暂停显示'}}</n-button>
                  <n-button size="small" quaternary @click="openLogFullscreen">全屏</n-button>
                  <n-button size="tiny" quaternary @click="clearLogLines">清空</n-button>
                </div>
              </div>
              <div class="log-filters"><n-input v-model:value="logQuery" size="small" clearable placeholder="搜索路径、IP、域名…"/><n-select v-model:value="logStatus" size="small" :options="logStatusOptions"/><small>{{visibleLogLines.length}} / {{displayLogLines.length}} 条</small></div>
              <ProxyAccessLogBox
                ref="logBox"
                :lines="visibleLogLines"
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
  </n-drawer>

  <Teleport to="body">
    <div v-if="logFullscreen && selectedRule" class="proxy-log-fullscreen">
      <div class="log-panel-head proxy-log-fullscreen__head">
        <div class="proxy-log-fullscreen__title">
          <span class="text-muted">实时访问日志</span>
          <span class="proxy-log-fullscreen__rule">{{ ruleName(selectedRule) }}</span>
        </div>
        <div class="log-panel-actions">
          <n-button size="small" quaternary @click="toggleLogPause">{{logPaused?'继续显示':'暂停显示'}}</n-button>
          <n-button size="small" quaternary @click="clearLogLines">清空</n-button>
          <n-button size="small" quaternary @click="logFullscreen = false">退出全屏</n-button>
        </div>
      </div>
      <ProxyAccessLogBox
        ref="logBoxFullscreen"
        :lines="visibleLogLines"
        fullscreen
        @scroll="onLogBoxScroll"
      />
    </div>
  </Teleport>

  <ProxyEditorFrame :show="!!deleteTarget" title="删除独立代理规则" subtitle="删除后仅该规则关联域名停止转发" :icon="TrashOutline" :busy="deleteBusy" @close="deleteTarget=null">
    <div v-if="deleteTarget" class="delete-review">
      <div class="delete-identity"><ServiceFavicon :rule-id="deleteTarget.id"/><div><strong>{{ruleName(deleteTarget)}}</strong><small>规则 #{{deleteTarget.id}} · {{deleteTarget.enabled?'已启用':'已停用'}}</small></div></div>
      <dl><dt>前端域名</dt><dd v-for="host in ruleHosts(deleteTarget)" :key="host">{{host}}</dd><dt>目标地址</dt><dd>{{deleteTarget.upstream}}</dd><dt>监听</dt><dd>{{deleteTarget.listen_port}} · {{deleteTarget.https_enabled?'HTTPS':'HTTP'}}</dd></dl>
      <n-alert type="warning" :bordered="false">不会删除其他独立规则、业务分组、证书或 DDNS 配置。后端发布失败时保留原规则。</n-alert>
      <n-alert v-if="deleteError" type="error">{{deleteError}}</n-alert>
    </div>
    <template #footer><n-button :disabled="deleteBusy" @click="deleteTarget=null">取消</n-button><n-button type="error" :loading="deleteBusy" @click="deleteRule">确认删除</n-button></template>
  </ProxyEditorFrame>

  <ProxyEditorFrame :show="showModal" :title="modalTitle" :subtitle="editing?'规则 #'+editing.id+' · 独立配置':copySource?'复制后为全新独立规则':'每个域名创建独立代理规则'" :icon="copySource?CopyOutline:editing?CreateOutline:AddOutline" :busy="saving||nginxSaving" wide @close="closeModal">
    <template #navigation>
      <n-tabs :value="formTab" role="tablist" aria-label="代理配置" type="line" animated @update:value="value=>switchFormTab(value as EditorTab)">
        <n-tab v-for="tab in editorTabs" :key="tab.key" :name="tab.key" role="tab" :aria-selected="formTab===tab.key" :tabindex="saving||nginxSaving?-1:0" :disabled="saving||nginxSaving" @keydown.enter.prevent="switchFormTab(tab.key)" @keydown.space.prevent="switchFormTab(tab.key)">{{tab.label}}<span v-if="tab.key==='nginx'&&nginxDirty" class="editor-dirty-dot" aria-hidden="true"/></n-tab>
      </n-tabs>
    </template>
    <ProxyBasicEditor v-show="formTab==='basic'" :certificates="editorCertificates" :certificates-loading="editorCertificatesLoading" :certificates-error="editorCertificatesError" :form="form" :business-group="draftBusinessGroup" :business-options="editorBusinessOptions" :copy-source="copySource" :ddns-hosts-loading="ddnsHostsLoading" :host-select-options="hostSelectOptions" :filter-ddns-host-option="filterDdnsHostOption" @patch="patchBasicDraft" @business="draftBusinessGroup=$event" @load-hosts="loadDdnsHostOptions"/>
    <ProxySecurityEditor v-if="formTab==='security'" :key="editorSession" :form="form" :active-security-features="activeSecurityFeatures" v-model:security-expanded="securityExpanded" :has-password="!!editing?.security?.basic_auth?.has_password" @patch="Object.assign(form,$event)" @preset="applySecurityPreset" @settings="openTrustSettings" :custom-nginx="editing?.nginx_mode==='custom'"/>
    <ProxyNginxEditor v-if="formTab==='nginx'" :editing="!!editing" :nginx-edit-mode="nginxEditMode" :nginx-server-mode="nginxServerMode" :nginx-draft="nginxDraft" :nginx-dirty="nginxDirty" :nginx-saving="nginxSaving" :nginx-loading="nginxLoading" :nginx-backups="nginxBackups" :rule-dirty="ruleDraftDirty" @basic="switchFormTab('basic')" @mode="setNginxEditMode" @save="saveRuleNginx" @rollback="rollbackRuleNginx()" @reset="resetRuleNginxAuto" @update:nginx-draft="nginxDraft=$event;onNginxDraftInput()"/>
    <n-alert v-if="saveError" type="error" :bordered="false" class="editor-save-error">{{saveError}}</n-alert>
    <template #footer>
      <span class="editor-footer-info">{{formTab==='nginx'?'Nginx 文本独立保存，配置发布失败会回滚':ruleDraftDirty?'有未保存的配置':'配置保存后自动发布'}}</span>
      <n-button :disabled="saving||nginxSaving" @click="closeModal">取消</n-button>
      <n-button v-if="formTab!=='nginx'" type="primary" :loading="saving" :disabled="nginxSaving" @click="save">{{editing?'保存配置':copySource?'创建副本':'创建代理'}}</n-button>
      <n-button v-else-if="ruleDraftDirty" type="primary" :loading="saving" @click="save">保存规则配置</n-button>
    </template>
  </ProxyEditorFrame>

  <ProxyEntryEditor
    v-model:show-entry-modal="showEntryModal"
    :entry-modal-title="entryModalTitle"
    :entry-form="entryForm"
    :cloning-source-id="cloningSourceId"
    :editing-entry-id="editingEntryId"
    :clone-rule-count="cloneRuleCount"
    :saving-entry="savingEntry"
    @patch="Object.assign(entryForm, $event)"
    @save="saveEntry"
  />
  <ProxyDiscovery
    v-model:show-discovery-modal="showDiscoveryModal"
    v-model:discovery-host="discoveryHost"
    v-model:listen-port="discoveryListenPort"
    :https-enabled="discoveryHttpsEnabled"
    @update:https-enabled="setDiscoveryProtocol"
    :error="discoveryError"
    :scanned="discoveryScanned"
    :business-options="[{label:'未分组',value:0},...business.groups.map(g=>({label:g.name,value:g.id}))]"
    v-model:business-group="discoveryBusinessGroup"
    :discovery-loading="discoveryLoading"
    :discovery-scan-host="discoveryScanHost"
    :discovery-summary="discoverySummary"
    :discovery-rows="discoveryRows"
    :discovery-select-all-checked="discoverySelectAllChecked"
    :discovery-select-all-indeterminate="discoverySelectAllIndeterminate"
    :discovery-adding="discoveryAdding"
    :has-selected-discovery-rows="hasSelectedDiscoveryRows"
    @scan="runDiscoveryScan"
    @select-all="onDiscoverySelectAll"
    @patch-row="patchDiscoveryRow"
    @submit="submitDiscovery"
  />
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, h, nextTick, onMounted, onUnmounted, reactive, ref, watch, type Component, type VNode } from 'vue'
import Sortable from 'sortablejs'
import {useRouter} from 'vue-router'
import {
  NAlert,
  NButton,
  NDrawer,
  NIcon,
  NInput,
  NSelect,
  NPopover,
  NSpace,
  NSpin,
  NSwitch,
  NTag,
  NTabs,
  NTab,
  useDialog,
  useMessage,
  type DataTableColumns,
  type SelectGroupOption,
} from 'naive-ui'
import {
  AddOutline,
  GlobeOutline,
  CloseOutline,
  CopyOutline,
  CreateOutline,
  OpenOutline,
  ReorderThreeOutline,
  InformationCircleOutline,
  PeopleOutline,
  TrashOutline,
} from '@vicons/ionicons5'
import { api, asList } from '../api/client'
import { createProxyPoll } from '../composables/proxy/useProxyPoll'
import { useProxyLogStream } from '../composables/proxy/useProxyLogStream'
import type { CertificateRecord, DiscoveredService, ProxyClientConn, ProxyEntry, ProxyRule, ProxySavePayload, ProxyTraffic, ProxyTrafficSnapshot, ProxyBusinessState, ProxyBusinessChange } from '../api/types'
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

import LoadError from '../components/LoadError.vue'
 import NginxApplyBanner from '../components/NginxApplyBanner.vue'
import ProxyStats from '../components/ProxyStats.vue'
import ProxyServiceWorkspace from '../components/proxy/ProxyServiceWorkspace.vue'
import ServiceFavicon from '../components/ServiceFavicon.vue'
import ProxyToolbar from '../components/proxy/ProxyToolbar.vue'
import ProxyEntryEditor from '../components/proxy/ProxyEntryEditor.vue'
import ProxyDiscovery from '../components/proxy/ProxyDiscovery.vue'
import ProxySecurityEditor from '../components/proxy/ProxySecurityEditor.vue'
import ProxyBasicEditor from '../components/proxy/ProxyBasicEditor.vue'
import ProxyEditorFrame from '../components/proxy/ProxyEditorFrame.vue'
import ProxyNginxEditor from '../components/proxy/ProxyNginxEditor.vue'
import ProxyOverviewPanel from '../components/proxy/ProxyOverviewPanel.vue'
import ProxyDiagnosticsPanel from '../components/proxy/ProxyDiagnosticsPanel.vue'
import ProxyEntryGroupCard from '../components/proxy/ProxyEntryGroup.vue'
import ProxyRuleTable from '../components/proxy/ProxyRuleTable.vue'
import { defaultSecurityForm } from '../composables/proxy/securityDraft'
import StatusBadge from '../components/StatusBadge.vue'
import { copyToClipboard } from '../utils/clipboard'
import {
  collectDdnsHostOptions,
  normalizeFrontendHostValues,
  validateUpstreamInput,
  type DdnsHostSelectOption,
} from '../utils/ddnsHosts'
import { formatBytes, formatRate } from '../utils/format'
import {
  formHostsList,
  mergeHostSelectOptions,
  setFormHostsFromText,
} from '../utils/proxyFormHosts'
import { renderTableRowActions } from '../utils/tableActions'

const router=useRouter()
const message = useMessage()
type ProxyPresentation = 'service' | 'port'
function readPresentation(): ProxyPresentation { try { return localStorage.getItem('fonu.proxy.presentation')==='port' ? 'port' : 'service' } catch { return 'service' } }
const presentation=ref<ProxyPresentation>(readPresentation())
watch(presentation,value=>{try{localStorage.setItem('fonu.proxy.presentation',value)}catch{/* storage unavailable */}})
const editorCertificates=ref<CertificateRecord[]>([])
const editorCertificatesLoading=ref(false)
const editorCertificatesError=ref(false)
let editorCertificateGeneration=0
async function loadEditorCertificates(){
  const generation=++editorCertificateGeneration
  editorCertificatesLoading.value=true
  editorCertificatesError.value=false
  editorCertificates.value=[]
  try{const records=await api.listCertificates();if(generation===editorCertificateGeneration)editorCertificates.value=asList(records)}
  catch{if(generation===editorCertificateGeneration)editorCertificatesError.value=true}
  finally{if(generation===editorCertificateGeneration)editorCertificatesLoading.value=false}
}
const business=ref<ProxyBusinessState>({groups:[],members:[]})
const businessError=ref('')
const businessBusy=ref(false)
let businessGeneration=0
const activeBusinessGroup=ref(-1)
const draftBusinessGroup=ref(0)
const discoveryBusinessGroup=ref(0)
async function loadBusiness(){
  if(businessBusy.value)return
  const generation=++businessGeneration
  try{const next=await api.getProxyBusiness();if(generation===businessGeneration){business.value=next;businessError.value=''}}
  catch(error){if(generation===businessGeneration)businessError.value=error instanceof Error?error.message:'读取业务分组失败'}
}
async function changeBusiness(change:ProxyBusinessChange):Promise<boolean>{
  if(businessBusy.value||businessError.value)return false
  businessBusy.value=true
  ++businessGeneration
  try{business.value=await api.changeProxyBusiness(change);message.success('业务分组已更新');return true}
  catch(error){message.error(error instanceof Error?error.message:'分组操作失败');return false}
  finally{businessBusy.value=false}
}
async function assignSavedRules(saved:ProxyRule[],groupId:number){
  if(groupId<=0||!saved.length)return
  if(businessBusy.value||businessError.value){message.warning('代理规则已保存，但业务分组尚未关联，可在列表中重新分组');return}
  businessBusy.value=true
  ++businessGeneration
  try{business.value=await api.changeProxyBusiness({action:'assign',id:groupId,rule_ids:saved.map(r=>r.id)})}
  catch{message.warning('代理规则已保存，但业务分组关联失败，可在列表中重新分组')}
  finally{businessBusy.value=false}
}
function ruleBusinessGroup(id:number){return business.value.members.find(m=>m.rule_id===id)?.group_id??0}
const dialog = useDialog()
const showDiscoveryModal = ref(false)
const discoveryHost = ref('127.0.0.1')
const discoveryScanHost = ref('127.0.0.1')
const discoveryLoading = ref(false)
const discoveryAdding = ref(false)
const discoveryListenPort = ref(443)
const discoveryHttpsEnabled = ref(true)
function setDiscoveryProtocol(value:boolean){if(discoveryListenPort.value===defaultListenPort(discoveryHttpsEnabled.value))discoveryListenPort.value=defaultListenPort(value);discoveryHttpsEnabled.value=value}

interface DiscoveryRow {
  id: string
  selected: boolean
  service: DiscoveredService
  ruleName: string
  hostname: string
}

const discoveryRows = ref<DiscoveryRow[]>([])

const ddnsHostOptionGroups = ref<SelectGroupOption[]>([])
const ddnsHostsLoading = ref(false)

const hostSelectOptions = computed(() =>
  mergeHostSelectOptions(ddnsHostOptionGroups.value, form.hostsSelected),
)


async function loadDdnsHostOptions() {
  if (ddnsHostsLoading.value) return
  ddnsHostsLoading.value = true
  try {
    const configs = asList(await api.listDDNSLite())
    ddnsHostOptionGroups.value = collectDdnsHostOptions(configs)
  } catch {
    ddnsHostOptionGroups.value = []
  } finally {
    ddnsHostsLoading.value = false
  }
}

function filterDdnsHostOption(pattern: string, option: DdnsHostSelectOption): boolean {
  const p = pattern.trim().toLowerCase()
  if (!p) return true
  const domain = String(option.value ?? option.label ?? '').toLowerCase()
  const provider = (option.providerLabel ?? '').toLowerCase()
  const task = (option.taskLabel ?? '').toLowerCase()
  return domain.includes(p) || provider.includes(p) || task.includes(p)
}

const discoverySummary = computed(() => {
  if (discoveryLoading.value || discoveryRows.value.length === 0) return ''
  return `扫描完成，发现 ${discoveryRows.value.length} 个 HTTP 服务`
})

const hasSelectedDiscoveryRows = computed(() =>
  discoveryRows.value.some(row=>row.selected) && discoveryRows.value.every(row=>!row.selected||!!row.hostname.trim()),
)

const discoverySelectAllChecked = computed(() => {
  const rows = discoveryRows.value
  if (rows.length === 0) return false
  return rows.every((row) => row.selected)
})

const discoverySelectAllIndeterminate = computed(() => {
  const rows = discoveryRows.value
  if (rows.length === 0) return false
  const selectedCount = rows.filter((row) => row.selected).length
  return selectedCount > 0 && selectedCount < rows.length
})

const showEntryModal = ref(false)
const savingEntry = ref(false)
const editingEntryId = ref<number | null>(null)
const cloningSourceId = ref<number | null>(null)
const cloneRuleCount = ref(0)
const entryForm = reactive({
  name: '',
  listen_port: 443,
  listen_ipv4: true,
  listen_ipv6: false,
  https_enabled: true,
  http_redirect: true,
})
const entryModalTitle = computed(() => {
  if (cloningSourceId.value) return '复制入口'
  if (editingEntryId.value) return '编辑入口'
  return '新建入口'
})
const copySource=ref<ProxyRule|null>(null)
const modalTitle = computed(() => editing.value?'编辑代理':copySource.value?'复制代理规则':'新增代理')
const rules = ref<ProxyRule[]>([])
const entries = ref<ProxyEntry[]>([])
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const showModal = ref(false)
const deleteTarget=ref<ProxyRule|null>(null)
const deleteBusy=ref(false)
const deleteError=ref('')
const discoveryError=ref('')
const discoveryScanned=ref(false)
type EditorTab='basic'|'security'|'nginx'
const editorTabs:{key:EditorTab;label:string}[]=[{key:'basic',label:'基础配置'},{key:'security',label:'安全设置'},{key:'nginx',label:'Nginx'}]
const formTab=ref<EditorTab>('basic')
const saveError=ref('')
const draftBaseline=ref('')
const nginxLoadedFor=ref<number|null>(null)
let editorRequest=0
const editorSession=ref(0)
const securityExpanded = ref<string[]>(['ip'])
const editing = ref<ProxyRule | null>(null)
const search = ref('')
const statusFilter = ref<string | null>(null)
const httpsFilter = ref<string | null>(null)
const layoutRef = ref<HTMLElement | null>(null)
const compactTable = ref(false)
let layoutObserver: ResizeObserver | undefined
watch(layoutRef, el => {
  layoutObserver?.disconnect()
  if (!el) return
  compactTable.value = el.clientWidth < 960
  if (typeof ResizeObserver !== 'undefined') {
    layoutObserver = new ResizeObserver(([entry]) => { if (entry) compactTable.value = entry.contentRect.width < 960 })
    layoutObserver.observe(el)
  }
}, { flush: 'post' })
const selectedRuleId = ref<number | null>(null)
const showDetailPanel = ref(false)
const detailTab = ref<'overview' | 'diagnostics' | 'logs' | 'nginx'>('overview')
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
const { lines: logLines, state: logStreamState, start: startLogStream, stop: stopLogStream, clear: clearStreamLines } = useProxyLogStream()
const logBox = ref<InstanceType<typeof ProxyAccessLogBox> | null>(null)
const logBoxFullscreen = ref<InstanceType<typeof ProxyAccessLogBox> | null>(null)
const logFullscreen = ref(false)
const logStickToBottom = ref(true)
const logPaused=ref(false),pausedLines=ref<string[]>([]),logQuery=ref(''),logStatus=ref('all')
const logStatusOptions=[{label:'全部状态',value:'all'},...['2','3','4','5'].map(v=>({label:v+'xx',value:v}))]
const displayLogLines=computed(()=>logPaused.value?pausedLines.value:logLines.value)
const visibleLogLines=computed(()=>{const q=logQuery.value.trim().toLowerCase();return displayLogLines.value.filter(line=>{const fields=line.trim().split(/\s+/);const status=fields[/^\d+$/.test(fields[2]??'')?5:4]??'';return (!q||line.toLowerCase().includes(q))&&(logStatus.value==='all'||status.startsWith(logStatus.value))})})
function toggleLogPause(){if(!logPaused.value){pausedLines.value=[...logLines.value];logPaused.value=true}else{logPaused.value=false;pausedLines.value=[];logStickToBottom.value=true;void nextTick().then(()=>scrollLogToBottom(true))}}
const LOG_SCROLL_BOTTOM_THRESHOLD = 24
const trafficByRule = ref<Record<number, ProxyTraffic>>({})
const clientRows = ref<ProxyClientConn[]>([])
const clientsLoading = ref(false)
const clientsError=ref('')
let clientsController:AbortController|undefined
let clientsGeneration=0
const trafficSnapshot=ref<ProxyTrafficSnapshot|null>(null)
const trafficError=ref('')
const trafficSampledAt=computed(()=>trafficSnapshot.value?.sampled_at??'')
let viewAlive=true
type RatePoint = { at: number; upload: number; download: number }
const rateHistory = ref<RatePoint[]>([])
const RATE_HISTORY_MS = 5 * 60 * 1000
const trafficPoll = createProxyPoll(refreshTraffic, 2000)
const clientsPoll = createProxyPoll(loadClients, 5000)

const nginxHttpPort = ref(80)
const nginxHttpsPort = ref(443)

function defaultListenPort(httpsEnabled: boolean) {
  return httpsEnabled ? nginxHttpsPort.value : nginxHttpPort.value
}

const statusOptions = [
  { label: '已启用', value: 'enabled' },
  { label: '已停用', value: 'disabled' },
]
const httpsOptions = [
  { label: 'HTTPS', value: 'on' },
  { label: 'HTTP', value: 'off' },
]

const form = reactive({
  listen_port: 80,
  listen_ipv4: true,
  listen_ipv6: false,
  hostsSelected: [] as string[],
  upstream: '',
  https_enabled: true,
  http_redirect: true,
  enabled: true,
  name: '',
  ...defaultSecurityForm(),
})

const entryStackRef = ref<HTMLElement | null>(null)
const expandedEntryKeys = ref<Set<string>>(new Set())
const formEntryId = ref<number | null>(null)
const editorBusinessOptions=computed(()=>[{label:'未分组',value:0},...business.value.groups.map(g=>({label:g.name,value:g.id}))])
function patchBasicDraft(patch:Partial<typeof form>){
  const changeProtocol=patch.https_enabled!==undefined&&patch.https_enabled!==form.https_enabled
  const standardPort=form.listen_port===nginxHttpPort.value||form.listen_port===nginxHttpsPort.value
  Object.assign(form,patch)
  if(changeProtocol&&standardPort)form.listen_port=defaultListenPort(form.https_enabled)
  if(patch.https_enabled===false)form.http_redirect=false
}
function openTrustSettings(){window.open(router.resolve({name:'settings',query:{tab:'security'}}).href,'_blank','noopener,noreferrer')}
function serializeRuleDraft(){return JSON.stringify({form,entry:formEntryId.value,group:draftBusinessGroup.value})}
const ruleDraftDirty=computed(()=>draftBaseline.value!==''&&serializeRuleDraft()!==draftBaseline.value)
function initializeEditor(){++editorRequest;editorSession.value++;nginxLoadedFor.value=null;nginxDirty.value=false;nginxDraft.value='';nginxGenerated.value='';nginxLoading.value=false;nginxBackups.value=[];saveError.value='';draftBaseline.value=serializeRuleDraft()}
function finishEditor(){++editorRequest;showModal.value=false;form.basic_auth_password='';nginxLoadedFor.value=null;nginxDirty.value=false;draftBaseline.value='';copySource.value=null;saveError.value=''}

const reordering = ref(false)
const togglingRuleId = ref<number | null>(null)

const enabledCount = computed(() => rules.value.filter((r) => r.enabled).length)
const disabledCount = computed(() => rules.value.length - enabledCount.value)

const trafficTotals = computed(() => {
 const t=trafficSnapshot.value?.total
 return {upload:t?.upload_total??0,download:t?.download_total??0,uploadRate:t?.upload_rate??0,downloadRate:t?.download_rate??0,connections:t?.connections??0}
})

const hasFilters = computed(() => !!(search.value.trim() || statusFilter.value || httpsFilter.value))
const canReorder = computed(() => presentation.value==='port' && !hasFilters.value && !reordering.value && rules.value.length > 1)
const canReorderInGroups = computed(() => canReorder.value)
const canReorderEntries = computed(() => presentation.value==='port' && !hasFilters.value && !reordering.value &&
  entries.value.length > 1 && entryGroups.value.length === entries.value.length &&
  entryGroups.value.every(group => group.entryId != null))

const entryGroups = computed(() => buildProxyEntryGroups(rules.value, entries.value))
const ruleEntryNames = computed(() => {
  const names = new Map<number,string>()
  for (const group of entryGroups.value) for (const rule of group.rules) names.set(rule.id,entryGroupDisplayName(group))
  return names
})
const filteredRules = computed(() => rules.value.filter(rule => {
  const q = search.value.trim().toLowerCase()
  const haystack = [rule.name, rule.upstream, ruleHosts(rule).join(' '), ruleEntryNames.value.get(rule.id), String(rule.listen_port)].join(' ').toLowerCase()
  if (q && !haystack.includes(q)) return false
  if (statusFilter.value === 'enabled' && !rule.enabled) return false
  if (statusFilter.value === 'disabled' && rule.enabled) return false
  if (httpsFilter.value === 'on' && !rule.https_enabled) return false
  if (httpsFilter.value === 'off' && rule.https_enabled) return false
  return true
}))
const visibleEntryGroups = computed(() => {
  if (!hasFilters.value) return entryGroups.value
  const ids = new Set(filteredRules.value.map(rule => rule.id))
  return entryGroups.value.map(group => {
    const matched = group.rules.filter(rule => ids.has(rule.id))
    return {...group,rules:matched,enabledCount:matched.filter(rule=>rule.enabled).length}
  }).filter(group => group.rules.length > 0 || (!statusFilter.value && !httpsFilter.value && group.name.toLowerCase().includes(search.value.trim().toLowerCase()) && entryGroups.value.find(original=>original.key===group.key)?.rules.length===0))
})
// Filtering uses a temporary expansion state, never overwrites the user's normal groups.
const filterCollapsedKeys = ref(new Set<string>())
watch([search,statusFilter,httpsFilter], () => { filterCollapsedKeys.value = new Set() })
function isEntryCollapsed(key: string): boolean {
  return hasFilters.value ? filterCollapsedKeys.value.has(key) : !expandedEntryKeys.value.has(key)
}
function toggleEntryCollapsed(key: string) {
  const target = hasFilters.value ? filterCollapsedKeys : expandedEntryKeys
  const next = new Set(target.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  target.value = next
}
const allEntriesExpanded = computed(() => visibleEntryGroups.value.length > 0 && visibleEntryGroups.value.every(group => !isEntryCollapsed(group.key)))
function toggleAllEntries() {
  const keys = visibleEntryGroups.value.map(group => group.key)
  if (hasFilters.value) filterCollapsedKeys.value = new Set(allEntriesExpanded.value ? keys : [])
  else expandedEntryKeys.value = new Set(allEntriesExpanded.value ? [] : keys)
}
function clearFilters() { search.value='';statusFilter.value=null;httpsFilter.value=null }
function openQuickCreate() {
  openCreateWithEntry({listen_port:nginxHttpsPort.value,https_enabled:true,http_redirect:true,listen_ipv4:true,listen_ipv6:false})
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


const activeSecurityFeatures = computed(() => {
  const tags: string[] = []
  if (form.ip_blacklist_mode) tags.push('黑名单')
  if (form.ip_whitelist_mode) tags.push('白名单')
  if (form.china_only) tags.push('大陆 IP')
  if (form.basic_auth_enabled) tags.push('Auth')
  if (form.rate_limit_enabled) tags.push('限流')
  if (form.conn_limit_enabled) tags.push('连接限制')
  if (form.security_headers || form.safe_headers) tags.push('响应头')
  if (form.block_sensitive_paths || form.max_body_mb>0 || form.allowed_methods.length) tags.push('请求防护')
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
  if (form.security_headers || form.safe_headers) expanded.add('headers')
  if (form.block_sensitive_paths || form.max_body_mb>0 || form.allowed_methods.length) expanded.add('request')
  if (form.proxy_ssl_verify_off || form.proxy_host_upstream || form.tls_min_13_only) expanded.add('advanced')
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
  if (sec.security_headers || sec.safe_headers) tags.push('响应头')
  if (sec.block_sensitive_paths || sec.max_body_mb || sec.allowed_methods?.length) tags.push('请求防护')
  return tags
}

const selectedSecurityFeatures = computed(() =>
  selectedRule.value ? securityFeatureLabels(selectedRule.value) : [],
)

const trafficChart = computed(() => {
  const now = Date.now()
  const labels: string[] = []
  const upload: (number|null)[] = []
  const download: (number|null)[] = []
  for (let i = 29; i >= 0; i--) {
    const bucketEnd = now - i * 10 * 1000
    const bucketStart = bucketEnd - 10 * 1000
    const points = rateHistory.value.filter((p) => p.at > bucketStart && p.at <= bucketEnd)
    const avg = (key: 'upload' | 'download') =>
      points.length ? points.reduce((sum, p) => sum + p[key], 0) / points.length : null
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
  copySource.value=null
  draftBusinessGroup.value=presentation.value==='service'?Math.max(0,activeBusinessGroup.value):0
  editingEntryId.value = null
  editing.value = null
  formTab.value = 'basic'
  resetForm()
  applyEntryListen(form, listen)
  formEntryId.value = entryId ?? null
  securityExpanded.value = ['ip']
  initializeEditor()
  void loadDdnsHostOptions()
  showModal.value = true
}


function openEntryCreate() {
  editing.value = null
  editingEntryId.value = null
  cloningSourceId.value = null
  Object.assign(entryForm, {
    name: '',
    listen_port: 443,
    listen_ipv4: true,
    listen_ipv6: false,
    https_enabled: true,
    http_redirect: true,
  })
  showEntryModal.value = true
}

function openEntryEdit(group: ProxyEntryGroup) {
  if (!group.entryId) return
  editing.value = null
  cloningSourceId.value = null
  editingEntryId.value = group.entryId
  const stored = entries.value.find((entry) => entry.id === group.entryId)
  Object.assign(entryForm, {
    name: stored?.name?.trim() ?? '',
    listen_port: group.listen.listen_port,
    listen_ipv4: group.listen.listen_ipv4,
    listen_ipv6: group.listen.listen_ipv6,
    https_enabled: group.listen.https_enabled,
    http_redirect: group.listen.http_redirect,
  })
  showEntryModal.value = true
}

function suggestClonePort(start: number): number {
  const used = new Set(entries.value.map((entry) => entry.listen_port))
  let port = start + 1
  while (port <= 65535 && used.has(port)) {
    port++
  }
  return port > 65535 ? start : port
}

function entryPortConflict(port: number, excludeId?: number | null): string | null {
  const other = entries.value.find(
    (entry) => entry.listen_port === port && entry.id !== (excludeId ?? undefined),
  )
  if (!other) return null
  const label = other.name?.trim() || `端口 ${other.listen_port}`
  return `监听端口 ${port} 已被入口「${label}」占用，请更换端口`
}

async function saveEntry() {
  if(savingEntry.value)return
  if (!entryForm.listen_ipv4 && !entryForm.listen_ipv6) {
    message.error('至少需要启用 IPv4 或 IPv6 监听')
    return
  }
  const port = entryForm.listen_port
  if (!port || port < 1 || port > 65535) {
    message.error('监听端口无效')
    return
  }
  const portErr = entryPortConflict(port, editingEntryId.value)
  if (portErr) {
    message.warning(portErr)
    return
  }
  savingEntry.value = true
  try {
    const payload = {
      name: entryForm.name.trim(),
      listen_port: port,
      listen_ipv4: entryForm.listen_ipv4,
      listen_ipv6: entryForm.listen_ipv6,
      https_enabled: entryForm.https_enabled,
      http_redirect: entryForm.http_redirect,
    }
    if (cloningSourceId.value) {
      const cloned = await api.cloneProxyEntry(cloningSourceId.value, payload)
      const count = cloned.rule_count ?? cloneRuleCount.value
      message.success(`入口已复制，${count} 条规则已同步创建`)
    } else if (editingEntryId.value) {
      await api.updateProxyEntry(editingEntryId.value, payload)
      message.success('入口已保存')
    } else {
      await api.createProxyEntry(payload)
      message.success('入口已创建')
    }
    showEntryModal.value = false
    editingEntryId.value = null
    await load()
  } catch (error) {
    const msg = error instanceof Error ? error.message : '保存失败'
    const partialPrefixes = ['入口已保存', '入口已创建', '入口已删除', '入口已复制']
    if (partialPrefixes.some((prefix) => msg.startsWith(prefix))) {
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

function duplicateEntry(group: ProxyEntryGroup) {
  if (!group.entryId) return
  editingEntryId.value = null
  cloningSourceId.value = group.entryId
  cloneRuleCount.value = group.rules.length
  Object.assign(entryForm, {
    name: group.name ? `${group.name} 副本` : '',
    listen_port: suggestClonePort(group.listen.listen_port),
    listen_ipv4: group.listen.listen_ipv4,
    listen_ipv6: group.listen.listen_ipv6,
    https_enabled: group.listen.https_enabled,
    http_redirect: group.listen.http_redirect,
  })
  showEntryModal.value = true
  message.info('将连同该入口下的全部规则一起复制，请确认监听端口')
}

function confirmDeleteEntry(group: ProxyEntryGroup) {
  if (!group.entryId) return
  const title = entryGroupDisplayName(group)
  dialog.error({
    title: `确定删除入口「${title}」？`,
    content: `将删除该入口下的 ${group.rules.length} 条规则，相关域名将停止反向代理。`,
    positiveText: '删除',
    negativeText: '取消',
    positiveButtonProps: { type: 'error' },
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

function buildDiscoveryRows(items: DiscoveredService[]): DiscoveryRow[] {
  return items
    .filter((service) => service.detected)
    .map((service, idx) => ({
      id: `${service.platform}-${service.port}-${idx}`,
      selected: false,
      service,
      ruleName: service.name,
      hostname: '',
    }))
}

function openDiscoveryModal(group?: ProxyEntryGroup) {
  discoveryBusinessGroup.value=presentation.value==='service'?Math.max(0,activeBusinessGroup.value):0
  discoveryHttpsEnabled.value = group?.listen.https_enabled ?? true
  discoveryListenPort.value = group?.listen.listen_port ?? nginxHttpsPort.value
  discoveryHost.value = '127.0.0.1'
  discoveryRows.value = []
  discoveryError.value=''
  discoveryScanned.value=false
  showDiscoveryModal.value = true
  void runDiscoveryScan()
}

function patchDiscoveryRow(id: string, patch: Partial<DiscoveryRow>) {
  const row = discoveryRows.value.find(row => row.id === id)
  if (row && !discoveryAdding.value && !discoveryLoading.value) Object.assign(row, patch)
}

function onDiscoverySelectAll(checked: boolean) {
  if (discoveryLoading.value||discoveryAdding.value) return
  for (const row of discoveryRows.value) {
    row.selected = checked
  }
}

async function runDiscoveryScan() {
  if (discoveryLoading.value||discoveryAdding.value) return
  discoveryError.value=''
  const host = discoveryHost.value.trim() || '127.0.0.1'
  discoveryScanHost.value = host
  discoveryLoading.value = true
  const loadingMsg = message.loading(`正在扫描 ${host} …`, { duration: 0 })

  try {
    const items = asList(await api.scanDiscovery(host))
    const previous=new Map(discoveryRows.value.map(row=>[row.service.upstream,row]))
    discoveryRows.value = buildDiscoveryRows(items).map(row=>{const old=previous.get(row.service.upstream);return old?{...row,selected:old.selected,ruleName:old.ruleName,hostname:old.hostname}:row})
    discoveryScanned.value=true
    loadingMsg.destroy()
    if (discoveryRows.value.length === 0) {
      message.info(`扫描完成：${host} 未发现可添加的 HTTP 服务`)
    } else {
      message.success(`扫描完成：发现 ${discoveryRows.value.length} 个 HTTP 服务`)
    }
  } catch (error) {
    loadingMsg.destroy()
    discoveryError.value=error instanceof Error?error.message:'扫描失败'
    message.error(discoveryError.value)
  } finally {
    discoveryLoading.value = false
  }
}

async function submitDiscovery() {
  if(discoveryAdding.value||discoveryLoading.value)return
  discoveryError.value=''
  if(discoveryScanHost.value!==discoveryHost.value.trim()&&discoveryHost.value.trim()){discoveryError.value='扫描地址已变更，请先重新扫描';return}
  const selected = discoveryRows.value.filter(row=>row.selected)
  if(selected.some(row=>!row.hostname.trim())){discoveryError.value='请为每个已选服务填写前端域名';return}
  if (selected.length === 0) {
    message.warning('请至少选择一项并填写前端域名')
    return
  }

  discoveryAdding.value = true
  try {
    if(!Number.isInteger(discoveryListenPort.value)||discoveryListenPort.value<1||discoveryListenPort.value>65535)throw new Error('监听端口必须在 1–65535 范围内')
    const seen = new Set<string>()
    const create = selected.map(row => {
      const hosts=normalizeFrontendHostValues([row.hostname.trim()])
      const host=hosts[0]
      if (/(^|\.)example\.(com|net|org)(:\d+)?$/i.test(host)) throw new Error('请填写实际域名，不要使用示例域名')
      const key=hostBindingKey(parseHostname(host),effectiveListenPort(host,discoveryListenPort.value))
      if(seen.has(key))throw new Error(`前端域名 ${host} 重复`)
      seen.add(key)
      const conflict=findHostConflict(hosts,discoveryListenPort.value)
      if(conflict)throw new Error(formatHostConflictMessage(conflict))
      const payload: ProxySavePayload = {
        upstream: row.service.upstream,
        hosts,
        name: row.ruleName.trim() || defaultRuleNameForHost(row.hostname) || row.service.name,
        enabled: true,
      }
      payload.listen_port=discoveryListenPort.value
      payload.https_enabled=discoveryHttpsEnabled.value
      payload.http_redirect=discoveryHttpsEnabled.value
      payload.listen_ipv4=true
      payload.listen_ipv6=false
      return payload
    })
    const saved=await api.saveProxyBatch({ create })
    if(saved.length!==create.length)throw new Error('服务端未返回完整的创建结果，请刷新规则列表核对后重试')
    await assignSavedRules(saved,discoveryBusinessGroup.value)
    message.success(`已添加 ${selected.length} 条规则`)
    showDiscoveryModal.value = false
    await load()
  } catch (error) {
    discoveryError.value=error instanceof Error?error.message:'添加失败'
    message.error(discoveryError.value)
  } finally {
    discoveryAdding.value = false
  }
}

watch(entryGroups, (groups) => {
  const validKeys = new Set(groups.map((group) => group.key))
  expandedEntryKeys.value = new Set([...expandedEntryKeys.value].filter((key) => validKeys.has(key)))
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

function openDetail(rule: ProxyRule, tab: 'overview' | 'diagnostics' | 'logs' | 'nginx' = 'overview') {
  selectedRuleId.value = rule.id
  detailTab.value = tab
  showDetailPanel.value = true
  clearRateHistory()
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

async function switchDetailTab(tab: 'overview' | 'diagnostics' | 'logs' | 'nginx') {
  if (tab === detailTab.value) return
  detailTab.value = tab
  if (tab === 'nginx') {
    await loadRuleNginxPreview()
  }
}

async function switchFormTab(tab:EditorTab) {
  if(tab===formTab.value||saving.value||nginxSaving.value)return
  formTab.value=tab
  if(tab==='nginx'&&editing.value&&nginxLoadedFor.value!==editing.value.id)await loadRuleNginxForEdit(editing.value.id)
}
function closeModal(){
  if(saving.value||nginxSaving.value)return
  if(ruleDraftDirty.value||nginxDirty.value){
    dialog.warning({title:'放弃未保存的配置？',content:'当前表单或 Nginx 文本尚未保存，关闭后将丢失。',positiveText:'放弃更改',negativeText:'继续编辑',onPositiveClick:finishEditor})
  }else finishEditor()
}

let detailRequest=0
async function loadRuleNginxPreview() {
  const rule = selectedRule.value
  if (!rule||!showDetailPanel.value||showModal.value) return
  const request=++detailRequest
  nginxLoading.value = true
  try {
    const view = await api.getProxyNginx(rule.id)
    if(request!==detailRequest||!showDetailPanel.value||selectedRuleId.value!==rule.id||showModal.value)return
    nginxServerMode.value = view.mode
    nginxGenerated.value = view.generated
    nginxEnabled.value = view.enabled
    nginxDraft.value = view.mode === 'custom' ? view.content : view.generated
  } catch (error) {
    if(request===detailRequest&&showDetailPanel.value)message.error(error instanceof Error ? error.message : '加载 Nginx 配置失败')
  } finally {
    if(request===detailRequest&&!showModal.value)nginxLoading.value = false
  }
}

async function loadRuleNginxForEdit(ruleId: number) {
  const request=++editorRequest
  nginxLoading.value = true
  try {
    const view = await api.getProxyNginx(ruleId)
    if(request!==editorRequest||editing.value?.id!==ruleId||!showModal.value)return
    nginxLoadedFor.value=ruleId
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
    if(request===editorRequest)message.error(error instanceof Error ? error.message : '加载 Nginx 配置失败')
  } finally {
    if(request===editorRequest)nginxLoading.value = false
  }
}

function scheduleNginxRefresh() {
  if (detailTab.value !== 'nginx' || showDetailPanel.value === false) return
  if (nginxRefreshTimer) clearTimeout(nginxRefreshTimer)
  nginxRefreshTimer = setTimeout(() => {
    void loadRuleNginxPreview()
  }, 300)
}

async function setNginxEditMode(mode: 'auto' | 'custom') {
  if (mode === nginxEditMode.value||nginxSaving.value||nginxLoading.value) return
  if(mode==='auto'&&nginxDirty.value&&!(await confirmDiscardNginxDraft()))return
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
  if (!ruleId||nginxSaving.value||nginxLoading.value) return
  if(ruleDraftDirty.value){message.warning('请先保存规则配置，再编辑 Nginx 文本');return}
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
  if (!ruleId||nginxSaving.value||nginxLoading.value||ruleDraftDirty.value) return
  if(!(await confirmDiscardNginxDraft()))return
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
  if(nginxSaving.value||nginxLoading.value||ruleDraftDirty.value)return
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
  ++detailRequest
  if(nginxRefreshTimer){clearTimeout(nginxRefreshTimer);nginxRefreshTimer=null}
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

function hostsToText(rule: ProxyRule): string {
  return ruleHosts(rule).join('\n')
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

  const defaultPort = rule.https_enabled ? 443 : 80
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
  if (await copyToClipboard(url)) {
    message.success('链接已复制')
  } else {
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
  /** 入口分组内表格：链接文案仅显示域名，完整 URL 保留在 href/title */
  displayHostOnly?: boolean
}

function accessLinkPairs(rule: ProxyRule, options: DomainLinkOptions = {}) {
  return ruleHosts(rule).map((host) => {
    const href = options.hideListenPort
      ? hostAccessUrlWithoutListenPort(rule, host)
      : hostAccessUrl(rule, host)
    let display = href
    if (options.displayHostOnly) {
      display = host
    } else if (options.hideListenPort) {
      display = hostAccessUrlWithoutListenPort(rule, host)
    }
    return { href, display }
  })
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

function defaultRuleNameForHost(host: string): string {
  const trimmed = host.trim()
  if (!trimmed) return ''
  return parseHostname(trimmed) || trimmed
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

function formatHostConflictMessage(conflict: { host: string; port: number; rule: ProxyRule }): string {
  const label = ruleName(conflict.rule)
  const status = conflict.rule.enabled ? '运行中' : '已停用'
  return (
    `域名 ${conflict.host}:${conflict.port} 已被规则「${label}」占用（${status}）。` +
    `同一监听端口下每个域名只能对应一条规则：若要改反代目标请编辑占用该域名的那一行；` +
    `若仅需修改名称或上游，请保持「前端域名」不变。`
  )
}

function listenPortForConflictCheck(payload: ProxySavePayload): number {
  if (payload.listen_port != null && payload.listen_port > 0) return payload.listen_port
  if (formEntryId.value != null) {
    const group = entryGroups.value.find((g) => g.entryId === formEntryId.value)
    if (group) return group.listen.listen_port
  }
  return form.listen_port
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
    if (excludeRuleId != null && rule.id === excludeRuleId) continue
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
    tags.push(h(NTag, { size: 'small', type: 'success', bordered: false }, () => 'HTTPS'))
  } else {
    tags.push(h(NTag, { size: 'small', bordered: false }, () => 'HTTP'))
  }
  return h('div', { class: 'proto-tags' }, tags)
}

function rowProps(row: ProxyRule) {
  const classes: string[] = []
  if (showDetailPanel.value && selectedRuleId.value === row.id) {
    classes.push('proxy-row--active')
  }
  if (!row.enabled) {
    classes.push('proxy-row--disabled')
  }
  return {
    class: classes.join(' '),
    'data-rule-id': String(row.id),
  }
}

function renderRuleEnabledSwitch(row: ProxyRule): VNode {
  return h(
    'div',
    {
      class: 'proxy-enable-cell',
      onClick: (e: Event) => e.stopPropagation(),
    },
    [
      h(NSwitch, {
        value: row.enabled,
        size: 'small',
        loading: togglingRuleId.value === row.id,
        onUpdateValue: (enabled: boolean) => toggleRuleEnabled(row, enabled),
        'aria-label': row.enabled ? '停用规则' : '启用规则',
      }),
    ],
  )
}

function renderServiceCell(row: ProxyRule, nested: boolean): VNode {
  return h('div',{class:'proxy-service-cell'},[
    h(ServiceFavicon,{ruleId:row.id}),
    h('div',{class:'proxy-service-cell__content'},[
      h('button',{type:'button',class:'proxy-service-name',title:ruleName(row),onClick:()=>openDetail(row)},ruleName(row)),
      renderDomainAccessLinks(row,{displayHostOnly:true,showWhenSingleHost:true}),
      !nested && h('span',{class:'proxy-service-entry'},ruleEntryNames.value.get(row.id) ?? listenLabel(row)),
    ]),
  ])
}
function renderTrafficCell(row:ProxyRule):VNode {
  const traffic=trafficByRule.value[row.id]
  return h('div',{class:'proxy-cell-lines proxy-traffic-cell'},[
    h('span',{title:'当前上传 / 下载'},'↑ '+formatRate(traffic?.upload_rate ?? 0)+'  ↓ '+formatRate(traffic?.download_rate ?? 0)),
    h('span',{class:'proxy-cell-muted',title:'累计上传 / 下载'},'↑ '+formatBytes(traffic?.upload_total ?? 0)+'  ↓ '+formatBytes(traffic?.download_total ?? 0)),
    h('span',{class:'proxy-cell-muted'},'近期访问 IP '+String(traffic?.connections ?? 0)),
  ])
}
function renderSecurityCell(row:ProxyRule):VNode {
  const tags=securityTags(row)
  return h('div',{class:'proxy-security-tags'},tags.length?tags.map(tag=>h(NTag,{size:'small',bordered:false},{default:()=>tag})): [h('span',{class:'proxy-cell-muted'},'—')])
}
function renderActions(row:ProxyRule):VNode {
  return renderTableRowActions([
    {label:'详情',icon:InformationCircleOutline,onClick:()=>openDetail(row)},
    {label:'编辑',icon:CreateOutline,type:'primary',onClick:()=>openEdit(row)},
    {label:'复制',icon:CopyOutline,onClick:()=>openDuplicate(row)},
    {label:'删除',icon:TrashOutline,type:'error',onClick:()=>confirmDelete(row)},
  ])
}
function makeColumns(nested:boolean, sortable:boolean):DataTableColumns<ProxyRule>{
  const cols:DataTableColumns<ProxyRule>=[]
  if(sortable)cols.push({title:'',key:'sort',width:30,render:()=>h('span',{class:'proxy-drag-handle',title:'拖动排序'},[h(NIcon,{component:ReorderThreeOutline,size:16})])})
  if(compactTable.value){
    cols.push({title:'服务 / 域名 / 目标',key:'service',render:row=>h('div',{class:'proxy-mobile-service'},[
      renderServiceCell(row,nested),
      h('div',{class:'proxy-mobile-meta'},[h('span',{class:'proxy-target',title:row.upstream},row.upstream),renderProtocol(row),h('span',{class:'proxy-cell-muted'},listenLabel(row))]),
      renderSecurityCell(row),renderTrafficCell(row)
    ])},
    {title:'状态 / 操作',key:'actions',width:116,render:row=>h('div',{class:'proxy-mobile-actions'},[renderRuleEnabledSwitch(row),renderActions(row)])})
    return cols
  }
  cols.push(
    {title:'服务 / 前端域名',key:'name',minWidth:190,render:row=>renderServiceCell(row,nested)},
    {title:'目标地址',key:'upstream',minWidth:130,render:row=>h('div',{class:'proxy-cell-lines'},[h('span',{class:'proxy-target',title:row.upstream},row.upstream),...(!nested?[h('span',{class:'proxy-cell-muted'},listenLabel(row))]:[])])},
    {title:'协议 / 安全',key:'security',width:112,render:row=>h('div',{class:'proxy-cell-lines'},[renderProtocol(row),renderSecurityCell(row)])},
    {title:'流量',key:'traffic',width:160,render:renderTrafficCell},
    {title:'启用',key:'enabled',width:62,render:renderRuleEnabledSwitch},
    {title:'操作',key:'actions',width:132,render:renderActions},
  )
  return cols
}
function entryGroupColumnsFor(group:ProxyEntryGroup):DataTableColumns<ProxyRule>{
  return makeColumns(true,canReorderInGroups.value && group.rules.length>1)
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

function clearLogLines() {
  pausedLines.value=[]
  clearStreamLines()
  logStickToBottom.value = true
}

watch([showDetailPanel, selectedRuleId, detailTab], async ([visible, id, tab]) => {
  stopLogStream()
  logPaused.value=false;pausedLines.value=[];logQuery.value='';logStatus.value='all'
  if (!visible || !id || tab !== 'logs') return
  clearLogLines()
  startLogStream(id)
  await nextTick()
  scrollLogToBottom(true)
})

watch(
  () => visibleLogLines.value,
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

let trafficPending = false
async function refreshTraffic() {
  if (document.hidden || trafficPending) return
  trafficPending = true
  try {
    const snapshot=await api.getProxyTrafficSnapshot()
    if(!viewAlive)return
    const rows = asList(snapshot.rules)
    const next: Record<number, ProxyTraffic> = {}
    for (const row of rows) {
      next[row.rule_id] = row
    }
    trafficByRule.value = next
    trafficSnapshot.value=snapshot
    trafficError.value=''
    if (showDetailPanel.value && selectedRuleId.value) {
      recordRateSample(selectedRuleId.value)
    }
  } catch(error) {
    if(viewAlive)trafficError.value=error instanceof Error?error.message:'流量采集请求失败'
  } finally {
    trafficPending = false
  }
}

async function refreshAll() {
  await Promise.all([load(), refreshTraffic()])
}

function startTrafficPoll() {
  trafficPoll.start()
}

function stopTrafficPoll() {
  trafficPoll.stop()
}

async function loadClients() {
 const rule=selectedRule.value
 if(!rule||document.hidden||clientsLoading.value)return
 const c=new AbortController();clientsController=c
 const g=++clientsGeneration;clientsLoading.value=true
 try{
  const rows=asList(await api.getProxyClients(rule.id,c.signal))
  if(g===clientsGeneration&&showDetailPanel.value&&selectedRuleId.value===rule.id&&detailTab.value==='overview'){clientRows.value=rows;clientsError.value=''}
 }catch(e){if(g===clientsGeneration&&!c.signal.aborted)clientsError.value=e instanceof Error?e.message:'读取客户端失败'}
 finally{if(g===clientsGeneration)clientsLoading.value=false}
}
function startClientsPoll(){clientsPoll.start(true);void loadClients()}
function stopClientsPoll(){clientsPoll.stop();++clientsGeneration;clientsController?.abort();clientsLoading.value=false}
watch([showDetailPanel,selectedRuleId,detailTab],([visible,id,tab],[,previousId])=>{
 stopClientsPoll()
 if(!visible||id!==previousId){clientRows.value=[];clientsError.value=''}
 if(visible&&id&&tab==='overview')startClientsPoll()
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

const NginxCodeEditor = defineAsyncComponent(() => import('../components/NginxCodeEditor.vue'))

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
    await loadBusiness()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.listen_port = defaultListenPort(true)
  form.listen_ipv4 = true
  form.listen_ipv6 = false
  form.hostsSelected = []
  form.upstream = ''
  form.https_enabled = true
  form.http_redirect = true
  form.enabled = true
  form.name = ''
  Object.assign(form, defaultSecurityForm())
}

function loadSecurityToForm(rule?: ProxyRule | null) {
  Object.assign(form,defaultSecurityForm())
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
  form.safe_headers = sec.safe_headers ?? false
  form.block_sensitive_paths = sec.block_sensitive_paths ?? false
  form.max_body_mb = sec.max_body_mb ?? 0
  form.allowed_methods = [...(sec.allowed_methods ?? [])]
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
    safe_headers: form.safe_headers,
    block_sensitive_paths: form.block_sensitive_paths,
    max_body_mb: form.max_body_mb,
    allowed_methods: form.allowed_methods,
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
  if(!form.rate_limit_enabled){form.rate_limit_enabled=true;form.rate_limit_rate=10;form.rate_limit_burst=20}
  form.safe_headers=true
  formTab.value='security'
  securityExpanded.value=['traffic','headers']
  message.success('已启用请求限流及基础安全响应头；其他配置保持不变')
}

function securityTags(rule: ProxyRule): string[] {
  const sec = rule.security ?? {}
  const tags: string[] = []
  if (sec.basic_auth?.enabled) tags.push('Auth')
  if ((sec.ip_blacklist?.length ?? 0) > 0 || sec.ip_whitelist_mode) tags.push('IP')
  if (sec.china_only) tags.push('CN')
  if (sec.rate_limit?.enabled || sec.conn_limit?.enabled) tags.push('Limit')
  if(sec.block_sensitive_paths || sec.max_body_mb || sec.allowed_methods?.length) tags.push('防护')
  if(sec.safe_headers || sec.security_headers) tags.push('响应头')
  return tags
}

function buildPayload(): ProxySavePayload {
  const hosts = normalizeFrontendHostValues(formHostsList(form.hostsSelected))
  if (hosts.length === 0) {
    throw new Error('至少需要一个前端域名')
  }
  if (!form.listen_ipv4 && !form.listen_ipv6) {
    throw new Error('至少需要启用 IPv4 或 IPv6 监听')
  }
  if(form.basic_auth_enabled&&(!editing.value?.security?.basic_auth?.has_password||hosts.length>1)&&!form.basic_auth_password.trim())throw new Error('启用访问认证的新增规则需要设置独立密码')
  const upstream = validateUpstreamInput(form.upstream)
  return {
    upstream,
    listen_port: form.listen_port,
    listen_ipv4: form.listen_ipv4,
    listen_ipv6: form.listen_ipv6,
    https_enabled: form.https_enabled,
    http_redirect: form.http_redirect,
    hosts,
    enabled: form.enabled,
    name: form.name.trim(),
    security: buildSecurityPayload(),
  }
}

async function toggleRuleEnabled(row: ProxyRule, enabled: boolean) {
  if (togglingRuleId.value !== null || row.enabled === enabled) return
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
  copySource.value=rule
  draftBusinessGroup.value=ruleBusinessGroup(rule.id)
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

  form.hostsSelected=[]
  Object.assign(form, {
    upstream: rule.upstream,
    enabled: rule.enabled,
    name: rule.name ?? '',
  })
  loadSecurityToForm(rule)
  // A copy is a new independent rule. Never inherit credentials.
  form.basic_auth_enabled=false
  form.basic_auth_username=''
  form.basic_auth_password=''
  syncSecurityExpanded()
  formTab.value = 'basic'
  initializeEditor()
  void loadDdnsHostOptions()
  showModal.value = true
}

function openEdit(rule: ProxyRule, tab: 'basic' | 'security' | 'nginx' = 'basic') {
  copySource.value=null
  draftBusinessGroup.value=ruleBusinessGroup(rule.id)
  closeDetail()
  editingEntryId.value = null
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

  setFormHostsFromText(form.hostsSelected, hostsToText(rule))
  Object.assign(form, {
    upstream: rule.upstream,
    enabled: rule.enabled,
    name: rule.name ?? '',
  })
  loadSecurityToForm(rule)
  syncSecurityExpanded()
  if (tab === 'nginx') {
    formTab.value = 'nginx'
  } else if (tab === 'security') {
    formTab.value = 'security'
  } else {
    formTab.value = 'basic'
  }
  initializeEditor()
  void loadDdnsHostOptions()
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
    // 与服务器再同步一次，确保展示顺序与持久化结果一致
    await load()
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

function attachSortable(tbody: HTMLElement, onEnd: (orderedIds: number[]) => void) {
  rowSortables.push(
    Sortable.create(tbody, {
      handle: '.proxy-drag-handle',
      animation: 150,
      draggable: '.n-data-table-tr',
      onEnd: async (evt) => {
        if (evt.oldIndex == null || evt.newIndex == null || evt.oldIndex === evt.newIndex || reordering.value) {
          return
        }
        // 以 DOM 实际行顺序为准：traffic 轮询重渲染时 evt.oldIndex/newIndex 可能与数据索引错位
        const orderedIds = Array.from(evt.from.querySelectorAll<HTMLElement>('tr[data-rule-id]'))
          .map((el) => Number(el.dataset.ruleId))
          .filter((id) => Number.isFinite(id))
        onEnd(orderedIds)
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
    await load()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '入口排序保存失败')
    await load()
  } finally {
    reordering.value = false
  }
}

async function setupEntryStackSortable(generation: number) {
  destroyEntryStackSortable()
  if (!canReorderEntries.value) return
  await nextTick()
  const stack = entryStackRef.value
  if (!stack || generation!==sortableGeneration || !canReorderEntries.value) return

  entryStackSortable = Sortable.create(stack, {
    handle: '.proxy-entry-drag-handle',
    animation: 150,
    draggable: '.proxy-entry-card',
    onEnd: (evt) => {
      if (evt.oldIndex == null || evt.newIndex == null || evt.oldIndex === evt.newIndex || reordering.value) {
        return
      }
      // 以 DOM 实际卡片顺序为准
      const orderedIds = Array.from(stack.querySelectorAll<HTMLElement>('.proxy-entry-card[data-entry-id]'))
        .map((el) => Number(el.dataset.entryId))
        .filter((id) => Number.isFinite(id))
      const byId = new Map(entryGroups.value.map((group) => [group.entryId, group]))
      const groups = orderedIds.map((id) => byId.get(id)).filter((group): group is ProxyEntryGroup => group != null)
      if (groups.length !== entryGroups.value.length) return
      void persistEntryOrder(groups)
    },
  })
}

let sortableGeneration = 0
async function setupRowSortable() {
  const generation = ++sortableGeneration
  destroyRowSortable()
  destroyEntryStackSortable()
  if (!canReorder.value && !canReorderEntries.value) return
  await nextTick()
  if(generation!==sortableGeneration)return

  if (!canReorderInGroups.value) { await setupEntryStackSortable(generation); return }

  for (const group of entryGroups.value) {
    if (group.rules.length < 2 || isEntryCollapsed(group.key)) continue
    const body = entryStackRef.value?.querySelector(
      `.proxy-entry-card__body[data-entry-key="${CSS.escape(group.key)}"] .n-data-table-tbody`,
    ) as HTMLElement | null
    if (!body) continue
    const groupKey = group.key
    attachSortable(body, (orderedIds) => {
      const groupRules = entryGroups.value.find((item) => item.key === groupKey)?.rules ?? []
      if (orderedIds.length !== groupRules.length) return
      const byId = new Map(groupRules.map((rule) => [rule.id, rule]))
      const nextGroupRules = orderedIds.map((id) => byId.get(id)).filter((rule): rule is ProxyRule => rule != null)
      if (nextGroupRules.length !== groupRules.length) return
      void persistRuleOrder(mergeGroupOrder(groupKey, nextGroupRules))
    })
  }

  await setupEntryStackSortable(generation)
}

watch([presentation,canReorder, canReorderInGroups, canReorderEntries, () => rules.value.length, entryGroups, expandedEntryKeys, compactTable,entryStackRef], () => {
  void setupRowSortable()
})

watch(showModal, (open) => {
  if (open) {
    void loadEditorCertificates()
    void loadDdnsHostOptions()
  } else {
    ++editorCertificateGeneration
    editorCertificatesLoading.value=false
    editing.value = null
    formEntryId.value = null
  }
})

watch(showEntryModal, (open) => {
  if (!open) {
    editingEntryId.value = null
    cloningSourceId.value = null
  }
})

function payloadForSingleHost(base: ProxySavePayload, host: string): ProxySavePayload {
  const customName = base.name?.trim() ?? ''
  return {
    ...base,
    hosts: [host],
    name: customName || defaultRuleNameForHost(host),
  }
}

async function save() {
  if(saving.value||nginxSaving.value)return
  saveError.value=''
  if(nginxDirty.value&&!(await confirmDiscardNginxDraft()))return
  if(saving.value||nginxSaving.value)return
  saving.value = true
  try {
    const basePayload = buildPayload()
    const hosts = basePayload.hosts
    const listenPort = listenPortForConflictCheck(basePayload)
    const excludeRuleId = editing.value?.id

    for (const host of hosts) {
      const conflict = findHostConflict([host], listenPort, excludeRuleId)
      if (conflict) {
        saveError.value=formatHostConflictMessage(conflict)
        formTab.value='basic'
        message.error(saveError.value)
        return
      }
    }

    if (editing.value) {
      const saved=await api.saveProxyBatch({
        update_id: editing.value.id,
        update: payloadForSingleHost(basePayload, hosts[0]),
        create: hosts.slice(1).map(host => payloadForSingleHost(basePayload, host)),
      })
      if(saved.length!==hosts.length||!saved.some(r=>r.id===editing.value?.id))throw new Error('服务端未返回完整保存结果，请刷新后核对')
      const currentId=editing.value.id
      if(draftBusinessGroup.value!==ruleBusinessGroup(currentId)){
        if(!(await changeBusiness({action:'assign',id:draftBusinessGroup.value,rule_ids:saved.map(r=>r.id)})))message.warning('规则已保存，但业务分组关联失败')
      }else await assignSavedRules(saved.filter(r=>r.id!==currentId),draftBusinessGroup.value)
      message.success(hosts.length > 1 ? `已保存 1 条并新增 ${hosts.length - 1} 条规则` : '规则已保存')
    } else {
      const saved=await api.saveProxyBatch({ create: hosts.map(host => payloadForSingleHost(basePayload, host)) })
      if(saved.length!==hosts.length)throw new Error('服务端未返回完整创建结果，请刷新后核对')
      await assignSavedRules(saved,draftBusinessGroup.value)
      message.success(hosts.length > 1 ? `已创建 ${hosts.length} 条规则` : '规则已创建')
    }
    finishEditor()
    await load()
  } catch (error) {
    const msg = error instanceof Error ? error.message : '保存失败'
    if (msg.startsWith('规则已保存')) {
      message.warning(msg)
      finishEditor()
      await load()
    } else {
      saveError.value=msg
      message.error(msg)
    }
  } finally {
    saving.value = false
  }
}

function confirmDelete(rule:ProxyRule){if(deleteBusy.value)return;deleteTarget.value=rule;deleteError.value=''}
async function deleteRule(){
  const rule=deleteTarget.value
  if(!rule||deleteBusy.value)return
  deleteBusy.value=true;deleteError.value=''
  try{
    await api.deleteProxy(rule.id)
    if(selectedRuleId.value===rule.id){closeDetail();selectedRuleId.value=null}
    rules.value=rules.value.filter(r=>r.id!==rule.id)
    deleteTarget.value=null
    message.success('规则已删除')
    await load()
  }catch(error){
    const msg=error instanceof Error?error.message:'删除失败'
    if(msg.startsWith('规则已删除')){deleteTarget.value=null;message.warning(msg);await load()}
    else{deleteError.value=msg;message.error(msg)}
  }finally{deleteBusy.value=false}
}

onMounted(async () => {
  await loadDefaults()
  await load()
  if(!viewAlive)return
  await refreshTraffic()
  if(viewAlive)startTrafficPoll()
})
onUnmounted(() => {
  viewAlive=false
  ++editorRequest;++detailRequest
  if(nginxRefreshTimer)clearTimeout(nginxRefreshTimer)
  layoutObserver?.disconnect()
  sortableGeneration++
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
  outline: 2px solid var(--fonu-brand);
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
  color: var(--fonu-brand-text);
}

.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn--accent:hover) {
  color: var(--fonu-brand-text);
  background: var(--fonu-brand-soft);
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

















































.entry-batch {
  margin-top: var(--fonu-space-3, 12px);
  padding-top: var(--fonu-space-3, 12px);
  border-top: 1px dashed var(--fonu-border, rgba(128, 128, 128, 0.35));
}

.entry-batch__title {
  font-size: 13px;
  font-weight: 600;
  opacity: 0.85;
  margin-bottom: var(--fonu-space-2, 8px);
}





.proxy-entry-card__body {
  padding: 0;
}

.proxy-table--nested {
  width: 100%;
}

.proxy-table--nested :deep(.n-data-table-wrapper) {
  padding: 0;
}

.proxy-table--nested :deep(.n-data-table-base-table) {
  width: 100%;
  table-layout: fixed;
}

.proxy-table--nested :deep(.n-data-table-th) {
  background: var(--fonu-bg, #f8fafc);
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.proxy-table--nested :deep(.n-data-table-td) {
  padding-top: 10px;
  padding-bottom: 10px;
}

.proxy-table--nested :deep(.domain-cell__main) {
  font-size: 13px;
}

.proxy-table--nested :deep(.domain-cell__link) {
  font-size: 13px;
  font-weight: 500;
  flex: 0 1 auto;
  max-width: calc(100% - 52px);
}

.proxy-table--nested :deep(.domain-cell__link-row) {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
}

.proxy-table--nested :deep(.domain-cell--compact) {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 0;
}

.proxy-table--nested :deep(.domain-cell__main--one-line) {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.35;
}

.proxy-table--nested :deep(.domain-cell__hosts-line) {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  line-height: 1.35;
}

.proxy-table--nested :deep(.domain-cell--hosts .domain-cell__links) {
  margin-top: 0;
}

.proxy-table--nested :deep(.domain-cell__fallback-host) {
  font-size: 12px;
}

.proxy-entry-card__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-5) var(--fonu-space-4);
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
  min-width: 0;
}

.proxy-table { width: 100%; }

.proxy-table :deep(.proxy-row--active td) {
  background: color-mix(in srgb, var(--fonu-brand) 8%, var(--fonu-surface, #fff));
}

.proxy-table :deep(.proxy-row--disabled td) {
  background: var(--fonu-surface, #fff);
}

.proxy-table :deep(.n-data-table-td--fixed-right),
.proxy-table :deep(.n-data-table-td--fixed-left) {
  background: var(--fonu-surface, #fff);
}

.proxy-table :deep(.n-data-table-th--fixed-right),
.proxy-table :deep(.n-data-table-th--fixed-left) {
  background: var(--fonu-bg, #f8fafc);
}

.proxy-table :deep(.n-data-table-tr:hover .n-data-table-td--fixed-right),
.proxy-table :deep(.n-data-table-tr:hover .n-data-table-td--fixed-left) {
  background: var(--fonu-bg, #f8fafc);
}

.proxy-table :deep(.proxy-row--active .n-data-table-td--fixed-right),
.proxy-table :deep(.proxy-row--active .n-data-table-td--fixed-left) {
  background: color-mix(in srgb, var(--fonu-brand) 8%, var(--fonu-surface, #fff));
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
  display: inline-flex;
  align-items: center;
  gap: 2px;
  max-width: 100%;
}

.proxy-table :deep(.domain-cell__link) {
  flex: 0 1 auto;
  max-width: calc(100% - 52px);
  min-width: 0;
  font-size: 12px;
  font-family: var(--fonu-mono);
  color: var(--fonu-brand-text);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.proxy-table :deep(.domain-cell__link:hover) {
  color: var(--fonu-brand-text);
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
  color: var(--fonu-brand-text);
  cursor: pointer;
  border-radius: 4px;
}

.proxy-table :deep(.domain-cell__link-action:hover) {
  color: var(--fonu-brand-text);
  background: var(--fonu-brand-soft);
}

.proxy-table :deep(.domain-cell__more) {
  align-self: flex-start;
  margin-top: 2px;
  padding: 0;
  border: none;
  background: none;
  font: inherit;
  font-size: 12px;
  color: var(--fonu-brand-text);
  cursor: pointer;
}

.proxy-table :deep(.domain-cell__more:hover) {
  color: var(--fonu-brand-text);
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
  color: var(--fonu-brand-text);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.domain-cell__popover-links :deep(.domain-cell__link:hover) {
  color: var(--fonu-brand-text);
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
  color: var(--fonu-brand-text);
  cursor: pointer;
  border-radius: 4px;
}

.domain-cell__popover-links :deep(.domain-cell__link-action:hover) {
  color: var(--fonu-brand-text);
  background: var(--fonu-brand-soft);
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
  border-color: color-mix(in srgb,var(--fonu-brand) 25%,var(--fonu-border));
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





@keyframes live-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
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

























/* naive-ui 首个 collapse-item 默认 padding-top: 0，导致首项偏矮 */






























.w-full {
  width: 100%;
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
  background: var(--fonu-bg);
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

.proxy-modal__help-ol--compact {
  font-size: 12px;
  line-height: 1.55;
  color: var(--fonu-text-secondary);
}

.proxy-modal__help-ol--compact li + li {
  margin-top: 6px;
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
  background: var(--fonu-brand-soft);
  color: var(--fonu-brand-text);
  font-size: 12px;
  line-height: 1.6;
}

.proxy-modal__tip-icon {
  flex-shrink: 0;
  margin-top: 1px;
  font-size: 16px;
}

.proxy-modal__tip--warn {
  background: color-mix(in srgb,var(--fonu-warning) 10%,var(--fonu-surface));
  color: var(--fonu-warning);
}







.proxy-secret-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

















.proxy-modal__pane :deep(.n-form-item .n-form-item-blank) {
  display: block;
  width: 100%;
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
}

@media (max-width: 767px) {
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
@media (max-width: 760px) {
}
.proxy-panel :deep(.fonu-card__header) { padding-bottom:12px; }
.proxy-entry-card { border-radius:var(--fonu-radius); }
.proxy-entry-card__head { background:var(--fonu-bg-muted); padding:10px 12px; }
.proxy-entry-card__running { color:var(--fonu-success); }
.proxy-entry-card__warn { color:var(--fonu-warning); }
.proxy-entry-card__head:focus-within .proxy-entry-card__intro:focus-visible { outline-color:var(--fonu-brand); }
.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn--accent) { color:var(--fonu-brand); }
.proxy-entry-card__actions :deep(.proxy-entry-card__action-btn--accent:hover) { color:var(--fonu-brand-hover); background:var(--fonu-brand-soft); }
@media(max-width:760px) { .proxy-entry-card__head { flex-wrap:wrap; } .proxy-entry-card__intro { flex:1 1 150px; } .proxy-entry-card__actions { max-width:100%; } }

.proxy-layout { width:100%; min-width:0; }
.proxy-entry-stack { padding:12px; gap:10px; }
.proxy-entry-card__body { min-width:0; }
.proxy-result-count { font-size:12px; color:var(--fonu-text-secondary); }
.proxy-entry-picker { display:grid; gap:6px; max-height:300px; overflow:auto; min-width:180px; }
.proxy-panel :deep(.fonu-card__header) { align-items:center; padding:12px 14px; }
.proxy-panel :deep(.fonu-card__body) { padding:0; }
@media(max-width:760px){.proxy-entry-stack{padding:8px;gap:8px}}

.business-error{margin-bottom:12px}
.port-workspace{display:grid;gap:10px;min-width:0}
.port-workspace__note{display:flex;flex-wrap:wrap;justify-content:space-between;gap:6px 12px;font-size:12px;color:var(--fonu-text-secondary)}
.port-workspace .proxy-panel{min-width:0}

.service-page-header{display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap;margin-bottom:16px;padding-bottom:12px;border-bottom:1px solid var(--fonu-border)}
.service-page-title,.service-page-actions{display:flex;align-items:center;gap:10px;flex-wrap:wrap;min-width:0}
.service-page-title{flex:1 1 260px}.service-page-title h1{font-size:25px;line-height:1.25;font-weight:650;margin:0;color:var(--fonu-text)}
.service-page-actions{justify-content:flex-end}
@media(max-width:760px){.service-page-header{gap:8px;margin-bottom:12px}.service-page-title h1{font-size:21px}.service-page-actions{justify-content:flex-start;gap:6px}}

.editor-tabs{display:flex;flex-wrap:wrap;gap:5px}.editor-tabs button{font:inherit;font-size:13px;border:0;background:transparent;border-radius:6px;padding:7px 12px;color:var(--fonu-text-secondary);cursor:pointer;display:flex;align-items:center;gap:5px}.editor-tabs button[aria-selected=true]{background:var(--fonu-surface);color:var(--fonu-brand-text);font-weight:600;box-shadow:0 1px 3px rgba(0,0,0,.08)}.editor-tabs button:focus-visible{outline:2px solid var(--fonu-brand)}.editor-count{font-size:11px}.editor-dirty-dot{width:6px;height:6px;background:var(--fonu-brand);border-radius:50%}.editor-footer-info{font-size:12px;color:var(--fonu-text-secondary);margin-right:auto;flex:1 1 150px}.editor-save-error{margin-top:12px}
@media(max-width:600px){.editor-tabs button{font-size:12px;padding:6px 8px}}

.proxy-detail-modal{width:100%;max-height:none;height:100%;border-radius:0;box-shadow:none;container:proxy-detail / inline-size}.proxy-detail-modal__header{padding:16px;flex-wrap:wrap;gap:10px}.proxy-detail-modal__intro{flex:1 1 200px;flex-wrap:wrap;gap:8px}.detail-title{min-width:0;flex:1 1 130px}.detail-title small{display:block;font-size:11px;color:var(--fonu-text-secondary);overflow-wrap:anywhere}.proxy-detail-modal__title{overflow-wrap:anywhere}.proxy-detail__scroll{min-height:0;overflow-y:auto;overscroll-behavior:contain}.proxy-detail__tabbar{padding-inline:16px}.proxy-detail__pane--logs,.proxy-detail__pane--nginx{min-height:350px;flex-shrink:0}.delete-review{display:grid;gap:12px}.delete-identity{display:flex;align-items:center;gap:10px}.delete-identity strong,.delete-identity small{display:block}.delete-identity small{font-size:12px;color:var(--fonu-text-secondary)}.delete-review dl{margin:0;display:grid;grid-template-columns:105px minmax(0,1fr);gap:8px;border:1px solid var(--fonu-border);border-radius:8px;padding:12px;font-size:13px}.delete-review dt{color:var(--fonu-text-secondary)}.delete-review dd{grid-column:2;margin:0;overflow-wrap:anywhere}
.log-filters{display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-bottom:8px}.log-filters :deep(.n-input){flex:1 1 140px;min-width:0}.log-filters :deep(.n-select){width:110px}.log-filters small{font-size:11px;color:var(--fonu-text-secondary)}.proxy-detail__tabbar{display:block;padding-top:0;border-bottom:0;min-width:0}.proxy-detail__tabbar :deep(.n-tabs){width:100%}
</style>
