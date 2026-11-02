package app

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fonu/fonu/internal/acme"
	"github.com/fonu/fonu/internal/analytics"
	"github.com/fonu/fonu/internal/api"
	"github.com/fonu/fonu/internal/auth"
	"github.com/fonu/fonu/internal/awareness"
	"github.com/fonu/fonu/internal/backup"
	"github.com/fonu/fonu/internal/certexport"
	"github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/chinacidr"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/ddns"
	"github.com/fonu/fonu/internal/discovery"
	"github.com/fonu/fonu/internal/dnsherenew"
	"github.com/fonu/fonu/internal/frp"
	"github.com/fonu/fonu/internal/ingest"
	"github.com/fonu/fonu/internal/iplocate"
	"github.com/fonu/fonu/internal/logstore"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/notify"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/scheduler"
	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/service"
	"github.com/fonu/fonu/internal/settings"
	"github.com/fonu/fonu/internal/traffic"
)

type App struct {
	cfg        config.Config
	logger     *slog.Logger
	server     *http.Server
	nginx      *nginx.Manager
	frp        *frp.Manager
	scheduler  *scheduler.Scheduler
	shutdownFn context.CancelFunc
	db         interface{ Close() error }
}

func New(cfg config.Config, staticFS fs.FS, migrationsDir string) (*App, error) {
	appLogWriter := logstore.NewAppLogWriter(cfg.LogsDir())
	logger := slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stdout, appLogWriter), &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	for _, dir := range []string{cfg.DataDir, cfg.NginxDir(), cfg.LogsDir(), cfg.CertsDir(), cfg.FrpDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	conn, err := db.Open(cfg.DBPath())
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(context.Background(), conn, migrationsDir); err != nil {
		conn.Close()
		return nil, err
	}

	secretBox, err := secret.NewBox(cfg.SessionSecret)
	if err != nil {
		conn.Close()
		return nil, err
	}

	authSvc := auth.New(conn)
	if err := authSvc.ApplyInstallAdmin(context.Background(), cfg.DataDir+"/admin.bootstrap", logger); err != nil {
		conn.Close()
		return nil, fmt.Errorf("bootstrap admin: %w", err)
	}
	proxyStore := proxy.NewStore(conn)
	settingsStore := settings.NewStore(conn)
	ddnsStore := ddns.NewStore(conn)
	certStore := certificate.NewStore(conn)
	nginxMgr := nginx.NewManager(cfg, logger.With("module", "NGINX"))
	if err := nginxMgr.EnsureDirs(); err != nil {
		conn.Close()
		return nil, err
	}

	proxySvc := service.NewProxyService(cfg, conn, proxyStore, certStore, settingsStore, nginxMgr)
	chinaCIDRSvc := chinacidr.New(cfg, settingsStore, nginxMgr, proxyStore, logger.With("module", "CHINA_CIDR"))
	notifySvc := notify.New(settingsStore, secretBox, logger)
	proxySvc.SetNotify(notifySvc)
	ddnsSvc := ddns.NewService(ddnsStore, settingsStore, secretBox, logger, notifySvc)
	dnsheRenewSvc := dnsherenew.NewService(settingsStore, secretBox, logger)
	certExportSvc := certexport.New(cfg, certStore, logger)
	acmeSvc := acme.NewService(cfg, certStore, ddnsSvc, settingsStore, proxySvc, certExportSvc, logger, notifySvc)
	backupSvc := backup.New(cfg.DataDir)
	discoverySvc := discovery.New()
	frpStore := frp.NewStore(settingsStore, secretBox)
	frpMgr := frp.NewManager(cfg, frpStore, logger)
	if err := frpMgr.EnsureDirs(); err != nil {
		conn.Close()
		return nil, err
	}
	startedAt := time.Now().UTC().Format(time.RFC3339)
	accessLogPath := filepath.Join(cfg.LogsDir(), "access.log")
	ipLocateSvc := iplocate.New(cfg)
	ingestStore := ingest.NewStore(conn)
	ingestSvc := ingest.NewService(ingestStore, accessLogPath, settingsStore, ipLocateSvc, notifySvc, proxySvc.List, logger)
	trafficCollector := traffic.NewCollector(conn, accessLogPath, notifySvc)
	trafficCollector.AttachIngest(ingestSvc)
	analyticsSvc := analytics.New(conn, settingsStore, ingestSvc)
	serverGeo := analytics.NewServerGeo(ddnsSvc, ipLocateSvc)
	serverGeo.Warm()
	analyticsSvc.SetServerGeo(serverGeo.Resolve)
	analyticsSvc.SetEventGeo(func(events []awareness.Event) {
		awareness.EnrichEventsGeo(events, ipLocateSvc)
	})

	handler := api.NewRouter(api.Deps{
		Config:     cfg,
		Logger:     logger,
		Auth:       authSvc,
		Proxy:      proxySvc,
		DDNS:       ddnsSvc,
		ACME:       acmeSvc,
		CertExport: certExportSvc,
		Settings:   settingsStore,
		Notify:     notifySvc,
		ChinaCIDR:  chinaCIDRSvc,
		Backup:     backupSvc,
		Discovery:  discoverySvc,
		DNSHERenew: dnsheRenewSvc,
		FRP:        frpMgr,
		Traffic:    trafficCollector,
		Analytics:  analyticsSvc,
		Ingest:     ingestSvc,
		IPLocate:   ipLocateSvc,
		StaticFS:   staticFS,
		StartedAt:  startedAt,
	})
	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	intervalMinutes, _ := settingsStore.GetInt(ctx, settings.KeyDDNSCheckInterval)
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}
	chinaHours, _ := settingsStore.GetInt(ctx, settings.KeyChinaCIDRUpdateHours)
	if chinaHours <= 0 {
		chinaHours = 24
	}
	sched := scheduler.New(
		scheduler.Job{
			Name:     "ddns",
			Interval: time.Duration(intervalMinutes) * time.Minute,
			IntervalOf: func(ctx context.Context) time.Duration {
				m, _ := settingsStore.GetInt(ctx, settings.KeyDDNSCheckInterval)
				if m <= 0 {
					m = 5
				}
				return time.Duration(m) * time.Minute
			},
			Run: ddnsSvc.Tick,
		},
		scheduler.Job{
			Name:     "acme",
			Interval: 24 * time.Hour,
			Run:      acmeSvc.Tick,
		},
		scheduler.Job{
			Name:     "china_cidr",
			Interval: time.Duration(chinaHours) * time.Hour,
			IntervalOf: func(ctx context.Context) time.Duration {
				h, _ := settingsStore.GetInt(ctx, settings.KeyChinaCIDRUpdateHours)
				if h <= 0 {
					h = 24
				}
				return time.Duration(h) * time.Hour
			},
			Run: chinaCIDRSvc.Tick,
		},
		scheduler.Job{
			Name:     "dnshe_renew",
			Interval: 24 * time.Hour,
			Run:      dnsheRenewSvc.Tick,
		},
	)
	sched.Start(ctx)
	ingestSvc.Start(ctx)
	trafficCollector.Start(ctx)
	go func() {
		if err := ipLocateSvc.Ensure(ctx); err != nil {
			logger.Warn("ip2region init failed; geo labels disabled until database is available", "module", "IPLOCATE", "error", err.Error())
		} else {
			logger.Info("ip2region ready", "module", "IPLOCATE", "path", ipLocateSvc.DBPath())
		}
	}()
	go certExportSvc.SyncAllEnabled(ctx)

	app := &App{
		cfg:        cfg,
		logger:     logger,
		server:     server,
		nginx:      nginxMgr,
		frp:        frpMgr,
		scheduler:  sched,
		shutdownFn: cancel,
		db:         conn,
	}

	if err := app.bootstrapNginx(context.Background(), proxySvc); err != nil {
		cancel()
		conn.Close()
		return nil, err
	}
	if err := frpMgr.Bootstrap(context.Background()); err != nil {
		logger.Warn("initial frpc bootstrap skipped", "module", "FRP", "error", err.Error())
	}

	logger.Info("application started", "module", "SYSTEM", "listen", cfg.ListenAddr)
	return app, nil
}

func (a *App) bootstrapNginx(ctx context.Context, proxySvc *service.ProxyService) error {
	if err := proxySvc.ReloadAll(ctx); err != nil {
		a.logger.Warn("initial nginx apply skipped", "module", "NGINX", "error", err.Error())
	}
	return nil
}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("application shutting down", "module", "SYSTEM")
	if a.shutdownFn != nil {
		a.shutdownFn()
	}
	if err := a.server.Shutdown(ctx); err != nil {
		return err
	}
	if err := a.nginx.Stop(ctx); err != nil {
		a.logger.Error("nginx stop failed", "module", "NGINX", "error", err.Error())
	}
	if a.frp != nil {
		if err := a.frp.Stop(ctx); err != nil {
			a.logger.Error("frpc stop failed", "module", "FRP", "error", err.Error())
		}
	}
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func ResolveMigrationsDir() (string, error) {
	if dir := os.Getenv("FONU_MIGRATIONS_DIR"); dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, nil
		}
		return "", fmt.Errorf("FONU_MIGRATIONS_DIR is not a directory: %s", dir)
	}
	candidates := []string{
		"migrations",
		filepath.Join("..", "migrations"),
		filepath.Join("..", "..", "migrations"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("migrations directory not found")
}
