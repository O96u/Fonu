package notify

import (
	"context"
	"crypto/tls"
	"net"
	"strconv"
	"time"
	"fmt"
	"net/smtp"
	"strings"
)

func SendEmail(cfg RuntimeConfig, title, message string) error {
	return sendEmailContext(context.Background(), cfg, title, message)
}

func sendEmailContext(ctx context.Context, cfg RuntimeConfig, title, message string) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if strings.TrimSpace(cfg.Email.Host) == "" {
		return fmt.Errorf("SMTP 主机未配置")
	}
	if len(cfg.Email.To) == 0 {
		return fmt.Errorf("收件人未配置")
	}
	if strings.TrimSpace(cfg.SMTPPassword) == "" && strings.TrimSpace(cfg.Email.Username) != "" {
		return fmt.Errorf("SMTP 密码未配置")
	}

	from := strings.TrimSpace(cfg.Email.From)
	if from == "" {
		from = strings.TrimSpace(cfg.Email.Username)
	}
	if from == "" {
		return fmt.Errorf("发件人未配置")
	}

	body := buildMailBody(from, cfg.Email.To, title, message)
	host := strings.Trim(cfg.Email.Host, "[]")
	if cfg.Email.Port < 1 || cfg.Email.Port > 65535 {
		return fmt.Errorf("SMTP 端口无效")
	}
	addr := net.JoinHostPort(host, strconv.Itoa(cfg.Email.Port))
	var auth smtp.Auth
	if cfg.Email.Username != "" {
		auth = smtp.PlainAuth("", cfg.Email.Username, cfg.SMTPPassword, host)
	}
	return sendSMTPContext(ctx, addr, host, auth, from, cfg.Email.To, body, cfg.Email.Port == 465, cfg.Email.TLS)
}

func buildMailBody(from string, to []string, title, message string) []byte {
	headers := []string{
		"From: " + from,
		"To: " + strings.Join(to, ", "),
		"Subject: " + title,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}
	bodyText := strings.TrimSpace(message)
	if bodyText == "" {
		bodyText = title
	}
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + bodyText + "\r\n")
}

func sendSMTPContext(ctx context.Context, addr, host string, auth smtp.Auth, from string, to []string, body []byte, implicitTLS, requireTLS bool) error {
	raw, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("SMTP 连接失败")
	}
	defer raw.Close()
	stop := context.AfterFunc(ctx, func() { _ = raw.Close() })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		if err := raw.SetDeadline(deadline); err != nil { return err }
	}
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	var conn net.Conn = raw
	if implicitTLS {
		tlsConn := tls.Client(raw, tlsCfg)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("SMTP TLS 握手失败")
		}
		conn = tlsConn
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP 握手失败")
	}
	defer client.Close()
	if !implicitTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("SMTP STARTTLS 失败")
			}
		} else if requireTLS {
			return fmt.Errorf("SMTP 服务不支持 STARTTLS")
		}
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 操作失败，请检查配置及服务状态")
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP 操作失败，请检查配置及服务状态")
	}
	for _, rcpt := range to {
		if err := client.Rcpt(strings.TrimSpace(rcpt)); err != nil {
			return fmt.Errorf("SMTP 操作失败，请检查配置及服务状态")
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP 操作失败，请检查配置及服务状态")
	}
	if _, err := writer.Write(body); err != nil {
		return fmt.Errorf("SMTP 操作失败，请检查配置及服务状态")
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP 操作失败，请检查配置及服务状态")
	}
	if err := client.Quit(); err != nil { return fmt.Errorf("SMTP 结束会话失败") }; return nil
}
