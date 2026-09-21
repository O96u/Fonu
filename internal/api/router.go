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
	logsHandler := NewLogsHandler(deps.Config)
	r := &Router{
		authHandler:     NewAuthHandler(deps.Auth, deps.Notify),
		proxyHandler:    NewProxyHandler(deps.Config, deps.Proxy, logsHandler, deps.Traffic),
		nginxHandler:    NewNginxHandler(deps.Proxy),
		statusHandler:   NewStatusHandler(deps.Config, deps.Proxy, deps.DDNS, deps.ACME, deps.StartedAt, deps.Config.NginxPIDFile),
		ddnsHandler:     NewDDNSHandler(deps.DDNS),
		certHandler:     NewCertHandler(deps.ACME),
		settingsHandler:  NewSettingsHandler(deps.Settings, deps.Proxy, deps.Notify),
		chinaCIDRHandler: NewChinaCIDRHandler(deps.ChinaCIDR),
		logsHandler:      logsHandler,
		backupHandler:    NewBackupHandler(deps.Backup),
		discoveryHandler: NewDiscoveryHandler(deps.Discovery),
		authSvc:         deps.Auth,
		staticFS:        deps.StaticFS,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/version", Version(deps.Config))
	mux.HandleFunc("GET /api/auth/status", r.authHandler.Status)
	mux.HandleFunc("POST /api/auth/login", r.authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", r.authHandler.Logout)

	protect := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, SessionMiddleware(deps.Auth)(RequireAuth(deps.Auth)(http.HandlerFunc(handler))))
	}

	protect("POST /api/auth/password", r.authHandler.ChangePassword)
	protect("GET /api/status", r.statusHandler.Get)
	protect("GET /api/proxies", r.proxyHandler.List)
	protect("GET /api/proxy-entries", r.proxyHandler.ListEntries)
	protect("POST /api/proxy-entries", r.proxyHandler.CreateEntry)
	protect("PUT /api/proxy-entries/reorder", r.proxyHandler.ReorderEntries)
	protect("PUT /api/proxy-entries/{id}", r.proxyHandler.UpdateEntry)
	protect("DELETE /api/proxy-entries/{id}", r.proxyHandler.DeleteEntry)
	protect("GET /api/proxies/traffic", r.proxyHandler.Traffic)
	protect("POST /api/proxies", r.proxyHandler.Create)
	protect("PUT /api/proxies/reorder", r.proxyHandler.Reorder)
	protect("PUT /api/proxies/{id}", r.proxyHandler.Update)
	protect("DELETE /api/proxies/{id}", r.proxyHandler.Delete)
	protect("GET /api/proxies/{id}/logs/stream", r.proxyHandler.StreamLogs)
	protect("GET /api/proxies/{id}/clients", r.proxyHandler.Clients)
	protect("GET /api/proxies/{id}/nginx", r.nginxHandler.GetRule)
	protect("PUT /api/proxies/{id}/nginx", r.nginxHandler.PutRule)
	protect("POST /api/proxies/{id}/nginx/rollback", r.nginxHandler.RollbackRule)
	protect("GET /api/settings/nginx/global", r.nginxHandler.GetGlobal)
	protect("PUT /api/settings/nginx/global", r.nginxHandler.PutGlobal)
	protect("POST /api/settings/nginx/global/rollback", r.nginxHandler.RollbackGlobal)
	protect("GET /api/ddns", r.ddnsHandler.List)
	protect("POST /api/ddns", r.ddnsHandler.Create)
	protect("PUT /api/ddns/{id}", r.ddnsHandler.Update)
	protect("DELETE /api/ddns/{id}", r.ddnsHandler.Delete)
	protect("POST /api/ddns/test", r.ddnsHandler.Test)
	protect("POST /api/ddns/update", r.ddnsHandler.UpdateAll)
	protect("POST /api/ddns/{id}/update", r.ddnsHandler.UpdateOne)
	protect("GET /api/certificates", r.certHandler.List)
	protect("GET /api/certificates/ca-options", r.certHandler.Options)
	protect("POST /api/certificates/apply", r.certHandler.Apply)
	protect("GET /api/certificates/jobs/{id}/stream", r.certHandler.ApplyJobStream)
	protect("POST /api/certificates/import", r.certHandler.Import)
	protect("POST /api/certificates/renew", r.certHandler.Renew)
	protect("GET /api/certificates/{domain}/download", r.certHandler.Download)
	protect("DELETE /api/certificates/{domain}", r.certHandler.Delete)
	notifyHandler := NewNotifyHandler(deps.Notify)
	protect("GET /api/settings", r.settingsHandler.Get)
	protect("PUT /api/settings", r.settingsHandler.Put)
	protect("POST /api/settings/notify/test", notifyHandler.Test)
	if deps.ChinaCIDR != nil {
		protect("GET /api/settings/china-cidr", r.chinaCIDRHandler.Status)
		protect("POST /api/settings/china-cidr/refresh", r.chinaCIDRHandler.Refresh)
	}
	protect("GET /api/logs/access", r.logsHandler.Access)
	protect("GET /api/logs/error", r.logsHandler.Error)
	protect("GET /api/logs/system", r.logsHandler.System)
	protect("GET /api/logs/stream", r.logsHandler.Stream)
	protect("GET /api/backup/export", r.backupHandler.Export)
	protect("POST /api/backup/restore", r.backupHandler.Restore)
	protect("GET /api/discovery/scan", r.discoveryHandler.Scan)
	if deps.FRP != nil {
		frpHandler := NewFRPHandler(deps.FRP, deps.Proxy)
		protect("GET /api/frp", frpHandler.Get)
		protect("PUT /api/frp", frpHandler.Put)
		protect("GET /api/frp/status", frpHandler.Status)
		protect("POST /api/frp/sync-domains", frpHandler.SyncDomains)
		protect("GET /api/frp/logs", frpHandler.Logs)
	}

	if deps.StaticFS != nil {
		mux.Handle("/", r.spaHandler())
	}

	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return loggingMiddleware(logger, SessionMiddleware(deps.Auth)(mux))
}

type Router struct {
	authHandler     *AuthHandler
	proxyHandler    *ProxyHandler
	nginxHandler    *NginxHandler
	statusHandler   *StatusHandler
	ddnsHandler     *DDNSHandler
	certHandler     *CertHandler
	settingsHandler  *SettingsHandler
	chinaCIDRHandler *ChinaCIDRHandler
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
			// SPA fallback: unknown routes serve index.html
			req.URL.Path = "/index.html"
		}
		fileServer.ServeHTTP(w, req)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if strings.HasPrefix(r.URL.Path, "/api/") &&
			!strings.Contains(r.URL.Path, "/logs/stream") &&
			!strings.Contains(r.URL.Path, "/certificates/jobs/") &&
			rec.status >= 400 {
			level := slog.LevelWarn
			msg := "api request completed with client error"
			if rec.status >= 500 {
				level = slog.LevelError
				msg = "api request failed"
			}
			logger.Log(r.Context(), level, msg,
				"module", "HTTP",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}
	})
}
