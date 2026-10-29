package certexport

import (
	"testing"

	"github.com/fonu/fonu/internal/config"
)

func TestSuggestExportTargetsPrefersCertShares(t *testing.T) {
	shares := []string{
		"/vol1/@appshare/fonu/etc",
		"/vol1/@appshare/fonu/certs",
		"/vol1/@appshare/fonu/www",
	}
	got := suggestExportTargets(shares, nil)
	if len(got) == 0 || got[0] != "/vol1/@appshare/fonu/certs" {
		t.Fatalf("got %v", got)
	}
}

func TestPathHintsForFNOS(t *testing.T) {
	t.Setenv("TRIM_PKGVAR", "/var/apps/fonu/var")
	t.Setenv("TRIM_DATA_SHARE_PATHS", "/vol1/@appshare/fonu/certs:/vol1/@appshare/fonu/www")

	cfg := config.Config{DataDir: t.TempDir()}
	h := PathHintsFor(cfg)
	if h.DeployKind != "fnos" {
		t.Fatalf("deploy kind %q", h.DeployKind)
	}
	if len(h.SuggestedExportTargets) == 0 {
		t.Fatal("expected suggested targets")
	}
}
