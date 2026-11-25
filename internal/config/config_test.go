package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNginxDefaultPorts(t *testing.T) {
	t.Setenv("FONU_NGINX_HTTP_PORT", "18080")
	t.Setenv("FONU_NGINX_HTTPS_PORT", "9443")

	cfg := Load()
	if cfg.NginxDefaultHTTPPort != 18080 {
		t.Fatalf("http port %d", cfg.NginxDefaultHTTPPort)
	}
	if cfg.NginxDefaultHTTPSPort != 9443 {
		t.Fatalf("https port %d", cfg.NginxDefaultHTTPSPort)
	}
	if cfg.DefaultListenPort(false) != 18080 {
		t.Fatalf("default http listen %d", cfg.DefaultListenPort(false))
	}
	if cfg.DefaultListenPort(true) != 9443 {
		t.Fatalf("default https listen %d", cfg.DefaultListenPort(true))
	}
}

func TestLoadNginxDefaultPortsFallback(t *testing.T) {
	t.Setenv("FONU_NGINX_HTTP_PORT", "")
	t.Setenv("FONU_NGINX_HTTPS_PORT", "")

	cfg := Load()
	if cfg.NginxDefaultHTTPPort != 80 || cfg.NginxDefaultHTTPSPort != 443 {
		t.Fatalf("ports %d %d", cfg.NginxDefaultHTTPPort, cfg.NginxDefaultHTTPSPort)
	}
}

func TestResolveNginxMimeTypesIgnoresMissingDockerPath(t *testing.T) {
	dir := t.TempDir()
	mime := filepath.Join(dir, "mime.types")
	if err := os.WriteFile(mime, []byte("types { text/html html; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FONU_NGINX_MIME_TYPES", filepath.Join(dir, "missing", "mime.types"))
	t.Setenv("FONU_DATA_DIR", dir)

	cfg := Load()
	if cfg.NginxMimeTypes != mime {
		t.Fatalf("mime types %q want %q", cfg.NginxMimeTypes, mime)
	}
}

func TestResolveNginxMimeTypesPrefersExistingExplicitPath(t *testing.T) {
	dir := t.TempDir()
	fallback := filepath.Join(dir, "mime.types")
	explicit := filepath.Join(dir, "explicit.types")
	for _, path := range []string{fallback, explicit} {
		if err := os.WriteFile(path, []byte("types { text/html html; }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FONU_DATA_DIR", dir)
	t.Setenv("FONU_NGINX_MIME_TYPES", explicit)
	if got := Load().NginxMimeTypes; got != explicit {
		t.Fatalf("mime types %q want %q", got, explicit)
	}
}
