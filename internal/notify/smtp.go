package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
	"unicode"
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

	// Reject control characters in envelope addresses before opening a connection.
	for _, address := range append([]string{from}, cfg.Email.To...) {
		if strings.IndexFunc(address, unicode.IsControl) >= 0 {
			return fmt.Errorf("SMTP 邮件地址包含非法控制字符")
		}
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

// sanitizeMailHeader keeps untrusted title text on a single logical header.
// Encoded-word folding below is the only place a subject can introduce CRLF.
func sanitizeMailHeader(value string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value))
}

func mailSubjectHeader(title string) string {
	title = sanitizeMailHeader(title)
	encoded := mime.BEncoding.Encode("UTF-8", title)
	if encoded == title {
		return "Subject: " + title
	}
	// Go's MIME encoder splits long UTF-8 subjects into RFC 2047 encoded
	// words of at most 75 characters. Fold only at encoded-word boundaries.
	var b strings.Builder
	b.WriteString("Subject:")
	lineLen := len("Subject:")
	for _, word := range strings.Fields(encoded) {
		if lineLen+1+len(word) > 78 {
			b.WriteString("\r\n ")
			lineLen = 1
		} else {
			b.WriteByte(' ')
			lineLen++
		}
		b.WriteString(word)
		lineLen += len(word)
	}
	return b.String()
}

func buildMailBody(from string, to []string, title, message string) []byte {
	recipients := make([]string, len(to))
	for i, address := range to {
		recipients[i] = sanitizeMailHeader(address)
	}
	headers := []string{
		"From: " + sanitizeMailHeader(from),
		"To: " + strings.Join(recipients, ", "),
		mailSubjectHeader(title),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: quoted-printable",
	}
	bodyText := strings.TrimSpace(message)
	if bodyText == "" {
		bodyText = sanitizeMailHeader(title)
	}
	var body bytes.Buffer
	body.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")
	// A bytes.Buffer cannot return write errors.
	writer := quotedprintable.NewWriter(&body)
	_, _ = writer.Write([]byte(bodyText + "\r\n"))
	_ = writer.Close()
	return body.Bytes()
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
