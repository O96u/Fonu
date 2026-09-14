export interface ProxyHost {
  id: number
  hostname: string
  listen_port?: number | null
}

export interface BasicAuthConfig {
  enabled?: boolean
  username?: string
  has_password?: boolean
}

export interface RateLimitConfig {
  enabled?: boolean
  rate?: number
  burst?: number
}

export interface ConnLimitConfig {
  enabled?: boolean
  max?: number
}

export interface ProxySecurityConfig {
  ip_blacklist?: string[]
  ip_whitelist?: string[]
  ip_whitelist_mode?: boolean
  china_only?: boolean
  basic_auth?: BasicAuthConfig
  rate_limit?: RateLimitConfig
  conn_limit?: ConnLimitConfig
  proxy_ssl_verify_off?: boolean
  proxy_host_upstream?: boolean
  tls_min_13_only?: boolean
  security_headers?: boolean
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
  name: string
  sort_order: number
  security: ProxySecurityConfig
  created_at: string
  updated_at: string
}

export interface ChinaCIDRStatus {
  updated_at?: string
  entry_count_v4: number
  entry_count_v6: number
  last_error?: string
  updating?: boolean
  ready?: boolean
}

export interface ProxyTraffic {
  rule_id: number
  upload_total: number
  download_total: number
  upload_rate: number
  download_rate: number
  connections: number
}

export interface ProxyClientConn {
  ip: string
  last_seen: string
}

export interface ProxySecurityPayload {
  ip_blacklist?: string[]
  ip_whitelist?: string[]
  ip_whitelist_mode?: boolean
  ip_blacklist_text?: string
  ip_whitelist_text?: string
  china_only?: boolean
  basic_auth?: {
    enabled?: boolean
    username?: string
    password?: string
  }
  rate_limit?: RateLimitConfig
  conn_limit?: ConnLimitConfig
  proxy_ssl_verify_off?: boolean
  proxy_host_upstream?: boolean
  tls_min_13_only?: boolean
  security_headers?: boolean
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
  name?: string
  security?: ProxySecurityPayload
}

export interface AuthStatus {
  initialized: boolean
  authenticated: boolean
}

export interface AppVersion {
  version: string
  nginx_http_port: number
  nginx_https_port: number
}

export interface ApiError {
  error: string
}

export interface DashboardStatus {
  public_ipv4: string
  public_ipv6: string
  public_ip_source?: 'ddns' | 'detect' | 'none'
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
  requests_hourly: number[]
  error_today: number
  avg_response_ms: number
  started_at: string
  uptime_seconds: number
}

export interface DDNSDomainRecord {
  domain: string
  ipv4?: string
  ipv6?: string
  status: string
  message?: string
}

export interface DDNSConfig {
  id: number
  provider: string
  root_domain: string
  record_name: string
  record_names?: string[]
  ipv4_enabled: boolean
  ipv6_enabled: boolean
  enabled: boolean
  has_token: boolean
  last_ipv4?: string
  last_ipv6?: string
  last_status?: string
  last_error?: string
  domain_records?: DDNSDomainRecord[]
  last_updated_at?: string
}

export interface CertificateCAOption {
  value: string
  label: string
}

export interface CertificateJobDone {
  ok: boolean
  error?: string
  domain?: string
  cert_path?: string
  key_path?: string
  cert_dir?: string
  expires_at?: string
}

export interface CertificateJobEvent {
  type: 'log' | 'done'
  level?: string
  message?: string
  result?: CertificateJobDone
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
  record_names?: string[]
  domains?: string[]
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
