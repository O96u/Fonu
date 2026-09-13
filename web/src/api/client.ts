import type {
  AccessLogEntry,
  ApiError,
  AppVersion,
  AuthStatus,
  CertificateCAOption,
  CertificateRecord,
  DashboardStatus,
  DDNSConfig,
  DDNSSavePayload,
  DDNSTestPayload,
  DiscoveredService,
  ProxyClientConn,
  ProxyRule,
  ProxySavePayload,
  ProxyTraffic,
  SettingsMap,
  SystemLogEntry,
} from './types'

export function asList<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

const REQUEST_TIMEOUT_MS = 30_000
const CERT_REQUEST_TIMEOUT_MS = 10 * 60_000

async function request<T>(path: string, init?: RequestInit, timeoutMs = REQUEST_TIMEOUT_MS): Promise<T> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), timeoutMs)
  try {
    return await requestWithSignal<T>(path, init, controller.signal)
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new Error(
        timeoutMs > REQUEST_TIMEOUT_MS
          ? '证书操作超时，请稍后在证书列表查看状态，或在「日志 → 实时日志」查看 Nginx 输出'
          : '请求超时，请检查 Fonu 服务或 Nginx 状态',
      )
    }
    throw error
  } finally {
    clearTimeout(timeout)
  }
}

