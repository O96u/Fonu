<template>
  <PageHeader title="DNSHE 域名" description="管理多个 DNSHE 账户的免费域名，支持续期、删除、转赠与接收">
    <template #actions>
      <n-button type="primary" @click="openAddAccount">
        <template #icon><n-icon :component="AddOutline" /></template>
        添加账户
      </n-button>
    </template>
  </PageHeader>

  <LoadError v-if="loadError" :message="loadError" @retry="init" />

  <template v-else>
    <n-alert class="dnshe-tip" type="info" :bordered="false" title="规则说明">
      域名到期前 <strong>180 天</strong>起可续期，每次延长 1 年。转赠域名需注册满 <strong>30 天</strong>，
      转赠码 <strong>3 天</strong>内有效；接收方输入码即可将域名转入自己账户。账户分类可拖动左侧手柄排序。
    </n-alert>

    <div class="dnshe-layout">
      <!-- 账户侧栏 -->
      <aside class="dnshe-sidebar">
        <FonuCard title="账户列表" subtitle="拖动手柄排序">
          <button class="add-account-btn" @click="openAddAccount">
            <n-icon :component="AddOutline" />
            添加账户
          </button>

          <div v-if="accounts.length === 0" class="sidebar-empty">还没有账户，点击上方按钮添加</div>

          <div
            v-for="account in accounts"
            :key="account.id"
            class="account-item"
            :class="{
              'account-item--active': account.id === selectedId,
              'account-item--dragging': account.id === dragId,
              'account-item--over': account.id === overId && account.id !== dragId,
            }"
            draggable="true"
            @click="selectAccount(account.id)"
            @dragstart="onDragStart($event, account.id)"
            @dragover="onDragOver($event, account.id)"
            @dragleave="onDragLeave(account.id)"
            @drop="onDrop($event)"
            @dragend="onDragEnd"
          >
            <n-icon class="account-item__grip" :component="ReorderThreeOutline" />
            <div class="account-item__main">
              <div class="account-item__name">{{ account.name }}</div>
              <div class="account-item__meta">
                <template v-if="account.auto_renew">自动续期已开启</template>
                <template v-else>自动续期已关闭</template>
                <template v-if="account.last_run_at"> · {{ account.last_run_at }}</template>
              </div>
            </div>
            <div class="account-item__ops" @click.stop>
              <n-button quaternary size="tiny" title="查看凭据" @click="openCredentials(account)">
                <template #icon><n-icon :component="KeyOutline" /></template>
              </n-button>
              <n-button quaternary size="tiny" title="推送到 DDNS" @click="confirmPushToDDNS(account)">
                <template #icon><n-icon :component="CloudUploadOutline" /></template>
              </n-button>
              <n-button quaternary size="tiny" @click="openEditAccount(account)">
                <template #icon><n-icon :component="CreateOutline" /></template>
              </n-button>
              <n-button quaternary size="tiny" type="error" @click="confirmDeleteAccount(account)">
                <template #icon><n-icon :component="TrashOutline" /></template>
              </n-button>
            </div>
          </div>
        </FonuCard>
      </aside>

      <!-- 域名详情 -->
      <main class="dnshe-content">
        <FonuCard class="dnshe-card">
          <template #title>
            <span class="detail-title">{{ selectedAccount ? selectedAccount.name : '域名列表' }}</span>
          </template>
          <template #header>
            <n-space :size="8">
              <n-button size="small" :loading="loadingDetail" @click="loadDetail">
                <template #icon><n-icon :component="RefreshOutline" /></template>
                刷新
              </n-button>
              <n-button size="small" :disabled="!selectedId" @click="openRegister">
                <template #icon><n-icon :component="AddCircleOutline" /></template>
                注册域名
              </n-button>
              <n-button size="small" :disabled="!selectedId" @click="openAccept">
                <template #icon><n-icon :component="DownloadOutline" /></template>
                接收域名
              </n-button>
            </n-space>
          </template>

          <div v-if="!selectedId" class="detail-empty">
            <EmptyState title="请选择账户" description="在左侧选择一个 DNSHE 账户，或添加新账户。" />
          </div>
          <div v-else-if="loadingDetail" class="loading-wrap"><n-spin size="medium" /></div>
          <EmptyState
            v-else-if="domains.length === 0"
            title="暂无域名"
            description="该账户下还没有免费域名，可注册或通过转赠码接收。"
          />
          <n-data-table
            v-else
            :columns="columns"
            :data="domains"
            :row-key="(row: DNSHEDomain) => row.id"
            :row-props="domainRowProps"
            :pagination="false"
            size="small"
          />
        </FonuCard>

        <FonuCard v-if="selectedId" title="转赠记录" subtitle="该账户发出与接收的域名转赠" class="dnshe-card">
          <div v-if="loadingDetail" class="loading-wrap"><n-spin size="small" /></div>
          <EmptyState v-else-if="gifts.length === 0" title="暂无转赠记录" description="转赠或接收域名后，记录会显示在这里。" />
          <n-data-table
            v-else
            :columns="giftColumns"
            :data="gifts"
            :row-key="(row: DNSHEGift) => row.id"
            :pagination="false"
            size="small"
          />
        </FonuCard>
      </main>
    </div>

    <!-- 账户新增/编辑 -->
    <n-modal v-model:show="accountModal.show" preset="card" style="width: 480px" :title="accountModal.editing ? '编辑账户' : '添加账户'">
      <n-form label-placement="top">
        <n-form-item label="账户名称">
          <n-input v-model:value="accountModal.name" placeholder="例如：主账户" />
        </n-form-item>
        <n-form-item label="API Key">
          <n-input v-model:value="accountModal.apiKey" type="password" show-password-on="click"
            :placeholder="accountModal.editing ? '留空表示不修改' : '请输入 DNSHE API Key'" />
        </n-form-item>
        <n-form-item label="API Secret">
          <n-input v-model:value="accountModal.apiSecret" type="password" show-password-on="click"
            :placeholder="accountModal.editing ? '留空表示不修改' : '请输入 DNSHE API Secret'" />
        </n-form-item>
        <n-form-item label="自动续期">
          <n-switch v-model:value="accountModal.autoRenew" />
          <span class="switch-hint">开启后按续期窗口自动安排，只在可续期时发起请求</span>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="accountModal.show = false">取消</n-button>
          <n-button type="primary" :loading="accountModal.saving" @click="saveAccount">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 转赠码 -->
    <n-modal v-model:show="giftModal.show" preset="card" style="width: 440px" title="转赠码已生成">
      <p class="gift-tip">将下方转赠码发给接收方，对方在本页面点击「接收域名」输入码即可完成转赠。</p>
      <div class="gift-code-box">
        <span class="gift-code">{{ giftModal.code }}</span>
        <n-button size="small" @click="copyCode">
          <template #icon><n-icon :component="CopyOutline" /></template>
          复制
        </n-button>
      </div>
      <p class="gift-meta">域名：{{ giftModal.domain }}<template v-if="giftModal.expiresAt"> · 有效期至 {{ giftModal.expiresAt }}</template></p>
      <template #footer>
        <n-space justify="end">
          <n-button @click="giftModal.show = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 接收域名 -->
    <n-modal v-model:show="acceptModal.show" preset="card" style="width: 440px" title="接收域名">
      <p class="gift-tip">输入转赠方提供的转赠码，域名将转入当前账户。</p>
      <n-input v-model:value="acceptModal.code" placeholder="请输入转赠码" class="accept-input" />
      <template #footer>
        <n-space justify="end">
          <n-button @click="acceptModal.show = false">取消</n-button>
          <n-button type="primary" :loading="acceptModal.submitting" @click="submitAccept">确认接收</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 查看凭据 -->
    <n-modal v-model:show="credModal.show" preset="card" style="width: 480px" :title="`API 凭据 - ${credModal.name}`">
      <n-spin :show="credModal.loading">
        <n-form label-placement="top">
          <n-form-item label="API Key">
            <n-input :value="credModal.apiKey" readonly>
              <template #suffix>
                <n-button quaternary size="tiny" @click="copyText(credModal.apiKey)">
                  <template #icon><n-icon :component="CopyOutline" /></template>
                </n-button>
              </template>
            </n-input>
          </n-form-item>
          <n-form-item label="API Secret">
            <n-input :value="credModal.apiSecret" readonly>
              <template #suffix>
                <n-button quaternary size="tiny" @click="copyText(credModal.apiSecret)">
                  <template #icon><n-icon :component="CopyOutline" /></template>
                </n-button>
              </template>
            </n-input>
          </n-form-item>
        </n-form>
      </n-spin>
      <template #footer>
        <n-space justify="end">
          <n-button @click="credModal.show = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 注册域名 -->
    <n-modal v-model:show="registerModal.show" preset="card" style="width: 480px" title="注册新域名">
      <n-form label-placement="top">
        <n-form-item label="主域名后缀">
          <n-select
            v-model:value="registerModal.root"
            :options="rootDomainOptions"
            filterable
          />
        </n-form-item>
        <n-form-item label="域名前缀">
          <n-input
            v-model:value="registerModal.prefix"
            placeholder="4-30 位小写字母或数字，例如 myhome"
            @input="registerModal.prefix = registerModal.prefix.toLowerCase()"
          />
        </n-form-item>
        <div v-if="registerModal.prefix" class="register-preview">
          完整域名：<strong>{{ registerPreview }}</strong>
        </div>
        <p class="register-note">
          免费前缀至少 4 位；过短的前缀 DNSHE 会要求付费。若域名已被占用或后缀不可注册，请根据提示更换。
        </p>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="registerModal.show = false">取消</n-button>
          <n-button type="primary" :loading="registerModal.submitting" @click="submitRegister">注册</n-button>
        </n-space>
      </template>
    </n-modal>
  </template>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NAlert,
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NTooltip,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import {
  AddCircleOutline,
  AddOutline,
  CloudUploadOutline,
  CopyOutline,
  CreateOutline,
  DownloadOutline,
  KeyOutline,
  ReorderThreeOutline,
  RefreshOutline,
  TrashOutline,
} from '@vicons/ionicons5'
import { api } from '../api/client'
import type { DNSHEAccount, DNSHEDomain, DNSHEGift } from '../api/types'
import PageHeader from '../components/PageHeader.vue'
import FonuCard from '../components/FonuCard.vue'
import EmptyState from '../components/EmptyState.vue'
import LoadError from '../components/LoadError.vue'
import StatusBadge from '../components/StatusBadge.vue'
import type { StatusKind } from '../utils/status'

