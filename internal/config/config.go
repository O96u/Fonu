package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	ListenAddr            string
	DataDir               string
	SessionSecret         string
	NginxBin              string
	NginxPIDFile          string
	NginxMimeTypes        string
	NginxDefaultHTTPPort  int
	NginxDefaultHTTPSPort int
	FrpcBin               string
	FrpPIDFile            string
}

func Load() Config {
	dataDir := envOr("FONU_DATA_DIR", defaultDataDir())
	if abs, err := filepath.Abs(dataDir); err == nil {
		dataDir = abs
	}
	return Config{
		ListenAddr:            envOr("FONU_LISTEN", ":6893"),
		DataDir:               dataDir,
		SessionSecret:         envOr("FONU_SESSION_SECRET", "change-me-in-production"),
		NginxBin:              envOr("FONU_NGINX_BIN", "nginx"),
		NginxPIDFile:          envOr("FONU_NGINX_PID", dataDir+"/nginx/nginx.pid"),
		NginxMimeTypes:        envOr("FONU_NGINX_MIME_TYPES", defaultMimeTypes()),
		NginxDefaultHTTPPort:  envIntOr("FONU_NGINX_HTTP_PORT", 80),
		NginxDefaultHTTPSPort: envIntOr("FONU_NGINX_HTTPS_PORT", 443),
		FrpcBin:               envOr("FONU_FRPC_BIN", "frpc"),
		FrpPIDFile:            envOr("FONU_FRP_PID", dataDir+"/frp/frpc.pid"),
	}
}

func (c Config) DefaultListenPort(httpsEnabled bool) int {
	if httpsEnabled {
		return c.NginxDefaultHTTPSPort
	}
	return c.NginxDefaultHTTPPort
}

func (c Config) DBPath() string {
	return c.DataDir + "/fonu.db"
}

func (c Config) NginxDir() string {
	return c.DataDir + "/nginx"
}

func (c Config) NginxConfigPath() string {
	return c.NginxDir() + "/nginx.conf"
}

func (c Config) NginxCustomDir() string {
	return c.NginxDir() + "/custom"
}

func (c Config) NginxGlobalCustomPath() string {
	return c.NginxCustomDir() + "/global.conf"
}

func (c Config) NginxRulesDir() string {
	return c.NginxDir() + "/rules"
}

func (c Config) NginxRuleCustomPath(ruleID int64) string {
	return filepath.Join(c.NginxRulesDir(), fmt.Sprintf("rule_%d.conf", ruleID))
}

func (c Config) NginxBackupsDir() string {
	return c.NginxDir() + "/backups"
}

func (c Config) NginxGlobalBackupsDir() string {
	return c.NginxBackupsDir() + "/global"
}

func (c Config) NginxRuleBackupsDir(ruleID int64) string {
	return filepath.Join(c.NginxBackupsDir(), "rules", fmt.Sprintf("%d", ruleID))
}

func (c Config) LogsDir() string {
	return c.DataDir + "/logs"
}

func (c Config) CertsDir() string {
	return c.DataDir + "/certs"
}

func (c Config) ErrorsDir() string {
	return c.NginxDir() + "/errors"
}

func (c Config) ChinaCIDRPath() string {
	return c.NginxDir() + "/china_cidr.conf"
}

func (c Config) ChinaCIDRTempPath() string {
	return c.NginxDir() + "/china_cidr.conf.tmp"
}

func (c Config) FrpDir() string {
	return c.DataDir + "/frp"
}

func (c Config) FrpConfigPath() string {
	return c.FrpDir() + "/frpc.toml"
}

func (c Config) FrpLogPath() string {
	return filepath.Join(c.LogsDir(), "frpc.log")
}

func defaultDataDir() string {
	if _, err := os.Stat("/data"); err == nil {
		return "/data"
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, ".data")
	}
	return "/data"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultMimeTypes() string {
	candidates := []string{
		"/etc/nginx/mime.types",
		"deploy/nginx/mime.types",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "/etc/nginx/mime.types"
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
