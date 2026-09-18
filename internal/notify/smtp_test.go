package notify

import (
	"strings"
	"testing"
)

func TestBuildMailBody(t *testing.T) {
	content := FormatAlert("test", "标题", "正文")
	body := string(buildMailBody("from@example.com", []string{"to@example.com"}, content.Subject, content.PlainBody))
	if !strings.Contains(body, "Subject: [Fonu]") {
		t.Fatalf("missing subject: %s", body)
	}
	if !strings.Contains(body, "正文") {
		t.Fatalf("missing message: %s", body)
	}
}
