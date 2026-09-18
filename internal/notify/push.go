package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func SendWebhook(ctx context.Context, client *http.Client, cfg RuntimeConfig, content AlertContent) error {
	switch cfg.Webhook.Provider {
	case WebhookBark:
		return sendBark(ctx, client, cfg, content.PushTitle, content.barkMessage())
	case WebhookNtfy:
		return sendNtfy(ctx, client, cfg, content.PushTitle, content.PlainBody)
	case WebhookGotify:
		return sendGotify(ctx, client, cfg, content.PushTitle, content.PlainBody)
	case WebhookCustom:
		return sendCustomWebhook(ctx, client, cfg, content)
	default:
		return fmt.Errorf("不支持的 Webhook 预设")
	}
}

func webhookSecret(cfg RuntimeConfig) string {
	if strings.TrimSpace(cfg.WebhookSecret) != "" {
		return strings.TrimSpace(cfg.WebhookSecret)
	}
	return strings.TrimSpace(cfg.Webhook.Key)
}

func sendBark(ctx context.Context, client *http.Client, cfg RuntimeConfig, title, message string) error {
	key := webhookSecret(cfg)
	if key == "" {
		return fmt.Errorf("Bark Device Key 未配置")
	}
	server := normalizeServerURL(cfg.Webhook.Server)
	if server == "" {
		server = DefaultBarkServer
	}
	endpoint := fmt.Sprintf("%s/%s/%s/%s", server, url.PathEscape(key), url.PathEscape(title), url.PathEscape(message))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	return doRequest(client, req)
}

func sendNtfy(ctx context.Context, client *http.Client, cfg RuntimeConfig, title, message string) error {
	topic := strings.TrimSpace(cfg.Webhook.Topic)
	if topic == "" {
		return fmt.Errorf("ntfy Topic 未配置")
	}
	server := normalizeServerURL(cfg.Webhook.Server)
	if server == "" {
		server = "https://ntfy.sh"
	}
	endpoint := joinURL(server, topic)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Title", title)
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if token := webhookSecret(cfg); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return doRequest(client, req)
}

func sendGotify(ctx context.Context, client *http.Client, cfg RuntimeConfig, title, message string) error {
	token := webhookSecret(cfg)
	if token == "" {
		return fmt.Errorf("Gotify App Token 未配置")
	}
	server := normalizeServerURL(cfg.Webhook.Server)
	if server == "" {
		return fmt.Errorf("Gotify 服务地址未配置")
	}
	endpoint := joinURL(server, "message") + "?token=" + url.QueryEscape(token)
	payload := map[string]any{
		"title":    title,
		"message":  message,
		"priority": 5,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return doRequest(client, req)
}

func sendCustomWebhook(ctx context.Context, client *http.Client, cfg RuntimeConfig, content AlertContent) error {
	target := strings.TrimSpace(cfg.Webhook.URL)
	if target == "" {
		return fmt.Errorf("Webhook URL 未配置")
	}
	body, _ := json.Marshal(content.webhookPayload())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return doRequest(client, req)
}

func doRequest(client *http.Client, req *http.Request) error {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if len(body) > 0 {
			return fmt.Errorf("请求失败 (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return fmt.Errorf("请求失败 (%d)", resp.StatusCode)
	}
	return nil
}

func normalizeServerURL(server string) string {
	server = strings.TrimSpace(server)
	server = strings.TrimRight(server, "/")
	if server == "" {
		return ""
	}
	if !strings.Contains(server, "://") {
		server = "https://" + server
	}
	return server
}

func joinURL(base string, parts ...string) string {
	base = strings.TrimRight(normalizeServerURL(base), "/")
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part != "" {
			base += "/" + part
		}
	}
	return base
}
