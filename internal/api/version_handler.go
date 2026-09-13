package api

import (
	"net/http"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/version"
)

type versionResponse struct {
	Version           string `json:"version"`
	NginxHTTPPort     int    `json:"nginx_http_port"`
	NginxHTTPSPort    int    `json:"nginx_https_port"`
}

func Version(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, versionResponse{
			Version:        version.Version,
			NginxHTTPPort:  cfg.NginxDefaultHTTPPort,
			NginxHTTPSPort: cfg.NginxDefaultHTTPSPort,
		})
	}
}
