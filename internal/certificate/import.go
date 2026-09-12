package certificate

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type ImportInfo struct {
	Primary   string
	Domains   []string
	Wildcard  bool
	ExpiresAt time.Time
}

func InspectImport(certPEM, keyPEM []byte) (ImportInfo, error) {
	cert, err := parseCertificate(certPEM)
	if err != nil {
		return ImportInfo{}, err
	}
	key, err := parsePrivateKey(keyPEM)
	if err != nil {
		return ImportInfo{}, err
	}
	if !publicKeysMatch(cert.PublicKey, key) {
		return ImportInfo{}, fmt.Errorf("证书与私钥不匹配")
	}
	domains, err := ExtractCertDomains(cert)
	if err != nil {
		return ImportInfo{}, err
	}
	return ImportInfo{
		Primary:   primaryDomain(domains),
		Domains:   domains,
		Wildcard:  hasWildcardDomain(domains),
		ExpiresAt: cert.NotAfter.UTC(),
	}, nil
}

func ValidatePEM(certPEM, keyPEM []byte) (time.Time, error) {
	info, err := InspectImport(certPEM, keyPEM)
	if err != nil {
		return time.Time{}, err
	}
	return info.ExpiresAt, nil
}

func ExtractCertDomains(cert *x509.Certificate) ([]string, error) {
	seen := map[string]bool{}
	var domains []string
	add := func(name string) error {
		name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
		if name == "" || net.ParseIP(name) != nil {
			return nil
		}
		if seen[name] {
			return nil
		}
		if strings.HasPrefix(name, "*.") {
			if err := validateDomainLabel(strings.TrimPrefix(name, "*.")); err != nil {
				return err
			}
		} else if err := validateDomainLabel(name); err != nil {
			return err
		}
		seen[name] = true
		domains = append(domains, name)
		return nil
	}

	for _, name := range cert.DNSNames {
		if err := add(name); err != nil {
			return nil, err
		}
	}
	if err := add(cert.Subject.CommonName); err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("证书中未包含有效域名")
	}
	return domains, nil
}

func primaryDomain(domains []string) string {
	for _, domain := range domains {
		if !strings.HasPrefix(domain, "*.") {
			return domain
		}
	}
	return strings.TrimPrefix(domains[0], "*.")
}

func hasWildcardDomain(domains []string) bool {
	for _, domain := range domains {
		if strings.HasPrefix(domain, "*.") {
			return true
		}
	}
	return false
}

func validateDomainLabel(domain string) error {
	if domain == "" {
		return fmt.Errorf("域名格式无效")
	}
	parts := strings.Split(domain, ".")
	for _, part := range parts {
		if part == "" || len(part) > 63 {
			return fmt.Errorf("域名格式无效")
		}
	}
	return nil
}

func parseCertificate(certPEM []byte) (*x509.Certificate, error) {
	rest := certPEM
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			return nil, fmt.Errorf("证书 PEM 格式无效")
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("解析证书失败：%w", err)
			}
			return cert, nil
		}
		rest = remaining
		if len(rest) == 0 {
			break
		}
	}
	return nil, fmt.Errorf("证书 PEM 中未找到有效证书")
}

func parsePrivateKey(keyPEM []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("私钥 PEM 格式无效")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		return x509.ParsePKCS8PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("不支持的私钥类型：%s", block.Type)
	}
}

func publicKeysMatch(certKey crypto.PublicKey, privateKey crypto.PrivateKey) bool {
	switch pub := certKey.(type) {
	case *rsa.PublicKey:
		priv, ok := privateKey.(*rsa.PrivateKey)
		return ok && pub.N.Cmp(priv.N) == 0
	case *ecdsa.PublicKey:
		priv, ok := privateKey.(*ecdsa.PrivateKey)
		return ok && pub.X.Cmp(priv.X) == 0 && pub.Y.Cmp(priv.Y) == 0
	default:
		return false
	}
}

func NormalizePEM(v string) []byte {
	return []byte(strings.TrimSpace(v))
}

func ResolvePEMSource(content, path string) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path != "" {
		return ReadPEMFile(path)
	}
	data := NormalizePEM(content)
	if len(data) == 0 {
		return nil, fmt.Errorf("内容不能为空")
	}
	return data, nil
}

func ReadPEMFile(path string) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("文件路径不能为空")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取文件 %s：%w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s 是目录，不是文件", path)
	}
	if info.Size() > 1<<20 {
		return nil, fmt.Errorf("文件过大（最大 1MB）")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败：%w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("文件内容为空")
	}
	return data, nil
}
