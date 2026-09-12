package nginx

import (
	"fmt"
	"strings"
)

func certUsable(cert *certFiles) bool {
	if cert == nil {
		return false
	}
	if strings.TrimSpace(cert.CertPath) == "" || strings.TrimSpace(cert.KeyPath) == "" {
		return false
	}
	return fileExists(cert.CertPath) && fileExists(cert.KeyPath)
}

func assertValidSSLBlocks(content string) error {
	inServer := false
	depth := 0
	serverSSL := false
	serverCert := false

	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !inServer {
			if strings.HasPrefix(line, "server") && strings.Contains(line, "{") {
				inServer = true
				depth = braceDelta(line)
				serverSSL = false
				serverCert = false
			}
			continue
		}

		depth += braceDelta(line)
		if isSSLListenLine(line) {
			serverSSL = true
		}
		if isSSLCertificateLine(line) {
			serverCert = true
		}

		if depth <= 0 {
			if serverSSL && !serverCert {
				return fmt.Errorf("检测到未配置证书的 HTTPS 监听，请为域名申请或导入证书，或关闭 HTTPS")
			}
			inServer = false
		}
	}

	if inServer && serverSSL && !serverCert {
		return fmt.Errorf("检测到未配置证书的 HTTPS 监听，请为域名申请或导入证书，或关闭 HTTPS")
	}
	return nil
}

func braceDelta(line string) int {
	return strings.Count(line, "{") - strings.Count(line, "}")
}

func isSSLListenLine(line string) bool {
	if !strings.HasPrefix(line, "listen ") {
		return false
	}
	return strings.Contains(line, " ssl;") || strings.HasSuffix(line, " ssl")
}

func isSSLCertificateLine(line string) bool {
	return strings.HasPrefix(line, "ssl_certificate ") && !strings.HasPrefix(line, "ssl_certificate_key ")
}
