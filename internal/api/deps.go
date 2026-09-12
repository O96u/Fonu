package api

import (
	"io/fs"
	"log/slog"

	"github.com/fonu/fonu/internal/acme"
	"github.com/fonu/fonu/internal/auth"
	"github.com/fonu/fonu/internal/backup"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/ddns"
	"github.com/fonu/fonu/internal/discovery"
	"github.com/fonu/fonu/internal/service"
	"github.com/fonu/fonu/internal/settings"
	"github.com/fonu/fonu/internal/traffic"
)

type Deps struct {
	Config     config.Config
	Logger     *slog.Logger
	Auth       *auth.Service
	Proxy      *service.ProxyService
	DDNS       *ddns.Service
	ACME       *acme.Service
	Settings   *settings.Store
	Backup     *backup.Service
	Discovery  *discovery.Service
	Traffic    *traffic.Collector
	StaticFS   fs.FS
	StartedAt  string
}
