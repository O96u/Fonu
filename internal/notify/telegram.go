package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func SendTelegram(ctx context.Context, client *http.Client, cfg RuntimeConfig, content AlertContent) error {
	token := strings.TrimSpace(cfg.TelegramToken)
	if token == "" {
		return fmt.Errorf("Telegram Bot Token 未配置")
	}
	chatID := strings.TrimSpace(cfg.Telegram.ChatID)
	if chatID == "" {
		return fmt.Errorf("Telegram Chat ID 未配置")
	}

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       content.HTMLBody,
		"parse_mode": "HTML",
	}
	body, _ := json.Marshal(payload)
	endpoint := "https://api.telegram.org/bot" + token + "/sendMessage"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("Telegram 请求构造失败")
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if proxyURL, err := normalizeProxyURL(cfg.Telegram.ProxyURL); err != nil {
		return err
	} else if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return fmt.Errorf("Telegram 代理地址无效")
		}
		base := client
		if base == nil {
			base = &http.Client{Timeout: 15 * time.Second}
		}
		transport := cloneHTTPTransport(base)
		transport.Proxy = http.ProxyURL(parsed)
		defer transport.CloseIdleConnections()
		httpClient = &http.Client{
			Timeout:   base.Timeout,
			Transport: transport,
		}
		if httpClient.Timeout == 0 {
			httpClient.Timeout = 15 * time.Second
		}
	}

	return doRequest(httpClient, req)
}
