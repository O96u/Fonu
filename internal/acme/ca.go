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
	CABuypassTest        = "buypass-test"
	CAGoogle             = "google"
	CASSLcom             = "sslcom"
	CAFreeSSL            = "freessl"
	CAActalis            = "actalis"
	CACustom             = "custom"
)

const (
	zeroSSLDirectoryURL    = "https://acme.zerossl.com/v2/DV90"
	buypassDirectoryURL    = "https://api.buypass.com/acme/directory"
	buypassTestDirectoryURL = "https://api.test4.buypass.no/acme/directory"
	googleDirectoryURL     = "https://dv.acme-v02.api.pki.goog/directory"
	sslcomDirectoryURL     = "https://acme.ssl.com/sslcom-dv-ecc"
	freesslDirectoryURL    = FreeSSLDirectoryURLDefault
	actalisDirectoryURL    = "https://acme-api.actalis.com/acme/directory"
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
	case CABuypassTest, "buypass-test4", "buypass_test":
		return CABuypassTest
	case CAGoogle, "googletrust", "google-trust", "gts":
		return CAGoogle
	case CASSLcom, "ssl.com":
		return CASSLcom
	case CAFreeSSL, "litessl", "freessl.cn":
		return CAFreeSSL
	case CAActalis, "actalis.com":
		return CAActalis
	case CACustom, "custom-acme", "other":
		return CACustom
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
	case CABuypassTest:
		return buypassTestDirectoryURL, nil
	case CAGoogle:
		return googleDirectoryURL, nil
	case CASSLcom:
		return sslcomDirectoryURL, nil
	case CAFreeSSL:
		return freesslDirectoryURL, nil
	case CAActalis:
		return actalisDirectoryURL, nil
	case CACustom:
		// 自定义 ACME 的 Directory URL 在申请时从设置读取
		return "", nil
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
	case CABuypassTest:
		return "Buypass 测试"
	case CAGoogle:
		return "Google Trust Services"
	case CASSLcom:
		return "SSL.com"
	case CAFreeSSL:
		return "FreeSSL / LiteSSL"
	case CAActalis:
		return "Actalis"
	case CACustom:
		return "自定义 ACME"
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
	case CABuypass, CABuypassTest:
		if len(domains) > 5 {
			return fmt.Errorf("Buypass 单张证书最多支持 5 个域名")
		}
		for _, domain := range domains {
			if strings.HasPrefix(domain, "*.") {
				return fmt.Errorf("Buypass 不支持通配符证书")
			}
		}
	case CAActalis:
		if len(domains) > 5 {
			return fmt.Errorf("Actalis 单张证书最多支持 5 个域名")
		}
		for _, domain := range domains {
			if strings.HasPrefix(domain, "*.") {
				return fmt.Errorf("Actalis 不支持通配符证书（免费套餐仅支持单域名）")
			}
		}
	}
	return nil
}

