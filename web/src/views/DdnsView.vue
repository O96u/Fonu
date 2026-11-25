<template>
  <PageHeader title="DDNS" description="通过映射公网 IP 到 DNS 解析，支持多个域名及服务商（含 DNSHE）">
    <template #actions>
      <n-button :loading="updatingAll" @click="updateAll">
        <template #icon><n-icon :component="RefreshOutline" /></template>
        全部刷新
      </n-button>
    </template>
  </PageHeader>

  <LoadError v-if="loadError" :message="loadError" @retry="init" />

  <n-alert
    v-if="!loadError && frpEnabled"
    class="frp-ddns-alert"
    type="info"
    :bordered="false"
    title="FRP 内网穿透已启用"
  >
    请将域名 DNS 解析指向 <strong>FRP 服务器（VPS）的公网 IP</strong>，而非 NAS 本机 IP。可在
    <router-link :to="{ name: 'frp' }">内网穿透</router-link> 页管理 FRP 配置。
  </n-alert>

  <template v-if="!loadError">
    <div class="stats-row">
      <div class="stat-card stat-card--ip">
        <div class="stat-card__head">
          <div class="stat-card__title">
            <n-icon :component="GlobeOutline" class="stat-card__icon stat-card__icon--blue" />
            <span>外网 IP 检测</span>
          </div>
          <n-button size="small" quaternary :loading="refreshingIP" @click="refreshPublicIP">刷新检测</n-button>
        </div>
        <div class="ip-lines">
          <div class="ip-line">
            <span class="ip-line__label">IPv4</span>
            <code class="ip-line__value">{{ publicIPv4 || '-' }}</code>
            <n-button v-if="publicIPv4" size="tiny" quaternary @click="copyText(publicIPv4)">
              <template #icon><n-icon :component="CopyOutline" /></template>
            </n-button>
          </div>
          <div class="ip-line">
            <span class="ip-line__label">IPv6</span>
            <code class="ip-line__value ip-line__value--v6">{{ publicIPv6 || '-' }}</code>
            <n-button v-if="publicIPv6" size="tiny" quaternary @click="copyText(publicIPv6)">
              <template #icon><n-icon :component="CopyOutline" /></template>
            </n-button>
          </div>
        </div>
        <div class="stat-card__foot">上次检测 {{ ipCheckedLabel }}</div>
      </div>

      <div class="ddns-summary">
        <div v-for="(stat, index) in taskStats" :key="stat.label" class="ddns-summary__item" :class="'ddns-stat--' + index">
          <span class="ddns-stat-label"><i aria-hidden="true" />{{ stat.label }}</span><strong>{{ stat.count }}</strong>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-wrap">
      <n-spin size="medium" />
    </div>

    <EmptyState
      v-else-if="configs.length === 0 && dnsheAccounts.length === 0 && !providerDraft && taskEditMode !== 'create'"
      title="还没有 DDNS 配置"
      description="添加 DNS 服务商后，在任务卡片中管理解析记录。"
    >
      <template #action>
        <n-button type="primary" @click="startCreateTask">添加任务</n-button>
      </template>
    </EmptyState>

    <div v-else class="ddns-layout">
      <aside class="task-sidebar">
        <div class="task-sidebar__head">
          <div class="task-sidebar__head-row">
            <h3 class="task-sidebar__title">DDNS 任务列表</h3>
            <n-button size="small" type="primary" @click="startCreateTask">
              <template #icon><n-icon :component="AddOutline" /></template>
              添加任务
            </n-button>
          </div>
        </div>

        <div v-if="taskEditMode === 'create'" ref="createTaskPanelRef" class="task-sidebar__create">
          <div class="task-item task-item--editing task-item--editing-create">
            <TaskProviderForm
              :form="taskForm"
              :editing="null"
              :saving="savingTask"
              :testing="testing"
              @save="saveTask"
              @test="testTask"
              @cancel="cancelTaskEdit"
            />
          </div>
        </div>

        <div class="task-sidebar__list">
          <article v-for="entry in taskEntries" :key="entry.key" class="task-card" :class="{ 'task-card--open': selectedKey === entry.key && !detailCollapsed }">
            <div v-if="entry.cfg && taskEditMode === entry.cfg.id" class="task-item--editing">
              <TaskProviderForm :form="taskForm" :editing="entry.cfg" :saving="savingTask" :testing="testing" @save="saveTask" @test="testTask" @cancel="cancelTaskEdit" />
            </div>
            <div v-else-if="entry.account && taskEditMode === entry.key" class="task-item--editing">
              <TaskProviderForm :form="taskForm" :editing="dnsheAccountEditingStub(entry.account)" :saving="savingTask" :testing="testing" @save="saveTask" @test="testTask" @cancel="cancelTaskEdit" />
            </div>
            <div v-else class="task-card__header">
              <button type="button" class="task-card__select" :aria-expanded="selectedKey === entry.key && !detailCollapsed" @click="entry.cfg ? selectTask(entry.cfg.id) : entry.account ? selectDnsheAccount(entry.account.id) : selectDraft()">
                <span class="provider-logo provider-logo--md" :class="{ 'provider-logo--dnshe': entry.provider === 'dnshe' }"><img :src="providerIcon(entry.provider)" :alt="providerLabel(entry.provider)" /></span>
                <span class="task-card__identity">
                  <strong>{{ entry.name }}</strong>
                  <span>{{ entry.subtitle }}</span>
                </span>
              </button>
              <StatusBadge v-if="entry.cfg" :value="taskStatusKind(entry.cfg)" :text="taskStatusText(entry.cfg)" />
              <StatusBadge v-else-if="entry.account" :value="dnsheAccountStatusKind(entry.account)" :text="dnsheAccountStatusText(entry.account)" />
              <span v-else class="task-item__draft-tag">待添加记录</span>
              <n-switch v-if="entry.cfg" :value="entry.cfg.enabled" size="small" :loading="togglingId === entry.cfg.id" @update:value="(v: boolean) => entry.cfg && toggleEnabled(entry.cfg, v)" />
              <n-switch v-else-if="entry.account" :value="dnsheAccountEnabled(entry.account)" size="small" :loading="togglingDnsheId === entry.account.id" @update:value="(v: boolean) => entry.account && toggleDnsheAccountEnabled(entry.account, v)" />
              <n-button v-else size="tiny" quaternary @click="clearDraft">取消</n-button>
              <n-button v-if="entry.cfg" size="small" quaternary title="编辑配置" aria-label="编辑配置" @click="startEditTask(entry.cfg)"><template #icon><n-icon :component="CreateOutline" /></template></n-button>
              <n-button v-if="entry.account" size="small" quaternary title="编辑配置" aria-label="编辑配置" @click="startEditDnsheAccount(entry.account)"><template #icon><n-icon :component="CreateOutline" /></template></n-button>
              <n-button v-if="entry.cfg" size="small" quaternary type="error" title="删除任务" aria-label="删除任务" @click="confirmDelete(entry.cfg)"><template #icon><n-icon :component="TrashOutline" /></template></n-button>
              <n-button v-if="entry.account" size="small" quaternary type="error" title="删除账户" aria-label="删除账户" @click="confirmDeleteDnsheAccount(entry.account)"><template #icon><n-icon :component="TrashOutline" /></template></n-button>
              <n-button class="task-chevron" size="small" quaternary :aria-label="selectedKey === entry.key && !detailCollapsed ? '收起任务' : '展开任务'" @click="entry.cfg ? selectTask(entry.cfg.id) : entry.account ? selectDnsheAccount(entry.account.id) : selectDraft()">
                <n-icon size="19" :component="selectedKey === entry.key && !detailCollapsed ? ChevronUpOutline : ChevronDownOutline" />
              </n-button>
            </div>
      <section v-if="selectedKey === entry.key && detailVisible" v-show="!detailCollapsed" class="task-detail">
        <div v-if="isDnsheAccountSelected && selectedDnsheAccount" class="task-detail__head">
          <div class="task-detail__toolbar">
            <div class="settings-switches">
              <div class="settings-switch">
                <span>IPv4</span>
                <n-switch
                  :value="panelTask?.ipv4_enabled ?? true"
                  size="small"
                  :loading="headIpToggleLoading"
                  :disabled="!headIpTogglesEnabled"
                  @update:value="(v: boolean) => headIpTogglesEnabled && patchPanelTaskFlags({ ipv4_enabled: v })"
                />
              </div>
              <div class="settings-switch">
                <span>IPv6</span>
                <n-switch
                  :value="panelTask?.ipv6_enabled ?? false"
                  size="small"
                  :loading="headIpToggleLoading"
                  :disabled="!headIpTogglesEnabled"
                  @update:value="(v: boolean) => headIpTogglesEnabled && patchPanelTaskFlags({ ipv6_enabled: v })"
                />
              </div>
            </div>
            <div class="settings-interval">
              <span class="settings-interval__label">同步周期</span>
              <n-input-number v-model:value="updateInterval" :min="1" :max="1440" size="small" class="settings-interval__input" />
              <span class="interval-unit">分钟</span>
              <n-button size="small" :loading="savingInterval" @click="saveInterval">保存</n-button>
            </div>
          </div>
        </div>

        <div v-if="!isDnsheAccountSelected && panelTask" class="task-detail__head">
          <div class="task-detail__toolbar">
            <div class="settings-switches">
              <div class="settings-switch">
                <span>IPv4</span>
                <n-switch
                  :value="panelTask.ipv4_enabled"
                  size="small"
                  :loading="headIpToggleLoading"
                  :disabled="!headIpTogglesEnabled"
                  @update:value="(v: boolean) => headIpTogglesEnabled && patchPanelTaskFlags({ ipv4_enabled: v })"
                />
              </div>
              <div class="settings-switch">
                <span>IPv6</span>
                <n-switch
                  :value="panelTask.ipv6_enabled"
                  size="small"
                  :loading="headIpToggleLoading"
                  :disabled="!headIpTogglesEnabled"
                  @update:value="(v: boolean) => headIpTogglesEnabled && patchPanelTaskFlags({ ipv6_enabled: v })"
                />
              </div>
            </div>
            <div class="settings-interval">
              <span class="settings-interval__label">同步周期</span>
              <n-input-number v-model:value="updateInterval" :min="1" :max="1440" size="small" class="settings-interval__input" />
              <span class="interval-unit">分钟</span>
              <n-button size="small" :loading="savingInterval" @click="saveInterval">保存</n-button>
            </div>
          </div>
        </div>

        <div v-if="isDnsheAccountSelected" class="dnshe-tabs" role="group" aria-label="DNSHE 管理">
          <n-button v-for="tab in dnsheTabs" :key="tab.key" size="small" :type="dnsheTab === tab.key ? 'primary' : 'default'" :aria-pressed="dnsheTab === tab.key" @click="dnsheTab = tab.key">{{ tab.label }}</n-button>
        </div>
        <DdnsDnsheDetail
          v-show="dnsheTab === 'domains'"
          v-if="isDnsheAccountSelected && selectedDnsheAccount"
                    :account="selectedDnsheAccount"
          :pushing-domain="pushingDnsheDomain"
          @refresh-accounts="loadDnsheAccounts"
          @domains-changed="syncDnsheDdnsFromAccount"
          @gifts-changed="dnsheGiftsRef?.reload()"
          @push-domain="pushDnsheDomainToRecords"
          @deleted="onDnsheAccountDeleted"
        />

        <FonuCard
          v-if="panelTask || isDnsheAccountSelected"
          ref="dnsheRecordsRefs"
          v-show="!isDnsheAccountSelected || dnsheTab === 'records'"
          flush
          class="detail-section"
        >
          <div class="records-head">
            <h4 class="detail-section__title">解析记录（{{ mergedDisplayRecords.length }}）</h4>
            <n-space :size="8">
              <n-button v-if="!isDnsheAccountSelected && !isDraftSelected && panelTask" size="small" :loading="updatingId === panelTask.id" @click="updateOne(panelTask)">立即同步</n-button>
              <n-button
                v-if="isDnsheAccountSelected && linkedDnsheConfigs.length"
                size="small"
                ghost
                type="primary"
                :loading="updatingDnsheLinked"
                @click="updateDnsheLinked"
              >
                <template #icon><n-icon :component="RefreshOutline" /></template>
                立即同步
              </n-button>
              <n-button
                size="small"
                type="primary"
                :disabled="recordEditing !== null"
                @click="startAddRecord"
              >
                <template #icon><n-icon :component="AddOutline" /></template>
                添加记录
              </n-button>
            </n-space>
          </div>

          <div class="record-table-wrap">
            <table class="record-table">
              <thead>
                <tr>
                  <th>域名 / 解析值</th><th>类型</th><th>上次结果</th><th>上次更新</th><th>状态</th><th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="recordEditing === '__new__'" class="record-row--editing">
                  <td>
                    <n-input
                      v-model:value="recordDraft.domain"
                      size="small"
                      placeholder="例如 s.example.com 或 www.sub.example.com"
                      @keyup.enter="saveRecord"
                    />
                  </td>
                  <td>
                    <div class="type-tags">
                      <n-tag v-if="recordTypeFlags.ipv4" size="tiny" :bordered="false">A</n-tag>
                      <n-tag v-if="recordTypeFlags.ipv6" size="tiny" :bordered="false" type="info">AAAA</n-tag>
                    </div>
                  </td>
                  <td class="record-table__muted">-</td>
                  <td class="record-table__muted">-</td>
                  <td class="record-table__muted">-</td>
                  <td>
                    <div class="row-actions">
                      <n-button size="tiny" type="primary" :loading="savingRecord" @click="saveRecord">保存</n-button>
                      <n-button size="tiny" quaternary @click="cancelRecordEdit">取消</n-button>
                    </div>
                  </td>
                </tr>

                <tr
                  v-for="record in mergedDisplayRecords"
                  :key="record.domain"
                  :class="{ 'record-row--editing': recordEditing === record.domain }"
                >
                  <template v-if="recordEditing === record.domain">
                    <td>
                      <n-input
                        v-model:value="recordDraft.domain"
                        size="small"
                        placeholder="例如 s.example.com 或 www.sub.example.com"
                        @keyup.enter="saveRecord"
                      />
                    </td>
                    <td>
                      <div class="type-tags">
                        <n-tag v-if="recordTypeFlags.ipv4" size="tiny" :bordered="false">A</n-tag>
                        <n-tag v-if="recordTypeFlags.ipv6" size="tiny" :bordered="false" type="info">AAAA</n-tag>
                      </div>
                    </td>
                  <td class="record-table__muted">-</td>
                  <td class="record-table__muted">-</td>
                  <td class="record-table__muted">-</td>
                    <td>
                      <div class="row-actions">
                        <n-button size="tiny" type="primary" :loading="savingRecord" @click="saveRecord">保存</n-button>
                        <n-button size="tiny" quaternary @click="cancelRecordEdit">取消</n-button>
                      </div>
                    </td>
                  </template>
                  <template v-else>
                    <td class="record-domain"><span class="record-table__mono">{{ record.domain }}</span><span class="record-ip">{{ recordValue(record) || '-' }}</span></td>
                    <td>
                      <div class="type-tags">
                        <n-tag v-if="recordTypeFlagsFor(record).ipv4" size="tiny" :bordered="false">A</n-tag>
                        <n-tag v-if="recordTypeFlagsFor(record).ipv6" size="tiny" :bordered="false" type="info">AAAA</n-tag>
                      </div>
                    </td>

                    <td class="record-table__result">{{ record.message || recordResultLabel(record.status) }}</td>
                    <td>{{ formatRelativeTime(recordTaskOf(record)?.last_updated_at) || '从未' }}</td>
                    <td :title="record.message || recordResultLabel(record.status)">
                      <n-tag size="small" round :bordered="false" :type="recordTagType(record.status)">
                        {{ recordStatusLabel(record.status) }}
                      </n-tag>
                    </td>
                    <td>
                      <div class="row-actions">
                        <n-button size="small" quaternary title="编辑记录" aria-label="编辑记录" :disabled="recordEditing !== null" @click="startEditRecord(record)"><template #icon><n-icon :component="CreateOutline" /></template></n-button>
                        <n-button size="small" quaternary type="error" title="删除记录" aria-label="删除记录" :disabled="recordEditing !== null" @click="confirmDeleteRecordFor(record)"><template #icon><n-icon :component="TrashOutline" /></template></n-button>
                      </div>
                    </td>
                  </template>
                </tr>

                <tr v-if="mergedDisplayRecords.length === 0 && recordEditing !== '__new__'">
                  <td colspan="6" class="record-table__empty">暂无解析记录，点击「添加记录」创建</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-if="mergedDisplayRecords.length" class="record-feedback">{{ recordFeedback }}</div>
        </FonuCard>

        <div v-if="detailError" class="alert alert--error">
          <n-icon :component="AlertCircleOutline" class="alert__icon" />
          <div class="alert__body">
            <div class="alert__title">{{ detailError.title }}</div>
            <div class="alert__text">{{ detailError.text }}</div>
            <router-link :to="{ name: 'logs', query: { tab: 'system' } }" class="alert__link">查看日志</router-link>
          </div>
        </div>

        <div v-if="isDraftSelected" class="task-footer"><n-button quaternary size="small" @click="startEditDraft">编辑配置</n-button></div>

        <DdnsDnsheGifts
          v-show="dnsheTab === 'gifts'"
          v-if="isDnsheAccountSelected && selectedDnsheAccount"
          ref="dnsheGiftsRefs"
          :account="selectedDnsheAccount"
        />
      </section>
          </article>
        </div>
      </aside>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NIcon,
  NInput,
  NInputNumber,
  NSpin,
  NSwitch,
  NSpace,
  NTag,
  useDialog,
  useMessage,
} from 'naive-ui'
import {
  AddOutline,
  AlertCircleOutline,
  CopyOutline,
  ChevronDownOutline,
  ChevronUpOutline,
  CreateOutline,
  TrashOutline,
  GlobeOutline,
  RefreshOutline,
} from '@vicons/ionicons5'
import aliyunIcon from '../assets/brand/dns/aliyun.png'
import cloudflareIcon from '../assets/brand/dns/cloudflare.png'
import dnspodIcon from '../assets/brand/dns/dnspod.png'
import tencentcloudIcon from '../assets/brand/dns/tencentcloud.png'
import volcengineIcon from '../assets/brand/dns/volcengine.png'
import huaweicloudIcon from '../assets/brand/dns/huaweicloud.png'
import dnsheIcon from '../assets/brand/dns/dnshe.png'
import { api, asList } from '../api/client'
import type { DDNSConfig, DDNSDomainRecord, DNSHEAccount } from '../api/types'
import DdnsDnsheDetail from '../components/DdnsDnsheDetail.vue'
import DdnsDnsheGifts from '../components/DdnsDnsheGifts.vue'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import TaskProviderForm, { type ProviderForm } from '../components/TaskProviderForm.vue'
import { copyToClipboard } from '../utils/clipboard'
import { formatRelativeTime } from '../utils/format'
import { statusLabel } from '../utils/status'

