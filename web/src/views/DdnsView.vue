<template>
  <PageHeader title="DDNS" description="通过映射公网 IP 到 DNS 解析，支持多个域名及服务商">
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

  <template v-else-if="!loadError">
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

      <div class="stat-card">
        <n-icon :component="ListOutline" class="stat-card__icon stat-card__icon--gray" />
        <div class="stat-card__value">{{ taskCount }}</div>
        <div class="stat-card__label">任务总数</div>
        <div class="stat-card__sub">已配置的 DDNS 任务</div>
      </div>

      <div class="stat-card">
        <n-icon :component="CheckmarkCircleOutline" class="stat-card__icon stat-card__icon--green" />
        <div class="stat-card__value stat-card__value--green">{{ normalTaskCount }}</div>
        <div class="stat-card__label">正常任务</div>
        <div class="stat-card__sub">运行正常</div>
      </div>

      <div class="stat-card">
        <n-icon :component="AlertCircleOutline" class="stat-card__icon stat-card__icon--red" />
        <div class="stat-card__value stat-card__value--red">{{ abnormalTaskCount }}</div>
        <div class="stat-card__label">异常任务</div>
        <div class="stat-card__sub">需要处理</div>
      </div>

      <div class="stat-card">
        <n-icon :component="TimeOutline" class="stat-card__icon stat-card__icon--blue" />
        <div class="stat-card__value stat-card__value--sm">{{ lastSyncRelative }}</div>
        <div class="stat-card__label">最后同步</div>
        <div class="stat-card__sub">{{ lastSyncAbsolute }}</div>
      </div>
    </div>

    <div v-if="loading" class="loading-wrap">
      <n-spin size="medium" />
    </div>

    <EmptyState
      v-else-if="configs.length === 0 && !providerDraft && taskEditMode !== 'create'"
      title="还没有 DDNS 配置"
      description="添加 DNS 服务商后，在右侧表格中管理解析记录。"
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

        <div class="task-sidebar__list">
          <div v-if="taskEditMode === 'create'" class="task-item task-item--editing">
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

          <button
            v-if="providerDraft && showDraftInList"
            type="button"
            class="task-item task-item--draft"
            :class="{ 'task-item--active': selectedKey === DRAFT_KEY }"
            @click="selectDraft()"
          >
            <div class="task-item__top">
              <div class="task-item__brand">
                <span class="provider-logo">
                  <img :src="providerIcon(providerDraft.provider)" :alt="providerLabel(providerDraft.provider)" />
                </span>
                <span class="task-item__name">{{ providerLabel(providerDraft.provider) }}</span>
              </div>
              <span class="task-item__draft-tag">待添加记录</span>
            </div>
            <div class="task-item__row task-item__muted">保存服务商后，请在右侧添加解析记录</div>
            <div class="task-item__foot">
              <n-button size="tiny" quaternary @click.stop="startEditDraft()">编辑凭据</n-button>
              <n-button size="tiny" quaternary type="error" @click.stop="clearDraft()">取消</n-button>
            </div>
          </button>

          <template v-for="cfg in configs" :key="cfg.id">
            <div v-if="taskEditMode === cfg.id" class="task-item task-item--editing">
              <TaskProviderForm
                :form="taskForm"
                :editing="cfg"
                :saving="savingTask"
                :testing="testing"
                @save="saveTask"
                @test="testTask"
                @cancel="cancelTaskEdit"
              />
            </div>
            <button
              v-else
              type="button"
              class="task-item"
              :class="{ 'task-item--active': selectedKey === String(cfg.id) }"
              @click="selectTask(cfg.id)"
            >
              <div class="task-item__top">
                <div class="task-item__brand">
                  <span class="provider-logo">
                    <img :src="providerIcon(cfg.provider)" :alt="providerLabel(cfg.provider)" />
                  </span>
                  <span class="task-item__name">{{ providerLabel(cfg.provider) }}</span>
                </div>
                <StatusBadge :value="taskStatusKind(cfg)" :text="taskStatusText(cfg)" />
              </div>
              <div class="task-item__row">
                <span class="task-item__muted">上次同步</span>
                <span>{{ formatRelativeTime(cfg.last_updated_at) || '从未' }}</span>
              </div>
              <div class="task-item__row">
                <span class="task-item__muted">（共 {{ domainRecordsOf(cfg).length }} 条记录）</span>
              </div>
              <div class="task-item__foot">
                <div class="task-item__tags">
                  <span v-if="cfg.ipv4_enabled && cfg.last_ipv4" class="ip-tag">IPv4 {{ cfg.last_ipv4 }}</span>
                  <span v-if="cfg.ipv6_enabled && cfg.last_ipv6" class="ip-tag">IPv6 {{ shortIPv6(cfg.last_ipv6) }}</span>
                </div>
                <div class="task-item__foot-actions">
                  <n-button size="tiny" quaternary title="编辑服务商" @click.stop="startEditTask(cfg)">
                    <template #icon><n-icon :component="CreateOutline" /></template>
                  </n-button>
                  <n-switch
                    :value="cfg.enabled"
                    size="small"
                    :loading="togglingId === cfg.id"
                    @update:value="(v: boolean) => toggleEnabled(cfg, v)"
                    @click.stop
                  />
                </div>
              </div>
            </button>
          </template>
        </div>
      </aside>

      <section v-if="selectedTask" class="task-detail">
        <div class="task-detail__head">
          <div class="task-detail__title-wrap">
            <span class="provider-logo provider-logo--lg">
              <img :src="providerIcon(selectedTask.provider)" :alt="providerLabel(selectedTask.provider)" />
            </span>
            <div>
              <h3 class="task-detail__title">{{ providerLabel(selectedTask.provider) }}</h3>
              <div class="task-detail__meta">
                <StatusBadge
                  v-if="!isDraftSelected"
                  :value="taskStatusKind(selectedTask)"
                  :text="taskStatusText(selectedTask)"
                />
                <span v-if="isDraftSelected">待添加解析记录</span>
                <template v-else>
                  <span>上次同步 {{ formatRelativeTime(selectedTask.last_updated_at) || '从未' }}</span>
                  <span>（共 {{ domainRecordsOf(selectedTask).length }} 条记录）</span>
                </template>
              </div>
            </div>
          </div>
          <div v-if="!isDraftSelected" class="task-detail__actions">
            <n-button
              size="small"
              type="primary"
              ghost
              :loading="updatingId === selectedTask.id"
              @click="updateOne(selectedTask)"
            >
              <template #icon><n-icon :component="RefreshOutline" /></template>
              立即同步
            </n-button>
            <n-button size="small" quaternary type="error" @click="confirmDelete(selectedTask)">删除</n-button>
          </div>
        </div>

        <FonuCard class="detail-section">
          <h4 class="detail-section__title">基本设置</h4>
          <div class="settings-grid">
            <div class="settings-field">
              <label class="settings-field__label">DNS 服务商</label>
              <n-select :value="selectedTask.provider" :options="providerOptions" disabled />
            </div>
            <div class="settings-field">
              <label class="settings-field__label">API Token</label>
              <n-input value="••••••••••••" type="password" disabled>
                <template #suffix><n-icon :component="EyeOutline" /></template>
              </n-input>
            </div>
            <div class="settings-row">
              <div class="settings-switches">
                <div class="settings-switch">
                  <span>IPv4</span>
                  <n-switch
                    :value="selectedTask.ipv4_enabled"
                    size="small"
                    :loading="togglingId === selectedTask.id"
                    :disabled="isDraftSelected"
                    @update:value="(v: boolean) => !isDraftSelected && patchTaskFlags(selectedTask!, { ipv4_enabled: v })"
                  />
                </div>
                <div class="settings-switch">
                  <span>IPv6</span>
                  <n-switch
                    :value="selectedTask.ipv6_enabled"
                    size="small"
                    :loading="togglingId === selectedTask.id"
                    :disabled="isDraftSelected"
                    @update:value="(v: boolean) => !isDraftSelected && patchTaskFlags(selectedTask!, { ipv6_enabled: v })"
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
        </FonuCard>

        <FonuCard flush class="detail-section">
          <div class="records-head">
            <h4 class="detail-section__title">解析记录（{{ displayRecords.length }}）</h4>
            <n-button
              size="small"
              type="primary"
              :disabled="recordEditing !== null"
              @click="startAddRecord"
            >
              <template #icon><n-icon :component="AddOutline" /></template>
              添加记录
            </n-button>
          </div>

          <div class="record-table-wrap">
            <table class="record-table">
              <thead>
                <tr>
                  <th>主机记录</th>
                  <th>类型</th>
                  <th>当前解析值</th>
                  <th>上次结果</th>
                  <th>上次更新</th>
                  <th>状态</th>
                  <th>操作</th>
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
                      <n-tag v-if="selectedTask.ipv4_enabled" size="tiny" :bordered="false">A</n-tag>
                      <n-tag v-if="selectedTask.ipv6_enabled" size="tiny" :bordered="false" type="info">AAAA</n-tag>
                    </div>
                  </td>
                  <td class="record-table__muted">-</td>
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
                  v-for="record in displayRecords"
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
                        <n-tag v-if="selectedTask.ipv4_enabled" size="tiny" :bordered="false">A</n-tag>
                        <n-tag v-if="selectedTask.ipv6_enabled" size="tiny" :bordered="false" type="info">AAAA</n-tag>
                      </div>
                    </td>
                    <td class="record-table__muted">-</td>
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
                    <td class="record-table__mono">{{ record.domain }}</td>
                    <td>
                      <div class="type-tags">
                        <n-tag v-if="selectedTask.ipv4_enabled" size="tiny" :bordered="false">A</n-tag>
                        <n-tag v-if="selectedTask.ipv6_enabled" size="tiny" :bordered="false" type="info">AAAA</n-tag>
                      </div>
                    </td>
                    <td>
                      <span v-if="recordValue(record)" class="record-table__mono">{{ recordValue(record) }}</span>
                      <span v-else class="record-table__muted">-</span>
                    </td>
                    <td class="record-table__result">{{ record.message || recordResultLabel(record.status) }}</td>
                    <td>{{ formatRelativeTime(selectedTask.last_updated_at) || '-' }}</td>
                    <td>
                      <n-tag size="small" round :bordered="false" :type="recordTagType(record.status)">
                        {{ recordStatusLabel(record.status) }}
                      </n-tag>
                    </td>
                    <td>
                      <div class="row-actions">
                        <n-button
                          size="tiny"
                          quaternary
                          title="编辑"
                          :disabled="recordEditing !== null"
                          @click="startEditRecord(record)"
                        >
                          <template #icon><n-icon :component="CreateOutline" /></template>
                        </n-button>
                        <n-button
                          size="tiny"
                          quaternary
                          type="error"
                          title="删除"
                          :disabled="recordEditing !== null"
                          @click="confirmDeleteRecord(selectedTask, record)"
                        >
                          <template #icon><n-icon :component="TrashOutline" /></template>
                        </n-button>
                      </div>
                    </td>
                  </template>
                </tr>

                <tr v-if="displayRecords.length === 0 && recordEditing !== '__new__'">
                  <td colspan="7" class="record-table__empty">暂无解析记录，点击「添加记录」创建</td>
                </tr>
              </tbody>
            </table>
          </div>
        </FonuCard>

        <div v-if="detailError" class="alert alert--error">
          <n-icon :component="AlertCircleOutline" class="alert__icon" />
          <div class="alert__body">
            <div class="alert__title">{{ detailError.title }}</div>
            <div class="alert__text">{{ detailError.text }}</div>
            <router-link :to="{ name: 'logs', query: { tab: 'system' } }" class="alert__link">查看日志</router-link>
          </div>
        </div>

        <div class="alert alert--info">
          <n-icon :component="InformationCircleOutline" class="alert__icon" />
          <div class="alert__body">
            <div class="alert__title">提示</div>
            <div class="alert__text">
              每条记录请填写完整域名（如 s.example.com、www.sub.example.com）。同一服务商任务可管理多个不同域名，Fonu 会按同步周期自动检测公网 IP 并更新对应解析记录。
            </div>
          </div>
        </div>
      </section>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NIcon,
  NInput,
  NInputNumber,
  NSelect,
  NSpin,
  NSwitch,
  NTag,
  useDialog,
  useMessage,
} from 'naive-ui'
import {
  AddOutline,
  AlertCircleOutline,
  CheckmarkCircleOutline,
  CopyOutline,
  CreateOutline,
  EyeOutline,
  GlobeOutline,
  InformationCircleOutline,
  ListOutline,
  RefreshOutline,
  TimeOutline,
  TrashOutline,
} from '@vicons/ionicons5'
import aliyunIcon from '../assets/brand/dns/aliyun.png'
import cloudflareIcon from '../assets/brand/dns/cloudflare.png'
import dnspodIcon from '../assets/brand/dns/dnspod.png'
import tencentcloudIcon from '../assets/brand/dns/tencentcloud.png'
import volcengineIcon from '../assets/brand/dns/volcengine.png'
import { api, asList } from '../api/client'
import type { DDNSConfig, DDNSDomainRecord } from '../api/types'
import EmptyState from '../components/EmptyState.vue'
import FonuCard from '../components/FonuCard.vue'
import LoadError from '../components/LoadError.vue'
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import TaskProviderForm, { type ProviderForm } from '../components/TaskProviderForm.vue'
import { formatDate, formatRelativeTime } from '../utils/format'
import { statusLabel } from '../utils/status'

