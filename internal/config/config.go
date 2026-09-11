package config

import (
	"os"
	"strconv"
)

type Config struct {
	ListenAddr      string
	DataDir         string
	SessionSecret   string
	NginxBin        string
	NginxPIDFile    string
	NginxMimeTypes  string
}

func Load() Config {
	dataDir := envOr("FONU_DATA_DIR", "/data")
	return Config{
		ListenAddr:    envOr("FONU_LISTEN", ":6893"),
		DataDir:       dataDir,
		SessionSecret: envOr("FONU_SESSION_SECRET", "change-me-in-production"),
		NginxBin:       envOr("FONU_NGINX_BIN", "nginx"),
		NginxPIDFile:   envOr("FONU_NGINX_PID", dataDir+"/nginx/nginx.pid"),
		NginxMimeTypes: envOr("FONU_NGINX_MIME_TYPES", defaultMimeTypes()),
	}
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

func (c Config) LogsDir() string {
	return c.DataDir + "/logs"
}

func (c Config) CertsDir() string {
	return c.DataDir + "/certs"
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