const message = useMessage()
const dialog = useDialog()

const accounts = ref<DNSHEAccount[]>([])
const selectedId = ref('')
const domains = ref<DNSHEDomain[]>([])
const gifts = ref<DNSHEGift[]>([])

const loadError = ref('')
const loadingDetail = ref(false)

const renewingId = ref<number | null>(null)
const deletingId = ref<number | null>(null)
const giftingId = ref<number | null>(null)
const cancellingGiftId = ref<number | null>(null)

const selectedAccount = computed(() => accounts.value.find((a) => a.id === selectedId.value))

// ---- 账户弹窗 ----
const accountModal = reactive({
  show: false,
  saving: false,
  editing: '',
  name: '',
  apiKey: '',
  apiSecret: '',
  autoRenew: false,
})

function openAddAccount() {
  accountModal.editing = ''
  accountModal.name = ''
  accountModal.apiKey = ''
  accountModal.apiSecret = ''
  accountModal.autoRenew = true
  accountModal.show = true
}

function openEditAccount(account: DNSHEAccount) {
  accountModal.editing = account.id
  accountModal.name = account.name
  accountModal.apiKey = ''
  accountModal.apiSecret = ''
  accountModal.autoRenew = account.auto_renew
  accountModal.show = true
}

async function saveAccount() {
  if (!accountModal.name.trim()) {
    message.warning('请填写账户名称')
    return
  }
  if (!accountModal.editing && (!accountModal.apiKey.trim() || !accountModal.apiSecret.trim())) {
    message.warning('请填写 API Key 和 API Secret')
    return
  }
  accountModal.saving = true
  try {
    if (accountModal.editing) {
      await api.updateDNSHEAccount(accountModal.editing, {
        name: accountModal.name,
        api_key: accountModal.apiKey || undefined,
        api_secret: accountModal.apiSecret || undefined,
        auto_renew: accountModal.autoRenew,
      })
      message.success('已保存')
    } else {
      const created = await api.createDNSHEAccount({
        name: accountModal.name,
        api_key: accountModal.apiKey,
        api_secret: accountModal.apiSecret,
        auto_renew: accountModal.autoRenew,
      })
      message.success('账户已添加')
      selectedId.value = created.id
    }
    accountModal.show = false
    await loadAccounts()
    await loadDetail()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    accountModal.saving = false
  }
}

