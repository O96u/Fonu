package certexport

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	certstore "github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
)

type Service struct {
	cfg    config.Config
	certs  *certstore.Store
	logger *slog.Logger
}

func New(cfg config.Config, certs *certstore.Store, logger *slog.Logger) *Service {
	return &Service{
		cfg:    cfg,
		certs:  certs,
		logger: logger.With("module", "CERT_EXPORT"),
	}
}

func (s *Service) SourceDir() string {
	return s.cfg.CertsDir()
}

func (s *Service) SyncAllEnabled(ctx context.Context) {
	records, err := s.certs.List(ctx)
	if err != nil {
		return
	}
	for _, rec := range records {
		if !exportSyncEnabled(rec.Export) {
			continue
		}
		if err := s.SyncRecord(ctx, rec.Domain); err != nil {
			s.logger.Warn("cert export startup sync failed", "domain", rec.Domain, "error", err.Error())
		}
	}
}

func (s *Service) SyncRecord(ctx context.Context, domain string) error {
	rec, err := s.certs.GetByDomain(ctx, domain)
	if err != nil {
		return err
	}
	return s.syncRecord(ctx, rec, false)
}

func (s *Service) SyncFromRecord(ctx context.Context, rec certstore.Record) {
	if err := s.syncRecord(ctx, rec, false); err != nil {
		s.logger.Warn("cert export sync failed", "domain", rec.Domain, "error", err.Error())
	}
}

func (s *Service) RemoveFromRecord(ctx context.Context, rec certstore.Record) {
	if err := s.syncRecord(ctx, rec, true); err != nil {
		s.logger.Warn("cert export remove failed", "domain", rec.Domain, "error", err.Error())
	}
}

func (s *Service) SaveExport(ctx context.Context, domain string, cfg certstore.ExportConfig) (certstore.Record, error) {
	existing, err := s.certs.GetByDomain(ctx, domain)
	if err == nil {
		cfg = certstore.MergeExportSecrets(cfg, existing.Export)
	}
	norm, err := cfg.Normalize(s.cfg.CertsDir())
	if err != nil {
		return certstore.Record{}, err
	}
	if err := s.certs.SetExport(ctx, domain, norm); err != nil {
		return certstore.Record{}, err
	}
	return s.certs.GetByDomain(ctx, domain)
}

func exportSyncEnabled(cfg certstore.ExportConfig) bool {
	if cfg.SSH.Enabled {
		return true
	}
	return cfg.Enabled && len(cfg.Targets) > 0
}

func (s *Service) syncRecord(ctx context.Context, rec certstore.Record, remove bool) error {
	cfg := rec.Export
	if !exportSyncEnabled(cfg) {
		if remove {
			return nil
		}
		return fmt.Errorf("未启用本地映射或 SSH 上传")
	}
	names := certstore.RecordStorageNames(rec)
	display := displayName(rec)
	var firstErr error

	if cfg.Enabled && len(cfg.Targets) > 0 {
		err := s.syncStorages(ctx, cfg, names, display, remove)
		s.recordLocalSyncResult(ctx, rec.Domain, err)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if cfg.SSH.Enabled {
		err := syncSSHUpload(ctx, cfg.SSH, s.cfg.CertsDir(), names, display, remove)
		s.recordSSHSyncResult(ctx, rec.Domain, err)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (s *Service) syncStorages(ctx context.Context, cfg certstore.ExportConfig, storageNames []string, displayName string, remove bool) error {
	if len(storageNames) == 0 {
		return nil
	}
	certsDir := s.cfg.CertsDir()
	var firstErr error
	for _, storageName := range storageNames {
		srcDir := filepath.Join(certsDir, storageName)
		for _, targetRoot := range cfg.Targets {
			targetRoot = cleanAbs(targetRoot)
			dstDir := filepath.Join(targetRoot, storageName)
			if remove {
				if err := os.RemoveAll(dstDir); err != nil && firstErr == nil {
					firstErr = err
				}
				continue
			}
			if err := copyPEMPair(srcDir, dstDir); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	meta := metaForStorage(certsDir, storageNames[0], displayName, remove)
	runOnChangeShell(ctx, cfg.OnChangeShell, meta, s.logger)
	return firstErr
}

func displayName(rec certstore.Record) string {
	if strings.TrimSpace(rec.Name) != "" {
		return rec.Name
	}
	return rec.Domain
}

func (s *Service) recordLocalSyncResult(ctx context.Context, domain string, err error) {
	rec, getErr := s.certs.GetByDomain(ctx, domain)
	if getErr != nil {
		return
	}
	cfg := rec.Export
	now := time.Now().UTC().Format(time.RFC3339)
	cfg.LastSyncAt = now
	if err != nil {
		cfg.LastError = err.Error()
	} else {
		cfg.LastError = ""
	}
	_ = s.certs.SetExport(ctx, domain, cfg)
}

func (s *Service) recordSSHSyncResult(ctx context.Context, domain string, err error) {
	rec, getErr := s.certs.GetByDomain(ctx, domain)
	if getErr != nil {
		return
	}
	cfg := rec.Export
	now := time.Now().UTC().Format(time.RFC3339)
	cfg.SSH.LastSyncAt = now
	if err != nil {
		cfg.SSH.LastError = err.Error()
	} else {
		cfg.SSH.LastError = ""
	}
	_ = s.certs.SetExport(ctx, domain, cfg)
}
