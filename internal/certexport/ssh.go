package certexport

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	certstore "github.com/fonu/fonu/internal/certificate"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func syncSSHUpload(ctx context.Context, cfg certstore.SSHExportConfig, certsDir string, storageNames []string, displayName string, remove bool) error {
	if len(storageNames) == 0 {
		return nil
	}
	client, err := sshClientConn(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("SFTP 连接失败：%w", err)
	}
	defer sftpClient.Close()

	var firstErr error
	for _, storageName := range storageNames {
		remoteDir := path.Join(cfg.RemoteDir, storageName)
		if remove {
			if err := sftpClient.RemoveAll(remoteDir); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("删除远程目录 %s：%w", remoteDir, err)
			}
			continue
		}
		localDir := filepath.Join(certsDir, storageName)
		if err := uploadPEMPairSFTP(sftpClient, localDir, remoteDir); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if !remove && strings.TrimSpace(cfg.OnUploadShell) != "" && firstErr == nil {
		meta := sshMetaForStorage(cfg.RemoteDir, storageNames[0], displayName)
		if err := runSSHCommand(client, cfg.OnUploadShell, meta); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type sshCertMeta struct {
	Name     string
	CertDir  string
	CertFile string
	KeyFile  string
}

func sshMetaForStorage(remoteRoot, storageName, displayName string) sshCertMeta {
	certDir := path.Join(remoteRoot, storageName)
	return sshCertMeta{
		Name:     displayName,
		CertDir:  certDir,
		CertFile: path.Join(certDir, certFileName),
		KeyFile:  path.Join(certDir, keyFileName),
	}
}

func uploadPEMPairSFTP(client *sftp.Client, localDir, remoteDir string) error {
	if err := client.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("创建远程目录 %s：%w", remoteDir, err)
	}
	if err := uploadFileSFTP(client, filepath.Join(localDir, certFileName), path.Join(remoteDir, certFileName)); err != nil {
		return err
	}
	if err := uploadFileSFTP(client, filepath.Join(localDir, keyFileName), path.Join(remoteDir, keyFileName)); err != nil {
		return err
	}
	return nil
}

func uploadFileSFTP(client *sftp.Client, localPath, remotePath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("读取 %s：%w", localPath, err)
	}
	remote, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("上传 %s：%w", remotePath, err)
	}
	defer remote.Close()
	if _, err := remote.Write(data); err != nil {
		return fmt.Errorf("写入 %s：%w", remotePath, err)
	}
	return remote.Chmod(0o600)
}

func sshAuthMethods(cfg certstore.SSHExportConfig) ([]ssh.AuthMethod, error) {
	switch cfg.AuthType {
	case certstore.SSHAuthPassword, "":
		if cfg.Password == "" {
			return nil, fmt.Errorf("SSH 密码未配置")
		}
		return []ssh.AuthMethod{ssh.Password(cfg.Password)}, nil
	case certstore.SSHAuthPrivateKey:
		signer, err := loadSSHSigner(cfg)
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	default:
		return nil, fmt.Errorf("SSH 认证方式无效")
	}
}

func loadSSHSigner(cfg certstore.SSHExportConfig) (ssh.Signer, error) {
	var pemBytes []byte
	if cfg.PrivateKey != "" {
		pemBytes = []byte(cfg.PrivateKey)
	} else if cfg.PrivateKeyPath != "" {
		data, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("读取私钥文件：%w", err)
		}
		pemBytes = data
	} else {
		return nil, fmt.Errorf("SSH 私钥未配置")
	}
	if cfg.PrivateKeyPassphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(cfg.PrivateKeyPassphrase))
	}
	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("解析 SSH 私钥：%w", err)
	}
	return signer, nil
}

func runSSHCommand(client *ssh.Client, script string, meta sshCertMeta) error {
	script = strings.TrimSpace(script)
	if script == "" {
		return nil
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	wrapped := fmt.Sprintf(
		"export certName=%s certDir=%s certFile=%s keyFile=%s; %s",
		shellQuote(meta.Name),
		shellQuote(meta.CertDir),
		shellQuote(meta.CertFile),
		shellQuote(meta.KeyFile),
		script,
	)
	out, err := session.CombinedOutput("sh -c " + shellQuote(wrapped))
	if err != nil {
		return fmt.Errorf("远程命令失败：%s", strings.TrimSpace(string(out)))
	}
	_ = out
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
