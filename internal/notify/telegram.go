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
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if proxyURL := strings.TrimSpace(cfg.Telegram.ProxyURL); proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return fmt.Errorf("Telegram 代理地址无效")
		}
		httpClient = &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(parsed),
			},
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if len(raw) > 0 {
			return fmt.Errorf("Telegram 请求失败 (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		return fmt.Errorf("Telegram 请求失败 (%d)", resp.StatusCode)
	}
	return nil
}
