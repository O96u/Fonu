package acme

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	certstore "github.com/fonu/fonu/internal/certificate"
)

type DownloadPart string

const (
	DownloadZIP  DownloadPart = "zip"
	DownloadCert DownloadPart = "cert"
	DownloadKey  DownloadPart = "key"
)

func (s *Service) Download(ctx context.Context, domain string, part string) (filename string, contentType string, data []byte, err error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return "", "", nil, fmt.Errorf("域名不能为空")
	}

	rec, err := s.store.GetByDomain(ctx, domain)
	if err != nil {
		return "", "", nil, err
	}
	if rec.CertPath == "" || rec.KeyPath == "" {
		return "", "", nil, fmt.Errorf("证书文件不存在，可能尚未申请成功")
	}
	if err := validateCertFilePath(s.cfg.CertsDir(), rec.CertPath); err != nil {
		return "", "", nil, err
	}
	if err := validateCertFilePath(s.cfg.CertsDir(), rec.KeyPath); err != nil {
		return "", "", nil, err
	}

	certPEM, err := os.ReadFile(rec.CertPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("读取证书失败")
	}
	keyPEM, err := os.ReadFile(rec.KeyPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("读取私钥失败")
	}

	base := downloadBaseName(rec)
	switch DownloadPart(strings.ToLower(strings.TrimSpace(part))) {
	case "", DownloadZIP:
		data, err = zipCertBundle(base, certPEM, keyPEM)
		if err != nil {
			return "", "", nil, err
		}
		return base + ".zip", "application/zip", data, nil
	case DownloadCert:
		return base + "-fullchain.pem", "application/x-pem-file", certPEM, nil
	case DownloadKey:
		return base + "-privatekey.pem", "application/x-pem-file", keyPEM, nil
	default:
		return "", "", nil, fmt.Errorf("不支持的下载类型")
	}
}

func downloadBaseName(rec certstore.Record) string {
	name := strings.TrimPrefix(rec.Domain, "*.")
	name = strings.ReplaceAll(name, "*", "wildcard")
	name = strings.ReplaceAll(name, "/", "_")
	if name == "" {
		name = "certificate"
	}
	return name
}

func validateCertFilePath(certsDir, path string) error {
	absCerts, err := filepath.Abs(certsDir)
	if err != nil {
		return fmt.Errorf("证书目录无效")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("证书路径无效")
	}
	rel, err := filepath.Rel(absCerts, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("证书路径无效")
	}
	return nil
}

func zipCertBundle(base string, certPEM, keyPEM []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := writeZipEntry(zw, "fullchain.pem", certPEM); err != nil {
		return nil, err
	}
	if err := writeZipEntry(zw, "privatekey.pem", keyPEM); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
