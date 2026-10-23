import type { CertExportSettings } from '../api/types'

export type CertSSHExportFormModel = {
  enabled: boolean
  host: string
  port: number
  username: string
  auth_type: 'password' | 'private_key'
  password: string
  has_password?: boolean
  private_key: string
  has_private_key?: boolean
  private_key_passphrase: string
  has_passphrase?: boolean
  remote_dir: string
  on_upload_shell: string
  proxy_enabled: boolean
  proxy_type: 'socks5' | 'http'
  proxy_host: string
  proxy_port: number
  proxy_username: string
  proxy_password: string
  has_proxy_password?: boolean
  last_sync_at?: string
  last_error?: string
}

export type CertExportFormModel = {
  enabled: boolean
  targets: string[]
  on_change_shell: string
  last_sync_at?: string
  last_error?: string
  ssh: CertSSHExportFormModel
}

export function emptySSHExportForm(): CertSSHExportFormModel {
  return {
    enabled: false,
    host: '',
    port: 22,
    username: '',
    auth_type: 'password',
    password: '',
    private_key: '',
    private_key_passphrase: '',
    remote_dir: '',
    on_upload_shell: '',
    proxy_enabled: false,
    proxy_type: 'socks5',
    proxy_host: '',
    proxy_port: 1080,
    proxy_username: '',
    proxy_password: '',
  }
}

export function emptyExportForm(): CertExportFormModel {
  return { enabled: false, targets: [''], on_change_shell: '', ssh: emptySSHExportForm() }
}

export function loadExportFormFromSettings(settings?: CertExportSettings): CertExportFormModel {
  const ssh = settings?.ssh
  return {
    enabled: settings?.enabled ?? false,
    targets: settings?.targets?.length ? [...settings.targets] : [''],
    on_change_shell: settings?.on_change_shell ?? '',
    last_sync_at: settings?.last_sync_at,
    last_error: settings?.last_error,
    ssh: {
      enabled: ssh?.enabled ?? false,
      host: ssh?.host ?? '',
      port: ssh?.port && ssh.port > 0 ? ssh.port : 22,
      username: ssh?.username ?? '',
      auth_type: ssh?.auth_type === 'private_key' ? 'private_key' : 'password',
      password: '',
      has_password: ssh?.has_password,
      private_key: '',
      has_private_key: ssh?.has_private_key,
      private_key_passphrase: '',
      has_passphrase: ssh?.has_passphrase,
      remote_dir: ssh?.remote_dir ?? '',
      on_upload_shell: ssh?.on_upload_shell ?? '',
      proxy_enabled: ssh?.proxy_enabled ?? false,
      proxy_type: ssh?.proxy_type === 'http' ? 'http' : 'socks5',
      proxy_host: ssh?.proxy_host ?? '',
      proxy_port: ssh?.proxy_port && ssh.proxy_port > 0 ? ssh.proxy_port : 1080,
      proxy_username: ssh?.proxy_username ?? '',
      proxy_password: '',
      has_proxy_password: ssh?.has_proxy_password,
      last_sync_at: ssh?.last_sync_at,
      last_error: ssh?.last_error,
    },
  }
}

export function exportPayload(form: CertExportFormModel): CertExportSettings {
  return {
    enabled: form.enabled,
    targets: form.targets.map((t) => t.trim()).filter(Boolean),
    on_change_shell: form.on_change_shell.trim(),
    ssh: {
      enabled: form.ssh.enabled,
      host: form.ssh.host.trim(),
      port: form.ssh.port > 0 ? form.ssh.port : 22,
      username: form.ssh.username.trim(),
      auth_type: form.ssh.auth_type,
      password: form.ssh.password,
      private_key: form.ssh.private_key.trim(),
      private_key_passphrase: form.ssh.private_key_passphrase,
      remote_dir: form.ssh.remote_dir.trim(),
      on_upload_shell: form.ssh.on_upload_shell.trim(),
      proxy_enabled: form.ssh.proxy_enabled,
      proxy_type: form.ssh.proxy_type,
      proxy_host: form.ssh.proxy_host.trim(),
      proxy_port: form.ssh.proxy_port > 0 ? form.ssh.proxy_port : form.ssh.proxy_type === 'http' ? 8080 : 1080,
      proxy_username: form.ssh.proxy_username.trim(),
      proxy_password: form.ssh.proxy_password,
    },
  }
}
