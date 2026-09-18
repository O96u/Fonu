package notify

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

func SendEmail(cfg RuntimeConfig, title, message string) error {
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
	addr := fmt.Sprintf("%s:%d", cfg.Email.Host, cfg.Email.Port)
	auth := smtp.PlainAuth("", cfg.Email.Username, cfg.SMTPPassword, cfg.Email.Host)

	if cfg.Email.Port == 465 {
		return sendSMTPS(addr, cfg.Email.Host, auth, from, cfg.Email.To, body)
	}
	return smtp.SendMail(addr, auth, from, cfg.Email.To, body)
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

func sendSMTPS(addr, host string, auth smtp.Auth, from string, to []string, body []byte) error {
	tlsCfg := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(strings.TrimSpace(rcpt)); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(body); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