function confirmDeleteAccount(account: DNSHEAccount) {
  dialog.warning({
    title: '删除账户',
    content: `确定删除账户「${account.name}」吗？仅删除本地保存的凭据，不影响 DNSHE 线上域名。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.deleteDNSHEAccount(account.id)
        message.success('账户已删除')
        if (selectedId.value === account.id) selectedId.value = ''
        await loadAccounts()
        if (accounts.value.length > 0) {
          if (!selectedId.value) selectedId.value = accounts.value[0].id
          await loadDetail()
        }
      } catch (e) {
        message.error(e instanceof Error ? e.message : '删除失败')
      }
    },
  })
}

// ---- 拖拽排序 ----
const dragId = ref('')
const overId = ref('')

function onDragStart(e: DragEvent, id: string) {
  dragId.value = id
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', id)
  }
}

function onDragOver(e: DragEvent, id: string) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  if (overId.value !== id) overId.value = id
}

function onDragLeave(id: string) {
  if (overId.value === id) overId.value = ''
}

async function onDrop(e: DragEvent) {
  e.preventDefault()
  const from = dragId.value
  const to = overId.value
  dragId.value = ''
  overId.value = ''
  if (!from || !to || from === to) return

  const ids = accounts.value.map((a) => a.id)
  const fromIdx = ids.indexOf(from)
  const toIdx = ids.indexOf(to)
  if (fromIdx < 0 || toIdx < 0) return

  const snapshot = accounts.value.slice()
  const [moved] = ids.splice(fromIdx, 1)
  ids.splice(toIdx, 0, moved)
  applyOrder(ids)

  try {
    await api.reorderDNSHEAccounts(ids)
    message.success('排序已保存')
  } catch (err) {
    accounts.value = snapshot
    message.error(err instanceof Error ? err.message : '排序保存失败')
  }
}

function onDragEnd() {
  dragId.value = ''
  overId.value = ''
}

function applyOrder(ids: string[]) {
  const byID = new Map(accounts.value.map((a) => [a.id, a]))
  accounts.value = ids.map((id) => byID.get(id)!).filter(Boolean)
}

// ---- 状态映射 ----

function domainStatusKind(status: string): StatusKind {
  const s = (status || '').toLowerCase()
  if (['active', 'ok', 'normal', 'running', 'enabled', 'registered'].includes(s)) return 'success'
  if (['pending', 'pending_delete', 'transferring'].includes(s)) return 'warning'
  if (['expired', 'suspended', 'deleted', 'disabled', 'locked'].includes(s)) return 'error'
  return 'unknown'
}

function domainStatusLabel(status: string): string {
  const map: Record<string, string> = {
    active: '正常',
    ok: '正常',
    normal: '正常',
    running: '正常',
    enabled: '正常',
    registered: '正常',
    pending: '待审核',
    pending_delete: '待删除',
    transferring: '转移中',
    expired: '已过期',
    suspended: '已暂停',
    deleted: '已删除',
    disabled: '已停用',
    locked: '已锁定',
  }
  return map[(status || '').toLowerCase()] ?? (status || '未知')
}

function giftStatusKind(status: string): StatusKind {
  const s = (status || '').toLowerCase()
  if (s === 'accepted') return 'success'
  if (s === 'pending') return 'warning'
  if (s === 'expired') return 'error'
  return 'unknown'
}

function giftStatusLabel(status: string): string {
  const map: Record<string, string> = {
    pending: '待接收',
    accepted: '已接收',
    cancelled: '已取消',
    expired: '已过期',
  }
  return map[(status || '').toLowerCase()] ?? (status || '未知')
}

// ---- 域名表格 ----

// parseFlexibleTime 解析 DNSHE 时间字符串，兼容 "2006-01-02 15:04:05"。
function parseFlexibleTime(raw: string): number | null {
  const t = new Date(raw.trim().replace(' ', 'T')).getTime()
  return Number.isNaN(t) ? null : t
}

// domainAgeDays 返回域名注册天数。
function domainAgeDays(row: DNSHEDomain): number | null {
  if (!row.created_at) return null
  const ms = parseFlexibleTime(row.created_at)
  if (ms === null) return null
  return Math.floor((Date.now() - ms) / 86_400_000)
}

// canGift 表示域名是否可转赠（官方要求注册满 30 天）。
function canGift(row: DNSHEDomain): boolean {
  const age = domainAgeDays(row)
  return age !== null && age >= 30
}

function daysLeftCell(row: DNSHEDomain) {
  if (row.never_expires) return h('span', { class: 'muted' }, '永不过期')
  if (row.days_left === null || row.days_left === undefined) return h('span', { class: 'muted' }, '-')
  const cls = row.days_left <= 7 ? 'days days--error' : row.days_left <= 30 ? 'days days--warning' : 'days'
  return h('span', { class: cls }, `${row.days_left} 天`)
}

function actionButton(text: string, options: Record<string, unknown>, type: 'default' | 'primary' | 'error' = 'default') {
  return h(
    NButton,
    { size: 'small', text: true, type, ...options },
    { default: () => text },
  )
}

const columns: DataTableColumns<DNSHEDomain> = [
  {
    key: 'drag',
    width: 32,
    render: () =>
      h(
        NIcon,
        { class: 'domain-grip', size: 15 },
        { default: () => h(ReorderThreeOutline) },
      ),
  },
  {
    title: '域名',
    key: 'full_domain',
    width: 210,
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'domain-name' }, row.full_domain),
  },
  {
    title: '状态',
    key: 'status',
    width: 80,
    render: (row) =>
      h(StatusBadge, { kind: domainStatusKind(row.status), text: domainStatusLabel(row.status) }),
  },
  { title: '注册时间', key: 'created_at', width: 160, render: (row) => row.created_at || '-' },
  {
    title: '到期时间',
    key: 'expires_at',
    width: 160,
    render: (row) => (row.never_expires ? '永不过期' : row.expires_at || '-'),
  },
  { title: '剩余天数', key: 'days_left', width: 90, render: daysLeftCell },
  {
    title: '自动续期',
    key: 'auto_renew',
    width: 90,
    render: (row) =>
      h(NSwitch, {
        size: 'small',
        value: row.auto_renew,
        disabled: !row.renewable,
        'onUpdate:value': (v: boolean) => toggleDomainAutoRenew(row, v),
      }),
  },
  {
    title: '操作',
    key: 'actions',
    width: 205,
    render: (row) => {
      const giftable = canGift(row)
      const giftBtn = actionButton('转赠', {
        disabled: !giftable,
        loading: giftingId.value === row.id,
        onClick: () => giftDomain(row),
      })
      const giftTip = h(NTooltip, { placement: 'top' }, {
        trigger: () =>
          // disabled 按钮不触发鼠标事件，需套一层 span 承载 tooltip
          h('span', { class: 'gift-trigger' }, [giftBtn]),
        default: () => {
          const age = domainAgeDays(row)
          return giftable
            ? '转赠域名给其他 DNSHE 账户'
            : `域名注册满 30 天后才可转赠（当前 ${age ?? '?'} 天）`
        },
      })
      return h('div', { class: 'row-actions' }, [
        actionButton('续期', {
          disabled: !row.renewable,
          loading: renewingId.value === row.id,
          onClick: () => renewDomain(row),
        }, 'primary'),
        giftTip,
        actionButton('删除', {
          loading: deletingId.value === row.id,
          onClick: () => confirmDeleteDomain(row),
        }, 'error'),
      ])
    },
  },
]

const giftColumns: DataTableColumns<DNSHEGift> = [
  {
    title: '转赠码',
    key: 'code',
    width: 210,
    render: (row) =>
      h('span', { class: 'gift-code-cell', title: '点击复制' }, [
        h(
          'span',
          { class: 'domain-name', onClick: () => copyText(row.code) },
          row.code,
        ),
      ]),
  },
  { title: '域名', key: 'full_domain', width: 200, ellipsis: { tooltip: true } },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row) =>
      h(StatusBadge, { kind: giftStatusKind(row.status), text: giftStatusLabel(row.status) }),
  },
  { title: '创建时间', key: 'created_at', width: 160, render: (row) => row.created_at || '-' },
  { title: '有效期至', key: 'expires_at', width: 160, render: (row) => row.expires_at || '-' },
  {
    title: '操作',
    key: 'actions',
    width: 90,
    render: (row) =>
      row.status === 'pending'
        ? actionButton('取消', {
            loading: cancellingGiftId.value === row.id,
            onClick: () => cancelGift(row),
          }, 'error')
        : h('span', { class: 'muted' }, '-'),
  },
]

// ---- 域名行拖拽 ----

const dragDomainId = ref<number | null>(null)
const overDomainId = ref<number | null>(null)

function domainRowProps(row: DNSHEDomain) {
  return {
    draggable: true,
    class: {
      'domain-row--dragging': row.id === dragDomainId.value,
      'domain-row--over':
        row.id === overDomainId.value && row.id !== dragDomainId.value,
    },
    onDragstart: (e: DragEvent) => {
      dragDomainId.value = row.id
      e.dataTransfer?.setData('text/plain', String(row.id))
      if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
    },
    onDragover: (e: DragEvent) => {
      e.preventDefault()
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
      if (overDomainId.value !== row.id) overDomainId.value = row.id
    },
    onDragleave: () => {
      if (overDomainId.value === row.id) overDomainId.value = null
    },
    onDrop: (e: DragEvent) => onDomainDrop(e, row.id),
    onDragend: () => {
      dragDomainId.value = null
      overDomainId.value = null
    },
  }
}

async function onDomainDrop(e: DragEvent, targetId: number) {
  e.preventDefault()
  const from = dragDomainId.value
  dragDomainId.value = null
  overDomainId.value = null
  if (from === null || from === targetId) return

  const ids = domains.value.map((d) => d.id)
  const fromIdx = ids.indexOf(from)
  const toIdx = ids.indexOf(targetId)
  if (fromIdx < 0 || toIdx < 0) return

  const snapshot = domains.value.slice()
  const [moved] = ids.splice(fromIdx, 1)
  ids.splice(toIdx, 0, moved)

  const byID = new Map(domains.value.map((d) => [d.id, d]))
  domains.value = ids.map((id) => byID.get(id)!)

  try {
    await api.reorderDNSHEDomains(selectedId.value, ids)
    message.success('域名排序已保存')
  } catch (err) {
    domains.value = snapshot
    message.error(err instanceof Error ? err.message : '排序保存失败')
  }
}

// ---- 数据加载 ----

async function loadAccounts() {
  accounts.value = await api.listDNSHEAccounts()
}

function selectAccount(id: string) {
  if (id === selectedId.value) return
  selectedId.value = id
  loadDetail()
}

async function loadDetail() {
  if (!selectedId.value) return
  loadingDetail.value = true
  try {
    const [domainList, giftList] = await Promise.all([
      api.listDNSHEDomains(selectedId.value),
      api.listDNSHEGifts(selectedId.value),
    ])
    domains.value = domainList
    gifts.value = giftList
  } catch (e) {
    message.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loadingDetail.value = false
  }
}

async function init() {
  loadError.value = ''
  try {
    await loadAccounts()
    if (accounts.value.length > 0) {
      selectedId.value = accounts.value[0].id
      await loadDetail()
    }
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : '加载失败'
  }
}

// ---- 域名操作 ----

async function renewDomain(row: DNSHEDomain) {
  renewingId.value = row.id
  try {
    const result = await api.renewDNSHEDomain(selectedId.value, row.id)
    message.success(result.new_expires_at ? `续期成功，新到期时间：${result.new_expires_at}` : '续期成功')
    await loadDetail()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '续期失败')
  } finally {
    renewingId.value = null
  }
}

function confirmDeleteDomain(row: DNSHEDomain) {
  dialog.warning({
    title: '删除域名',
    content: `确定删除域名「${row.full_domain}」吗？域名及其 DNS 记录将被永久删除，无法恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: () => deleteDomain(row),
  })
}

