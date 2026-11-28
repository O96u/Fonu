package notify

import (
	"bytes"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
)

func TestBuildMailBody(t *testing.T) {
	content := FormatAlert("test", "标题", "正文")
	raw := buildMailBody("from@example.com", []string{"to@example.com", "other@example.com"}, content.Subject, content.PlainBody)
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.Header.Get("Subject"), "=?UTF-8?b?") {
		t.Fatalf("subject not RFC 2047 encoded: %s", m.Header.Get("Subject"))
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject"))
	if err != nil || subject != content.Subject {
		t.Fatalf("decoded subject = %q, err = %v", subject, err)
	}
	if m.Header.Get("Content-Transfer-Encoding") != "quoted-printable" {
		t.Fatal("missing body transfer encoding")
	}
	body, err := io.ReadAll(quotedprintable.NewReader(m.Body))
	if err != nil || strings.ReplaceAll(strings.TrimSpace(string(body)), "\r\n", "\n") != content.PlainBody {
		t.Fatalf("decoded body = %q, err = %v", body, err)
	}
	for _, b := range raw {
		if b >= 128 {
			t.Fatal("message is not 7-bit transport safe")
		}
	}
}

func TestMailSubjectEncodingAndFolding(t *testing.T) {
	for _, title := range []string{
		"[Fonu] DDNS update",
		"[Fonu] ⚠️ DDNS 更新失败",
		"[Fonu] " + strings.Repeat("证书续签提醒🌐", 30),
	} {
		raw := buildMailBody("from@example.com", []string{"to@example.com"}, title, "")
		m, err := mail.ReadMessage(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		subject, err := new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject"))
		if err != nil || subject != title {
			t.Fatalf("subject round trip: %q != %q, err=%v", subject, title, err)
		}
		headers := strings.SplitN(string(raw), "\r\n\r\n", 2)[0]
		for _, line := range strings.Split(headers, "\r\n") {
			if len(line) > 78 {
				t.Fatalf("header line too long: %d", len(line))
			}
			for _, word := range strings.Fields(line) {
				if strings.HasPrefix(word, "=?") && len(word) > 75 {
					t.Fatalf("encoded word too long: %d", len(word))
				}
			}
		}
	}
}

func TestMailSubjectCannotInjectHeaders(t *testing.T) {
	title := "测试\r\nBcc: injected@example.com\x00"
	raw := buildMailBody("from@example.com", []string{"to@example.com"}, title, "正文")
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if m.Header.Get("Bcc") != "" {
		t.Fatal("injected Bcc header")
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject"))
	if err != nil || subject != sanitizeMailHeader(title) {
		t.Fatalf("unexpected subject: %q %v", subject, err)
	}
}
