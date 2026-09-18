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
  nginx_mode?: string
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
  request_yesterday?: number
  request_trend?: number | null
  requests_hourly: number[]
  requests_hourly_labels?: string[]
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
  server_port?: number
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

export type NotifyType = '' | 'email' | 'webhook' | 'telegram'
export type WebhookProvider = 'bark' | 'ntfy' | 'gotify' | 'custom'

export interface NotifyEmailConfig {
  host: string
  port: number
  username: string
  from: string
  to: string[]
  tls: boolean
  has_password?: boolean
}

export interface NotifyWebhookConfig {
  provider: WebhookProvider
  server: string
  key: string
  topic: string
  url: string
  has_secret?: boolean
}

export interface NotifyTelegramConfig {
  chat_id: string
  proxy_url: string
  has_bot_token?: boolean
}

export interface NotifyTestPayload {
  type: NotifyType
  email: NotifyEmailConfig
  webhook: NotifyWebhookConfig
  telegram: NotifyTelegramConfig
  on_ddns_ip_change: boolean
  on_ddns_failure: boolean
  on_cert_expiry: boolean
  on_cert_renew_success: boolean
  on_cert_renew_failure: boolean
  on_ip_frequent_access: boolean
  on_login_failure: boolean
  on_nginx_reload_failure: boolean
  ip_frequent_threshold: number
  ip_frequent_window_sec: number
  login_failure_threshold: number
  login_failure_window_sec: number
  smtp_password?: string
  webhook_secret?: string
  telegram_token?: string
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

export interface FRPStatus {
  enabled: boolean
  connected: boolean
  message: string
  last_error?: string
  server_endpoint?: string
  connected_at?: string
  uptime_seconds?: number
  client_version?: string
  http_gateway_enabled?: boolean
  https_gateway_enabled?: boolean
  synced_domain_count?: number
}

export interface FRPConfig {
  enabled: boolean
  server_addr: string
  server_port: number
  auth_token: string
  has_auth_token: boolean
  tls_enabled: boolean
  custom_domains: string[]
}

export interface FRPResponse {
  enabled: boolean
  server_addr: string
  server_port: number
  auth_token: string
  has_auth_token: boolean
  tls_enabled: boolean
  custom_domains: string[]
  status: FRPStatus
  frps_config: string
  nginx_http_port: number
  nginx_https_port: number
}

export interface FRPSaveResponse {
  message: string
  config: FRPConfig
  status: FRPStatus
  frps_config: string
  nginx_http_port: number
  nginx_https_port: number
}

export interface FRPSavePayload {
  enabled: boolean
  server_addr: string
  server_port: number
  auth_token?: string
  tls_enabled: boolean
  custom_domains: string[]
}

export interface NginxBackupEntry {
  name: string
  created_at: string
}

export interface RuleNginxView {
  mode: 'auto' | 'custom'
  enabled: boolean
  active: boolean
  generated: string
  content: string
  backups: NginxBackupEntry[]
}

export interface GlobalNginxView {
  mode: 'auto' | 'custom'
  generated_framework: string
  generated_snippet: string
  content: string
  backups: NginxBackupEntry[]
}