async function deleteDomain(row: DNSHEDomain) {
  deletingId.value = row.id
  try {
    await api.deleteDNSHEDomain(selectedId.value, row.id)
    message.success('域名已删除')
    await loadDetail()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '删除失败')
  } finally {
    deletingId.value = null
  }
}

async function giftDomain(row: DNSHEDomain) {
  giftingId.value = row.id
  try {
    const gift = await api.initiateDNSHEGift(selectedId.value, row.id)
    giftModal.code = gift.code
    giftModal.domain = gift.full_domain
    giftModal.expiresAt = gift.expires_at ?? ''
    giftModal.show = true
    await loadDetail()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '生成转赠码失败')
  } finally {
    giftingId.value = null
  }
}

async function cancelGift(row: DNSHEGift) {
  cancellingGiftId.value = row.id
  try {
    await api.cancelDNSHEGift(selectedId.value, row.id)
    message.success('转赠已取消')
    await loadDetail()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '取消失败')
  } finally {
    cancellingGiftId.value = null
  }
}

async function toggleDomainAutoRenew(row: DNSHEDomain, enabled: boolean) {
  try {
    await api.setDNSHEDomainAutoRenew(row.full_domain, enabled)
    row.auto_renew = enabled
    message.success(enabled ? `已开启 ${row.full_domain} 的自动续期` : `已关闭 ${row.full_domain} 的自动续期`)
  } catch (e) {
    message.error(e instanceof Error ? e.message : '保存失败')
  }
}