async function requestWithSignal<T>(path: string, init?: RequestInit, signal?: AbortSignal): Promise<T> {
  const response = await fetch(path, {
    credentials: 'include',
    signal,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  if (!response.ok) {
    let message = '请求失败'
    try {
      const body = (await response.json()) as ApiError
      if (body.error) {
        message = body.error
      }
    } catch {
      // ignore
    }
    throw new Error(message)
  }

  if (response.status === 204) {
    return undefined as T
  }
  return (await response.json()) as T
}

export const api = {
  getVersion: () => request<AppVersion>('/api/version'),
  authStatus: () => request<AuthStatus>('/api/auth/status'),
  login: (username: string, password: string) =>
    request<{ message: string }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  logout: () => request<{ message: string }>('/api/auth/logout', { method: 'POST' }),
  changePassword: (old_password: string, new_password: string) =>
    request<{ message: string }>('/api/auth/password', {
      method: 'POST',
      body: JSON.stringify({ old_password, new_password }),
    }),

  getStatus: () => request<DashboardStatus>('/api/status'),
  listProxies: () => request<ProxyRule[]>('/api/proxies'),
  createProxy: (payload: ProxySavePayload) =>
    request<ProxyRule>('/api/proxies', { method: 'POST', body: JSON.stringify(payload) }),
  updateProxy: (id: number, payload: Partial<ProxySavePayload>) =>
    request<ProxyRule>(`/api/proxies/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteProxy: (id: number) => request<void>(`/api/proxies/${id}`, { method: 'DELETE' }),
  getProxyTraffic: () => request<ProxyTraffic[]>('/api/proxies/traffic'),
  getProxyClients: (id: number) => request<ProxyClientConn[]>(`/api/proxies/${id}/clients`),

  listDDNS: () => request<DDNSConfig[]>('/api/ddns'),
  listDDNSLite: () => request<DDNSConfig[]>('/api/ddns?lite=1'),
  createDDNS: (payload: DDNSSavePayload) =>
    request<DDNSConfig>('/api/ddns', { method: 'POST', body: JSON.stringify(payload) }),
  updateDDNS: (id: number, payload: DDNSSavePayload) =>
    request<DDNSConfig>(`/api/ddns/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteDDNS: (id: number) => request<void>(`/api/ddns/${id}`, { method: 'DELETE' }),
  testDDNS: (payload: DDNSTestPayload) =>
    request<{ message: string }>('/api/ddns/test', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  updateAllDDNS: () => request<DDNSConfig[]>('/api/ddns/update', { method: 'POST' }),
  updateDDNSOne: (id: number) => request<DDNSConfig>(`/api/ddns/${id}/update`, { method: 'POST' }),

  listCertificates: () => request<CertificateRecord[]>('/api/certificates'),
  listCertificateCAOptions: () => request<CertificateCAOption[]>('/api/certificates/ca-options'),
  applyCertificate: (payload: {
    dns_zone?: string
    ddns_config_id?: number
    domains: string[]
    ca?: string
    email?: string
  }) =>
    request<{ job_id: string }>('/api/certificates/apply', {
      method: 'POST',
      body: JSON.stringify({
        dns_zone: payload.dns_zone ?? '',
        ddns_config_id: payload.ddns_config_id ?? 0,
        domains: payload.domains,
        ca: payload.ca ?? '',
        email: payload.email ?? '',
      }),
    }),
  certificateApplyStreamURL: (jobId: string) => `/api/certificates/jobs/${encodeURIComponent(jobId)}/stream`,
  importCertificate: (payload: {
    certificate?: string
    private_key?: string
    cert_path?: string
    key_path?: string
  }) =>
    request<CertificateRecord>('/api/certificates/import', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  async downloadCertificate(domain: string, part: 'zip' | 'cert' | 'key' = 'zip') {
    const response = await fetch(
      `/api/certificates/${encodeURIComponent(domain)}/download?part=${part}`,
      { credentials: 'include' },
    )
    if (!response.ok) {
      let message = '下载证书失败'
      try {
        const body = (await response.json()) as ApiError
        if (body.error) message = body.error
      } catch {
        // ignore
      }
      throw new Error(message)
    }
    const blob = await response.blob()
    const disposition = response.headers.get('Content-Disposition') ?? ''
    const match = disposition.match(/filename="?([^";\n]+)"?/)
    const filename = match?.[1] ?? `${domain}.zip`
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    link.click()
    URL.revokeObjectURL(url)
  },
  deleteCertificate: (domain: string) =>
    request<{ message: string }>(`/api/certificates/${encodeURIComponent(domain)}`, { method: 'DELETE' }),
  renewCertificate: (domain?: string, ca?: string) =>
    request<CertificateRecord>(
      '/api/certificates/renew',
      {
        method: 'POST',
        body: JSON.stringify({ domain: domain ?? '', ca: ca ?? '' }),
      },
      CERT_REQUEST_TIMEOUT_MS,
    ),

  getSettings: () => request<SettingsMap>('/api/settings'),
  saveSettings: (payload: SettingsMap) =>
    request<SettingsMap>('/api/settings', { method: 'PUT', body: JSON.stringify(payload) }),

  getAccessLogs: (params?: { limit?: number; keyword?: string; status?: number }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.keyword) q.set('keyword', params.keyword)
    if (params?.status) q.set('status', String(params.status))
    return request<AccessLogEntry[]>(`/api/logs/access?${q}`)
  },
  getErrorLogs: (params?: { limit?: number; keyword?: string }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.keyword) q.set('keyword', params.keyword)
    return request<string[]>(`/api/logs/error?${q}`)
  },
  getSystemLogs: (params?: { limit?: number; level?: string; keyword?: string }) => {
    const q = new URLSearchParams()
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.level) q.set('level', params.level)
    if (params?.keyword) q.set('keyword', params.keyword)
    return request<SystemLogEntry[]>(`/api/logs/system?${q}`)
  },

  scanDiscovery: (host?: string) => {
    const q = new URLSearchParams()
    if (host) q.set('host', host)
    return request<DiscoveredService[]>(`/api/discovery/scan?${q}`)
  },

  async exportBackup() {
    const response = await fetch('/api/backup/export', { credentials: 'include' })
    if (!response.ok) {
      throw new Error('导出备份失败')
    }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `fonu-backup-${new Date().toISOString().slice(0, 10)}.tar.gz`
    link.click()
    URL.revokeObjectURL(url)
  },

  async restoreBackup(file: File) {
    const form = new FormData()
    form.append('file', file)
    const response = await fetch('/api/backup/restore', {
      method: 'POST',
      credentials: 'include',
      body: form,
    })
    if (!response.ok) {
      let message = '恢复备份失败'
      try {
        const body = (await response.json()) as ApiError
        if (body.error) message = body.error
      } catch {
        // ignore
      }
      throw new Error(message)
    }
    return (await response.json()) as { message: string }
  },
}
