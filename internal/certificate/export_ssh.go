package certificate

import (
	"fmt"
	"strings"
)

const (
	SSHAuthPassword   = "password"
	SSHAuthPrivateKey = "private_key"
	SSHProxySOCKS5    = "socks5"
	SSHProxyHTTP      = "http"
)

type SSHExportConfig struct {
	Enabled              bool   `json:"enabled"`
	Host                 string `json:"host"`
	Port                 int    `json:"port"`
	Username             string `json:"username"`
	AuthType             string `json:"auth_type"`
	Password             string `json:"password,omitempty"`
	HasPassword          bool   `json:"has_password,omitempty"`
	PrivateKey           string `json:"private_key,omitempty"`
	HasPrivateKey        bool   `json:"has_private_key,omitempty"`
	PrivateKeyPath       string `json:"private_key_path,omitempty"`
	PrivateKeyPassphrase string `json:"private_key_passphrase,omitempty"`
	HasPassphrase        bool   `json:"has_passphrase,omitempty"`
	RemoteDir            string `json:"remote_dir"`
	OnUploadShell        string `json:"on_upload_shell"`
	ProxyEnabled         bool   `json:"proxy_enabled"`
	ProxyType            string `json:"proxy_type"`
	ProxyHost            string `json:"proxy_host"`
	ProxyPort            int    `json:"proxy_port"`
	ProxyUsername        string `json:"proxy_username,omitempty"`
	ProxyPassword        string `json:"proxy_password,omitempty"`
	HasProxyPassword     bool   `json:"has_proxy_password,omitempty"`
	LastSyncAt           string `json:"last_sync_at,omitempty"`
	LastError            string `json:"last_error,omitempty"`
}

func (s SSHExportConfig) Normalize() SSHExportConfig {
	port := s.Port
	if port <= 0 {
		port = 22
	}
	auth := strings.TrimSpace(s.AuthType)
	if auth == "" {
		auth = SSHAuthPassword
	}
	proxyType := strings.TrimSpace(s.ProxyType)
	if proxyType == "" {
		proxyType = SSHProxySOCKS5
	}
	proxyPort := s.ProxyPort
	if s.ProxyEnabled && proxyPort <= 0 {
		if proxyType == SSHProxyHTTP {
			proxyPort = 8080
		} else {
			proxyPort = 1080
		}
	}
	return SSHExportConfig{
		Enabled:              s.Enabled,
		Host:                 strings.TrimSpace(s.Host),
		Port:                 port,
		Username:             strings.TrimSpace(s.Username),
		AuthType:             auth,
		Password:             s.Password,
		HasPassword:          s.HasPassword,
		PrivateKey:           strings.TrimSpace(s.PrivateKey),
		HasPrivateKey:        s.HasPrivateKey,
		PrivateKeyPath:       strings.TrimSpace(s.PrivateKeyPath),
		PrivateKeyPassphrase: s.PrivateKeyPassphrase,
		HasPassphrase:        s.HasPassphrase,
		RemoteDir:            cleanRemoteDir(s.RemoteDir),
		OnUploadShell:        strings.TrimSpace(s.OnUploadShell),
		ProxyEnabled:         s.ProxyEnabled,
		ProxyType:            proxyType,
		ProxyHost:            strings.TrimSpace(s.ProxyHost),
		ProxyPort:            proxyPort,
		ProxyUsername:        strings.TrimSpace(s.ProxyUsername),
		ProxyPassword:        s.ProxyPassword,
		HasProxyPassword:     s.HasProxyPassword,
		LastSyncAt:           strings.TrimSpace(s.LastSyncAt),
		LastError:            strings.TrimSpace(s.LastError),
	}
}

func (s SSHExportConfig) Validate() error {
	if !s.Enabled {
		return nil
	}
	if s.Host == "" {
		return fmt.Errorf("SSH 主机不能为空")
	}
	if s.Username == "" {
		return fmt.Errorf("SSH 用户名不能为空")
	}
	if s.RemoteDir == "" {
		return fmt.Errorf("SSH 远程目录不能为空")
	}
	if !strings.HasPrefix(s.RemoteDir, "/") {
		return fmt.Errorf("SSH 远程目录须为绝对路径（以 / 开头）")
	}
	switch s.AuthType {
	case SSHAuthPassword:
		if s.Password == "" && !s.HasPassword {
			return fmt.Errorf("请填写 SSH 密码")
		}
	case SSHAuthPrivateKey:
		if s.PrivateKey == "" && s.PrivateKeyPath == "" && !s.HasPrivateKey {
			return fmt.Errorf("请填写 SSH 私钥内容")
		}
	default:
		return fmt.Errorf("SSH 认证方式无效")
	}
	if s.ProxyEnabled {
		if s.ProxyHost == "" {
			return fmt.Errorf("启用代理时请填写代理主机")
		}
		switch s.ProxyType {
		case SSHProxySOCKS5, SSHProxyHTTP:
		default:
			return fmt.Errorf("代理类型须为 SOCKS5 或 HTTP")
		}
	}
	return nil
}

func cleanRemoteDir(dir string) string {
	dir = strings.TrimSpace(dir)
	dir = strings.ReplaceAll(dir, "\\", "/")
	for strings.HasSuffix(dir, "/") && dir != "/" {
		dir = strings.TrimSuffix(dir, "/")
	}
	return dir
}

func MergeExportSecrets(newCfg, oldCfg ExportConfig) ExportConfig {
	newCfg.SSH = mergeSSHSecrets(newCfg.SSH, oldCfg.SSH)
	return newCfg
}

func mergeSSHSecrets(newSSH, oldSSH SSHExportConfig) SSHExportConfig {
	if newSSH.Password == "" && oldSSH.Password != "" {
		newSSH.Password = oldSSH.Password
		newSSH.HasPassword = true
	}
	if newSSH.PrivateKey == "" && oldSSH.PrivateKey != "" {
		newSSH.PrivateKey = oldSSH.PrivateKey
		newSSH.HasPrivateKey = true
	}
	if newSSH.PrivateKeyPassphrase == "" && oldSSH.PrivateKeyPassphrase != "" {
		newSSH.PrivateKeyPassphrase = oldSSH.PrivateKeyPassphrase
		newSSH.HasPassphrase = true
	}
	if newSSH.Password != "" {
		newSSH.HasPassword = true
	}
	if newSSH.PrivateKey != "" {
		newSSH.HasPrivateKey = true
	}
	if newSSH.PrivateKeyPassphrase != "" {
		newSSH.HasPassphrase = true
	}
	if newSSH.ProxyPassword == "" && oldSSH.ProxyPassword != "" {
		newSSH.ProxyPassword = oldSSH.ProxyPassword
		newSSH.HasProxyPassword = true
	}
	if newSSH.ProxyPassword != "" {
		newSSH.HasProxyPassword = true
	}
	return newSSH
}

func (c ExportConfig) RedactSecrets() ExportConfig {
	out := c
	if out.SSH.Password != "" {
		out.SSH.HasPassword = true
		out.SSH.Password = ""
	}
	if out.SSH.PrivateKey != "" {
		out.SSH.HasPrivateKey = true
		out.SSH.PrivateKey = ""
	}
	if out.SSH.PrivateKeyPassphrase != "" {
		out.SSH.HasPassphrase = true
		out.SSH.PrivateKeyPassphrase = ""
	}
	if out.SSH.ProxyPassword != "" {
		out.SSH.HasProxyPassword = true
		out.SSH.ProxyPassword = ""
	}
	return out
}