// ---- 注册 ----

// DNSHE 未公开「可注册后缀列表」接口，内置已知免费后缀；
// 若后缀已停止注册，后端会返回 "root domain not allowed"。
const ROOT_DOMAINS = [
  'ccwu.cc',
  'us.ci',
  'bot.cd',
  'de5.net',
  'cc.cd',
  'l.cd',
  'ddns.ge',
  'bbroot.com',
]
const rootDomainOptions = ROOT_DOMAINS.map((value) => ({ label: value, value }))

const registerModal = reactive({
  show: false,
  submitting: false,
  prefix: '',
  root: ROOT_DOMAINS[0],
})

const registerPreview = computed(
  () => `${registerModal.prefix.trim().toLowerCase()}.${registerModal.root}`,
)

function openRegister() {
  registerModal.prefix = ''
  registerModal.root = ROOT_DOMAINS[0]
  registerModal.show = true
}

async function submitRegister() {
  const prefix = registerModal.prefix.trim().toLowerCase()
  if (!/^[a-z0-9][a-z0-9-]{2,28}[a-z0-9]$|^[a-z0-9]{4,30}$/.test(prefix)) {
    message.warning('前缀需 4-30 位小写字母、数字（可含连字符，不能以连字符开头/结尾）')
    return
  }
  registerModal.submitting = true
  try {
    const result = await api.registerDNSHEDomain(selectedId.value, {
      subdomain: prefix,
      domain: registerModal.root,
    })
    message.success(`注册成功：${result.full_domain}`)
    registerModal.show = false
    await loadDetail()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '注册失败')
  } finally {
    registerModal.submitting = false
  }
}

