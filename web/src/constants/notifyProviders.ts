import type { NotifyType, WebhookProvider } from '../api/types'

export const NOTIFY_TYPE_OPTIONS: { label: string; value: NotifyType }[] = [
  { label: '关闭', value: '' },
  { label: '邮件（SMTP）', value: 'email' },
  { label: 'Webhook', value: 'webhook' },
  { label: 'Telegram 机器人', value: 'telegram' },
]

export const WEBHOOK_PROVIDER_OPTIONS: { label: string; value: WebhookProvider }[] = [
  { label: 'Bark', value: 'bark' },
  { label: 'ntfy', value: 'ntfy' },
  { label: 'Gotify', value: 'gotify' },
  { label: '自定义 Webhook', value: 'custom' },
]

export const DEFAULT_BARK_SERVER = 'https://api.day.app'
export const DEFAULT_NTFY_SERVER = 'https://ntfy.sh'
export const MASKED_SECRET = '********'
