export interface ProxyHost {
  id: number
  hostname: string
  listen_port?: number | null
}

export interface ProxyRule {
  id: number
  domain: string
  upstream: string
  listen_port: number
  listen_ipv4: boolean
  listen_ipv6: boolean
  hosts: ProxyHost[]
  https_enabled: boolean
  http_redirect: boolean
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface ProxySavePayload {
  upstream: string
  listen_port?: number
  listen_ipv4?: boolean
  listen_ipv6?: boolean
  hosts: string[]
  https_enabled?: boolean
  http_redirect?: boolean
  enabled?: boolean
}

export interface AuthStatus {
  initialized: boolean
  authenticated: boolean
}

export interface ApiError {
  error: string
}

export interface DashboardStatus {
  public_ipv4: string
  public_ipv6: string
  ddns_status: string
  ddns_count: number
  ddns_last_updated?: string
  certificate_status: string
  certificate_days: number
  certificate_count: number
  certificate_summary?: string
  proxy_count: number
  nginx_status: string
  request_today: number
  error_today: number
  avg_response_ms: number
  started_at: string
  uptime_seconds: number
}

export interface DDNSConfig {
  id: number
  provider: string
  root_domain: string
  record_name: string
  ipv4_enabled: boolean
  ipv6_enabled: boolean
  enabled: boolean
  has_token: boolean
  last_ipv4?: string
  last_ipv6?: string
  last_status?: string
  last_error?: string
  last_updated_at?: string
}

export interface CertificateCAOption {
  value: string
  label: string
}

export interface CertificateRecord {
  id: number
  domain: string
  domains?: string[]
  wildcard: boolean
  acme_ca?: string
  status: string
  days_left: number
  last_error?: string
  expires_at?: string
  last_renew_at?: string
}

export interface AccessLogEntry {
  time: string
  domain: string
  method: string
  path: string
  status: number
  response_time: number
  client_ip: string
  upstream: string
}

export interface SystemLogEntry {
  time: string
  level: string
  module: string
  message: string
}

export type SettingsMap = Record<string, string>

export interface DiscoveredService {
  name: string
  port: number
  host: string
  upstream: string
  detected: boolean
  platform: string
  title?: string
  suggestion?: string
}

export interface DDNSSavePayload {
  provider?: string
  root_domain?: string
  record_name?: string
  ipv4_enabled?: boolean
  ipv6_enabled?: boolean
  enabled?: boolean
  api_token?: string
  api_token_id?: string
  api_secret?: string
}

export interface DDNSTestPayload {
  config_id?: number
  provider?: string
  api_token?: string
  api_token_id?: string
  api_secret?: string
}