// ---- 接收 ----
const acceptModal = reactive({ show: false, submitting: false, code: '' })

function openAccept() {
  acceptModal.code = ''
  acceptModal.show = true
}

async function submitAccept() {
  const code = acceptModal.code.trim()
  if (!code) {
    message.warning('请输入转赠码')
    return
  }
  acceptModal.submitting = true
  try {
    const gift = await api.acceptDNSHEGift(selectedId.value, code)
    message.success(`已接收域名：${gift.full_domain}`)
    acceptModal.show = false
    await loadDetail()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '接收失败')
  } finally {
    acceptModal.submitting = false
  }
}

// ---- 查看凭据 ----
const credModal = reactive({
  show: false,
  loading: false,
  name: '',
  apiKey: '',
  apiSecret: '',
})

async function openCredentials(account: DNSHEAccount) {
  credModal.name = account.name
  credModal.apiKey = ''
  credModal.apiSecret = ''
  credModal.show = true
  credModal.loading = true
  try {
    const cred = await api.revealDNSHECredentials(account.id)
    credModal.apiKey = cred.api_key
    credModal.apiSecret = cred.api_secret
  } catch (error) {
    credModal.show = false
    message.error(error instanceof Error ? error.message : '获取凭据失败')
  } finally {
    credModal.loading = false
  }
}

