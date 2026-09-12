package nginx

import (
	"fmt"
	"regexp"
	"strings"
)

var serverBlockRe = regexp.MustCompile(`(?s)server\s*\{([^}]*)\}`)

func assertValidSSLBlocks(content string) error {
	for _, block := range serverBlockRe.FindAllStringSubmatch(content, -1) {
		body := block[1]
		if !strings.Contains(body, " ssl") && !strings.Contains(body, " ssl;") {
			continue
		}
		if !strings.Contains(body, "ssl_certificate ") {
			return fmt.Errorf("检测到未配置证书的 HTTPS 监听，请为域名申请或导入证书，或关闭 HTTPS")
		}
	}
	return nil
}

func certUsable(cert *certFiles) bool {
	if cert == nil {
		return false
	}
	if strings.TrimSpace(cert.CertPath) == "" || strings.TrimSpace(cert.KeyPath) == "" {
		return false
	}
	return fileExists(cert.CertPath) && fileExists(cert.KeyPath)
}
