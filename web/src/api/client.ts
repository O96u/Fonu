import type {
  AccessLogEntry,
  ApiError,
  AuthStatus,
  CertificateCAOption,
  CertificateRecord,
  DashboardStatus,
  DDNSConfig,
  DDNSSavePayload,
  DDNSTestPayload,
  DiscoveredService,
  ProxyRule,
  SettingsMap,
  SystemLogEntry,
} from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    credentials: 'include',
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
  authStatus: () => request<AuthStatus>('/api/auth/status'),
  setup: (username: string, password: string) =>
    request<{ message: string }>('/api/auth/setup', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
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
  createProxy: (payload: Partial<ProxyRule> & { domain: string; upstream: string }) =>
    request<ProxyRule>('/api/proxies', { method: 'POST', body: JSON.stringify(payload) }),
  updateProxy: (id: number, payload: Partial<ProxyRule>) =>
    request<ProxyRule>(`/api/proxies/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteProxy: (id: number) => request<void>(`/api/proxies/${id}`, { method: 'DELETE' }),

  listDDNS: () => request<DDNSConfig[]>('/api/ddns'),
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
  applyCertificate: (payload: { dns_zone: string; domains: string[]; ca?: string; email?: string }) =>
    request<CertificateRecord[]>('/api/certificates/apply', {
      method: 'POST',
      body: JSON.stringify({
        dns_zone: payload.dns_zone,
        domains: payload.domains,
        ca: payload.ca ?? '',
        email: payload.email ?? '',
      }),
    }),
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
  deleteCertificate: (domain: string) =>
    request<{ message: string }>(`/api/certificates/${encodeURIComponent(domain)}`, { method: 'DELETE' }),
  renewCertificate: (domain?: string, ca?: string) =>
    request<CertificateRecord>('/api/certificates/renew', {
      method: 'POST',
      body: JSON.stringify({ domain: domain ?? '', ca: ca ?? '' }),
    }),

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
