export interface NotifyEventOption {
  key: string
  label: string
  hint?: string
  hasThreshold?: boolean
  thresholdKey?: 'ip_frequent' | 'login_failure'
}

export interface NotifyEventGroup {
  title: string
  events: NotifyEventOption[]
}

export const NOTIFY_EVENT_GROUPS: NotifyEventGroup[] = [
  {
    title: 'DDNS',
    events: [
      {
        key: 'on_ddns_ip_change',
        label: 'IP 变更',
        hint: '公网 IP 变更并成功同步到 DNS',
      },
      {
        key: 'on_ddns_failure',
        label: '更新失败',
      },
    ],
  },
  {
    title: '证书',
    events: [
      {
        key: 'on_cert_expiry',
        label: '即将到期',
        hint: '进入续签阈值后每天提醒一次',
      },
      {
        key: 'on_cert_renew_success',
        label: '续签成功',
      },
      {
        key: 'on_cert_renew_failure',
        label: '续签失败',
      },
    ],
  },
  {
    title: '安全',
    events: [
      {
        key: 'on_ip_frequent_access',
        label: 'IP 频繁访问',
        hint: '同一 IP 在统计窗口内超过阈值',
        hasThreshold: true,
        thresholdKey: 'ip_frequent',
      },
      {
        key: 'on_login_failure',
        label: '登录异常',
        hint: '同一 IP 在窗口内多次登录失败',
        hasThreshold: true,
        thresholdKey: 'login_failure',
      },
    ],
  },
  {
    title: '系统',
    events: [
      {
        key: 'on_nginx_reload_failure',
        label: 'Nginx 重载失败',
        hint: '保存反代或证书后 Nginx 未能成功重载',
      },
    ],
  },
]

export const DEFAULT_IP_FREQUENT_THRESHOLD = 100
export const DEFAULT_IP_FREQUENT_WINDOW_SEC = 60
export const DEFAULT_LOGIN_FAILURE_THRESHOLD = 5
export const DEFAULT_LOGIN_FAILURE_WINDOW_SEC = 300