const DRAFT_KEY = '__draft__'
const DNSHE_KEY_PREFIX = 'dnshe:'

const message = useMessage()
const dialog = useDialog()
const configs = ref<DDNSConfig[]>([])
const loading = ref(false)
const frpEnabled = ref(false)
const refreshingIP = ref(false)
const loadError = ref('')
const savingTask = ref(false)
const savingRecord = ref(false)
const testing = ref(false)
const updatingAll = ref(false)
const updatingId = ref<number | null>(null)
const savingInterval = ref(false)
const togglingId = ref<number | null>(null)
const togglingDnsheId = ref<string | null>(null)
const taskEditMode = ref<'create' | number | string | null>(null)
const providerDraft = ref<ProviderForm | null>(null)
const updateInterval = ref(5)
const selectedKey = ref<string | null>(null)
const publicIPv4 = ref('')
const publicIPv6 = ref('')
const ipCheckedAt = ref('')
const recordEditing = ref<string | null>(null)
const recordDraft = reactive({ domain: '', originalDomain: '' })
const dnsheAccounts = ref<DNSHEAccount[]>([])
const dnsheGiftsRefs = ref<InstanceType<typeof DdnsDnsheGifts>[]>([])
const dnsheGiftsRef = computed(() => dnsheGiftsRefs.value[0] ?? null)
const dnsheRecordsRefs = ref<InstanceType<typeof FonuCard>[]>([])
const dnsheRecordsRef = computed(() => dnsheRecordsRefs.value[0] ?? null)
const updatingDnsheLinked = ref(false)
const pushingDnsheDomain = ref('')

