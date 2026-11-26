package nginx

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

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

// A resolver lives for one generation/publication only. Each certificate/key
// pair is parsed at most once; subsequent host matches use its actual SANs.
type certificateResolver struct {
	dir string
	sources []CertSource
	leaves map[certFiles]*x509.Certificate
	disk []os.DirEntry
	scanned bool
}
func newCertificateResolver(dir string,sources []CertSource)*certificateResolver{
	return &certificateResolver{dir:dir,sources:sources,leaves:make(map[certFiles]*x509.Certificate)}
}
func (r *certificateResolver) find(hosts []string)*certFiles{
	if len(hosts)==0{return nil}
	for _,source:=range r.sources{
		if cert:=certFromSource(source);r.covers(cert,hosts){return cert}
		for _,domain:=range source.Domains{
			if cert:=certAt(r.dir,storageDirName(domain));r.covers(cert,hosts){return cert}
		}
	}
	if !r.scanned{r.disk,_=osReadDir(r.dir);r.scanned=true}
	for _,entry:=range r.disk{
		if !entry.IsDir(){continue}
		if cert:=certAt(r.dir,entry.Name());r.covers(cert,hosts){return cert}
	}
	return nil
}
func(r *certificateResolver) covers(cert *certFiles,hosts []string)bool{
	if len(hosts)==0||cert==nil{return false}
	leaf,ok:=r.leaves[*cert]
	if !ok{
		leaf=loadCertificateLeaf(cert)
		r.leaves[*cert]=leaf
	}
	if leaf==nil{return false}
	now:=time.Now()
	if now.Before(leaf.NotBefore)||!now.Before(leaf.NotAfter){return false}
	for _,host:=range hosts{if leaf.VerifyHostname(host)!=nil{return false}}
	return true
}
func loadCertificateLeaf(cert *certFiles)*x509.Certificate{
	if !certUsable(cert){return nil}
	pair,err:=tls.LoadX509KeyPair(cert.CertPath,cert.KeyPath)
	if err!=nil||len(pair.Certificate)==0{return nil}
	leaf,err:=x509.ParseCertificate(pair.Certificate[0])
	if err!=nil{return nil}
	return leaf
}
func findCertificateForHosts(certsDir string,hostnames []string,certs []CertSource)*certFiles{
	return newCertificateResolver(certsDir,certs).find(hostnames)
}
func certificateCovers(cert *certFiles,hosts []string)bool{
	return newCertificateResolver("",nil).covers(cert,hosts)
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
			if strings.HasSuffix(hostname, "."+suffix) && !strings.Contains(strings.TrimSuffix(hostname, "."+suffix), ".") {
				return true
			}
		}
	}
	return false
}

func scanCertDirectories(certsDir string, hostnames []string) *certFiles {
	entries, err := osReadDir(certsDir)
	if err != nil { return nil }
	for _, entry := range entries {
		if !entry.IsDir() { continue }
		if cert := certAt(certsDir, entry.Name()); certificateCovers(cert, hostnames) {
			return cert
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
	resolver:=newCertificateResolver(certsDir,certs)
	for _, rule := range rules {
		if !rule.Enabled || !rule.HTTPSEnabled {
			continue
		}
		for _, group := range rule.PortGroups() {
			if len(group.Hostnames) == 0 {
				return fmt.Errorf("启用 HTTPS 的规则缺少前端域名")
			}
			for _, hostname := range group.Hostnames {
				if resolver.find([]string{hostname})==nil {
					return HTTPSCoverageError(hostname, certs)
				}
			}
		}
	}
	return nil
}

func allHostsCovered(hosts, domains []string) bool {
	if len(hosts) == 0 { return false }
	for _, host := range hosts {
		if !hostCoveredByCert(host, domains) { return false }
	}
	return true
}
