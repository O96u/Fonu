package certexport

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func runOnChangeShell(ctx context.Context, shell string, meta CertMeta, logger *slog.Logger) {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		return
	}
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, shellCommand(), shellCommandArgs(shell)...)
	cmd.Env = append(os.Environ(),
		"certName="+meta.Name,
		"certDir="+meta.CertDir,
		"certFile="+meta.CertFile,
		"keyFile="+meta.KeyFile,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Warn("cert export on-change shell failed",
			"storage", meta.StorageName,
			"deleted", meta.Deleted,
			"error", err.Error(),
			"output", strings.TrimSpace(string(out)))
	}
}

func shellCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}

func shellCommandArgs(script string) []string {
	if runtime.GOOS == "windows" {
		return []string{"/C", script}
	}
	return []string{"-c", script}
}

func metaForStorage(certsDir, storageName, displayName string, deleted bool) CertMeta {
	certDir := filepath.Join(certsDir, storageName)
	meta := CertMeta{
		Name:        displayName,
		StorageName: storageName,
		Deleted:     deleted,
		CertDir:     certDir,
	}
	if !deleted {
		meta.CertFile = filepath.Join(certDir, certFileName)
		meta.KeyFile = filepath.Join(certDir, keyFileName)
	}
	return meta
}