const providerMap: Record<string, { label: string; icon: string }> = {
  cloudflare: { label: 'Cloudflare', icon: cloudflareIcon },
  dnspod: { label: 'DNSPod', icon: dnspodIcon },
  alidns: { label: '阿里云 DNS', icon: aliyunIcon },
  tencentcloud: { label: '腾讯云 DNS', icon: tencentcloudIcon },
  volcengine: { label: '火山引擎 DNS', icon: volcengineIcon },
  huaweicloud: { label: '华为云 DNS', icon: huaweicloudIcon },
  dnshe: { label: 'DNSHE', icon: dnsheIcon },
}

const taskForm = reactive<ProviderForm>({
  provider: 'cloudflare',
  api_token: '',
  api_token_id: '',
  api_secret: '',
  ipv4_enabled: true,
  ipv6_enabled: false,
  enabled: true,
  remark: '',
  auto_renew: false,
})

const isDraftSelected = computed(() => selectedKey.value === DRAFT_KEY)

const isDnsheAccountSelected = computed(() => selectedKey.value?.startsWith(DNSHE_KEY_PREFIX) ?? false)

function dnsheAccountKey(id: string) {
  return `${DNSHE_KEY_PREFIX}${id}`
}

const selectedDnsheAccount = computed(() => {
  if (!isDnsheAccountSelected.value || !selectedKey.value) return null
  const id = selectedKey.value.slice(DNSHE_KEY_PREFIX.length)
  return dnsheAccounts.value.find((a) => a.id === id) ?? null
})

function dnsheLinkedConfigs(acc: DNSHEAccount): DDNSConfig[] {
  const name = acc.name.trim()
  return configs.value.filter((c) => c.provider === 'dnshe' && (c.remark?.trim() ?? '') === name)
}

const linkedDnsheConfigs = computed(() => {
  const acc = selectedDnsheAccount.value
  return acc ? dnsheLinkedConfigs(acc) : []
})

const sidebarConfigs = computed(() => {
  const accountNames = new Set(dnsheAccounts.value.map((a) => a.name.trim()))
  return configs.value.filter((c) => {
    if (c.provider !== 'dnshe') return true
    return !accountNames.has(c.remark?.trim() ?? '')
  })
})

const detailVisible = computed(
  () => (isDnsheAccountSelected.value && !!selectedDnsheAccount.value) || !!selectedTask.value,
)

const draftAsConfig = computed((): DDNSConfig | null => {
  if (!providerDraft.value) return null
  return {
    id: -1,
    provider: providerDraft.value.provider,
    root_domain: '',
    record_name: '@',
    ipv4_enabled: providerDraft.value.ipv4_enabled,
    ipv6_enabled: providerDraft.value.ipv6_enabled,
    enabled: providerDraft.value.enabled,
    has_token: !!providerDraft.value.api_token,
  }
})

const selectedTask = computed(() => {
  if (isDnsheAccountSelected.value) return null
  if (isDraftSelected.value) return draftAsConfig.value
  const id = Number(selectedKey.value)
  if (!id) return null
  return configs.value.find((c) => c.id === id) ?? null
})

const panelTask = computed(() => {
  if (isDnsheAccountSelected.value && selectedDnsheAccount.value) {
    const linked = dnsheLinkedConfigs(selectedDnsheAccount.value)
    return linked[0] ?? null
  }
  return selectedTask.value
})

const headIpTogglesEnabled = computed(() => !!panelTask.value && !isDraftSelected.value)

const headIpToggleLoading = computed(() => {
  const task = panelTask.value
  if (!task || isDraftSelected.value) return false
  if (isDnsheAccountSelected.value && togglingDnsheId.value) return true
  return togglingId.value === task.id
})

const showDraftInList = computed(() => taskEditMode.value !== 'create')

type TaskBucket = 'healthy' | 'error' | 'paused' | 'pending'
function taskBucket(cfg: DDNSConfig): TaskBucket {
  if (!cfg.enabled) return 'paused'
  if (cfg.last_status === 'error' || cfg.last_status === 'warning') return 'error'
  return isTaskHealthy(cfg) ? 'healthy' : 'pending'
}
function accountBucket(acc: DNSHEAccount): TaskBucket {
  const rows = dnsheLinkedConfigs(acc)
  if (!rows.length) return 'pending'
  const buckets = rows.map(taskBucket)
  if (buckets.includes('error')) return 'error'
  if (buckets.includes('pending')) return 'pending'
  if (buckets.includes('healthy')) return 'healthy'
  return 'paused'
}
const taskStats = computed(() => {
  const buckets = [...sidebarConfigs.value.map(taskBucket), ...dnsheAccounts.value.map(accountBucket)]
  return [
    { label: '全部', count: buckets.length },
    { label: '正常', count: buckets.filter(b => b === 'healthy').length },
    { label: '异常', count: buckets.filter(b => b === 'error').length },
    { label: '暂停', count: buckets.filter(b => b === 'paused').length },
    { label: '待配置 / 同步', count: buckets.filter(b => b === 'pending').length },
  ]
})
type TaskEntry = { key: string; provider: string; name: string; subtitle: string; cfg?: DDNSConfig; account?: DNSHEAccount }
const taskEntries = computed<TaskEntry[]>(() => {
  const rows: TaskEntry[] = []
  if (providerDraft.value && showDraftInList.value) rows.push({ key: DRAFT_KEY, provider: providerDraft.value.provider, name: providerLabel(providerDraft.value.provider), subtitle: '请添加首条解析记录' })
  for (const cfg of sidebarConfigs.value) rows.push({ key: String(cfg.id), provider: cfg.provider, name: cfg.remark || providerLabel(cfg.provider), subtitle: `${domainRecordsOf(cfg).length} 条记录 · ${formatRelativeTime(cfg.last_updated_at) || '从未同步'}`, cfg })
  for (const account of dnsheAccounts.value) rows.push({ key: dnsheAccountKey(account.id), provider: 'dnshe', name: account.name + ' · DNSHE', subtitle: `${dnsheAccountRecordCount(account)} 条记录 · ${formatRelativeTime(dnsheAccountLastSync(account)) || '从未同步'}`, account })
  return rows
})
const recordFeedback = computed(() => {
  const rows = mergedDisplayRecords.value
  if (rows.some(r => r.status === 'error')) return '存在同步异常，请查看记录状态及错误提示'
  if (rows.every(r => r.status === 'unchanged')) return '上次同步成功 · 记录未变化'
  if (rows.every(r => r.status === 'ok' || r.status === 'unchanged')) return '上次同步成功'
  return '部分记录待同步或状态未知'
})
const ipCheckedLabel = computed(() => ipCheckedAt.value ? formatRelativeTime(ipCheckedAt.value) || '刚刚' : '尚未检测')
const detailCollapsed = ref(true)
const dnsheTab = ref<'domains' | 'records' | 'gifts'>('domains')
const dnsheTabs = [{ key: 'domains', label: '域名管理' }, { key: 'records', label: '解析记录' }, { key: 'gifts', label: 'Gift 管理' }] as const

