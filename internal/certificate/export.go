package certificate

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type ExportConfig struct {
	Enabled       bool             `json:"enabled"`
	Targets       []string         `json:"targets"`
	OnChangeShell string           `json:"on_change_shell"`
	LastSyncAt    string           `json:"last_sync_at,omitempty"`
	LastError     string           `json:"last_error,omitempty"`
	SSH           SSHExportConfig  `json:"ssh"`
}

func DefaultExportConfig() ExportConfig {
	return ExportConfig{}
}

func ParseExportConfig(raw string) (ExportConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultExportConfig(), nil
	}
	var cfg ExportConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ExportConfig{}, fmt.Errorf("证书映射配置 JSON 无效")
	}
	return cfg, nil
}

func (c ExportConfig) Marshal() (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c ExportConfig) Normalize(certsDir string) (ExportConfig, error) {
	out := ExportConfig{
		Enabled:       c.Enabled,
		Targets:       normalizeExportTargets(c.Targets),
		OnChangeShell: strings.TrimSpace(c.OnChangeShell),
		LastSyncAt:    strings.TrimSpace(c.LastSyncAt),
		LastError:     strings.TrimSpace(c.LastError),
		SSH:           c.SSH.Normalize(),
	}
	if err := out.Validate(certsDir); err != nil {
		return ExportConfig{}, err
	}
	return out, nil
}

func (c ExportConfig) Validate(certsDir string) error {
	if err := c.SSH.Validate(); err != nil {
		return err
	}
	if !c.Enabled {
		return nil
	}
	if len(c.Targets) == 0 {
		return fmt.Errorf("启用证书映射时请至少填写一个目标目录")
	}
	seen := map[string]bool{}
	for _, t := range c.Targets {
		t = strings.TrimSpace(t)
		if t == "" {
			return fmt.Errorf("目标目录不能为空")
		}
		if err := validateExportTargetPath(t, certsDir); err != nil {
			return err
		}
		clean := filepath.Clean(t)
		if seen[clean] {
			return fmt.Errorf("目标目录重复：%s", clean)
		}
		seen[clean] = true
	}
	return nil
}

func normalizeExportTargets(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		c := filepath.Clean(t)
		if seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

func validateExportTargetPath(target, certsDir string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("目标目录不能为空")
	}
	if !filepath.IsAbs(target) {
		return fmt.Errorf("目标目录必须为绝对路径：%s", target)
	}
	target = filepath.Clean(target)
	certsDir = filepath.Clean(certsDir)
	if target == certsDir {
		return fmt.Errorf("目标目录不能与证书源目录相同")
	}
	if exportPathUnderDir(target, certsDir) {
		return fmt.Errorf("目标目录不能位于证书源目录内：%s", target)
	}
	return nil
}

func exportPathUnderDir(path, parent string) bool {
	path = filepath.Clean(path)
	parent = filepath.Clean(parent)
	if path == parent {
		return true
	}
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
