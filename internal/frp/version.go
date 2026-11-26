package frp

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const defaultClientVersion = "未知"

func ResolveClientVersion(bin string) string {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return defaultClientVersion
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
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
