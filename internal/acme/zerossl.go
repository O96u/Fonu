package acme

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type zeroSSLEABResponse struct {
	Success    int    `json:"success"`
	EABKid     string `json:"eab_kid"`
	EABHmacKey string `json:"eab_hmac_key"`
	Error      struct {
		Code int    `json:"code"`
		Type string `json:"type"`
	} `json:"error"`
}

func zerosslEABCredentials(ctx context.Context, apiKey string) (kid, hmac string, err error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", "", fmt.Errorf("请在设置中配置 ZeroSSL API Key")
	}

	endpoint := "https://api.zerossl.com/acme/eab-credentials?access_key=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}

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
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("ZeroSSL API 返回 %d", resp.StatusCode)
	}

	var payload zeroSSLEABResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", fmt.Errorf("解析 ZeroSSL 响应失败：%w", err)
	}
	if payload.Success != 1 || payload.EABKid == "" || payload.EABHmacKey == "" {
		if payload.Error.Type != "" {
			return "", "", fmt.Errorf("ZeroSSL API 错误：%s", payload.Error.Type)
		}
		return "", "", fmt.Errorf("ZeroSSL API 未返回有效的 EAB 凭据")
	}
	return payload.EABKid, payload.EABHmacKey, nil
}
