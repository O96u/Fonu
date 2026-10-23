package certexport

import (
	"path/filepath"
	"testing"
)

func TestValidateTargetPath(t *testing.T) {
	root := t.TempDir()
	certs := filepath.Join(root, "certs")
	target := filepath.Join(root, "export")
	if err := ValidateTargetPath(target, certs); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	if err := ValidateTargetPath(certs, certs); err == nil {
		t.Fatal("same dir should fail")
	}
	if err := ValidateTargetPath(filepath.Join(certs, "extra"), certs); err == nil {
		t.Fatal("under certs should fail")
	}
	if err := ValidateTargetPath("relative/path", certs); err == nil {
		t.Fatal("relative should fail")
	}
}
