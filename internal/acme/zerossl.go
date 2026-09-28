package acme

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const zeroSSLEABCredentialsURL = "https://api.zerossl.com/acme/eab-credentials"

type zeroSSLEABResponse struct {
	Success    json.RawMessage `json:"success"`
	EABKid     string          `json:"eab_kid"`
	EABHmacKey string          `json:"eab_hmac_key"`
	Error      struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
}

func zeroSSLAPISuccess(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b
	}
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return n == 1
	}
	return false
}

func zerosslEABCredentials(ctx context.Context, apiKey string) (kid, hmac string, err error) {
	return zerosslEABCredentialsAt(ctx, apiKey, zeroSSLEABCredentialsURL)
}

func zerosslEABCredentialsAt(ctx context.Context, apiKey string, endpoint string) (kid, hmac string, err error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", "", fmt.Errorf("请在设置中配置 ZeroSSL API Key")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "ApiKey "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求 ZeroSSL API 失败：%w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", err
	}

	var payload zeroSSLEABResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		if resp.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("ZeroSSL API 返回 %d", resp.StatusCode)
		}
		return "", "", fmt.Errorf("解析 ZeroSSL 响应失败：%w", err)
	}

	if !zeroSSLAPISuccess(payload.Success) || payload.EABKid == "" || payload.EABHmacKey == "" {
		if msg := zeroSSLAPIErrorMessage(payload, resp.StatusCode); msg != "" {
			return "", "", fmt.Errorf("ZeroSSL API 错误：%s", msg)
		}
		return "", "", fmt.Errorf("ZeroSSL API 未返回有效的 EAB 凭据")
	}
	return payload.EABKid, payload.EABHmacKey, nil
}

func zeroSSLAPIErrorMessage(payload zeroSSLEABResponse, status int) string {
	if payload.Error.Info != "" {
		return payload.Error.Info
	}
	if payload.Error.Type != "" {
		return payload.Error.Type
	}
	if status != http.StatusOK {
		return fmt.Sprintf("HTTP %d", status)
	}
	return ""
}
