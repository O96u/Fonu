package frp

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
	"github.com/fonu/fonu/internal/nginx"
)

const (
	frpcCmdTimeout    = 8 * time.Second
	frpcStartWait     = 5 * time.Second
	frpcTerminateWait = 3 * time.Second
)

type Manager struct {
	cfg    config.Config
	store  *Store
	logger *slog.Logger
}

type ApplyResult struct {
	Message string
}

func NewManager(cfg config.Config, store *Store, logger *slog.Logger) *Manager {
	return &Manager{cfg: cfg, store: store, logger: logger.With("module", "FRP")}
}

func (m *Manager) EnsureDirs() error {
	return os.MkdirAll(m.cfg.FrpDir(), 0o755)
}

func (m *Manager) Apply(ctx context.Context, in SaveInput) (ApplyResult, error) {
	existing, err := m.store.Load(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	keepToken := existing.HasAuthToken && (strings.TrimSpace(in.AuthToken) == "" || in.AuthToken == maskedToken)
	if err := m.store.Save(ctx, in, keepToken); err != nil {
		return ApplyResult{}, err
	}

	cfg, err := m.store.Load(ctx)
	if err != nil {
		return ApplyResult{}, err
	}

	if !cfg.Enabled {
		if err := m.Stop(ctx); err != nil {
			return ApplyResult{}, err
		}
		m.clearStarted(ctx)
		_ = m.store.SetLastError(ctx, "")
		return ApplyResult{Message: "FRP 已停用"}, nil
	}

	runtimeCfg, token, err := m.store.LoadRuntime(ctx)
	if err != nil {
		_ = m.store.SetLastError(ctx, err.Error())
		return ApplyResult{}, err
	}

	if err := m.EnsureDirs(); err != nil {
		return ApplyResult{}, err
	}

	content, err := Generate(m.cfg, runtimeCfg, token, m.cfg.NginxDefaultHTTPPort, m.cfg.NginxDefaultHTTPSPort)
	if err != nil {
		_ = m.store.SetLastError(ctx, err.Error())
		return ApplyResult{}, err
	}

	tmpPath := m.cfg.FrpConfigPath() + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return ApplyResult{}, err
	}
	if err := os.Rename(tmpPath, m.cfg.FrpConfigPath()); err != nil {
		_ = os.Remove(tmpPath)
		return ApplyResult{}, err
	}

	if err := m.verifyConfig(ctx); err != nil {
		_ = m.store.SetLastError(ctx, err.Error())
		return ApplyResult{}, err
	}

	if !m.available() {
		msg := "frpc 未安装，配置已保存"
		_ = m.store.SetLastError(ctx, msg)
		return ApplyResult{Message: msg}, nil
	}

	running, _ := m.isRunning()
	if !running {
		if err := m.start(ctx); err != nil {
			_ = m.store.SetLastError(ctx, err.Error())
			return ApplyResult{}, err
		}
		m.markStarted(ctx)
		_ = m.store.SetLastError(ctx, "")
		return ApplyResult{Message: "FRP 已启动"}, nil
	}

	if err := m.reload(ctx); err != nil {
		m.logger.Warn("frpc reload failed, trying restart", "error", err)
		if err := m.forceRestart(ctx); err != nil {
			_ = m.store.SetLastError(ctx, err.Error())
			return ApplyResult{}, err
		}
		m.markStarted(ctx)
		_ = m.store.SetLastError(ctx, "")
		return ApplyResult{Message: "FRP 已重新启动"}, nil
	}
	_ = m.store.SetLastError(ctx, "")
	return ApplyResult{Message: "FRP 已重载"}, nil
}

