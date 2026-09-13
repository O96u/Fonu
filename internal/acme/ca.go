package acme

import (
	"fmt"
	"strings"

	"github.com/go-acme/lego/v4/lego"
)

const (
	CALetsEncrypt        = "letsencrypt"
	CALetsEncryptStaging = "letsencrypt-staging"
	CAZeroSSL            = "zerossl"
	CABuypass            = "buypass"
)

const (
	zeroSSLDirectoryURL = "https://acme.zerossl.com/v2/DV90"
	buypassDirectoryURL = "https://api.buypass.com/acme/directory"
)

func NormalizeCA(ca string) string {
	switch ca {
	case "", CALetsEncrypt, "production":
		return CALetsEncrypt
	case CALetsEncryptStaging, "staging", "letsencrypt-test":
		return CALetsEncryptStaging
	case CAZeroSSL:
		return CAZeroSSL
	case CABuypass, "buypass-go":
		return CABuypass
	default:
		return ca
	}
}

func DirectoryURL(ca string) (string, error) {
	switch NormalizeCA(ca) {
	case CALetsEncrypt:
		return lego.LEDirectoryProduction, nil
	case CALetsEncryptStaging:
		return lego.LEDirectoryStaging, nil
	case CAZeroSSL:
		return zeroSSLDirectoryURL, nil
	case CABuypass:
		return buypassDirectoryURL, nil
	default:
		return "", fmt.Errorf("不支持的颁发机构：%s", ca)
	}
}

func CALabel(ca string) string {
	switch NormalizeCA(ca) {
	case CALetsEncrypt:
		return "Let's Encrypt"
	case CALetsEncryptStaging:
		return "Let's Encrypt 测试"
	case CAZeroSSL:
		return "ZeroSSL"
	case CABuypass:
		return "Buypass"
	default:
		if ca == "" {
			return "手动导入"
		}
		return ca
	}
}

func ValidateCA(ca string) error {
	_, err := DirectoryURL(ca)
	return err
}

func ValidateCADomains(ca string, domains []string) error {
	switch NormalizeCA(ca) {
	case CABuypass:
		if len(domains) > 5 {
			return fmt.Errorf("Buypass 单张证书最多支持 5 个域名")
		}
		for _, domain := range domains {
			if strings.HasPrefix(domain, "*.") {
				return fmt.Errorf("Buypass 不支持通配符证书")
			}
		}
	}
	return nil
}

