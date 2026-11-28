package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/auth"
)

func NewRouter(deps Deps) http.Handler {
	logsHandler := NewLogsHandler(deps.Config, deps.Proxy, deps.IPLocate)
	r := &Router{
		authHandler:      NewAuthHandler(deps.Auth, deps.Notify),
		proxyHandler:     NewProxyHandler(deps.Config, deps.Proxy, logsHandler, deps.Traffic),
		nginxHandler:     NewNginxHandler(deps.Proxy),
		statusHandler:    NewStatusHandler(deps.Config, deps.Proxy, deps.DDNS, deps.ACME, deps.Analytics, deps.StartedAt, deps.Config.NginxPIDFile),
		ddnsHandler:      NewDDNSHandler(deps.DDNS),
		certHandler:      NewCertHandler(deps.ACME, deps.CertExport),
		settingsHandler:  NewSettingsHandler(deps.Settings, deps.Proxy, deps.Notify),
		chinaCIDRHandler: NewChinaCIDRHandler(deps.ChinaCIDR),
		logsHandler:      logsHandler,
		backupHandler:    NewBackupHandler(deps.Backup),
		discoveryHandler: NewDiscoveryHandler(deps.Discovery),
		authSvc:          deps.Auth,
		staticFS:         deps.StaticFS,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/version", Version(deps.Config))
	mux.HandleFunc("GET /api/auth/status", r.authHandler.Status)
	mux.HandleFunc("POST /api/auth/setup", r.authHandler.Setup)
	mux.HandleFunc("POST /api/auth/login", r.authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", r.authHandler.Logout)

	protect := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, SessionMiddleware(deps.Auth)(RequireAuth(deps.Auth)(http.HandlerFunc(handler))))
	}

	protect("POST /api/auth/password", r.authHandler.ChangePassword)
	protect("POST /api/auth/username", r.authHandler.ChangeUsername)
	protect("GET /api/status", r.statusHandler.Get)
 protect("GET /api/public-ip",r.statusHandler.PublicIP)
	protect("GET /api/runtime", r.statusHandler.Runtime)
	protect("GET /api/nginx/apply-status", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusOK, deps.Proxy.NginxApplyStatus()) })
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Cache-Control", "no-store"); w.WriteHeader(http.StatusOK); _, _ = w.Write([]byte("ok\n")) })
	if deps.Analytics != nil {
		awarenessHandler := NewAwarenessHandler(deps.Analytics, deps.IPLocate)
		protect("GET /api/awareness/live", awarenessHandler.Live)
		analyticsHandler := NewAnalyticsHandler(deps.Analytics)
		protect("GET /api/analytics/overview", analyticsHandler.Overview)
	}
	protect("GET /api/proxy-groups", r.proxyHandler.BusinessGroups)
	protect("POST /api/proxy-groups", r.proxyHandler.ChangeBusinessGroups)
	protect("GET /api/proxies", r.proxyHandler.List)
	protect("GET /api/proxies/{id}/favicon", r.proxyHandler.Favicon)
	protect("GET /api/proxy-entries", r.proxyHandler.ListEntries)
	protect("POST /api/proxy-entries", r.proxyHandler.CreateEntry)
	protect("PUT /api/proxy-entries/reorder", r.proxyHandler.ReorderEntries)
	protect("PUT /api/proxy-entries/{id}", r.proxyHandler.UpdateEntry)
	protect("POST /api/proxy-entries/{id}/clone", r.proxyHandler.CloneEntry)
	protect("DELETE /api/proxy-entries/{id}", r.proxyHandler.DeleteEntry)
	protect("GET /api/proxies/traffic", r.proxyHandler.Traffic)
	protect("GET /api/proxies/traffic/snapshot", r.proxyHandler.TrafficSnapshot)
	protect("GET /api/proxies/{id}/diagnostics", r.proxyHandler.Diagnostics)
	protect("POST /api/proxies/{id}/diagnostics", r.proxyHandler.RunDiagnostics)
	protect("POST /api/proxies", r.proxyHandler.Create)
	protect("POST /api/proxies/batch", r.proxyHandler.SaveBatch)
	protect("POST /api/proxies/access-blacklist", r.proxyHandler.AccessBlacklist)
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
	dnsheHandler := NewDNSHEHandler(deps.DNSHERenew, deps.DDNS)
	protect("GET /api/dnshe/accounts", dnsheHandler.ListAccounts)
	protect("POST /api/dnshe/accounts", dnsheHandler.CreateAccount)
	protect("PUT /api/dnshe/accounts/reorder", dnsheHandler.ReorderAccounts)
	protect("PUT /api/dnshe/accounts/{id}", dnsheHandler.UpdateAccount)
	protect("DELETE /api/dnshe/accounts/{id}", dnsheHandler.DeleteAccount)
	protect("GET /api/dnshe/accounts/{id}/credentials", dnsheHandler.RevealCredentials)
	protect("POST /api/dnshe/accounts/{id}/push-to-ddns", dnsheHandler.PushToDDNS)
	protect("GET /api/dnshe/accounts/{id}/domains", dnsheHandler.ListDomains)
	protect("POST /api/dnshe/accounts/{id}/domains/{subId}/renew", dnsheHandler.RenewDomain)
	protect("POST /api/dnshe/accounts/{id}/domains/{subId}/delete", dnsheHandler.DeleteDomain)
	protect("PUT /api/dnshe/accounts/{id}/domains/reorder", dnsheHandler.ReorderDomains)
	protect("POST /api/dnshe/accounts/{id}/domains/register", dnsheHandler.RegisterDomain)
	protect("PUT /api/dnshe/domains/auto-renew", dnsheHandler.SetDomainAutoRenew)
	protect("GET /api/dnshe/accounts/{id}/gifts", dnsheHandler.ListGifts)
	protect("POST /api/dnshe/accounts/{id}/gifts", dnsheHandler.InitiateGift)
	protect("POST /api/dnshe/accounts/{id}/gifts/accept", dnsheHandler.AcceptGift)
	protect("POST /api/dnshe/accounts/{id}/gifts/{giftId}/cancel", dnsheHandler.CancelGift)
	protect("GET /api/certificates", r.certHandler.List)
	protect("GET /api/certificates/export-paths", r.certHandler.ExportPaths)
	protect("GET /api/certificates/ca-options", r.certHandler.Options)
	protect("POST /api/certificates/apply", r.certHandler.Apply)
	protect("GET /api/certificates/jobs/{id}/stream", r.certHandler.ApplyJobStream)
	protect("POST /api/certificates/import", r.certHandler.Import)
	protect("POST /api/certificates/renew", r.certHandler.Renew)
	protect("PUT /api/certificates/{domain}", r.certHandler.Update)
	protect("PUT /api/certificates/{domain}/auto-renew", r.certHandler.SetAutoRenew)
	protect("PUT /api/certificates/{domain}/dns-config", r.certHandler.SetDNSConfig)
	protect("POST /api/certificates/ssh-test", r.certHandler.TestSSH)
	protect("GET /api/certificates/{domain}/deployment-history", r.certHandler.DeploymentHistory)
	protect("POST /api/certificates/{domain}/export/sync", r.certHandler.SyncExport)
	protect("GET /api/certificates/{domain}/download", r.certHandler.Download)
	protect("DELETE /api/certificates/{domain}", r.certHandler.Delete)
	notifyHandler := NewNotifyHandler(deps.Notify)
	protect("GET /api/settings", r.settingsHandler.Get)
	protect("PUT /api/settings", r.settingsHandler.Put)
	protect("POST /api/settings/notify/test", notifyHandler.Test)
 protect("GET /api/settings/notify/history",notifyHandler.History)
	if deps.ChinaCIDR != nil {
		protect("GET /api/settings/china-cidr", r.chinaCIDRHandler.Status)
		protect("POST /api/settings/china-cidr/refresh", r.chinaCIDRHandler.Refresh)
	}
	protect("GET /api/logs/summary", r.logsHandler.Summary)
 protect("GET /api/logs/history", r.logsHandler.History)
 protect("GET /api/logs/access", r.logsHandler.Access)
	protect("GET /api/logs/error", r.logsHandler.Error)
	protect("GET /api/logs/system", r.logsHandler.System)
	protect("GET /api/logs/stream", r.logsHandler.Stream)
	protect("GET /api/backup/export", r.backupHandler.Export)
	protect("POST /api/backup/restore", r.backupHandler.Restore)
 protect("GET /api/backup/local",r.backupHandler.ListLocal)
 protect("POST /api/backup/local",r.backupHandler.CreateLocal)
 protect("GET /api/backup/local/{name}",r.backupHandler.DownloadLocal)
 protect("DELETE /api/backup/local/{name}",r.backupHandler.DeleteLocal)
 protect("POST /api/backup/local/{name}/restore",r.backupHandler.RestoreLocal)
	protect("GET /api/discovery/scan", r.discoveryHandler.Scan)
	fsHandler := NewFSHandler()
	protect("GET /api/fs/directories", fsHandler.ListDirectories)
	if deps.FRP != nil {
		frpHandler := NewFRPHandler(deps.FRP, deps.Proxy)
		protect("GET /api/frp", frpHandler.Get)
		protect("PUT /api/frp", frpHandler.Put)
		protect("GET /api/frp/status", frpHandler.Status)
		protect("POST /api/frp/restart", frpHandler.Restart)
		protect("POST /api/frp/diagnostics", frpHandler.Diagnostics)
		protect("POST /api/frp/export-config", frpHandler.ExportConfig)
		protect("POST /api/frp/sync-domains", frpHandler.SyncDomains)
		protect("GET /api/frp/domain-candidates", frpHandler.DomainCandidates)
		protect("GET /api/frp/logs", frpHandler.Logs)
	}

	if deps.StaticFS != nil {
		mux.Handle("/", r.spaHandler())
	}

	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return loggingMiddleware(logger, protectRequests(SessionMiddleware(deps.Auth)(mux)))
}

