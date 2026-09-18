package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/config"
)

const maxCustomBackups = 10

type BackupEntry struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func EnsureCustomDirs(cfg config.Config) error {
	for _, dir := range []string{
		cfg.NginxCustomDir(),
		cfg.NginxRulesDir(),
		cfg.NginxGlobalBackupsDir(),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func ReadGlobalCustom(cfg config.Config) (string, error) {
	data, err := os.ReadFile(cfg.NginxGlobalCustomPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func ReadRuleCustom(cfg config.Config, ruleID int64) (string, error) {
	data, err := os.ReadFile(cfg.NginxRuleCustomPath(ruleID))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func ruleCustomContent(cfg config.Config, ruleID int64, opts GenerateOptions) string {
	if opts.RuleCustomOverrides != nil {
		if override, ok := opts.RuleCustomOverrides[ruleID]; ok {
			return override
		}
	}
	content, _ := ReadRuleCustom(cfg, ruleID)
	return content
}

func globalCustomPath(cfg config.Config, opts GenerateOptions) string {
	if opts.GlobalCustomPathOverride != "" {
		return opts.GlobalCustomPathOverride
	}
	return cfg.NginxGlobalCustomPath()
}

func globalCustomShouldInclude(cfg config.Config, opts GenerateOptions) bool {
	if strings.TrimSpace(opts.GlobalCustomOverride) != "" {
		return true
	}
	if opts.GlobalCustomPathOverride != "" {
		data, err := os.ReadFile(opts.GlobalCustomPathOverride)
		return err == nil && strings.TrimSpace(string(data)) != ""
	}
	content, _ := ReadGlobalCustom(cfg)
	return strings.TrimSpace(content) != ""
}

func BackupFile(srcPath, backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	name := time.Now().UTC().Format("20060102T150405.000Z") + ".conf"
	dst := filepath.Join(backupDir, name)
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", err
	}
	if err := pruneBackups(backupDir, maxCustomBackups); err != nil {
		return "", err
	}
	return name, nil
}

func ListBackups(backupDir string) ([]BackupEntry, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}
	var out []BackupEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		out = append(out, BackupEntry{Name: entry.Name(), CreatedAt: info.ModTime().UTC()})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name > out[j].Name
	})
	return out, nil
}

func RestoreBackup(backupDir, backupName, targetPath string) error {
	src := filepath.Join(backupDir, backupName)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("备份不存在")
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0o644)
}

func WriteGlobalCustom(cfg config.Config, content string) error {
	if err := EnsureCustomDirs(cfg); err != nil {
		return err
	}
	return os.WriteFile(cfg.NginxGlobalCustomPath(), []byte(content), 0o644)
}

func WriteRuleCustom(cfg config.Config, ruleID int64, content string) error {
	if err := os.MkdirAll(cfg.NginxRulesDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(cfg.NginxRuleCustomPath(ruleID), []byte(content), 0o644)
}

func RemoveRuleCustom(cfg config.Config, ruleID int64) error {
	path := cfg.NginxRuleCustomPath(ruleID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	backupDir := cfg.NginxRuleBackupsDir(ruleID)
	_ = os.RemoveAll(backupDir)
	return nil
}

func SaveGlobalCustomWithBackup(cfg config.Config, content string) (string, error) {
	path := cfg.NginxGlobalCustomPath()
	var backupName string
	if _, err := os.Stat(path); err == nil {
		name, err := BackupFile(path, cfg.NginxGlobalBackupsDir())
		if err != nil {
			return "", err
		}
		backupName = name
	}
	if err := WriteGlobalCustom(cfg, content); err != nil {
		return "", err
	}
	return backupName, nil
}

func SaveRuleCustomWithBackup(cfg config.Config, ruleID int64, content string) (string, error) {
	path := cfg.NginxRuleCustomPath(ruleID)
	var backupName string
	if _, err := os.Stat(path); err == nil {
		name, err := BackupFile(path, cfg.NginxRuleBackupsDir(ruleID))
		if err != nil {
			return "", err
		}
		backupName = name
	}
	if err := WriteRuleCustom(cfg, ruleID, content); err != nil {
		return "", err
	}
	return backupName, nil
}

func pruneBackups(dir string, keep int) error {
	entries, err := ListBackups(dir)
	if err != nil {
		return err
	}
	for i := keep; i < len(entries); i++ {
		_ = os.Remove(filepath.Join(dir, entries[i].Name))
	}
	return nil
}

func DefaultGlobalHTTPSnippet() string {
	return `# 全局 http 块片段（自动生成）
# 上传大小限制（全局默认 50M，单条规则可在 server 块覆盖）
client_max_body_size 50m;
`
}

func GenerateGlobalFramework(cfg config.Config) string {
	return fmt.Sprintf(`# 以下部分由 Fonu 自动生成，请勿在此编辑

# 主进程
worker_processes auto;
# 错误日志
error_log %s warn;
# PID 文件
pid %s;

# 事件模型
events {
    worker_connections 1024;
}

http {
    # MIME 类型
    include       %s;
    default_type  application/octet-stream;

    # 访问日志格式与路径
    log_format fonu_access '$time_iso8601 $host $server_port $request_method $request_uri $status $request_time $remote_addr $upstream_addr $request_length $bytes_sent';
    access_log %s fonu_access;

    # 传输优化
    sendfile on;
    keepalive_timeout 65;

    # WebSocket 升级映射
    # map $http_upgrade $connection_upgrade { ... }

    # limit_req_zone / geo / real_ip 等由规则与系统设置动态生成
    # 全局自定义片段插入在上述动态块之后、各 server {} 之前
}
`, absNginxPath(filepath.Join(cfg.LogsDir(), "error.log")),
		absNginxPath(cfg.NginxPIDFile),
		absNginxPath(cfg.NginxMimeTypes),
		absNginxPath(filepath.Join(cfg.LogsDir(), "access.log")))
}