func (m *Manager) Bootstrap(ctx context.Context) error {
	cfg, err := m.store.Load(ctx)
	if err != nil || !cfg.Enabled {
		return nil
	}
	if !m.available() {
		m.logger.Warn("frpc binary not found, skipping bootstrap", "bin", m.cfg.FrpcBin)
		return nil
	}
	_, err = m.Apply(ctx, SaveInput{
		Enabled:       cfg.Enabled,
		ServerAddr:    cfg.ServerAddr,
		ServerPort:    cfg.ServerPort,
		AuthToken:     maskedToken,
		TLSEnabled:    cfg.TLSEnabled,
		CustomDomains: cfg.CustomDomains,
	})
	if err != nil {
		m.logger.Warn("initial frpc apply skipped", "error", err.Error())
	}
	return nil
}

func (m *Manager) Load(ctx context.Context) (Config, error) {
	return m.store.Load(ctx)
}

func (m *Manager) Status(ctx context.Context) Status {
	cfg, err := m.store.Load(ctx)
	if err != nil {
		return Status{Message: "读取配置失败"}
	}
	return BuildStatus(
		m.cfg,
		cfg,
		cfg.Enabled,
		m.store.LastError(ctx),
		m.store.StartedAt(ctx),
		ResolveClientVersion(m.cfg.FrpcBin),
	)
}

func (m *Manager) SyncDomains(ctx context.Context, domains []string) (ApplyResult, error) {
	cfg, err := m.store.Load(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	merged := MergeDomains(cfg.CustomDomains, domains)
	return m.Apply(ctx, SaveInput{
		Enabled:       cfg.Enabled,
		ServerAddr:    cfg.ServerAddr,
		ServerPort:    cfg.ServerPort,
		AuthToken:     maskedToken,
		TLSEnabled:    cfg.TLSEnabled,
		CustomDomains: merged,
	})
}

func (m *Manager) Logs(ctx context.Context, limit int) ([]string, error) {
	return ReadLogTail(m.cfg.FrpLogPath(), limit)
}

func (m *Manager) FRPSConfig(ctx context.Context) (string, error) {
	cfg, err := m.store.Load(ctx)
	if err != nil {
		return "", err
	}
	_, token, err := m.store.LoadRuntime(ctx)
	if err != nil {
		return "", err
	}
	port := cfg.ServerPort
	if port <= 0 {
		port = 7000
	}
	return BuildFRPSConfig(port, m.cfg.NginxDefaultHTTPPort, m.cfg.NginxDefaultHTTPSPort, token), nil
}

func (m *Manager) NginxHTTPPort() int {
	return m.cfg.NginxDefaultHTTPPort
}

func (m *Manager) NginxHTTPSPort() int {
	return m.cfg.NginxDefaultHTTPSPort
}

func (m *Manager) verifyConfig(ctx context.Context) error {
	if !m.available() {
		return nil
	}
	return m.runFrpc(ctx, []string{"verify", "-c", m.cfg.FrpConfigPath()}, "FRP 配置校验失败")
}

func (m *Manager) markStarted(ctx context.Context) {
	_ = m.store.SetStartedAt(ctx, time.Now().UTC().Format(time.RFC3339))
}

func (m *Manager) clearStarted(ctx context.Context) {
	_ = m.store.SetStartedAt(ctx, "")
}

func (m *Manager) Stop(ctx context.Context) error {
	if !m.isRunningQuick() {
		removePIDFile(m.cfg.FrpPIDFile)
		return nil
	}
	pid, err := readPIDFile(m.cfg.FrpPIDFile)
	if err != nil {
		removePIDFile(m.cfg.FrpPIDFile)
		return nil
	}
	_ = terminateProcess(pid, syscall.SIGTERM)
	waitProcessExit(pid, frpcTerminateWait)
	if nginx.IsPIDAlive(pid) {
		_ = terminateProcess(pid, syscall.SIGKILL)
		waitProcessExit(pid, time.Second)
	}
	removePIDFile(m.cfg.FrpPIDFile)
	m.clearStarted(ctx)
	m.logger.Info("frpc stopped")
	return nil
}

func (m *Manager) available() bool {
	_, err := exec.LookPath(m.cfg.FrpcBin)
	return err == nil
}

func (m *Manager) start(ctx context.Context) error {
	if m.isRunningQuick() {
		return m.reload(ctx)
	}
	args := []string{"-c", m.cfg.FrpConfigPath()}
	if err := m.runFrpcStart(ctx, args, "FRP 启动失败"); err != nil {
		return err
	}
	m.logger.Info("frpc started")
	return nil
}

func (m *Manager) reload(ctx context.Context) error {
	args := []string{"reload", "-c", m.cfg.FrpConfigPath()}
	if err := m.runFrpc(ctx, args, "FRP 重载失败"); err != nil {
		return err
	}
	m.logger.Info("frpc reloaded")
	return nil
}

func (m *Manager) forceRestart(ctx context.Context) error {
	_ = m.Stop(ctx)
	return m.start(ctx)
}

func (m *Manager) runFrpc(ctx context.Context, args []string, prefix string) error {
	ctx, cancel := context.WithTimeout(ctx, frpcCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, m.cfg.FrpcBin, args...)
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		msg := frpcCommandError(stdout.String(), stderr.String(), err)
		if ctx.Err() == context.DeadlineExceeded {
			msg = "操作超时"
		}
		return fmt.Errorf("%s：%s", prefix, msg)
	}
	return nil
}

