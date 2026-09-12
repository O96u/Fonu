package nginx

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

type Manager struct {
	cfg    config.Config
	logger *slog.Logger
}

type ApplyResult struct {
	Reloaded bool
	Message  string
}

func NewManager(cfg config.Config, logger *slog.Logger) *Manager {
	return &Manager{cfg: cfg, logger: logger}
}

func (m *Manager) EnsureDirs() error {
	for _, dir := range []string{m.cfg.NginxDir(), m.cfg.LogsDir(), m.cfg.CertsDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Apply(ctx context.Context, rules []proxy.Rule) (ApplyResult, error) {
	if err := m.EnsureDirs(); err != nil {
		return ApplyResult{}, err
	}

	content, err := Generate(m.cfg, rules)
	if err != nil {
		return ApplyResult{}, err
	}

	tmpPath := m.cfg.NginxConfigPath() + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return ApplyResult{}, err
	}

	if err := m.validate(ctx, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return ApplyResult{}, err
	}

	currentPath := m.cfg.NginxConfigPath()
	backupPath := currentPath + ".bak"
	if _, err := os.Stat(currentPath); err == nil {
		if err := copyFile(currentPath, backupPath); err != nil {
			_ = os.Remove(tmpPath)
			return ApplyResult{}, err
		}
	}

	if err := os.Rename(tmpPath, currentPath); err != nil {
		_ = os.Remove(tmpPath)
		return ApplyResult{}, err
	}

	running, err := m.isRunning()
	if err != nil {
		return ApplyResult{}, err
	}

	if !running {
		if err := m.start(ctx); err != nil {
			return ApplyResult{}, err
		}
		return ApplyResult{Reloaded: true, Message: "Nginx 已启动"}, nil
	}

	if err := m.reload(ctx); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Reloaded: true, Message: "Nginx 已重载"}, nil
}

func (m *Manager) ValidateOnly(ctx context.Context, rules []proxy.Rule) error {
	content, err := Generate(m.cfg, rules)
	if err != nil {
		return err
	}
	tmpPath := filepath.Join(m.cfg.NginxDir(), "validate.tmp.conf")
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	return m.validate(ctx, tmpPath)
}

func (m *Manager) validate(ctx context.Context, configPath string) error {
	cmd := exec.CommandContext(ctx, m.cfg.NginxBin, "-t", "-c", configPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		m.logger.Error("nginx validate failed", "error", msg)
		return fmt.Errorf("Nginx 配置校验失败：%s", sanitizeNginxError(msg))
	}
	m.logger.Info("nginx validate succeeded")
	return nil
}

func (m *Manager) start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, m.cfg.NginxBin, "-c", m.cfg.NginxConfigPath())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("Nginx 启动失败：%s", sanitizeNginxError(msg))
	}
	m.logger.Info("nginx started")
	return nil
}

func (m *Manager) reload(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, m.cfg.NginxBin, "-c", m.cfg.NginxConfigPath(), "-s", "reload")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("Nginx 重载失败：%s", sanitizeNginxError(msg))
	}
	m.logger.Info("nginx reloaded")
	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	if !m.isRunningQuick() {
		return nil
	}
	cmd := exec.CommandContext(ctx, m.cfg.NginxBin, "-c", m.cfg.NginxConfigPath(), "-s", "quit")
	if err := cmd.Run(); err != nil {
		return err
	}
	m.logger.Info("nginx stopped")
	return nil
}

func (m *Manager) isRunning() (bool, error) {
	if _, err := os.Stat(m.cfg.NginxPIDFile); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *Manager) isRunningQuick() bool {
	ok, _ := m.isRunning()
	return ok
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func sanitizeNginxError(msg string) string {
	msg = strings.ReplaceAll(msg, "\r\n", " ")
	msg = strings.ReplaceAll(msg, "\n", " ")
	return strings.TrimSpace(msg)
}