// ---- 推送到 DDNS ----
function confirmPushToDDNS(account: DNSHEAccount) {
  dialog.warning({
    title: '推送到 DDNS',
    content: `将使用账户「${account.name}」的凭据，按主域名为其下所有域名创建 DDNS 任务；已存在 DDNS 任务的主域名会自动跳过。`,
    positiveText: '推送',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        const result = await api.pushDNSHEToDDNS(account.id)
        let text = result.message
        if (result.skipped.length > 0) {
          text += `；已跳过：${result.skipped.join('、')}`
        }
        message.success(text)
      } catch (error) {
        message.error(error instanceof Error ? error.message : '推送失败')
      }
    },
  })
}

// ---- 转赠码弹窗 ----
const giftModal = reactive({ show: false, code: '', domain: '', expiresAt: '' })

async function copyCode() {
  await copyText(giftModal.code)
}

async function copyText(text: string) {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const ta = document.createElement('textarea')
      ta.value = text
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    message.success('已复制')
  } catch {
    message.error('复制失败，请手动选择复制')
  }
}

onMounted(init)
</script>

<style scoped>
.dnshe-tip {
  margin-bottom: var(--fonu-space-4);
}

.dnshe-layout {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: var(--fonu-space-4);
  align-items: start;
}

.dnshe-sidebar {
  position: sticky;
  top: var(--fonu-space-4);
}

