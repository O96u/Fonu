package frp

import (
	"fmt"
	"strings"

	"github.com/fonu/fonu/internal/config"
)

func Generate(cfg config.Config, frpCfg Config, authToken string, httpPort, httpsPort int) (string, error) {
	if strings.TrimSpace(frpCfg.ServerAddr) == "" {
		return "", fmt.Errorf("FRP 服务器地址未配置")
	}
	if strings.TrimSpace(authToken) == "" {
		return "", fmt.Errorf("FRP 认证 Token 未配置")
	}
	domains := normalizeDomains(frpCfg.CustomDomains)
	if len(domains) == 0 {
		return "", fmt.Errorf("请至少填写一个穿透域名")
	}
	if httpPort <= 0 || httpsPort <= 0 {
		return "", fmt.Errorf("Nginx 端口未配置")
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("serverAddr = %q\n", frpCfg.ServerAddr))
	b.WriteString(fmt.Sprintf("serverPort = %d\n\n", frpCfg.ServerPort))

	b.WriteString(fmt.Sprintf("pidFile = %q\n", cfg.FrpPIDFile))
	b.WriteString(fmt.Sprintf("log.to = %q\n", cfg.FrpLogPath()))
	b.WriteString("log.level = \"info\"\n\n")

	b.WriteString("auth.method = \"token\"\n")
	b.WriteString(fmt.Sprintf("auth.token = %q\n\n", authToken))

	if frpCfg.TLSEnabled {
		b.WriteString("transport.tls.enable = true\n\n")
	}

	b.WriteString("[[proxies]]\n")
	b.WriteString("name = \"fonu-nginx-http\"\n")
	b.WriteString("type = \"http\"\n")
	b.WriteString("localIP = \"127.0.0.1\"\n")
	b.WriteString(fmt.Sprintf("localPort = %d\n", httpPort))
	b.WriteString(fmt.Sprintf("customDomains = %s\n\n", formatTOMLStringArray(domains)))

	b.WriteString("[[proxies]]\n")
	b.WriteString("name = \"fonu-nginx-https\"\n")
	b.WriteString("type = \"https\"\n")
	b.WriteString(fmt.Sprintf("customDomains = %s\n", formatTOMLStringArray(domains)))
	b.WriteString("[proxies.plugin]\n")
	b.WriteString("type = \"https2https\"\n")
	b.WriteString(fmt.Sprintf("localAddr = \"127.0.0.1:%d\"\n", httpsPort))

	return b.String(), nil
}

func formatTOMLStringArray(items []string) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%q", item))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
