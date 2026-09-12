package acme

import (
	"fmt"

	"github.com/go-acme/lego/v4/lego"
)

const (
	CALetsEncrypt        = "letsencrypt"
	CALetsEncryptStaging = "letsencrypt-staging"
)

func NormalizeCA(ca string) string {
	switch ca {
	case "", CALetsEncrypt, "production":
		return CALetsEncrypt
	case CALetsEncryptStaging, "staging", "letsencrypt-test":
		return CALetsEncryptStaging
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
