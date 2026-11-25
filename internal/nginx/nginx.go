package nginx

import (
	"bytes"
	"context"
		"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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
	opts   GenerateOptions
	mu sync.Mutex
	optsMu sync.RWMutex
	statusMu sync.RWMutex
	status ApplyStatus
	confirmedMaster int
	confirmedWorkers map[string]bool
}

type ApplyResult struct {
	Reloaded bool
	Confirmed bool
	Unchanged bool
	Message  string
}

func NewManager(cfg config.Config, logger *slog.Logger) *Manager {
	return &Manager{cfg: cfg, logger: logger}
}

func (m *Manager) SetGenerateOptions(opts GenerateOptions) {
	m.optsMu.Lock()
	defer m.optsMu.Unlock()
	m.opts = opts
}

func (m *Manager) CurrentOptions() GenerateOptions {
	m.optsMu.RLock()
	defer m.optsMu.RUnlock()
	return m.opts
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
	return BinAvailable(m.cfg.NginxBin)
}

func (m *Manager) Apply(ctx context.Context, rules []proxy.Rule, certs []CertSource) (ApplyResult, error) {
	return m.ApplyWithOptions(ctx, rules, certs, m.CurrentOptions())
}

func (m *Manager) ApplyWithOptions(ctx context.Context, rules []proxy.Rule, certs []CertSource, opts GenerateOptions) (result ApplyResult, applyErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusMu.Lock()
	m.status.State = "applying"
	m.status.LastError = ""
	m.status.Desired = ""
	m.statusMu.Unlock()
	defer func() {
		m.statusMu.Lock()
		m.status.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if applyErr != nil {
			m.status.State = "failed"; m.status.LastError = applyErr.Error()
			m.confirmedMaster = 0
			m.confirmedWorkers = nil
			m.status.Confirmed = ""
		} else if result.Confirmed {
			m.status.State = "confirmed"; m.status.Applied = m.status.Desired; m.status.Confirmed = m.status.Desired
		} else if result.Reloaded {
			m.status.State = "reload_requested"; m.status.Applied = m.status.Desired; m.status.Confirmed = ""
		} else { m.status.State = "saved"; m.status.Confirmed = "" }
		status := m.status
		m.statusMu.Unlock()
		if err := m.appendApplyHistory(status, result.Unchanged); err != nil {
			m.logger.Warn("persist nginx apply history", "error",err)
		}
	}()
	if err := ctx.Err(); err != nil { return ApplyResult{}, err }
	if err := m.EnsureDirs(); err != nil {
		return ApplyResult{}, err
	}

	if err := EnsureErrorPages(m.cfg); err != nil {
		return ApplyResult{}, err
	}
	if err := EnsureCustomDirs(m.cfg); err != nil {
		return ApplyResult{}, err
	}
	authSnapshot, err := snapshotHtpasswd(m.cfg)
	if err != nil { return ApplyResult{}, err }
	authRestored := false
	restoreAuth := func() error {
		if authRestored { return nil }
		err := restoreHtpasswd(m.cfg, authSnapshot)
		if err == nil { authRestored = true }
		return err
	}
	defer func() {
		if applyErr != nil {
			if err := restoreAuth(); err != nil {
				applyErr = errors.Join(applyErr, fmt.Errorf("restore htpasswd: %w", err))
			}
		}
	}()
	if err := SyncHtpasswdFiles(m.cfg, rules); err != nil {
		return ApplyResult{}, err
	}

		// Path override is only for pre-rename validation during China CIDR updates.
	opts.ChinaCIDRPathOverride = ""
	opts.ChinaCIDRAvailable = chinaCIDRExists(m.cfg)

	content, err := Generate(m.cfg, rules, certs, opts)
	if err != nil {
		return ApplyResult{}, err
	}

	revision, skippable := configRevision(m.cfg, content, rules, certs, opts)
	m.statusMu.Lock()
	m.status.Desired = revision
	confirmed := m.status.Confirmed
	m.statusMu.Unlock()
	if skippable && confirmed == revision && m.confirmedMaster != 0 && m.ownsMaster(m.confirmedMaster) {
		if disk, err := os.ReadFile(m.cfg.NginxConfigPath()); err == nil && string(disk) == content &&
			workersOverlap(m.confirmedWorkers, activeWorkers(m.confirmedMaster)) {
			return ApplyResult{Confirmed:true,Unchanged:true,Message:"配置未变化，无需重载"},nil
		}
	}
	tmpPath, err := writeTemporaryConfig(m.cfg.NginxDir(), content)
	if err != nil { return ApplyResult{}, err }
	defer os.Remove(tmpPath)

	if err := ValidateConfigSyntax(content); err != nil {
		_ = os.Remove(tmpPath)
		return ApplyResult{}, err
	}
	if m.available() {
		if err := m.validate(ctx, tmpPath); err != nil {
			_ = os.Remove(tmpPath)
			return ApplyResult{}, err
		}
	} else {
		m.logger.Warn("nginx binary not found, applied syntax-only validation", "bin", m.cfg.NginxBin)
	}

	currentPath := m.cfg.NginxConfigPath()
	backupPath := currentPath + ".bak"
	hadPrevious := false
	if _, err := os.Stat(currentPath); err == nil {
		hadPrevious = true
		if err := copyFile(currentPath, backupPath); err != nil {
			_ = os.Remove(tmpPath)
			return ApplyResult{}, err
		}
	} else if !os.IsNotExist(err) { return ApplyResult{}, err }

	if err := os.Rename(tmpPath, currentPath); err != nil {
		_ = os.Remove(tmpPath)
		return ApplyResult{}, err
	}

	defer func() {
		if applyErr == nil { return }
		if err := restoreConfig(currentPath, backupPath, hadPrevious); err != nil {
			applyErr = errors.Join(applyErr, fmt.Errorf("restore nginx config: %w", err))
			return
		}
		if err := restoreAuth(); err != nil {
			applyErr = errors.Join(applyErr, fmt.Errorf("restore htpasswd: %w", err))
		}
		if !hadPrevious && m.available() { m.terminateMaster() }
		if hadPrevious && m.available() {
			rollbackCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := m.start(rollbackCtx); err != nil { applyErr = errors.Join(applyErr, fmt.Errorf("restore nginx service: %w", err)) }
		}
	}()
	if !m.available() {
		return ApplyResult{Message: "Nginx 未安装，配置已保存"}, nil
	}

	running, err := m.isRunning()
	if err != nil {
		return ApplyResult{}, err
	}

	var previous map[string]bool
	if running {
		pid, _ := readPIDFile(m.cfg.NginxPIDFile)
		previous = activeWorkers(pid)
		if workerConfirmationSupported() && len(previous)==0 {
			return ApplyResult{},fmt.Errorf("无法识别当前 Nginx worker，拒绝未确认的重载")
		}
		if err := m.reload(ctx); err != nil {
			m.logger.Warn("nginx reload failed; restoring previous config", "error", err)
			return ApplyResult{}, err
		}
	} else if err := m.start(ctx); err != nil { return ApplyResult{}, err }
	if workerConfirmationSupported() {
		pid, workers, err := m.waitForWorkers(ctx,previous)
		if err != nil { return ApplyResult{},err }
		m.confirmedMaster, m.confirmedWorkers = pid,workers
		return ApplyResult{Reloaded:true,Confirmed:true,Message:"Nginx 新 worker 已就绪"},nil
	}
	return ApplyResult{Reloaded:true,Message:"已请求 Nginx 重载，尚未确认 worker"},nil
}

