package frp

import (
	"fmt"
	"sort"
	"strings"
)

// BuildFRPSConfig generates frps.toml for the user's VPS, aligned with Fonu Nginx ports and auth.
func BuildFRPSConfig(bindPort, httpPort, httpsPort int, token string, tcpRemotePorts []int) string {
	if bindPort <= 0 {
		bindPort = 7000
	}
	if httpPort <= 0 {
		httpPort = 80
	}
	if httpsPort <= 0 {
		httpsPort = 443
	}
	if token == "" {
		token = "your-secret-token"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`bindAddr = "0.0.0.0"
bindPort = %d

vhostHTTPPort = %d
vhostHTTPSPort = %d

`, bindPort, httpPort, httpsPort))

	ports := append([]int(nil), tcpRemotePorts...)
	sort.Ints(ports)
	if len(ports) > 0 {
		parts := make([]string, 0, len(ports))
		for _, port := range ports {
			parts = append(parts, fmt.Sprintf(`"%d"`, port))
		}
		b.WriteString("# TCP 隧道远程端口，需在 VPS 防火墙放行\n")
		b.WriteString(fmt.Sprintf("allowPorts = [%s]\n\n", strings.Join(parts, ", ")))
	}

	b.WriteString(fmt.Sprintf(`[auth]
method = "token"
token = "%s"
`, token))
	return b.String()
}