const DRAFT_KEY = '__draft__'

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
const taskEditMode = ref<'create' | number | null>(null)
const providerDraft = ref<ProviderForm | null>(null)
const updateInterval = ref(5)
const selectedKey = ref<string | null>(null)
const publicIPv4 = ref('')
const publicIPv6 = ref('')
const ipCheckedAt = ref('')
const recordEditing = ref<string | null>(null)
const recordDraft = reactive({ domain: '', originalDomain: '' })

const providerOptions = [
  { label: 'Cloudflare', value: 'cloudflare' },
  { label: 'DNSPod', value: 'dnspod' },
  { label: '阿里云 DNS', value: 'alidns' },
  { label: '腾讯云 DNS', value: 'tencentcloud' },
  { label: '火山引擎 DNS', value: 'volcengine' },
]

const providerMap: Record<string, { label: string; icon: string }> = {
  cloudflare: { label: 'Cloudflare', icon: cloudflareIcon },
  dnspod: { label: 'DNSPod', icon: dnspodIcon },
  alidns: { label: '阿里云 DNS', icon: aliyunIcon },
  tencentcloud: { label: '腾讯云 DNS', icon: tencentcloudIcon },
  volcengine: { label: '火山引擎 DNS', icon: volcengineIcon },
}

const taskForm = reactive<ProviderForm>({
  provider: 'cloudflare',
  api_token: '',
  api_token_id: '',
  api_secret: '',
  ipv4_enabled: true,
  ipv6_enabled: false,
  enabled: true,
})

