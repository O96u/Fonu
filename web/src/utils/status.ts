export type StatusKind = 'success' | 'warning' | 'error' | 'disabled' | 'unknown'

const LABELS: Record<string, string> = {
  ok: '正常',
  running: '运行中',
  warning: '部分异常',
  error: '异常',
  disabled: '停用',
  none: '未配置',
  stopped: '已停止',
  unknown: '未知',
}

export function statusLabel(value?: string): string {
  if (!value) return '未知'
  return LABELS[value] ?? value
}

export function statusKind(value?: string): StatusKind {
  switch (value) {
    case 'ok':
    case 'running':
      return 'success'
    case 'warning':
      return 'warning'
    case 'error':
      return 'error'
    case 'disabled':
    case 'stopped':
    case 'none':
      return 'disabled'
    default:
      return 'unknown'
  }
}

export function httpStatusKind(code: number): StatusKind {
  if (code >= 500) return 'error'
  if (code >= 400) return 'warning'
  if (code >= 300) return 'unknown'
  return 'success'
}
