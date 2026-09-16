package frp

import (
	"os/exec"
	"strings"
)

const defaultClientVersion = "frpc v0.66.0"

func ResolveClientVersion(bin string) string {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return defaultClientVersion
	}
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return defaultClientVersion
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return defaultClientVersion
	}
	if strings.HasPrefix(line, "frpc version ") {
		return "frpc v" + strings.TrimPrefix(line, "frpc version ")
	}
	return line
}
