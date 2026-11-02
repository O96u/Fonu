package iplocate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const defaultRemoteBatchURL = "http://ip-api.com/batch?fields=status,country,regionName,city,query&lang=zh-CN"

type remoteBatchItem struct {
	Status     string `json:"status"`
	Country    string `json:"country"`
	RegionName string `json:"regionName"`
	City       string `json:"city"`
	Query      string `json:"query"`
}

// RemoteEnrichBatch queries remote API for IPs (best-effort). Returns map ip -> Region.
func (s *Service) RemoteEnrichBatch(ctx context.Context, url string, ips []string) map[string]Region {
	out := map[string]Region{}
	if len(ips) == 0 {
		return out
	}
	if strings.TrimSpace(url) == "" {
		url = defaultRemoteBatchURL
	}
	payload, err := json.Marshal(ips)
	if err != nil {
		return out
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.client
	if client == nil {
		client = &http.Client{Timeout: 1200 * time.Millisecond}
	}
	resp, err := client.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return out
	}
	var items []remoteBatchItem
	if err := json.Unmarshal(body, &items); err != nil {
		return out
	}
	for _, item := range items {
		if strings.ToLower(item.Status) != "success" {
			continue
		}
		ip := strings.TrimSpace(item.Query)
		if ip == "" {
			continue
		}
		country := strings.TrimSpace(item.Country)
		province := normalizeProvinceKey(item.RegionName)
		city := strings.TrimSpace(item.City)
		label := formatRemoteLabel(country, province, city)
		out[ip] = Region{
			Country:  country,
			Province: province,
			City:     city,
			Label:    label,
		}
	}
	return out
}

func formatRemoteLabel(country, province, city string) string {
	if country == "中国" || country == "China" {
		return FormatRegion(fmt.Sprintf("中国|0|%s|%s|0", province, city))
	}
	if country == "" {
		return ""
	}
	if cn, ok := countryNames[strings.ToLower(country)]; ok {
		country = cn
	}
	if province != "" && city != "" && province != city {
		return country + " · " + province + " · " + city
	}
	if province != "" {
		return country + " · " + province
	}
	return country
}

// IsPublicIP reports whether ip should be geo-enriched.
func IsPublicIP(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	if parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() {
		return false
	}
	return true
}