type Router struct {
	authHandler      *AuthHandler
	proxyHandler     *ProxyHandler
	nginxHandler     *NginxHandler
	statusHandler    *StatusHandler
	ddnsHandler      *DDNSHandler
	certHandler      *CertHandler
	settingsHandler  *SettingsHandler
	chinaCIDRHandler *ChinaCIDRHandler
	logsHandler      *LogsHandler
	backupHandler    *BackupHandler
	discoveryHandler *DiscoveryHandler
	authSvc          *auth.Service
	staticFS         fs.FS
}

var hashedAsset = regexp.MustCompile(`-[A-Za-z0-9_-]{8,}\.[A-Za-z0-9]+$`)

func (r *Router) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(r.staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") {
			http.NotFound(w, req)
			return
		}

		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" { path = "index.html" }
		if _,err:=fs.Stat(r.staticFS,path);err!=nil {
			if strings.HasPrefix(path,"assets/")||filepath.Ext(path)!="" {
				http.NotFound(w,req)
				return
			}
			w.Header().Set("Cache-Control","no-cache")
			content,err:=fs.ReadFile(r.staticFS,"index.html")
			if err!=nil {http.NotFound(w,req);return}
			w.Header().Set("Content-Type","text/html; charset=utf-8")
			http.ServeContent(w,req,"index.html",time.Time{},strings.NewReader(string(content)))
			return
		}
		if strings.HasPrefix(path,"assets/")&&hashedAsset.MatchString(path) {
			w.Header().Set("Cache-Control","public, max-age=31536000, immutable")
		} else {w.Header().Set("Cache-Control","no-cache")}

		fileServer.ServeHTTP(w, req)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

// Unwrap lets http.ResponseController reach the underlying connection.
func (w *statusRecorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Flush() { _ = w.FlushError() }
func (w *statusRecorder) FlushError() error { return http.NewResponseController(w.ResponseWriter).Flush() }

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
