package service

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/fonu/fonu/internal/proxy"
)

// AccessBlacklistRequest binds a historical log endpoint to the current rule.
// Mutation requires the same rule ID the administrator inspected and confirmed.
type AccessBlacklistRequest struct {
	Host string `json:"host"`
	Port int `json:"port"`
	IP string `json:"ip"`
	Action string `json:"action"`
	RuleID int64 `json:"rule_id,omitempty"`
}

type AccessBlacklistResult struct {
	RuleID int64 `json:"rule_id"`
	RuleName string `json:"rule_name"`
	IP string `json:"ip"`
	Exact bool `json:"exact"`
	Covered bool `json:"covered"`
	Changed bool `json:"changed"`
	WhitelistMode bool `json:"whitelist_mode"`
}

func accessBlacklistIP(raw string) (netip.Addr, error) {
	ip, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil || ip.Zone() != "" {
		return netip.Addr{}, fmt.Errorf("只允许单个有效的 IPv4 或 IPv6 地址")
	}
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return netip.Addr{}, fmt.Errorf("不允许快捷封禁回环、链路本地、组播或未指定地址")
	}
	return ip, nil
}

func accessBlacklistRule(rules []proxy.Rule, host string, port int) (proxy.Rule, error) {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" || port < 1 || port > 65535 {
		return proxy.Rule{}, fmt.Errorf("日志缺少明确域名或监听端口，请在反代规则中手动管理黑名单")
	}
	var matches []proxy.Rule
	for _, rule := range rules {
		if !rule.Enabled { continue }
		for _, endpoint := range rule.Endpoints() {
			if endpoint.Port == port && strings.EqualFold(strings.TrimSuffix(endpoint.Hostname, "."), host) {
				matches = append(matches, rule)
				break
			}
		}
	}
	if len(matches) != 1 {
		return proxy.Rule{}, fmt.Errorf("无法唯一匹配当前启用的反代规则，请在反代页面手动确认")
	}
	rule := matches[0]
	if rule.NginxMode == "custom" {
		return proxy.Rule{}, fmt.Errorf("该规则使用手动 Nginx 配置，快捷黑名单不会可靠生效，请手动管理")
	}
	return rule, nil
}

// exactIP treats a host-length CIDR as the same single address. Wider prefixes
// are never removed by the quick action.
func exactBlacklistIP(raw string, ip netip.Addr) bool {
	if addr, err := netip.ParseAddr(strings.TrimSpace(raw)); err == nil {
		return addr.Unmap() == ip
	}
	prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
	return err == nil && prefix.Bits() == prefix.Addr().BitLen() && prefix.Addr().Unmap() == ip
}

func blacklistCoverage(list []string, ip netip.Addr) (exact, covered bool) {
	for _, raw := range list {
		if exactBlacklistIP(raw, ip) {
			exact = true
			continue
		}
		if prefix, err := netip.ParsePrefix(strings.TrimSpace(raw)); err == nil && prefix.Contains(ip) {
			covered = true
		}
	}
	return
}

func blacklistResult(rule proxy.Rule, ip netip.Addr) AccessBlacklistResult {
	name := strings.TrimSpace(rule.Name)
	if name == "" { name = rule.Domain }
	exact, covered := blacklistCoverage(rule.Security.IPBlacklist, ip)
	return AccessBlacklistResult{RuleID:rule.ID, RuleName:name, IP:ip.String(), Exact:exact, Covered:covered, WhitelistMode:rule.Security.WhitelistEnabled()}
}

func (s *ProxyService) AccessBlacklist(ctx context.Context, req AccessBlacklistRequest) (AccessBlacklistResult, error) {
	ip, err := accessBlacklistIP(req.IP)
	if err != nil { return AccessBlacklistResult{}, err }
	if req.Action != "inspect" && req.Action != "block" && req.Action != "unblock" {
		return AccessBlacklistResult{}, fmt.Errorf("无效的黑名单操作")
	}
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	if err := s.recoverPendingLocked(ctx); err != nil { return AccessBlacklistResult{}, err }
	rules, err := s.store.List(ctx)
	if err != nil { return AccessBlacklistResult{}, err }
	rule, err := accessBlacklistRule(rules, req.Host, req.Port)
	if err != nil { return AccessBlacklistResult{}, err }
	result := blacklistResult(rule, ip)
	if req.Action == "inspect" { return result, nil }
	if req.RuleID != rule.ID { return AccessBlacklistResult{}, fmt.Errorf("对应规则已变化，请重新检查日志并确认") }
	if req.Action == "block" && (result.Exact || result.Covered) || req.Action == "unblock" && !result.Exact {
		return result, nil
	}
	updated, err := mutateProxy(ctx, s, func(store *proxy.Store) (proxy.Rule,error) {
		current, err := store.Get(ctx, rule.ID)
		if err != nil { return proxy.Rule{},err }
		list := make([]string,0,len(current.Security.IPBlacklist)+1)
		for _, raw := range current.Security.IPBlacklist {
			if req.Action == "unblock" && exactBlacklistIP(raw,ip) {continue}
			list=append(list,raw)
		}
		if req.Action == "block" {list=append(list,ip.String())}
		return store.Update(ctx,rule.ID,proxy.UpdateInput{Security:&proxy.SecurityInput{IPBlacklist:list}})
	})
	if err != nil { return AccessBlacklistResult{},err }
	result=blacklistResult(updated,ip)
	result.Changed=true
	return result,nil
}