const displayRecords = computed(() => {
  const task = panelTask.value
  if (!task || isDraftSelected.value) return []
  return domainRecordsOf(task)
})

const mergedDisplayRecords = computed(() => {
  if (isDnsheAccountSelected.value && selectedDnsheAccount.value) {
    const rows: DDNSDomainRecord[] = []
    const seen = new Set<string>()
    for (const cfg of dnsheLinkedConfigs(selectedDnsheAccount.value)) {
      for (const r of domainRecordsOf(cfg)) {
        const key = r.domain.toLowerCase()
        if (seen.has(key)) continue
        seen.add(key)
        rows.push(r)
      }
    }
    return rows
  }
  return displayRecords.value
})

const recordTypeFlags = computed(() => {
  const task = panelTask.value
  return { ipv4: task?.ipv4_enabled ?? true, ipv6: task?.ipv6_enabled ?? false }
})

function recordTypeFlagsFor(record: DDNSDomainRecord) {
  const task = recordTaskOf(record)
  return { ipv4: task?.ipv4_enabled ?? true, ipv6: task?.ipv6_enabled ?? false }
}

function recordTaskOf(record: DDNSDomainRecord): DDNSConfig | null {
  const domain = record.domain.toLowerCase()
  if (isDnsheAccountSelected.value && selectedDnsheAccount.value) {
    for (const cfg of dnsheLinkedConfigs(selectedDnsheAccount.value)) {
      if (domainsOf(cfg).some((d) => d.toLowerCase() === domain)) return cfg
    }
    return panelTask.value
  }
  const task = selectedTask.value
  if (task && domainsOf(task).some((d) => d.toLowerCase() === domain)) return task
  return panelTask.value
}

const detailError = computed(() => {
  const tasks: DDNSConfig[] = []
  if (isDnsheAccountSelected.value && selectedDnsheAccount.value) {
    tasks.push(...dnsheLinkedConfigs(selectedDnsheAccount.value))
  } else if (selectedTask.value && !isDraftSelected.value) {
    tasks.push(selectedTask.value)
  }
  for (const cfg of tasks) {
    const failed = domainRecordsOf(cfg).find((r) => r.status === 'error')
    if (failed) {
      return {
        title: `${failed.domain} 解析失败`,
        text: failed.message || cfg.last_error || 'DNS 解析更新失败，请检查凭证与域名配置。',
      }
    }
    if (cfg.enabled && cfg.last_status === 'error' && cfg.last_error) {
      return { title: `${providerLabel(cfg.provider)} 同步失败`, text: cfg.last_error }
    }
  }
  return null
})

function providerMeta(v: string) {
  return providerMap[v] ?? { label: v, icon: '' }
}

function providerLabel(v: string) {
  return providerMeta(v).label
}

function providerIcon(v: string) {
  return providerMeta(v).icon
}

function isTaskHealthy(cfg: DDNSConfig) {
  if (!cfg.enabled) return false
  return cfg.last_status === 'ok' || cfg.last_status === 'unchanged'
}

function taskStatusKind(cfg: DDNSConfig) {
  if (!cfg.enabled) return 'disabled'
  if (cfg.last_status === 'error' || cfg.last_status === 'warning') return cfg.last_status
  return isTaskHealthy(cfg) ? 'ok' : 'unknown'
}

function taskStatusText(cfg: DDNSConfig) {
  if (!cfg.enabled) return '暂停'
  if (cfg.last_status === 'error' || cfg.last_status === 'warning') return '异常'
  return isTaskHealthy(cfg) ? '正常' : '待同步'
}

function dnsheAccountLastSync(acc: DNSHEAccount): string {
  const times = dnsheLinkedConfigs(acc).map((c) => c.last_updated_at).filter(Boolean) as string[]
  if (times.length === 0) return ''
  return times.sort((a, b) => new Date(b).getTime() - new Date(a).getTime())[0]
}

function dnsheAccountRecordCount(acc: DNSHEAccount): number {
  const seen = new Set<string>()
  let count = 0
  for (const cfg of dnsheLinkedConfigs(acc)) {
    for (const r of domainRecordsOf(cfg)) {
      const key = r.domain.toLowerCase()
      if (seen.has(key)) continue
      seen.add(key)
      count++
    }
  }
  return count
}

function dnsheAccountEnabled(acc: DNSHEAccount): boolean {
  const linked = dnsheLinkedConfigs(acc)
  return linked.length > 0 && linked.some((c) => c.enabled)
}

function dnsheAccountStatusKind(acc: DNSHEAccount) {
  const linked = dnsheLinkedConfigs(acc)
  if (linked.length === 0) return 'unknown'
  if (!linked.some((c) => c.enabled)) return 'disabled'
  if (linked.some((c) => c.enabled && (c.last_status === 'error' || c.last_status === 'warning'))) {
    return linked.some((c) => c.enabled && c.last_status === 'error') ? 'error' : 'warning'
  }
  return accountBucket(acc) === 'healthy' ? 'ok' : 'unknown'
}

function dnsheAccountStatusText(acc: DNSHEAccount) {
  const linked = dnsheLinkedConfigs(acc)
  if (linked.length === 0) return '待配置'
  if (!linked.some((c) => c.enabled)) return '暂停'
  if (linked.some((c) => c.enabled && (c.last_status === 'error' || c.last_status === 'warning'))) return '异常'
  return accountBucket(acc) === 'healthy' ? '正常' : '待同步'
}

function recordNamesOf(row: DDNSConfig): string[] {
  return row.record_names?.length ? row.record_names : [row.record_name || '@']
}

function formatDomain(root: string, record: string): string {
  if (!record || record === '@') return root
  if (record === '*') return `*.${root}`
  if (record.includes('.')) return record
  return `${record}.${root}`
}

function domainsOf(row: DDNSConfig): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const name of recordNamesOf(row)) {
    const domain = formatDomain(row.root_domain, name).trim().toLowerCase()
    if (!domain || seen.has(domain)) continue
    seen.add(domain)
    out.push(domain)
  }
  return out
}

function validateDomainFormat(domain: string): string | null {
  const raw = domain.trim().toLowerCase().replace(/\.$/, '')
  if (!raw) return '请输入完整域名'
  const bare = raw.startsWith('*.') ? raw.slice(2) : raw
  if (!bare.includes('.')) return '请填写完整域名，例如 s.example.com'
  if (!/^[a-z0-9.-]+$/.test(raw) && !/^\*\.[a-z0-9.-]+$/.test(raw)) return '域名包含非法字符'
  const parts = bare.split('.').filter(Boolean)
  if (parts.length < 2) return '域名格式无效'
  for (const part of parts) {
    if (part.length > 63 || part.startsWith('-') || part.endsWith('-')) return '域名格式无效'
  }
  return null
}

function domainRecordsOf(row: DDNSConfig): DDNSDomainRecord[] {
  if (row.domain_records?.length) return row.domain_records
  return domainsOf(row).map((domain) => ({ domain, status: 'unknown' }))
}

function recordStatusLabel(status: string): string {
  if (status === 'error') return '异常'
  if (status === 'ok' || status === 'unchanged') return '正常'
  return statusLabel(status)
}

function recordResultLabel(status: string): string {
  if (status === 'unchanged') return 'IP 并无变化'
  if (status === 'ok') return '已同步'
  if (status === 'error') return '同步失败'
  return statusLabel(status)
}

function recordTagType(status: string): 'success' | 'warning' | 'error' | 'default' {
  if (status === 'ok' || status === 'unchanged') return 'success'
  if (status === 'error') return 'error'
  return 'warning'
}

function recordValue(record: DDNSDomainRecord) {
  const task = recordTaskOf(record)
  if (!task) return ''
  const lines: string[] = []
  if (task.ipv4_enabled && record.ipv4) lines.push(record.ipv4)
  if (task.ipv6_enabled && record.ipv6) lines.push(record.ipv6)
  return lines.join(' / ')
}

function normalizeRecordDomain(input: string, rootDomain?: string): string {
  const raw = input.trim().toLowerCase().replace(/\.$/, '')
  if (!raw) return ''
  if (raw.includes('.')) return raw
  if (!rootDomain) return raw
  if (raw === '@') return rootDomain
  if (raw === '*') return `*.${rootDomain}`
  return `${raw}.${rootDomain}`
}

function resetTaskForm() {
  Object.assign(taskForm, {
    provider: 'cloudflare',
    api_token: '',
    api_token_id: '',
    api_secret: '',
    ipv4_enabled: true,
    ipv6_enabled: false,
    enabled: true,
    remark: '',
    auto_renew: false,
  })
}

function ensureSelection() {
  if (isDnsheAccountSelected.value && selectedDnsheAccount.value) return
  if (providerDraft.value) {
    if (!selectedKey.value) selectedKey.value = DRAFT_KEY
    return
  }
  if (configs.value.length === 0 && dnsheAccounts.value.length === 0) {
    selectedKey.value = null
    return
  }
  if (sidebarConfigs.value.some((c) => String(c.id) === selectedKey.value)) return
  if (isDnsheAccountSelected.value && selectedDnsheAccount.value) return
  if (sidebarConfigs.value.length > 0) {
    selectedKey.value = String(sidebarConfigs.value[0].id)
  } else if (dnsheAccounts.value[0]) {
    selectedKey.value = dnsheAccountKey(dnsheAccounts.value[0].id)
  }
}