func (m *Manager) ValidateOnly(ctx context.Context, rules []proxy.Rule, certs []CertSource) error {
	return m.ValidateOnlyWithOptions(ctx, rules, certs, m.CurrentOptions())
}

func (m *Manager) ValidateOnlyWithOptions(ctx context.Context, rules []proxy.Rule, certs []CertSource, opts GenerateOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if opts.ChinaCIDRPathOverride != "" {
		opts.ChinaCIDRAvailable = true
	} else {
		opts.ChinaCIDRAvailable = chinaCIDRExists(m.cfg)
	}
	content, err := Generate(m.cfg, rules, certs, opts)
	if err != nil {
		return err
	}
	if err := ValidateConfigSyntax(content); err != nil {
		return err
	}
	if !m.available() {
		m.logger.Warn("nginx binary not found, syntax-only validation", "bin", m.cfg.NginxBin)
		return nil
	}
	tmpPath, err := writeTemporaryConfig(m.cfg.NginxDir(), content)
	if err != nil { return err }
	defer os.Remove(tmpPath)
	return m.validate(ctx, tmpPath)
}

func (m *Manager) validate(ctx context.Context, configPath string) error {
	f, err := os.CreateTemp(m.cfg.NginxDir(), "validate-*.log")
	if err != nil { return err }
	tmpErr := f.Name()
	if err := f.Close(); err != nil { os.Remove(tmpErr); return err }
	defer os.Remove(tmpErr)
	args := []string{"-e", absNginxPath(tmpErr), "-t", "-c", configPath}
	err = m.runNginx(ctx, args, "Nginx 配置校验失败")
	if err != nil { return err }
	data, err := os.ReadFile(tmpErr)
	if err != nil { return err }
	for _, line := range strings.Split(string(data),"\n") {
		if strings.Contains(line,"conflicting server name") {
			return fmt.Errorf("Nginx 域名与监听冲突：%s",sanitizeNginxError(line))
		}
	}
	return nil
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
	pid, err := readPIDFile(m.cfg.NginxPIDFile)
	if err != nil || !m.ownsMaster(pid) { return false }
	ctx, cancel := context.WithTimeout(ctx, nginxCmdTimeout)
	defer cancel()

	fullArgs := append(m.nginxGlobalArgs(), "-c", m.cfg.NginxConfigPath(), "-s", "reload")
	cmd := exec.CommandContext(ctx, m.cfg.NginxBin, fullArgs...)
	return cmd.Run() == nil
}