.add-account-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  padding: 8px 0;
  border: 1px dashed var(--fonu-border);
  border-radius: var(--fonu-radius, 8px);
  background: transparent;
  color: var(--fonu-primary);
  font-size: 13px;
  cursor: pointer;
  margin-bottom: var(--fonu-space-2);
}

.add-account-btn:hover {
  border-color: var(--fonu-primary);
  background: var(--fonu-primary-soft, rgba(24, 160, 88, 0.08));
}

.sidebar-empty {
  padding: var(--fonu-space-4) 0;
  text-align: center;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.account-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
  border-radius: var(--fonu-radius, 8px);
  border: 1px solid transparent;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.account-item:hover {
  background: var(--fonu-surface-hover, rgba(0, 0, 0, 0.04));
}

.account-item--active {
  background: var(--fonu-primary-soft, rgba(24, 160, 88, 0.1));
  border-color: var(--fonu-primary);
}

.account-item--dragging {
  opacity: 0.45;
}

.account-item--over {
  border-color: var(--fonu-primary);
}

.account-item__grip {
  color: var(--fonu-text-muted);
  font-size: 16px;
  cursor: grab;
}

.account-item__main {
  flex: 1;
  min-width: 0;
}

.account-item__name {
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.account-item__meta {
  font-size: 11px;
  color: var(--fonu-text-muted);
  margin-top: 2px;
}

.account-item__ops {
  display: none;
  gap: 2px;
}

.account-item:hover .account-item__ops {
  display: flex;
}

.detail-title {
  font-weight: 600;
}

.detail-empty,
.loading-wrap {
  display: flex;
  justify-content: center;
  padding: var(--fonu-space-5) 0;
}

.dnshe-card {
  margin-bottom: var(--fonu-space-4);
}

.domain-name {
  font-weight: 500;
  font-family: var(--fonu-font-mono, monospace);
}

.days {
  font-weight: 600;
}

.days--warning {
  color: var(--fonu-warning);
}

.days--error {
  color: var(--fonu-error);
}

.muted {
  color: var(--fonu-text-muted);
}

.switch-hint {
  margin-left: 10px;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.gift-tip {
  margin: 0 0 var(--fonu-space-3);
  font-size: 13px;
  color: var(--fonu-text-muted);
}

.gift-code-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-3);
  border-radius: var(--fonu-radius, 8px);
  background: var(--fonu-surface-muted, rgba(0, 0, 0, 0.03));
}

.gift-code {
  font-family: var(--fonu-font-mono, monospace);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 1px;
  color: var(--fonu-primary);
}

.gift-meta {
  margin: var(--fonu-space-3) 0 0;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

.gift-code-cell .domain-name:hover {
  color: var(--fonu-primary);
  cursor: pointer;
}

.accept-input {
  margin-top: var(--fonu-space-2);
}

:deep(.row-actions) {
  display: flex;
  align-items: center;
  gap: 12px;
}

:deep(.gift-trigger) {
  display: inline-block;
}

:deep(.domain-grip) {
  color: var(--fonu-text-muted);
  cursor: grab;
  vertical-align: middle;
}

:deep(.domain-row) {
  cursor: grab;
}

:deep(.domain-row--dragging) {
  opacity: 0.4;
}

:deep(.domain-row--over td) {
  box-shadow: inset 0 2px 0 var(--fonu-primary);
}

.register-preview {
  margin-bottom: var(--fonu-space-2);
  font-size: 14px;
}

.register-note {
  margin: 0;
  font-size: 12px;
  color: var(--fonu-text-muted);
}

@media (max-width: 900px) {
  .dnshe-layout {
    grid-template-columns: 1fr;
  }

  .dnshe-sidebar {
    position: static;
  }
}
</style>
