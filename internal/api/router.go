package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/auth"
)

func NewRouter(deps Deps) http.Handler {
	r := &Router{
		authHandler:     NewAuthHandler(deps.Auth),
		proxyHandler:    NewProxyHandler(deps.Proxy),
		statusHandler:   NewStatusHandler(deps.Config, deps.Proxy, deps.DDNS, deps.ACME, deps.StartedAt, deps.Config.NginxPIDFile),
		ddnsHandler:     NewDDNSHandler(deps.DDNS),
		certHandler:     NewCertHandler(deps.ACME),
		settingsHandler:  NewSettingsHandler(deps.Settings),
		logsHandler:      NewLogsHandler(deps.Config),
		backupHandler:    NewBackupHandler(deps.Backup),
		discoveryHandler: NewDiscoveryHandler(deps.Discovery),
		authSvc:         deps.Auth,
		staticFS:        deps.StaticFS,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/auth/status", r.authHandler.Status)
	mux.HandleFunc("POST /api/auth/setup", r.authHandler.Setup)
	mux.HandleFunc("POST /api/auth/login", r.authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", r.authHandler.Logout)

	protect := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, SessionMiddleware(deps.Auth)(RequireAuth(deps.Auth)(http.HandlerFunc(handler))))
	}

	protect("POST /api/auth/password", r.authHandler.ChangePassword)
	protect("GET /api/status", r.statusHandler.Get)
	protect("GET /api/proxies", r.proxyHandler.List)
	protect("POST /api/proxies", r.proxyHandler.Create)
	protect("PUT /api/proxies/{id}", r.proxyHandler.Update)
	protect("DELETE /api/proxies/{id}", r.proxyHandler.Delete)
	protect("GET /api/ddns", r.ddnsHandler.Get)
	protect("PUT /api/ddns", r.ddnsHandler.Put)
	protect("POST /api/ddns/test", r.ddnsHandler.Test)
	protect("POST /api/ddns/update", r.ddnsHandler.Update)
	protect("GET /api/certificates", r.certHandler.List)
	protect("POST /api/certificates/apply", r.certHandler.Apply)
	protect("POST /api/certificates/renew", r.certHandler.Renew)
	protect("GET /api/settings", r.settingsHandler.Get)
	protect("PUT /api/settings", r.settingsHandler.Put)
	protect("GET /api/logs/access", r.logsHandler.Access)
	protect("GET /api/logs/error", r.logsHandler.Error)
	protect("GET /api/logs/system", r.logsHandler.System)
	protect("GET /api/logs/stream", r.logsHandler.Stream)
	protect("GET /api/backup/export", r.backupHandler.Export)
	protect("POST /api/backup/restore", r.backupHandler.Restore)
	protect("GET /api/discovery/scan", r.discoveryHandler.Scan)

	if deps.StaticFS != nil {
		mux.Handle("/", r.spaHandler())
	}

	return loggingMiddleware(SessionMiddleware(deps.Auth)(mux))
}

type Router struct {
	authHandler     *AuthHandler
	proxyHandler    *ProxyHandler
	statusHandler   *StatusHandler
	ddnsHandler     *DDNSHandler
	certHandler     *CertHandler
	settingsHandler *SettingsHandler
	logsHandler      *LogsHandler
	backupHandler    *BackupHandler
	discoveryHandler *DiscoveryHandler
	authSvc          *auth.Service
	staticFS        fs.FS
}

func (r *Router) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(r.staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") {
			http.NotFound(w, req)
			return
		}
		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(r.staticFS, path); err != nil {
			req.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, req)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	logger := slog.Default()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			logger.Info("api request",
				"module", "SYSTEM",
				"method", r.Method,
				"path", r.URL.Path,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}
	})
}