watch(configs, ensureSelection)

function selectTask(id: number) {
  if (selectedKey.value === String(id) && !recordEditing.value && taskEditMode.value === null) { detailCollapsed.value = !detailCollapsed.value; return }
  detailCollapsed.value = false
  selectedKey.value = String(id)
  cancelRecordEdit()
}

function selectDnsheAccount(id: string) {
  if (selectedKey.value === dnsheAccountKey(id) && !recordEditing.value && taskEditMode.value === null) { detailCollapsed.value = !detailCollapsed.value; return }
  detailCollapsed.value = false
  dnsheTab.value = 'domains'
  selectedKey.value = dnsheAccountKey(id)
  cancelRecordEdit()
}

function dnsheAccountEditingStub(acc: DNSHEAccount): DDNSConfig {
  return {
    id: -1,
    provider: 'dnshe',
    root_domain: '',
    record_name: '@',
    ipv4_enabled: true,
    ipv6_enabled: false,
    enabled: true,
    has_token: true,
    remark: acc.name,
  }
}

function startEditDnsheAccount(acc: DNSHEAccount) {
  detailCollapsed.value = false
  selectedKey.value = dnsheAccountKey(acc.id)
  cancelRecordEdit()
  Object.assign(taskForm, {
    provider: 'dnshe',
    api_token: '',
    api_token_id: '',
    api_secret: '',
    ipv4_enabled: true,
    ipv6_enabled: false,
    enabled: true,
    remark: acc.name,
    auto_renew: acc.auto_renew,
  })
  taskEditMode.value = dnsheAccountKey(acc.id)
}

async function toggleDnsheAccountEnabled(acc: DNSHEAccount, enabled: boolean) {
  const linked = dnsheLinkedConfigs(acc)
  if (linked.length === 0) {
    message.warning('暂无关联的解析任务，请先注册域名或添加解析记录')
    return
  }
  togglingDnsheId.value = acc.id
  try {
    for (const cfg of linked) {
      if (cfg.enabled === enabled) continue
      await patchTaskFlags(cfg, { enabled })
    }
  } finally {
    togglingDnsheId.value = null
  }
}

function selectDraft() {
  if (selectedKey.value === DRAFT_KEY && !recordEditing.value) { detailCollapsed.value = !detailCollapsed.value; return }
  detailCollapsed.value = false
  selectedKey.value = DRAFT_KEY
  cancelRecordEdit()
}

const createTaskPanelRef = ref<HTMLElement | null>(null)

