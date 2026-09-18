package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendCustomWebhook(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := RuntimeConfig{
		Config: Config{
			Type: NotifyTypeWebhook,
			Webhook: WebhookConfig{
				Provider: WebhookCustom,
				URL:      server.URL,
			},
		},
	}
	content := FormatAlert("test", "title", "message")
	if err := SendWebhook(context.Background(), server.Client(), cfg, content); err != nil {
		t.Fatalf("send custom webhook: %v", err)
	}
	if !strings.Contains(body, `"title":"title"`) || !strings.Contains(body, `"message":"message"`) {
		t.Fatalf("unexpected body: %s", body)
	}
	if !strings.Contains(body, `"formatted":`) {
		t.Fatalf("missing formatted body: %s", body)
	}
}

func TestSendNtfyUsesTopic(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := RuntimeConfig{
		Config: Config{
			Type: NotifyTypeWebhook,
			Webhook: WebhookConfig{
				Provider: WebhookNtfy,
				Server:   server.URL,
				Topic:    "alerts",
			},
		},
	}
	if err := SendWebhook(context.Background(), server.Client(), cfg, FormatAlert("test", "title", "message")); err != nil {
		t.Fatalf("send ntfy: %v", err)
	}
	if gotPath != "/alerts" {
		t.Fatalf("expected /alerts got %s", gotPath)
	}
}

func TestJoinURL(t *testing.T) {
	got := joinURL("https://ntfy.sh", "topic")
	if got != "https://ntfy.sh/topic" {
		t.Fatalf("unexpected url: %s", got)
	}
}
