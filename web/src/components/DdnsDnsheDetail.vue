<template>
  <div class="dnshe-detail">
    <FonuCard flush class="detail-section">
      <div class="records-head">
        <h4 class="detail-section__title">免费域名（{{ domains.length }}）</h4>
        <n-space :size="8" class="records-head__actions">
          <n-button size="small" quaternary :loading="loadingDetail" @click="loadDetail">
            <template #icon><n-icon :component="RefreshOutline" /></template>
            刷新
          </n-button>
          <n-button size="small" @click="openAccept">
            <template #icon><n-icon :component="DownloadOutline" /></template>
            接收域名
          </n-button>
          <n-button size="small" type="primary" @click="openRegister">
            <template #icon><n-icon :component="AddOutline" /></template>
            注册域名
          </n-button>
        </n-space>
      </div>

      <div v-if="loadingDetail" class="loading-wrap"><n-spin size="medium" /></div>
      <div v-else-if="domains.length === 0" class="detail-empty-wrap">
        <EmptyState title="暂无域名" description="可注册新域名，或通过转赠码接收。" />
      </div>
      <div v-else class="record-table-wrap">
        <table class="record-table">
          <thead>
            <tr>
              <th class="record-table__col-grip" aria-label="排序" />
              <th>域名</th>
              <th>状态</th>
              <th>注册时间</th>
              <th>到期时间</th>
              <th>剩余天数</th>
              <th>自动续期</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in domains"
              :key="row.id"
              :class="{
                'domain-row--dragging': row.id === dragDomainId,
                'domain-row--over': row.id === overDomainId && row.id !== dragDomainId,
              }"
              draggable="true"
              @dragstart="onDomainDragStart($event, row)"
              @dragover="onDomainDragOver($event, row)"
              @dragleave="onDomainDragLeave(row)"
              @drop="onDomainDrop($event, row.id)"
              @dragend="onDomainDragEnd"
            >
              <td class="record-table__col-grip">
                <n-icon :component="ReorderThreeOutline" class="domain-grip" :size="15" />
              </td>
              <td class="record-table__mono">{{ row.full_domain }}</td>
              <td>
                <StatusBadge :kind="domainStatusKind(row.status)" :text="domainStatusLabel(row.status)" />
              </td>
              <td>{{ row.created_at || '-' }}</td>
              <td>{{ row.never_expires ? '永不过期' : row.expires_at || '-' }}</td>
              <td>
                <span v-if="row.never_expires" class="record-table__muted">永不过期</span>
                <span v-else-if="row.days_left === null || row.days_left === undefined" class="record-table__muted">-</span>
                <span v-else :class="daysLeftClass(row)">{{ row.days_left }} 天</span>
              </td>
              <td>
                <n-switch
                  size="small"
                  :value="row.auto_renew"
                  :disabled="!row.renewable"
                  @update:value="(v: boolean) => toggleDomainAutoRenew(row, v)"
                />
              </td>
              <td>
                <div class="row-actions">
                  <n-button
                    size="tiny"
                    quaternary
                    type="primary"
                    :loading="pushingDomain === row.full_domain"
                    @click="emit('push-domain', row.full_domain)"
                  >
                    推送
                  </n-button>
                  <n-button
                    size="tiny"
                    quaternary
                    :disabled="!row.renewable"
                    :loading="renewingId === row.id"
                    @click="renewDomain(row)"
                  >
                    续期
                  </n-button>
                  <n-tooltip v-if="!canGift(row)" placement="top">
                    <template #trigger>
                      <span class="gift-trigger">
                        <n-button size="tiny" quaternary disabled>转赠</n-button>
                      </span>
                    </template>
                    域名注册满 30 天后才可转赠（当前 {{ domainAgeDays(row) ?? '?' }} 天）
                  </n-tooltip>
                  <n-button
                    v-else
                    size="tiny"
                    quaternary
                    :loading="giftingId === row.id"
                    @click="giftDomain(row)"
                  >
                    转赠
                  </n-button>
                  <n-button
                    size="tiny"
                    quaternary
                    type="error"
                    :loading="deletingId === row.id"
                    @click="confirmDeleteDomain(row)"
                  >
                    删除
                  </n-button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </FonuCard>

    <!-- 账户编辑 -->
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
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import {
  NButton,
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
} from 'naive-ui'
import {
  AddOutline,
  CopyOutline,
  DownloadOutline,
  ReorderThreeOutline,
  RefreshOutline,
} from '@vicons/ionicons5'
import { api } from '../api/client'
import type { DNSHEAccount, DNSHEDomain } from '../api/types'
import FonuCard from './FonuCard.vue'
import EmptyState from './EmptyState.vue'
import StatusBadge from '../components/StatusBadge.vue'
import type { StatusKind } from '../utils/status'

