package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
)

func TestEnsureErrorPagesWritesAssets(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{DataDir: dir}

	if err := EnsureErrorPages(cfg); err != nil {
		t.Fatal(err)
	}

	errorsDir := cfg.ErrorsDir()
	for _, name := range []string{"403.html", "404.html", "error.png", "429.png"} {
		path := filepath.Join(errorsDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}

	html, err := os.ReadFile(filepath.Join(errorsDir, "403.html"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(html)
	for _, want := range []string{
		"/fonu-errors/error.png",
		"background:url(/fonu-errors/error.png)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("403.html missing %q:\n%s", want, body)
		}
	}
}
