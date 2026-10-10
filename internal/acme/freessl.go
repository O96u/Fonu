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

const (
	freesslDirectoryURLLegacy = "https://acme.freessl.cn/v2/DV90/directory"
	freesslDirectoryURLPro    = "https://acmepro.freessl.cn/v2/DV"
)

type freeSSLEABResponse struct {
	EABKid     string `json:"eab_kid"`
	EABHmacKey string `json:"eab_hmac_key"`
	Kid        string `json:"kid"`
	Hmac       string `json:"hmac"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}

func resolveFreeSSLDirectoryURL(customURL, automationToken string) string {
	customURL = strings.Trim(strings.TrimSpace(customURL), "`")
	if customURL != "" {
		return customURL
	}
	token := strings.TrimSpace(automationToken)
	if token != "" {
		return freesslDirectoryURLLegacy + "/" + token
	}
	return freesslDirectoryURLPro
}

func resolveFreeSSLEAB(ctx context.Context, automationToken, configuredKid, configuredHmac string) (kid, hmac string, err error) {
	configuredKid = strings.TrimSpace(configuredKid)
	configuredHmac = strings.TrimSpace(configuredHmac)
	if configuredKid != "" && configuredHmac != "" {
		return configuredKid, configuredHmac, nil
	}
	if configuredKid != "" || configuredHmac != "" {
		return "", "", fmt.Errorf("请同时配置 FreeSSL EAB Kid 与 Hmac，或填写 Automation Token 自动获取")
	}
	token := strings.TrimSpace(automationToken)
	if token == "" {
		return "", "", fmt.Errorf("请配置 FreeSSL Automation Token（freessl.cn 自动化 / EAB 管理页），或在设置中填写 EAB Kid 与 Hmac")
	}
	kid, hmac, err = fetchFreeSSLEAB(ctx, token)
	if err != nil {
		return "", "", err
	}
	return kid, hmac, nil
}

func fetchFreeSSLEAB(ctx context.Context, token string) (kid, hmac string, err error) {
	endpoints := []string{
		"https://freessl.cn/api/acme/eab?token=" + token,
		"https://api.freessl.cn/acme/eab?token=" + token,
	}
	var lastErr error
	for _, endpoint := range endpoints {
		kid, hmac, err = fetchFreeSSLEABAt(ctx, endpoint)
		if err == nil && kid != "" && hmac != "" {
			return kid, hmac, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", "", fmt.Errorf("无法通过 Automation Token 获取 FreeSSL EAB：%v（仍可在设置中手动填写 EAB）", lastErr)
	}
	return "", "", fmt.Errorf("FreeSSL 未返回有效 EAB，请在设置中手动填写 EAB Kid 与 Hmac")
}

func fetchFreeSSLEABAt(ctx context.Context, endpoint string) (kid, hmac string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var parsed freeSSLEABResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", err
	}
	kid = strings.TrimSpace(parsed.EABKid)
	hmac = strings.TrimSpace(parsed.EABHmacKey)
	if kid == "" {
		kid = strings.TrimSpace(parsed.Kid)
	}
	if hmac == "" {
		hmac = strings.TrimSpace(parsed.Hmac)
	}
	if kid == "" || hmac == "" {
		return "", "", fmt.Errorf("响应缺少 EAB 字段")
	}
	return kid, hmac, nil
}