const props = defineProps<{
  account: DNSHEAccount
  pushingDomain?: string
}>()

const emit = defineEmits<{
  'refresh-accounts': []
  'domains-changed': []
  'gifts-changed': []
  'push-domain': [fullDomain: string]
  deleted: []
}>()

const message = useMessage()
const dialog = useDialog()

const selectedId = computed(() => props.account.id)
const domains = ref<DNSHEDomain[]>([])

const loadingDetail = ref(false)

function notifyDomainsChanged() {
  emit('domains-changed')
}

const renewingId = ref<number | null>(null)
const deletingId = ref<number | null>(null)
const giftingId = ref<number | null>(null)

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

function openEditAccount(account: DNSHEAccount = props.account) {
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
    }
    accountModal.show = false
    emit('refresh-accounts')
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
        emit('deleted')
        emit('refresh-accounts')
      } catch (e) {
        message.error(e instanceof Error ? e.message : '删除失败')
      }
    },
  })
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

function daysLeftClass(row: DNSHEDomain) {
  if (row.days_left === null || row.days_left === undefined) return 'days'
  if (row.days_left <= 7) return 'days days--error'
  if (row.days_left <= 30) return 'days days--warning'
  return 'days'
}

// ---- 域名行拖拽 ----

const dragDomainId = ref<number | null>(null)
const overDomainId = ref<number | null>(null)

function onDomainDragStart(e: DragEvent, row: DNSHEDomain) {
  dragDomainId.value = row.id
  e.dataTransfer?.setData('text/plain', String(row.id))
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
}

function onDomainDragOver(e: DragEvent, row: DNSHEDomain) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  if (overDomainId.value !== row.id) overDomainId.value = row.id
}

function onDomainDragLeave(row: DNSHEDomain) {
  if (overDomainId.value === row.id) overDomainId.value = null
}

function onDomainDragEnd() {
  dragDomainId.value = null
  overDomainId.value = null
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

async function loadDetail() {
  const id = selectedId.value
  if (!id) return
  loadingDetail.value = true
  try {
    domains.value = await api.listDNSHEDomains(id)
  } catch (e) {
    message.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loadingDetail.value = false
  }
}

watch(
  () => props.account.id,
  () => {
    void loadDetail()
  },
  { immediate: true },
)

// ---- 域名操作 ----

async function renewDomain(row: DNSHEDomain) {
  renewingId.value = row.id
  try {
    const result = await api.renewDNSHEDomain(selectedId.value, row.id)
    message.success(result.new_expires_at ? `续期成功，新到期时间：${result.new_expires_at}` : '续期成功')
    await loadDetail()
    notifyDomainsChanged()
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
    notifyDomainsChanged()
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
    emit('gifts-changed')
  } catch (e) {
    message.error(e instanceof Error ? e.message : '生成转赠码失败')
  } finally {
    giftingId.value = null
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
    notifyDomainsChanged()
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
    notifyDomainsChanged()
    emit('gifts-changed')
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

defineExpose({ openEditAccount, openCredentials, confirmDeleteAccount })
</script>

<style scoped>
.dnshe-detail {
  display: flex;
  flex-direction: column;
  gap: var(--fonu-space-4);
  min-width: 0;
}

.detail-section {
  min-width: 0;
}

.detail-section__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--fonu-text);
}

.records-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--fonu-space-3);
  padding: var(--fonu-space-4) var(--fonu-space-5) 0;
  flex-wrap: wrap;
}

.records-head__actions {
  flex-shrink: 0;
  margin-left: auto;
}

.record-table-wrap {
  overflow-x: auto;
  padding: var(--fonu-space-3) var(--fonu-space-5) var(--fonu-space-4);
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

.record-table__col-grip {
  width: 28px;
  padding-left: 8px;
  padding-right: 0;
}

.record-table__mono {
  font-family: var(--fonu-mono);
  font-size: 12px;
}

.record-table__muted {
  color: var(--fonu-text-muted);
}

.record-table__link {
  font-family: var(--fonu-mono);
  font-size: 12px;
  font-weight: 500;
  color: var(--fonu-text);
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
  vertical-align: middle;
}

.record-table__link:hover {
  color: var(--fonu-primary);
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: nowrap;
}

.gift-trigger {
  display: inline-block;
}

.domain-grip {
  color: var(--fonu-text-muted);
  cursor: grab;
  vertical-align: middle;
}

.domain-row--dragging {
  opacity: 0.4;
}

.domain-row--over td {
  box-shadow: inset 0 2px 0 var(--fonu-primary);
}

.detail-empty-wrap {
  padding: 0 var(--fonu-space-5) var(--fonu-space-4);
}

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

.accept-input {
  margin-top: var(--fonu-space-2);
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