function startCreateTask() {
  detailCollapsed.value = false
  resetTaskForm()
  taskEditMode.value = 'create'
  cancelRecordEdit()
  nextTick(() => {
    createTaskPanelRef.value?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
}

function startEditTask(cfg: DDNSConfig) {
  detailCollapsed.value = false
  selectedKey.value = String(cfg.id)
  Object.assign(taskForm, {
    provider: cfg.provider || 'cloudflare',
    api_token: '',
    api_token_id: '',
    api_secret: '',
    ipv4_enabled: cfg.ipv4_enabled,
    ipv6_enabled: cfg.ipv6_enabled,
    enabled: cfg.enabled,
    remark: cfg.remark ?? '',
  })
  taskEditMode.value = cfg.id
}

function startEditDraft() {
  if (!providerDraft.value) return
  Object.assign(taskForm, providerDraft.value)
  taskEditMode.value = 'create'
}

function cancelTaskEdit() {
  taskEditMode.value = null
}

function clearDraft() {
  providerDraft.value = null
  if (selectedKey.value === DRAFT_KEY) {
    selectedKey.value = configs.value[0] ? String(configs.value[0].id) : null
  }
}

function hasCredentialInput(form: ProviderForm) {
  return !!(form.api_token.trim() || form.api_token_id.trim() || form.api_secret.trim())
}

async function copyText(text: string) {
  if (await copyToClipboard(text)) {
    message.success('已复制')
  } else {
    message.error('复制失败')
  }
}

async function refreshPublicIP() {
  refreshingIP.value = true
  try {
    const s = await api.getStatus()
    publicIPv4.value = s.public_ipv4 || ''
    publicIPv6.value = s.public_ipv6 || ''
    ipCheckedAt.value = new Date().toISOString()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '刷新失败')
  } finally {
    refreshingIP.value = false
  }
}

async function loadCached() {
  const [list, settings] = await Promise.all([api.listDDNSLite(), api.getSettings()])
  configs.value = asList(list)
  updateInterval.value = Number(settings.ddns_check_interval_minutes ?? 5)
  ensureSelection()
}

async function refreshLiveDNS() {
  try {
    configs.value = asList(await api.listDDNS())
    ensureSelection()
  } catch {
    message.warning('实时解析刷新失败，继续显示上次数据')
  }
}

async function loadDnsheAccounts() {
  try {
    dnsheAccounts.value = await api.listDNSHEAccounts()
    ensureSelection()
  } catch {
    message.warning('DNSHE 账户加载失败，保留已加载数据')
  }
}

async function syncDnsheDdnsFromAccount() {
  const acc = selectedDnsheAccount.value
  if (!acc) return
  try {
    await api.pushDNSHEToDDNS(acc.id)
  } catch {
    // 无域名或已全部同步时忽略
  }
  await refreshLiveDNS()
}

function isDnsheDomainInRecords(fullDomain: string): boolean {
  const domain = fullDomain.trim().toLowerCase()
  const acc = selectedDnsheAccount.value
  if (!acc) return false
  return dnsheLinkedConfigs(acc).some((cfg) => domainsOf(cfg).some((d) => d.toLowerCase() === domain))
}

function scrollToDnsheRecords() {
  dnsheTab.value = 'records'
  detailCollapsed.value = false
  nextTick(() => {
    const el = dnsheRecordsRef.value?.$el as HTMLElement | undefined
    el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

async function pushDnsheDomainToRecords(fullDomain: string) {
  if (!selectedDnsheAccount.value) return
  const domain = fullDomain.trim().toLowerCase()
  if (isDnsheDomainInRecords(domain)) {
    scrollToDnsheRecords()
    message.info('该域名已在解析记录中')
    return
  }
  pushingDnsheDomain.value = fullDomain
  try {
    let task = await ensureDnsheTaskForDomain(domain)
    const merged = [...new Set([...domainsOf(task), domain])]
    if (merged.length !== domainsOf(task).length) {
      const updated = await api.updateDDNS(task.id, {
        provider: task.provider,
        domains: merged,
        enabled: task.enabled,
        ipv4_enabled: task.ipv4_enabled,
        ipv6_enabled: task.ipv6_enabled,
      })
      configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
      task = updated
    }
    await refreshLiveDNS()
    message.success('已推送到解析记录')
    scrollToDnsheRecords()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '推送失败')
  } finally {
    pushingDnsheDomain.value = ''
  }
}

function onDnsheAccountDeleted() {
  void loadDnsheAccounts().then(() => {
    if (dnsheAccounts.value[0]) {
      selectedKey.value = dnsheAccountKey(dnsheAccounts.value[0].id)
    } else if (configs.value[0]) {
      selectedKey.value = String(configs.value[0].id)
    } else {
      selectedKey.value = null
    }
  })
}

function rootDomainOfFull(domain: string): string {
  const parts = domain.toLowerCase().split('.').filter(Boolean)
  if (parts.length < 2) return domain.toLowerCase()
  return parts.slice(-2).join('.')
}

async function ensureDnsheTaskForDomain(domain: string): Promise<DDNSConfig> {
  const acc = selectedDnsheAccount.value
  if (!acc) throw new Error('未选择 DNSHE 账户')
  const root = rootDomainOfFull(domain)
  const existing = dnsheLinkedConfigs(acc).find((c) => c.root_domain.toLowerCase() === root)
  if (existing) return existing
  const creds = await api.revealDNSHECredentials(acc.id)
  const created = await api.createDDNS({
    provider: 'dnshe',
    api_token: creds.api_key,
    api_secret: creds.api_secret,
    remark: acc.name,
    domains: [domain],
    ipv4_enabled: true,
    ipv6_enabled: false,
    enabled: true,
  })
  configs.value = [...configs.value, created]
  return created
}

async function init() {
  loading.value = true
  loadError.value = ''
  try {
    await loadCached()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '请检查 Fonu 服务是否正常运行'
  } finally {
    loading.value = false
  }
  refreshPublicIP()
  refreshLiveDNS()
  void loadDnsheAccounts()
  try {
    const frp = await api.getFRP()
    frpEnabled.value = frp.enabled
  } catch {
    frpEnabled.value = false
  }
}

async function saveTask() {
  const dnsheEditingKey =
    typeof taskEditMode.value === 'string' &&
    taskEditMode.value.startsWith(DNSHE_KEY_PREFIX) &&
    taskEditMode.value !== DRAFT_KEY
      ? taskEditMode.value
      : null

  if (dnsheEditingKey) {
    const accountId = dnsheEditingKey.slice(DNSHE_KEY_PREFIX.length)
    savingTask.value = true
    try {
      await api.updateDNSHEAccount(accountId, {
        name: taskForm.remark.trim(),
        api_key: taskForm.api_token || undefined,
        api_secret: taskForm.api_secret || undefined,
        auto_renew: taskForm.auto_renew,
      })
      await loadDnsheAccounts()
      message.success('账户已更新')
      taskEditMode.value = null
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败')
    } finally {
      savingTask.value = false
    }
    return
  }

  const editingId = typeof taskEditMode.value === 'number' ? taskEditMode.value : null

  if (editingId) {
    const existing = configs.value.find((c) => c.id === editingId)
    if (!existing) return
    savingTask.value = true
    try {
      const updated = await api.updateDDNS(editingId, {
        provider: taskForm.provider,
        domains: domainsOf(existing),
        ipv4_enabled: taskForm.ipv4_enabled,
        ipv6_enabled: taskForm.ipv6_enabled,
        enabled: taskForm.enabled,
        api_token: taskForm.api_token,
        api_token_id: taskForm.api_token_id,
        api_secret: taskForm.api_secret,
        remark: taskForm.remark,
      })
      configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
      message.success('服务商已更新')
      taskEditMode.value = null
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败')
    } finally {
      savingTask.value = false
    }
    return
  }

  if (taskForm.provider === 'dnshe') {
    if (!hasCredentialInput(taskForm)) {
      message.error('请填写 DNSHE API 凭证')
      return
    }
    savingTask.value = true
    try {
      const created = await api.createDNSHEAccount({
        name: taskForm.remark.trim(),
        api_key: taskForm.api_token,
        api_secret: taskForm.api_secret,
        auto_renew: taskForm.auto_renew,
      })
      await loadDnsheAccounts()
      selectedKey.value = dnsheAccountKey(created.id)
      taskEditMode.value = null
      resetTaskForm()
      message.success('DNSHE 账户已添加')
      await syncDnsheDdnsFromAccount()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败')
    } finally {
      savingTask.value = false
    }
    return
  }

  if (!hasCredentialInput(taskForm)) {
    message.error('请填写 DNS API 凭证')
    return
  }

  providerDraft.value = { ...taskForm }
  selectedKey.value = DRAFT_KEY
  taskEditMode.value = null
  message.success('服务商已保存，请添加解析记录')
}

async function testTask() {
  testing.value = true
  try {
    const editingId = typeof taskEditMode.value === 'number' ? taskEditMode.value : undefined
    await api.testDDNS({
      config_id: editingId,
      provider: taskForm.provider,
      api_token: taskForm.api_token,
      api_token_id: taskForm.api_token_id,
      api_secret: taskForm.api_secret,
    })
    message.success('连接成功')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '测试失败')
  } finally {
    testing.value = false
  }
}

function startAddRecord() {
  if (!panelTask.value && !isDnsheAccountSelected.value) return
  recordEditing.value = '__new__'
  recordDraft.domain = ''
  recordDraft.originalDomain = ''
}

function startEditRecord(record: DDNSDomainRecord) {
  recordEditing.value = record.domain
  recordDraft.domain = record.domain
  recordDraft.originalDomain = record.domain
}

function cancelRecordEdit() {
  recordEditing.value = null
  recordDraft.domain = ''
  recordDraft.originalDomain = ''
}

async function saveRecord() {
  let task = panelTask.value
  if (!task && isDnsheAccountSelected.value) {
    const domainOnly = normalizeRecordDomain(recordDraft.domain)
    const formatEarly = validateDomainFormat(domainOnly)
    if (formatEarly) {
      message.error(formatEarly)
      return
    }
    savingRecord.value = true
    try {
      task = await ensureDnsheTaskForDomain(domainOnly)
      const domains = [...new Set([...domainsOf(task), domainOnly])]
      const updated = await api.updateDDNS(task.id, {
        provider: task.provider,
        domains,
        enabled: task.enabled,
        ipv4_enabled: task.ipv4_enabled,
        ipv6_enabled: task.ipv6_enabled,
      })
      configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
      message.success('记录已添加')
      cancelRecordEdit()
      refreshLiveDNS()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败')
    } finally {
      savingRecord.value = false
    }
    return
  }
  if (!task) return

  const domain = normalizeRecordDomain(recordDraft.domain, task.root_domain || undefined)
  const formatError = validateDomainFormat(domain)
  if (formatError) {
    message.error(formatError)
    return
  }

  savingRecord.value = true
  try {
    if (isDraftSelected.value && providerDraft.value) {
      const created = await api.createDDNS({
        ...providerDraft.value,
        domains: [domain],
      })
      configs.value = [...configs.value, created]
      providerDraft.value = null
      selectedKey.value = String(created.id)
      message.success('任务已创建')
    } else if (recordEditing.value === '__new__') {
      if (isDnsheAccountSelected.value) {
        task = await ensureDnsheTaskForDomain(domain)
      }
      const domains = [...new Set([...domainsOf(task), domain])]
      const updated = await api.updateDDNS(task.id, {
        provider: task.provider,
        domains,
        enabled: task.enabled,
        ipv4_enabled: task.ipv4_enabled,
        ipv6_enabled: task.ipv6_enabled,
      })
      configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
      message.success('记录已添加')
      refreshLiveDNS()
    } else {
      const domains = domainsOf(task).map((d) => (d === recordDraft.originalDomain ? domain : d))
      const unique = [...new Set(domains)]
      const updated = await api.updateDDNS(task.id, {
        provider: task.provider,
        domains: unique,
        enabled: task.enabled,
        ipv4_enabled: task.ipv4_enabled,
        ipv6_enabled: task.ipv6_enabled,
      })
      configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
      message.success('记录已更新')
      refreshLiveDNS()
    }
    cancelRecordEdit()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    savingRecord.value = false
  }
}

async function patchPanelTaskFlags(
  flags: Partial<Pick<DDNSConfig, 'ipv4_enabled' | 'ipv6_enabled' | 'enabled'>>,
) {
  const rows =
    isDnsheAccountSelected.value && selectedDnsheAccount.value
      ? dnsheLinkedConfigs(selectedDnsheAccount.value)
      : panelTask.value
        ? [panelTask.value]
        : []
  for (const row of rows) {
    await patchTaskFlags(row, flags)
  }
}

async function patchTaskFlags(
  row: DDNSConfig,
  flags: Partial<Pick<DDNSConfig, 'ipv4_enabled' | 'ipv6_enabled' | 'enabled'>>,
) {
  togglingId.value = row.id
  try {
    const updated = await api.updateDDNS(row.id, {
      provider: row.provider,
      domains: domainsOf(row),
      enabled: flags.enabled ?? row.enabled,
      ipv4_enabled: flags.ipv4_enabled ?? row.ipv4_enabled,
      ipv6_enabled: flags.ipv6_enabled ?? row.ipv6_enabled,
    })
    configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
  } finally {
    togglingId.value = null
  }
}

async function toggleEnabled(row: DDNSConfig, enabled: boolean) {
  await patchTaskFlags(row, { enabled })
}

function syncFeedbackMessage(cfg: DDNSConfig): { type: 'success' | 'warning' | 'error'; text: string } {
  if (cfg.last_status === 'error') return { type: 'error', text: cfg.last_error || '同步失败' }
  const records = domainRecordsOf(cfg)
  const updated = records.filter((r) => r.status === 'ok').length
  const unchanged = records.filter((r) => r.status === 'unchanged').length
  const failed = records.filter((r) => r.status === 'error').length
  if (cfg.last_status === 'warning' || failed > 0) {
    const parts: string[] = []
    if (updated > 0) parts.push(`${updated} 条已更新`)
    if (unchanged > 0) parts.push(`${unchanged} 条未变化`)
    if (failed > 0) parts.push(`${failed} 条失败`)
    return { type: 'warning', text: parts.join('，') || cfg.last_error || '部分域名同步失败' }
  }
  if (updated > 0 && unchanged === 0) return { type: 'success', text: `已更新 ${updated} 条记录` }
  if (updated > 0) return { type: 'success', text: `已更新 ${updated} 条，${unchanged} 条未变化` }
  if (unchanged > 0) return { type: 'success', text: `同步完成，${unchanged} 条记录未变化` }
  return { type: 'success', text: '同步完成' }
}

async function updateDnsheLinked() {
  const rows = linkedDnsheConfigs.value
  if (rows.length === 0) return
  updatingDnsheLinked.value = true
  try {
    let failures = 0
    for (const row of rows) {
      const updated = await api.updateDDNSOne(row.id)
      configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
      if (syncFeedbackMessage(updated).type !== 'success') failures++
    }
    if (failures) message.warning(`${failures} 个关联任务同步异常，请查看解析记录`)
    else message.success('DNSHE 解析已同步')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
    await refreshLiveDNS()
  } finally {
    updatingDnsheLinked.value = false
  }
}

async function updateOne(row: DDNSConfig) {
  updatingId.value = row.id
  try {
    const updated = await api.updateDDNSOne(row.id)
    configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
    const fb = syncFeedbackMessage(updated)
    message[fb.type](fb.text)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
    await refreshLiveDNS()
  } finally {
    updatingId.value = null
  }
}

async function updateAll() {
  updatingAll.value = true
  try {
    configs.value = await api.updateAllDDNS()
    ensureSelection()
    const errors = configs.value.filter((c) => c.enabled && c.last_status === 'error')
    const warnings = configs.value.filter((c) => c.enabled && c.last_status === 'warning')
    if (errors.length > 0) {
      message.error(errors.length === 1 ? errors[0].last_error || '部分任务更新失败' : `${errors.length} 个任务更新失败`)
    } else if (warnings.length > 0) {
      message.warning(warnings.length === 1 ? warnings[0].last_error || '部分域名同步失败' : `${warnings.length} 个任务存在域名同步失败`)
    } else {
      message.success('全部 DDNS 已更新')
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '更新失败')
    await refreshLiveDNS()
  } finally {
    updatingAll.value = false
  }
}

async function saveInterval() {
  savingInterval.value = true
  try {
    await api.saveSettings({ ddns_check_interval_minutes: String(updateInterval.value) })
    message.success('检查周期已保存')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    savingInterval.value = false
  }
}

function confirmDeleteDnsheAccount(account: DNSHEAccount) {
  dialog.error({
    title: '删除账户',
    content: `确定删除账户「${account.name}」吗？仅删除本地保存的凭据，不影响 DNSHE 线上域名。`,
    positiveText: '删除',
    negativeText: '取消',
    positiveButtonProps: { type: 'error' },
    onPositiveClick: async () => {
      try {
        await api.deleteDNSHEAccount(account.id)
        message.success('账户已删除')
        onDnsheAccountDeleted()
      } catch (error) {
        message.error(error instanceof Error ? error.message : '删除失败')
        return false
      }
    },
  })
}

function confirmDelete(row: DDNSConfig) {
  dialog.error({
    title: '删除 DDNS 配置',
    content: `确定删除 ${providerLabel(row.provider)} 的 DDNS 配置吗？`,
    positiveText: '删除',
    negativeText: '取消',
    positiveButtonProps: { type: 'error' },
    onPositiveClick: async () => {
      await api.deleteDDNS(row.id)
      configs.value = configs.value.filter((c) => c.id !== row.id)
      ensureSelection()
      message.success('已删除')
    },
  })
}

function confirmDeleteRecordFor(record: DDNSDomainRecord) {
  const row = recordTaskOf(record)
  if (!row) {
    message.error('找不到对应任务')
    return
  }
  confirmDeleteRecord(row, record)
}

function confirmDeleteRecord(row: DDNSConfig, record: DDNSDomainRecord) {
  dialog.error({
    title: '删除解析记录',
    content: `确定从 DDNS 中移除 ${record.domain} 吗？${domainsOf(row).length === 1 ? '这是最后一条记录，移除后对应 DDNS 配置也将被删除。' : ''}`,
    positiveText: '删除',
    negativeText: '取消',
    positiveButtonProps: { type: 'error' },
    onPositiveClick: async () => {
      const domains = domainsOf(row).filter((d) => d !== record.domain)
      if (domains.length === 0) {
        await api.deleteDDNS(row.id)
        configs.value = configs.value.filter((c) => c.id !== row.id)
      } else {
        const updated = await api.updateDDNS(row.id, {
          provider: row.provider,
          domains,
          enabled: row.enabled,
          ipv4_enabled: row.ipv4_enabled,
          ipv6_enabled: row.ipv6_enabled,
        })
        configs.value = configs.value.map((c) => (c.id === updated.id ? updated : c))
      }
      ensureSelection()
      message.success('已删除')
      refreshLiveDNS()
    },
  })
}

onMounted(init)
</script>

<style scoped>
.frp-ddns-alert {
  margin-bottom: var(--fonu-space-4);
}

.stats-row {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) repeat(4, minmax(0, 1fr));
  gap: var(--fonu-space-4);
  margin-bottom: var(--fonu-space-4);
}

.stat-card {
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  padding: var(--fonu-space-4) var(--fonu-space-5);
  min-height: 118px;
}

.stat-card--ip { min-height: 132px; }

.stat-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  margin-bottom: var(--fonu-space-3);
}

