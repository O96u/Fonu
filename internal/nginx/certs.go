package nginx

import (
	"fmt"
	"os"
	"strings"

	"github.com/fonu/fonu/internal/proxy"
)

func osReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

type CertSource struct {
	Domains  []string
	CertPath string
	KeyPath  string
}

func FindCertificateForHosts(certsDir string, hostnames []string, certs []CertSource) *certFiles {
	return findCertificateForHosts(certsDir, hostnames, certs)
}

func HasCertificateForHosts(certsDir string, hostnames []string, certs []CertSource) bool {
	return certUsable(findCertificateForHosts(certsDir, hostnames, certs))
}

func findCertificateForHosts(certsDir string, hostnames []string, certs []CertSource) *certFiles {
	for _, hostname := range hostnames {
		for _, source := range certs {
			if !hostCoveredByCert(hostname, source.Domains) {
				continue
			}
			if cert := certFromSource(source); cert != nil {
				return cert
			}
			for _, domain := range source.Domains {
				if cert := certAt(certsDir, storageDirName(domain)); cert != nil && certUsable(cert) {
					return cert
				}
			}
		}
		if cert := findCertificate(certsDir, hostname); cert != nil && certUsable(cert) {
			return cert
		}
	}
	return scanCertDirectories(certsDir, hostnames)
}

func storageDirName(domain string) string {
	domain = strings.TrimSpace(domain)
	if strings.HasPrefix(domain, "*.") {
		return "wildcard." + strings.TrimPrefix(domain, "*.")
	}
	return domain
}

func certFromSource(source CertSource) *certFiles {
	cert := &certFiles{
		CertPath: absNginxPath(source.CertPath),
		KeyPath:  absNginxPath(source.KeyPath),
	}
	if !certUsable(cert) {
		return nil
	}
	return cert
}

func hostCoveredByCert(hostname string, domains []string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}
		if domain == hostname {
			return true
		}
		if strings.HasPrefix(domain, "*.") {
			suffix := strings.TrimPrefix(domain, "*.")
			if hostname == suffix || strings.HasSuffix(hostname, "."+suffix) {
				return true
			}
		}
	}
	return false
}

func scanCertDirectories(certsDir string, hostnames []string) *certFiles {
	entries, err := osReadDir(certsDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pattern := dirNameToPattern(entry.Name())
		for _, hostname := range hostnames {
			if !hostCoveredByCert(hostname, []string{pattern, entry.Name()}) {
				continue
			}
			if cert := certAt(certsDir, pattern); cert != nil && certUsable(cert) {
				return cert
			}
			if cert := certAt(certsDir, entry.Name()); cert != nil && certUsable(cert) {
				return cert
			}
		}
	}
	return nil
}

func dirNameToPattern(name string) string {
	if strings.HasPrefix(name, "wildcard.") {
		return "*." + strings.TrimPrefix(name, "wildcard.")
	}
	return name
}

func certSummary(certs []CertSource) string {
	seen := map[string]struct{}{}
	var domains []string
	for _, cert := range certs {
		for _, domain := range cert.Domains {
			if domain == "" {
				continue
			}
			if _, ok := seen[domain]; ok {
				continue
			}
			seen[domain] = struct{}{}
			domains = append(domains, domain)
		}
	}
	if len(domains) == 0 {
		return "无"
	}
	return strings.Join(domains, ", ")
}

func HTTPSCoverageError(hostname string, certs []CertSource) error {
	return fmt.Errorf("域名 %s 未匹配到可用证书（当前证书覆盖：%s），请检查拼写或先在证书页导入/申请", hostname, certSummary(certs))
}

func ValidateHTTPSRules(rules []proxy.Rule, certs []CertSource, certsDir string) error {
	for _, rule := range rules {
		if !rule.Enabled || !rule.HTTPSEnabled {
			continue
		}
		for _, group := range rule.PortGroups() {
			cert := findCertificateForHosts(certsDir, group.Hostnames, certs)
			if certUsable(cert) {
				continue
			}
			if len(group.Hostnames) == 0 {
				return fmt.Errorf("启用 HTTPS 的规则缺少前端域名")
			}
			return HTTPSCoverageError(group.Hostnames[0], certs)
		}
	}
	return nil
}