const isDraftSelected = computed(() => selectedKey.value === DRAFT_KEY)

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
  if (isDraftSelected.value) return draftAsConfig.value
  const id = Number(selectedKey.value)
  if (!id) return null
  return configs.value.find((c) => c.id === id) ?? null
})

const showDraftInList = computed(() => taskEditMode.value !== 'create')

const taskCount = computed(() => configs.value.length + (providerDraft.value ? 1 : 0))

const normalTaskCount = computed(() => configs.value.filter((c) => c.enabled && isTaskHealthy(c)).length)
const abnormalTaskCount = computed(() => configs.value.filter((c) => c.enabled && !isTaskHealthy(c)).length)

const lastSyncIso = computed(() => {
  const times = configs.value.map((c) => c.last_updated_at).filter(Boolean) as string[]
  if (times.length === 0) return ''
  return times.sort((a, b) => new Date(b).getTime() - new Date(a).getTime())[0]
})

const lastSyncRelative = computed(() => formatRelativeTime(lastSyncIso.value) || '-')
const lastSyncAbsolute = computed(() => (lastSyncIso.value ? formatDate(lastSyncIso.value) : '暂无同步记录'))
const ipCheckedLabel = computed(() => formatRelativeTime(ipCheckedAt.value) || '刚刚')