.stat-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
}

.stat-card__icon { font-size: 18px; }
.stat-card__icon--blue { color: #3b82f6; }
.stat-card__icon--green { color: #10b981; }
.stat-card__icon--red { color: #ef4444; }
.stat-card__icon--gray { color: #94a3b8; }

.stat-card__value {
  margin-top: 4px;
  font-size: 30px;
  font-weight: 700;
  line-height: 1.1;
  color: var(--fonu-text);
  letter-spacing: -0.03em;
}

.stat-card__value--sm { font-size: 22px; }
.stat-card__value--green { color: #10b981; }
.stat-card__value--red { color: #ef4444; }

.stat-card__label {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--fonu-text);
}

.stat-card__sub,
.stat-card__foot {
  margin-top: 4px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.ip-lines { display: flex; flex-direction: column; gap: 8px; }

.ip-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.ip-line__label {
  width: 36px;
  flex-shrink: 0;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.ip-line__value {
  flex: 1;
  min-width: 0;
  font-family: var(--fonu-mono);
  font-size: 13px;
  color: var(--fonu-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.loading-wrap {
  display: flex;
  justify-content: center;
  padding: var(--fonu-space-6);
}

.ddns-layout {
  display: grid;
  grid-template-columns: minmax(280px, 34%) minmax(0, 1fr);
  gap: var(--fonu-space-4);
  align-items: start;
}

.task-sidebar {
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  overflow: hidden;
}

.task-sidebar__head {
  padding: var(--fonu-space-4);
  border-bottom: 1px solid var(--fonu-border);
}

.task-sidebar__head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
}

.task-sidebar__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--fonu-text);
  flex-shrink: 0;
}

.task-sidebar__create {
  padding: var(--fonu-space-3);
  border-bottom: 1px solid var(--fonu-border);
  max-height: min(72vh, 680px);
  overflow-y: auto;
}

.task-sidebar__create .task-item--editing-create {
  margin: 0;
}

.task-sidebar__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: var(--fonu-space-3);
  max-height: 720px;
  overflow: auto;
}

.task-sidebar__list--with-create {
  max-height: min(48vh, 420px);
}

.task-item {
  width: 100%;
  text-align: left;
  border: 1px solid var(--fonu-border);
  border-radius: 12px;
  background: var(--fonu-surface);
  padding: 14px;
  cursor: pointer;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.task-item:hover { border-color: rgba(16, 185, 129, 0.25); }

.task-item--active {
  border-color: var(--fonu-brand);
  box-shadow: 0 0 0 1px rgba(16, 185, 129, 0.15);
}

.task-item--editing,
.task-item--draft {
  cursor: default;
}

.task-item--draft {
  border-style: dashed;
}

.task-item--editing {
  border-color: var(--fonu-brand);
  padding: 12px;
  overflow: visible;
}

.task-item__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.task-item__brand {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.provider-logo {
  display: inline-flex;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
  background: #fff;
  border: 1px solid var(--fonu-border);
}

.provider-logo--md { width: 32px; height: 32px; }
.provider-logo--lg { width: 40px; height: 40px; border-radius: 10px; }

.provider-logo img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.provider-logo--dnshe {
  background: #e0f2fe;
  border-color: #bae6fd;
  padding: 3px;
}

.task-item__name {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
}

.task-item__provider {
  font-size: 11px;
  color: var(--fonu-text-muted);
  margin-left: 6px;
}

.task-item__row {
  margin-top: 8px;
  font-size: 12px;
  color: var(--fonu-text-secondary);
}

.task-item__domain { font-weight: 600; color: var(--fonu-text); }
.task-item__muted { color: var(--fonu-text-muted); }

.task-item__draft-tag {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(245, 158, 11, 0.12);
  color: #d97706;
}

.task-item__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 10px;
}

.task-item__foot-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.task-item__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.ip-tag {
  display: inline-block;
  max-width: 100%;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--fonu-bg);
  border: 1px solid var(--fonu-border);
  font-family: var(--fonu-mono);
  font-size: 11px;
  color: var(--fonu-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-detail {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
  min-width: 0;
}

.task-detail__head {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  padding: var(--fonu-space-4) var(--fonu-space-5);
}

.task-detail__head-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  flex-wrap: wrap;
}

.task-detail__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  margin-top: var(--fonu-space-4);
  padding-top: var(--fonu-space-4);
  border-top: 1px solid var(--fonu-border);
}

.task-detail__title-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.task-detail__title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--fonu-text);
}

.task-detail__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-top: 6px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.task-detail__actions {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  margin-left: auto;
}

.detail-section { min-width: 0; }

.detail-section__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--fonu-text);
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--fonu-space-4);
  margin-top: var(--fonu-space-4);
}

.settings-field__label {
  display: block;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.settings-row {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  flex-wrap: wrap;
}

.settings-switches {
  display: flex;
  flex-wrap: wrap;
  gap: var(--fonu-space-5);
  align-items: center;
}

.settings-interval {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.settings-interval__label {
  font-size: 13px;
  color: var(--fonu-text-secondary);
  white-space: nowrap;
}

.settings-interval__input {
  width: 120px;
}

.settings-switch {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--fonu-text);
}

.interval-unit {
  font-size: 13px;
  color: var(--fonu-text-secondary);
}

.records-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-4) var(--fonu-space-5) 0;
}

