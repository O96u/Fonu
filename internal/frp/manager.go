package frp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
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
	mu sync.Mutex
	version string
	versionChecked time.Time
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
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.applyLocked(ctx, in)
}

func (m *Manager) applyLocked(ctx context.Context, in SaveInput) (result ApplyResult, err error) {
	existing, err := m.store.Load(ctx)
	if err != nil { return result, err }
	snapshot, err := m.store.snapshot(ctx)
	if err != nil { return result, err }
	path := m.cfg.FrpConfigPath()
	old, readErr := os.ReadFile(path)
	if readErr != nil && !os.IsNotExist(readErr) { return result, readErr }
	hadFile := readErr == nil
	wasRunning := m.isRunningQuick()
	if wasRunning && !hadFile { return result, fmt.Errorf("运行配置丢失，无法安全应用") }
	keepToken := existing.HasAuthToken && (strings.TrimSpace(in.AuthToken) == "" || in.AuthToken == maskedToken)
	if err = m.store.Save(ctx, in, keepToken); err != nil { return result, err }

	fileChanged, processChanged := false, false
	defer func() {
		if err == nil { return }
		recovery, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		defer cancel()
		var restoreErr error
		if processChanged { restoreErr = errors.Join(restoreErr, m.stopLocked(recovery)) }
		if fileChanged {
			if hadFile { restoreErr = errors.Join(restoreErr, writePrivateConfig(path, old)) } else if e := os.Remove(path); e != nil && !os.IsNotExist(e) { restoreErr = errors.Join(restoreErr, e) }
		}
		restoreErr = errors.Join(restoreErr, m.store.restore(recovery, snapshot))
		if processChanged && wasRunning && restoreErr == nil {
			if e := m.start(recovery); e != nil { restoreErr = errors.Join(restoreErr, e) } else { m.markStarted(recovery) }
		}
		if restoreErr != nil { err = fmt.Errorf("%w；回滚未完全成功：%v", err, restoreErr) }
		_ = m.store.SetLastError(recovery, err.Error())
	}()

	cfg, err := m.store.Load(ctx)
	if err != nil { return result, err }
	if !cfg.Enabled || !hasActiveRoutes(cfg) {
		processChanged = wasRunning
		if err = m.stopLocked(ctx); err != nil { return result, err }
		m.clearStarted(ctx)
		_ = m.store.SetLastError(ctx, "")
		if !cfg.Enabled { return ApplyResult{Message: "FRP 已停用"}, nil }
		return ApplyResult{Message: "配置已保存，请添加穿透域名或 TCP/UDP 隧道后再连接"}, nil
	}
	runtimeCfg, token, err := m.store.LoadRuntime(ctx)
	if err != nil { return result, err }
	if err = m.EnsureDirs(); err != nil { return result, err }
	content, err := Generate(m.cfg, runtimeCfg, token, m.cfg.NginxDefaultHTTPPort, m.cfg.NginxDefaultHTTPSPort)
	if err != nil { return result, err }
	if hadFile && bytes.Equal(old, []byte(content)) && wasRunning {
		if err = os.Chmod(path, 0o600); err != nil { return result, err }
		_ = m.store.SetLastError(ctx, "")
		return ApplyResult{Message: "配置未变化，服务保持运行"}, nil
	}
	tmp, err := writePrivateTemp(path, []byte(content))
	if err != nil { return result, err }
	defer os.Remove(tmp)
	if m.available() {
		if err = m.runFrpc(ctx, []string{"verify", "-c", tmp}, "FRP 配置校验失败"); err != nil { return result, err }
	}
	if err = os.Rename(tmp, path); err != nil { return result, err }
	fileChanged = true
	if !m.available() {
		msg := "frpc 未安装，配置已保存"
		_ = m.store.SetLastError(ctx, msg)
		return ApplyResult{Message: msg}, nil
	}
	// Global connection parameters cannot be safely hot-reloaded. Restart
	// only when configuration changes, and restore the prior process on error.
	processChanged = true
	if err = m.forceRestart(ctx); err != nil { return result, err }
	m.markStarted(ctx)
	_ = m.store.SetLastError(ctx, "")
	return ApplyResult{Message: "配置已应用，FRP 正在连接"}, nil
}

func writePrivateTemp(path string, content []byte) (string, error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "frpc-*.toml")
	if err != nil { return "", err }
	name := tmp.Name()
	if _, err = tmp.Write(content); err == nil { err = tmp.Sync() }
	closeErr := tmp.Close()
	if err == nil { err = closeErr }
	if err != nil { _ = os.Remove(name); return "", err }
	return name, nil
}

func writePrivateConfig(path string, content []byte) error {
	tmp, err := writePrivateTemp(path, content)
	if err != nil { return err }
	defer os.Remove(tmp)
	return os.Rename(tmp, path)
}

// Restart operates on the saved configuration, never on browser drafts.
func (m *Manager) Restart(ctx context.Context) (ApplyResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, err := m.store.Load(ctx)
	if err != nil { return ApplyResult{}, err }
	if !cfg.Enabled || !hasActiveRoutes(cfg) { return ApplyResult{}, fmt.Errorf("请先启用 FRP 并配置穿透规则") }
	if !m.available() { return ApplyResult{}, fmt.Errorf("frpc 未安装") }
	if err := m.verifyConfig(ctx); err != nil { return ApplyResult{}, err }
	if err := m.forceRestart(ctx); err != nil { _ = m.store.SetLastError(ctx, err.Error()); return ApplyResult{}, err }
	m.markStarted(ctx)
	_ = m.store.SetLastError(ctx, "")
	return ApplyResult{Message: "FRP 已重启，正在连接"}, nil
}

