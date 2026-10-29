package certexport

import (
	"os"
	"strings"

	"github.com/fonu/fonu/internal/config"
)

type PathHints struct {
	CertsDir               string   `json:"certs_dir"`
	DeployKind             string   `json:"deploy_kind"`
	DataSharePaths         []string `json:"data_share_paths"`
	AccessiblePaths        []string `json:"accessible_paths"`
	SuggestedExportTargets []string `json:"suggested_export_targets"`
}

func PathHintsFor(cfg config.Config) PathHints {
	shares := splitEnvPaths(os.Getenv("TRIM_DATA_SHARE_PATHS"))
	accessible := splitEnvPaths(os.Getenv("TRIM_DATA_ACCESSIBLE_PATHS"))
	suggested := suggestExportTargets(shares, accessible)

	return PathHints{
		CertsDir:               cfg.CertsDir(),
		DeployKind:             detectDeployKind(),
		DataSharePaths:         shares,
		AccessiblePaths:        accessible,
		SuggestedExportTargets: suggested,
	}
}

func (s *Service) PathHints() PathHints {
	return PathHintsFor(s.cfg)
}

func detectDeployKind() string {
	if strings.TrimSpace(os.Getenv("TRIM_PKGVAR")) != "" {
		return "fnos"
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}
	return "generic"
}

func splitEnvPaths(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ":")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func suggestExportTargets(shares, accessible []string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, p := range shares {
		lower := strings.ToLower(strings.TrimRight(p, "/"))
		if strings.HasSuffix(lower, "/certs") || strings.HasSuffix(lower, "certs") {
			add(p)
		}
	}
	for _, p := range accessible {
		add(p)
	}
	for _, p := range shares {
		add(p)
	}
	return out
}