func (m *Manager) runFrpcStart(ctx context.Context, args []string, prefix string) error {
	cmd := exec.Command(m.cfg.FrpcBin, args...)
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s：%s", prefix, err.Error())
	}
	if cmd.Process != nil {
		if err := writePIDFile(m.cfg.FrpPIDFile, cmd.Process.Pid); err != nil {
			_ = cmd.Process.Kill()
			return fmt.Errorf("%s：%s", prefix, err.Error())
		}
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	deadline := time.Now().Add(frpcStartWait)
	for time.Now().Before(deadline) {
		if running, _ := m.isRunning(); running {
			return nil
		}
		select {
		case err := <-waitDone:
			if running, _ := m.isRunning(); running {
				return nil
			}
			msg := frpcCommandError(stdout.String(), stderr.String(), err)
			m.logger.Warn("frpc exited during start", "error", msg)
			return fmt.Errorf("%s：%s", prefix, msg)
		default:
		}
		if ctx.Err() != nil {
			return fmt.Errorf("%s：%s", prefix, "操作超时")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if running, _ := m.isRunning(); running {
		return nil
	}
	msg := frpcCommandError(stdout.String(), stderr.String(), nil)
	if msg == "" {
		msg = "未检测到运行中的进程"
	}
	return fmt.Errorf("%s：%s", prefix, msg)
}

func frpcCommandError(stdout, stderr string, err error) string {
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = strings.TrimSpace(stdout)
	}
	if msg == "" && err != nil {
		msg = err.Error()
	}
	if msg == "" {
		msg = "进程已退出"
	}
	return msg
}

func writePIDFile(path string, pid int) error {
	if pid <= 0 {
		return fmt.Errorf("无效的 frpc 进程 ID")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0o644)
}

func (m *Manager) isRunning() (bool, error) {
	pid, err := readPIDFile(m.cfg.FrpPIDFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		removePIDFile(m.cfg.FrpPIDFile)
		return false, nil
	}
	if !nginx.IsPIDAlive(pid) {
		removePIDFile(m.cfg.FrpPIDFile)
		return false, nil
	}
	return true, nil
}

func (m *Manager) isRunningQuick() bool {
	ok, _ := m.isRunning()
	return ok
}

func readPIDFile(path string) (int, error) {
	return nginx.ReadPID(path)
}

func removePIDFile(path string) {
	_ = os.Remove(path)
}

func waitProcessExit(pid int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !nginx.IsPIDAlive(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func terminateProcess(pid int, sig syscall.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(sig)
}

func FrpConfigDir(cfg config.Config) string {
	return filepath.Clean(cfg.FrpDir())
}
