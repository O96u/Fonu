package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/settings"
)

const (
	EventDDNSError  = "ddns_error"
	EventCertError  = "cert_error"
	EventNginxError = "nginx_error"
)

type Service struct {
	settings *settings.Store
	client   *http.Client
}

func New(settings *settings.Store) *Service {
	return &Service{
		settings: settings,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Service) Alert(ctx context.Context, event, title, message string) {
	if s == nil {
		return
	}
	if !s.enabledFor(ctx, event) {
		return
	}
	url, err := s.settings.Get(ctx, settings.KeyNotifyWebhookURL)
	if err != nil || strings.TrimSpace(url) == "" {
		return
	}
	payload := map[string]string{
		"event":   event,
		"title":   title,
		"message": message,
		"time":    time.Now().UTC().Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (s *Service) enabledFor(ctx context.Context, event string) bool {
	switch event {
	case EventDDNSError:
		return s.flag(ctx, settings.KeyNotifyOnDDNSError, true)
	case EventCertError:
		return s.flag(ctx, settings.KeyNotifyOnCertError, true)
	case EventNginxError:
		return s.flag(ctx, settings.KeyNotifyOnNginxError, false)
	default:
		return true
	}
}

func (s *Service) flag(ctx context.Context, key string, fallback bool) bool {
	raw, err := s.settings.Get(ctx, key)
	if err != nil || raw == "" {
		return fallback
	}
	return raw == "1" || strings.EqualFold(raw, "true")
}