func (m *Manager) Bootstrap(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, err := m.store.Load(ctx)
	if err != nil || !cfg.Enabled {
		return nil
	}
	if !m.available() {
		m.logger.Warn("frpc binary not found, skipping bootstrap", "bin", m.cfg.FrpcBin)
		return nil
	}
	_, err = m.applyLocked(ctx, saveInputFromConfig(cfg))
	if err != nil {
		m.logger.Warn("initial frpc apply skipped", "error", err.Error())
	}
	return nil
}

func (m *Manager) Load(ctx context.Context) (Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.Load(ctx)
}

func (m *Manager) Status(ctx context.Context) Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, err := m.store.Load(ctx)
	if err != nil {
		return Status{Message: "读取配置失败"}
	}
	if time.Since(m.versionChecked) > time.Minute {
		m.version = ResolveClientVersion(m.cfg.FrpcBin)
		m.versionChecked = time.Now()
	}
	return BuildStatus(
		m.cfg,
		cfg,
		cfg.Enabled,
		m.store.LastError(ctx),
		m.store.StartedAt(ctx),
		m.version,
	)
}

func (m *Manager) SyncDomains(ctx context.Context, domains []string) (ApplyResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, err := m.store.Load(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	merged := MergeDomains(cfg.CustomDomains, domains)
	in := saveInputFromConfig(cfg)
	in.CustomDomains = merged
	return m.applyLocked(ctx, in)
}

func saveInputFromConfig(cfg Config) SaveInput {
	return SaveInput{
		Enabled:       cfg.Enabled,
		ServerAddr:    cfg.ServerAddr,
		ServerPort:    cfg.ServerPort,
		AuthToken:     maskedToken,
		TLSEnabled:    cfg.TLSEnabled,
		CustomDomains: cfg.CustomDomains,
		TCPProxies:    cfg.TCPProxies,
	}
}

func (m *Manager) Logs(ctx context.Context, limit int) ([]string, error) {
	return ReadLogTail(m.cfg.FrpLogPath(), limit)
}

func (m *Manager) FRPSConfig(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	return BuildFRPSConfig(port, m.cfg.NginxDefaultHTTPPort, m.cfg.NginxDefaultHTTPSPort, token, tcpRemotePorts(cfg.TCPProxies)), nil
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
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked(ctx)
}

func (m *Manager) stopLocked(ctx context.Context) error {
	if !m.isRunningQuick() {
		removePIDFile(m.cfg.FrpPIDFile)
		m.clearStarted(ctx)
		return nil
	}
	pid, err := readPIDFile(m.cfg.FrpPIDFile)
	if err != nil { return err }
	if isOwnedProcess(m.cfg, pid) {
		if err := terminateProcess(pid, syscall.SIGTERM); err != nil && isOwnedProcess(m.cfg, pid) { return err }
	}
	deadline := time.Now().Add(frpcTerminateWait)
	for isOwnedProcess(m.cfg, pid) && time.Now().Before(deadline) { time.Sleep(100*time.Millisecond) }
	if isOwnedProcess(m.cfg, pid) {
		if err := terminateProcess(pid, syscall.SIGKILL); err != nil && isOwnedProcess(m.cfg, pid) { return err }
		deadline = time.Now().Add(time.Second)
		for isOwnedProcess(m.cfg, pid) && time.Now().Before(deadline) { time.Sleep(100*time.Millisecond) }
	}
	if isOwnedProcess(m.cfg, pid) { return fmt.Errorf("frpc 进程未退出") }
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
	if err := m.stopLocked(ctx); err != nil { return err }
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

const sessionMarker = "[FONU] frpc session started"

func (m *Manager) runFrpcStart(ctx context.Context, args []string, prefix string) error {
	if err := ctx.Err(); err != nil { return err }
	if err := os.MkdirAll(filepath.Dir(m.cfg.FrpLogPath()), 0o755); err != nil { return err }
	logFile, err := os.OpenFile(m.cfg.FrpLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil { return err }
	_, err = fmt.Fprintln(logFile, sessionMarker)
	_ = logFile.Close()
	if err != nil { return err }
	cmd := exec.Command(m.cfg.FrpcBin, args...)
	// frpc writes its own rotating log. Never retain unbounded stdout buffers.
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil { return fmt.Errorf("%s：%w", prefix, err) }
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	if err := writePIDFile(m.cfg.FrpPIDFile, cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case err := <-waitDone:
		removePIDFile(m.cfg.FrpPIDFile)
		return fmt.Errorf("%s：%s", prefix, frpcCommandError("", "", err))
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		removePIDFile(m.cfg.FrpPIDFile)
		return ctx.Err()
	case <-timer.C:
		if !m.isRunningQuick() { return fmt.Errorf("%s：进程未保持运行", prefix) }
		return nil
	}
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
	if !isOwnedProcess(m.cfg, pid) {
		removePIDFile(m.cfg.FrpPIDFile)
		return false, nil
	}
	return true, nil
}

// Verify the executable and config argument before trusting a recycled PID.
func isOwnedProcess(cfg config.Config, pid int) bool {
	if pid <= 1 || !nginx.IsPIDAlive(pid) { return false }
	bin, err := exec.LookPath(cfg.FrpcBin)
	if err != nil { return false }
	expected, err := os.Stat(bin)
	if err != nil { return false }
	actual, err := os.Stat(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil || !os.SameFile(expected, actual) { return false }
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil { return false }
	args := strings.Split(string(raw), "\x00")
	for i := 1; i+1 < len(args); i++ {
		if (args[i] == "-c" || args[i] == "--config") && filepath.Clean(args[i+1]) == filepath.Clean(cfg.FrpConfigPath()) { return true }
	}
	return false
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
