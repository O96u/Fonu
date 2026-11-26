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
	cmd:=exec.CommandContext(ctx,bin,"--version")
	var out commandOutput
	cmd.Stdout=&out
	cmd.WaitDelay=time.Second
	if err := cmd.Run(); err != nil {
		return defaultClientVersion
	}
	line := strings.TrimSpace(out.String())
	if line == "" {
		return defaultClientVersion
	}
	if strings.HasPrefix(line, "frpc version ") {
		return "frpc v" + strings.TrimPrefix(line, "frpc version ")
	}
	return line
}