const displayRecords = computed(() => {
  const task = selectedTask.value
  if (!task || isDraftSelected.value) return []
  return domainRecordsOf(task)
})

const detailError = computed(() => {
  const cfg = selectedTask.value
  if (!cfg || isDraftSelected.value) return null
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

function shortIPv6(ip: string) {
  return ip.length <= 24 ? ip : `${ip.slice(0, 18)}…`
}

function isTaskHealthy(cfg: DDNSConfig) {
  if (!cfg.enabled) return false
  return cfg.last_status === 'ok' || cfg.last_status === 'unchanged' || !cfg.last_status
}

function taskStatusKind(cfg: DDNSConfig) {
  if (!cfg.enabled) return 'disabled'
  if (cfg.last_status === 'error' || cfg.last_status === 'warning') return cfg.last_status
  return 'ok'
}

function taskStatusText(cfg: DDNSConfig) {
  if (!cfg.enabled) return '暂停'
  if (cfg.last_status === 'error' || cfg.last_status === 'warning') return '异常'
  return '正常'
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
  const task = selectedTask.value
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
  })
}

function ensureSelection() {
  if (providerDraft.value) {
    if (!selectedKey.value) selectedKey.value = DRAFT_KEY
    return
  }
  if (configs.value.length === 0) {
    selectedKey.value = null
    return
  }
  if (!configs.value.some((c) => String(c.id) === selectedKey.value)) {
    selectedKey.value = String(configs.value[0].id)
  }
}

