package app

import (
	"crypto/tls"
	"fmt"

	"github.com/fonu/fonu/internal/config"
)

// consoleTLSConfig validates both files before starting background services.
// HTTP remains the default only when neither file is configured.
func consoleTLSConfig(cfg config.Config) (*tls.Config, error) {
	if cfg.TLSCertFile == "" && cfg.TLSKeyFile == "" {
		return nil, nil
	}
	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
		return nil, fmt.Errorf("FONU_TLS_CERT_FILE and FONU_TLS_KEY_FILE must both be configured")
	}
	pair, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load console TLS certificate: %w", err)
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{pair},
	}, nil
}
