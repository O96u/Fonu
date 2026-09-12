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
	"syscall"
	"time"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

const (
	nginxCmdTimeout    = 8 * time.Second
	nginxStartWait     = 5 * time.Second
	nginxTerminateWait = 3 * time.Second
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
	workDirs := []string{
		"logs",
		"client_body_temp",
		"proxy_temp",
		"fastcgi_temp",
		"uwsgi_temp",
		"scgi_temp",
		filepath.Join("temp", "client_body_temp"),
		filepath.Join("temp", "proxy_temp"),
		filepath.Join("temp", "fastcgi_temp"),
		filepath.Join("temp", "uwsgi_temp"),
		filepath.Join("temp", "scgi_temp"),
	}
	for _, name := range workDirs {
		if err := os.MkdirAll(filepath.Join(m.cfg.NginxDir(), name), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) nginxGlobalArgs() []string {
	prefix := absNginxPath(m.cfg.NginxDir())
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	errorLog := absNginxPath(filepath.Join(m.cfg.LogsDir(), "error.log"))
	return []string{"-p", prefix, "-e", errorLog}
}

func (m *Manager) available() bool {
	_, err := exec.LookPath(m.cfg.NginxBin)
	return err == nil
}

func (m *Manager) Apply(ctx context.Context, rules []proxy.Rule, certs []CertSource) (ApplyResult, error) {
	if err := m.EnsureDirs(); err != nil {
		return ApplyResult{}, err
	}

	content, err := Generate(m.cfg, rules, certs)
	if err != nil {
		return ApplyResult{}, err
	}

	tmpPath := m.cfg.NginxConfigPath() + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return ApplyResult{}, err
	}

	if m.available() {
		if err := m.validate(ctx, tmpPath); err != nil {
			_ = os.Remove(tmpPath)
			return ApplyResult{}, err
		}
	} else {
		m.logger.Warn("nginx binary not found, skipping validation", "bin", m.cfg.NginxBin)
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

	if !m.available() {
		return ApplyResult{Message: "Nginx 未安装，配置已保存"}, nil
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
		m.logger.Warn("nginx reload failed, trying restart", "error", err)
		if err := m.forceRestart(ctx); err != nil {
			return ApplyResult{}, err
		}
		return ApplyResult{Reloaded: true, Message: "Nginx 已重新启动"}, nil
	}
	return ApplyResult{Reloaded: true, Message: "Nginx 已重载"}, nil
}

func (m *Manager) ValidateOnly(ctx context.Context, rules []proxy.Rule, certs []CertSource) error {
	content, err := Generate(m.cfg, rules, certs)
	if err != nil {
		return err
	}
	if !m.available() {
		m.logger.Warn("nginx binary not found, skipping validation", "bin", m.cfg.NginxBin)
		return nil
	}
	tmpPath := filepath.Join(m.cfg.NginxDir(), "validate.tmp.conf")
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	return m.validate(ctx, tmpPath)
}

func (m *Manager) validate(ctx context.Context, configPath string) error {
	tmpErr := filepath.Join(m.cfg.NginxDir(), "validate.error.log")
	args := []string{"-e", absNginxPath(tmpErr), "-t", "-c", configPath}
	err := m.runNginx(ctx, args, "Nginx 配置校验失败")
	_ = os.Remove(tmpErr)
	return err
}

func (m *Manager) start(ctx context.Context) error {
	running, err := m.isRunning()
	if err != nil {
		return err
	}
	if running {
		return m.reload(ctx)
	}
	if m.trySilentReload(ctx) {
		m.logger.Info("nginx reloaded existing master")
		return nil
	}
	prepareStartPlatform(m, ctx)
	if err := m.runNginxStart(ctx, []string{"-c", m.cfg.NginxConfigPath()}, "Nginx 启动失败"); err != nil {
		if strings.Contains(err.Error(), "conflicting server name") && m.trySilentReload(ctx) {
			m.logger.Warn("nginx start skipped duplicate master, reloaded instead")
			return nil
		}
		return err
	}
	m.logger.Info("nginx started")
	return nil
}

func (m *Manager) trySilentReload(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, nginxCmdTimeout)
	defer cancel()

	fullArgs := append(m.nginxGlobalArgs(), "-c", m.cfg.NginxConfigPath(), "-s", "reload")
	cmd := exec.CommandContext(ctx, m.cfg.NginxBin, fullArgs...)
	return cmd.Run() == nil
}

func (m *Manager) reload(ctx context.Context) error {
	pid, err := readPIDFile(m.cfg.NginxPIDFile)
	if err == nil && isPIDAlive(pid) {
		if err := reloadProcess(pid); err == nil {
			m.logger.Info("nginx reloaded")
			return nil
		}
		m.logger.Warn("nginx signal reload failed, falling back to cli", "pid", pid, "error", err)
	} else if err != nil && !os.IsNotExist(err) {
		removePIDFile(m.cfg.NginxPIDFile)
	}

	if err := m.runNginx(ctx, []string{"-c", m.cfg.NginxConfigPath(), "-s", "reload"}, "Nginx 重载失败"); err != nil {
		return err
	}
	m.logger.Info("nginx reloaded")
	return nil
}

func (m *Manager) forceRestart(ctx context.Context) error {
	m.terminateMaster()
	removePIDFile(m.cfg.NginxPIDFile)
	prepareStartPlatform(m, ctx)
	return m.start(ctx)
}

func (m *Manager) terminateMaster() {
	pid, err := readPIDFile(m.cfg.NginxPIDFile)
	if err != nil || !isPIDAlive(pid) {
		return
	}
	_ = terminateProcess(pid, syscall.SIGTERM)
	waitProcessExit(pid, nginxTerminateWait)
	if isPIDAlive(pid) {
		_ = terminateProcess(pid, syscall.SIGKILL)
		waitProcessExit(pid, time.Second)
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	if !m.isRunningQuick() {
		return nil
	}
	if err := m.runNginx(ctx, []string{"-c", m.cfg.NginxConfigPath(), "-s", "quit"}, "Nginx 停止失败"); err != nil {
		return err
	}
	m.logger.Info("nginx stopped")
	return nil
}

func (m *Manager) runNginx(ctx context.Context, args []string, prefix string) error {
	ctx, cancel := context.WithTimeout(ctx, nginxCmdTimeout)
	defer cancel()

	fullArgs := append(m.nginxGlobalArgs(), args...)
	cmd := exec.CommandContext(ctx, m.cfg.NginxBin, fullArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if ctx.Err() == context.DeadlineExceeded {
			msg = "操作超时"
		}
		m.logger.Error("nginx command failed", "args", strings.Join(fullArgs, " "), "error", msg)
		return fmt.Errorf("%s：%s", prefix, sanitizeNginxError(msg))
	}
	return nil
}

// runNginxStart launches nginx without waiting for the master process to exit.
// On Windows the master stays in the foreground and would block cmd.Run().
func (m *Manager) runNginxStart(ctx context.Context, args []string, prefix string) error {
	fullArgs := append(m.nginxGlobalArgs(), args...)
	cmd := exec.Command(m.cfg.NginxBin, fullArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s：%s", prefix, err.Error())
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	deadline := time.Now().Add(nginxStartWait)
	for time.Now().Before(deadline) {
		if running, err := m.isRunning(); err == nil && running {
			return nil
		}
		select {
		case err := <-waitDone:
			if running, checkErr := m.isRunning(); checkErr == nil && running {
				return nil
			}
			msg := strings.TrimSpace(stderr.String())
			if msg == "" && err != nil {
				msg = err.Error()
			}
			if msg == "" {
				msg = "进程已退出"
			}
			m.logger.Error("nginx start failed", "args", strings.Join(fullArgs, " "), "error", msg)
			return fmt.Errorf("%s：%s", prefix, sanitizeNginxError(msg))
		default:
		}
		if ctx.Err() != nil {
			return fmt.Errorf("%s：%s", prefix, "操作超时")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if running, err := m.isRunning(); err == nil && running {
		return nil
	}
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = "未检测到运行中的进程"
	}
	return fmt.Errorf("%s：%s", prefix, sanitizeNginxError(msg))
}

func (m *Manager) isRunning() (bool, error) {
	pid, err := readPIDFile(m.cfg.NginxPIDFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		removePIDFile(m.cfg.NginxPIDFile)
		return false, nil
	}
	if !isPIDAlive(pid) {
		removePIDFile(m.cfg.NginxPIDFile)
		return false, nil
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