watch(configs, ensureSelection)

function selectTask(id: number) {
  selectedKey.value = String(id)
  cancelRecordEdit()
}

function selectDraft() {
  selectedKey.value = DRAFT_KEY
  cancelRecordEdit()
}

function startCreateTask() {
  resetTaskForm()
  taskEditMode.value = 'create'
  cancelRecordEdit()
}

function startEditTask(cfg: DDNSConfig) {
  selectedKey.value = String(cfg.id)
  Object.assign(taskForm, {
    provider: cfg.provider || 'cloudflare',
    api_token: '',
    api_token_id: '',
    api_secret: '',
    ipv4_enabled: cfg.ipv4_enabled,
    ipv6_enabled: cfg.ipv6_enabled,
    enabled: cfg.enabled,
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
  try {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  } catch {
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
    // keep cache
  }
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
  try {
    const frp = await api.getFRP()
    frpEnabled.value = frp.enabled
  } catch {
    frpEnabled.value = false
  }
}

async function saveTask() {
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
  if (!selectedTask.value) return
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
  const task = selectedTask.value
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

function confirmDelete(row: DDNSConfig) {
  dialog.warning({
    title: '删除 DDNS 配置',
    content: `确定删除 ${providerLabel(row.provider)} 的 DDNS 配置吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await api.deleteDDNS(row.id)
      configs.value = configs.value.filter((c) => c.id !== row.id)
      ensureSelection()
      message.success('已删除')
    },
  })
}

function confirmDeleteRecord(row: DDNSConfig, record: DDNSDomainRecord) {
  dialog.warning({
    title: '删除解析记录',
    content: `确定从 DDNS 中移除 ${record.domain} 吗？`,
    positiveText: '删除',
    negativeText: '取消',
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

.task-sidebar__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: var(--fonu-space-3);
  max-height: 720px;
  overflow: auto;
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
  overflow: hidden;
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

.task-item__name {
  font-size: 14px;
  font-weight: 600;
  color: var(--fonu-text);
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
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--fonu-space-4);
  flex-wrap: wrap;
  background: var(--fonu-surface);
  border: 1px solid var(--fonu-border);
  border-radius: var(--fonu-radius);
  box-shadow: var(--fonu-shadow);
  padding: var(--fonu-space-4) var(--fonu-space-5);
}

.task-detail__title-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
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
  flex-wrap: wrap;
  gap: 6px;
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
</style>