.record-table-wrap {
  overflow-x: auto;
  padding: 0 var(--fonu-space-5) var(--fonu-space-4);
}

.record-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.record-table th {
  padding: 10px 12px;
  text-align: left;
  font-size: 12px;
  font-weight: 600;
  color: var(--fonu-text-secondary);
  border-bottom: 1px solid var(--fonu-border);
  white-space: nowrap;
}

.record-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--fonu-border);
  vertical-align: middle;
}

.record-table__mono {
  font-family: var(--fonu-mono);
  font-size: 12px;
}

.record-table__muted { color: var(--fonu-text-muted); }

.record-table__result {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.record-table__empty {
  text-align: center;
  color: var(--fonu-text-muted);
  padding: 28px 12px !important;
}

.record-row--editing {
  background: rgba(16, 185, 129, 0.04);
}

.alert {
  display: flex;
  gap: 12px;
  padding: 14px 16px;
  border-radius: 12px;
}

.alert--error {
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.18);
  color: #b91c1c;
}

.alert--info {
  background: rgba(59, 130, 246, 0.08);
  border: 1px solid rgba(59, 130, 246, 0.16);
  color: #1d4ed8;
}

.alert__icon {
  font-size: 20px;
  flex-shrink: 0;
  margin-top: 2px;
}

.alert__title { font-size: 14px; font-weight: 600; }

.alert__text {
  margin-top: 4px;
  font-size: 13px;
  line-height: 1.55;
  opacity: 0.92;
}

.alert__link {
  display: inline-block;
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
  color: inherit;
  text-decoration: underline;
}

.type-tags { display: flex; gap: 4px; }

.row-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

@media (max-width: 1199px) {
  .stats-row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .stat-card--ip { grid-column: 1 / -1; }
  .ddns-layout { grid-template-columns: 1fr; }
  .task-sidebar__list { max-height: none; }
}

@media (max-width: 767px) {
  .stats-row { grid-template-columns: 1fr; }
  .settings-grid { grid-template-columns: 1fr; }
  .settings-row { flex-direction: column; align-items: flex-start; }
  .task-sidebar__head-row { flex-direction: column; align-items: stretch; }
}
.stats-row { gap: 1px; background: var(--fonu-border); border: 1px solid var(--fonu-border); border-radius: var(--fonu-radius); overflow: hidden; }
.stats-row > .stat-card { border: 0; border-radius: 0; box-shadow: none; min-width: 0; background: var(--fonu-surface); }
.ddns-layout { grid-template-columns:minmax(240px, 29%) minmax(0,1fr); gap:12px; }
.task-sidebar__list { gap:4px; padding:8px; }
.task-sidebar__head { background:var(--fonu-bg-muted); padding:12px; }
.task-sidebar__head-row { flex-wrap:wrap; }
.task-item { border-radius:var(--fonu-radius-sm); box-shadow:none; }
.task-detail__toolbar { flex-wrap:wrap; gap:8px; }
.record-table-wrap { max-width:100%; overflow-x:auto; }
.stats-row > .stat-card { padding:12px 14px; min-height:104px; }
.stat-card__icon--blue { color:var(--fonu-brand); }
.stat-card__icon--green,.stat-card__value--green { color:var(--fonu-success); }
.stat-card__icon--red,.stat-card__value--red { color:var(--fonu-error); }
@media(max-width:1100px) { .ddns-layout { grid-template-columns:minmax(0,1fr); } .task-sidebar__list { max-height:300px; } }
@media(max-width:760px) { .stats-row { grid-template-columns:repeat(2,minmax(0,1fr)); } .stat-card--ip { grid-column:1/-1; } .stat-card__value { font-size:24px; } }

/* Beta 10: main shell owns vertical scrolling. */
.stats-row { display:flex; flex-direction:column; gap:0; background:var(--fonu-surface); }
.stats-row > .stat-card { min-height:0; padding:12px 16px; }
.stat-card--ip .ip-lines { display:grid; grid-template-columns:repeat(auto-fit,minmax(min(100%,250px),1fr)); gap:4px 24px; }
.stat-card--ip .ip-line { min-width:0; flex:none; height:auto; }
.ddns-summary { display:grid; grid-template-columns:repeat(5,minmax(0,1fr)); border-top:1px solid var(--fonu-border); }
.ddns-summary__item { display:flex; flex-direction:column; align-items:center; gap:3px; padding:10px 4px; color:var(--fonu-text); }
.ddns-summary__item + .ddns-summary__item { border-left:1px solid var(--fonu-border); }
.ddns-summary__item strong { font-size:20px; font-weight:600; }
.ddns-summary__item span { font-size:12px; color:var(--fonu-text-secondary); text-align:center; }
.task-sidebar__create, .task-sidebar__list, .task-sidebar__list--with-create { max-height:none; overflow:visible; }
.task-sidebar { min-width:0; }
.task-item { padding:10px 12px; }
.task-item__foot { margin-top:2px; justify-content:flex-end; }
.task-item__spacer { flex:1; }
.task-detail__head { padding:12px 16px; }
.dnshe-tabs { display:flex; flex-wrap:wrap; gap:8px; }
.ddns-help { color:var(--fonu-text-secondary); font-size:12px; padding:4px 8px; }
.ddns-help summary { cursor:pointer; }
@media(max-width:1199px) { .ddns-layout { grid-template-columns:minmax(0,1fr); } }
@media(max-width:480px) { .ddns-summary__item span { font-size:11px; } }

/* Accordion cards: one header, one inline detail, no nested vertical scroll. */
.ddns-layout { display:block; min-width:0; }
.task-sidebar { border:0; background:transparent; box-shadow:none; overflow:visible; }
.task-sidebar__head { background:transparent; padding:0 0 10px; border:0; }
.task-sidebar__head-row { flex-direction:row; align-items:center; }
.task-sidebar__list { display:grid; gap:12px; padding:0; }
.task-card { min-width:0; border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); background:var(--fonu-surface); }
.task-card__header { display:flex; align-items:center; gap:8px; padding:12px 14px; }
.task-card__select { display:flex; flex:1; align-items:center; gap:10px; min-width:0; border:0; background:transparent; text-align:left; color:var(--fonu-text); cursor:pointer; padding:0; }
.task-card__identity { display:grid; gap:3px; min-width:0; }
.task-card__identity strong { font-size:14px; overflow-wrap:anywhere; }
.task-card__identity > span { font-size:12px; color:var(--fonu-text-secondary); }
.task-card .task-detail { border-top:1px solid var(--fonu-border); padding:12px 14px; gap:12px; }
.task-card .task-detail__head { border:0; box-shadow:none; background:transparent; padding:0; }
.task-card .task-detail__toolbar { margin:0; padding:0; border:0; }
.task-card .detail-section { border:0; box-shadow:none; padding:0; background:transparent; }
.task-card .record-table { min-width:0; width:100%; table-layout:auto; }
.task-card .record-table th,.task-card .record-table td { padding:8px 5px; }
.record-domain { min-width:0; overflow-wrap:anywhere; }
.record-domain > span { display:block; white-space:normal; overflow-wrap:anywhere; }
.record-ip { font-size:12px; color:var(--fonu-text-secondary); }
.record-feedback { padding:10px 5px; color:var(--fonu-text-secondary); font-size:12px; }
.task-footer { display:flex; align-items:center; gap:6px; border-top:1px solid var(--fonu-border); padding-top:8px; }
@media(max-width:480px) { .task-card__header,.task-card .task-detail { padding:10px; } .task-card__header { gap:4px; } }

/* Compact status and icon actions, consistent with classic V1 theme. */
.stats-row { gap:10px; border:0; background:transparent; overflow:visible; }
.stats-row > .stat-card { border:1px solid var(--fonu-border); border-radius:var(--fonu-radius); padding:10px 14px; }
.stat-card--ip .ip-line { background:var(--fonu-bg-muted); border-radius:var(--fonu-radius-sm); padding:6px 10px; }
.ddns-summary { border:0; gap:8px; }
.ddns-summary__item { align-items:flex-start; padding:10px 12px; background:var(--fonu-surface); border:1px solid var(--fonu-border); border-radius:var(--fonu-radius-sm); }
.ddns-stat-label { display:flex; align-items:center; gap:6px; }
.ddns-stat-label i { width:7px; height:7px; border-radius:50%; background:var(--fonu-text-muted); }
.ddns-stat--1 strong { color:var(--fonu-success); }
.ddns-stat--1 i { background:var(--fonu-success); }
.ddns-stat--2 strong { color:var(--fonu-error); }
.ddns-stat--2 i { background:var(--fonu-error); }
.task-card__header { align-items:center; }
.task-chevron { flex:none; width:30px; }
.task-card .record-table { min-width:480px; }
.task-card .record-table th,.task-card .record-table td { vertical-align:middle; }
.task-card .record-table td:last-child { width:84px; }
.task-card .detail-section :deep(.fonu-card__body) { padding:0; }
@media(max-width:600px) { .ddns-summary { grid-template-columns:repeat(3,minmax(0,1fr)); } .task-card__header { flex-wrap:wrap; } }
</style>