func (m *Manager) reload(ctx context.Context) error {
	pid, err := readPIDFile(m.cfg.NginxPIDFile)
	if err == nil && m.ownsMaster(pid) {
		if err := reloadProcess(pid); err == nil {
			m.logger.Info("nginx reloaded")
			return nil
		}
		m.logger.Warn("nginx signal reload failed, falling back to cli", "pid", pid, "error", err)
	} else {
		return fmt.Errorf("未找到经过身份确认的 Fonu Nginx 主进程")
	}

	if !m.ownsMaster(pid) { return fmt.Errorf("Nginx 主进程身份已变化") }
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
	if err != nil || !m.ownsMaster(pid) {
		return
	}
	_ = terminateProcess(pid, syscall.SIGTERM)
	waitProcessExit(pid, nginxTerminateWait)
	if m.ownsMaster(pid) {
		_ = terminateProcess(pid, syscall.SIGKILL)
		waitProcessExit(pid, time.Second)
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.isRunningQuick() {
		return nil
	}
	m.confirmedMaster = 0
	m.confirmedWorkers = nil
	m.statusMu.Lock()
	m.status.Confirmed = ""
	m.status.State = "saved"
	m.statusMu.Unlock()
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
	var stderr lockedBuffer
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
	if !m.ownsMaster(pid) {
		removePIDFile(m.cfg.NginxPIDFile)
		return false, nil
	}
	return true, nil
}

func (m *Manager) isRunningQuick() bool {
	ok, _ := m.isRunning()
	return ok
}

func (m *Manager) ValidateContent(ctx context.Context, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ValidateConfigSyntax(content); err != nil {
		return err
	}
	if !m.available() {
		m.logger.Warn("nginx binary not found, syntax-only validation", "bin", m.cfg.NginxBin)
		return nil
	}
	tmpPath, err := writeTemporaryConfig(m.cfg.NginxDir(), content)
	if err != nil { return err }
	defer os.Remove(tmpPath)
	return m.validate(ctx, tmpPath)
}

func ChinaCIDRExists(cfg config.Config) bool {
	return chinaCIDRExists(cfg)
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

func writeTemporaryConfig(dir, content string) (string, error) {
	f, err := os.CreateTemp(dir, "nginx-*.conf")
	if err != nil { return "", err }
	name := f.Name()
	_, err = f.WriteString(content)
	closeErr := f.Close()
	if err == nil { err = closeErr }
	if err != nil { os.Remove(name); return "", err }
	return name, nil
}

// ApplyStatus reports the last apply attempt; a successful signal is not a worker acknowledgement.
type ApplyStatus struct {
	State string `json:"state"`
	Desired string `json:"desired_revision,omitempty"`
	Applied string `json:"last_requested_revision,omitempty"`
	LastError string `json:"last_error,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Confirmed string `json:"confirmed_revision,omitempty"`
	History []ApplyHistoryEntry `json:"history,omitempty"`
}

func (m *Manager) ApplyStatus() ApplyStatus {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()
	status := m.status
	if status.State == "" { status.State = "unknown" }
	status.History = m.readApplyHistory()
	return status
}

func restoreConfig(current, backup string, hadPrevious bool) error {
	if !hadPrevious { return os.Remove(current) }
	content, err := os.ReadFile(backup)
	if err != nil { return err }
	tmp, err := writeTemporaryConfig(filepath.Dir(current), string(content))
	if err != nil { return err }
	defer os.Remove(tmp)
	return os.Rename(tmp, current)
}

// exec.Cmd may still be copying stderr while startup is being polled.
type lockedBuffer struct { mu sync.Mutex; buf bytes.Buffer }
func (b *lockedBuffer) Write(p []byte) (int, error) { b.mu.Lock(); defer b.mu.Unlock(); return b.buf.Write(p) }
func (b *lockedBuffer) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.buf.String() }
