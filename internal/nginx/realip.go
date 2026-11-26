package nginx

import (
	"encoding/json"
	"fmt"
 "net/netip"
	"strings"

	"github.com/fonu/fonu/internal/validate"
)

type TrustedProxyConfig struct {
	Enabled    bool     `json:"enabled"`
	Preset     string   `json:"preset"` // "", "cloudflare", "custom"
	CIDRs      []string `json:"cidrs,omitempty"`
	Header     string   `json:"header,omitempty"` // CF-Connecting-IP, X-Real-IP, X-Forwarded-For
	Recursive  bool     `json:"recursive,omitempty"`
}

var cloudflareIPv4 = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
}

var cloudflareIPv6 = []string{
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

func ParseTrustedProxyJSON(raw string) TrustedProxyConfig {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return TrustedProxyConfig{}
	}
	var cfg TrustedProxyConfig
	_ = json.Unmarshal([]byte(raw), &cfg)
	return cfg
}

func (c TrustedProxyConfig) effectiveCIDRs() []string {
	if !c.Enabled {
		return nil
	}
	switch c.Preset {
	case "cloudflare":
		out := make([]string, 0, len(cloudflareIPv4)+len(cloudflareIPv6))
		out = append(out, cloudflareIPv4...)
		out = append(out, cloudflareIPv6...)
		return out
	default:
		return c.CIDRs
	}
}

func (c TrustedProxyConfig) effectiveHeader() string {
	if c.Preset == "cloudflare" {return "CF-Connecting-IP"}
	if c.Header != "" {return c.Header}
	return "X-Forwarded-For"
}

func writeRealIPDirectives(b *strings.Builder, cfg TrustedProxyConfig) {
	writeRealIPComment(b, cfg)
	cidrs := cfg.effectiveCIDRs()
	if len(cidrs) == 0 {
		b.WriteString("    map $remote_addr $fonu_client_ip {\n")
		b.WriteString("        default $remote_addr;\n")
		b.WriteString("    }\n\n")
		return
	}
	for _, cidr := range cidrs {
		b.WriteString("    set_real_ip_from " + cidr + ";\n")
	}
	header := cfg.effectiveHeader()
	b.WriteString("    real_ip_header " + header + ";\n")
	if header == "X-Forwarded-For" {
		b.WriteString("    real_ip_recursive on;\n")
	}
	b.WriteString("\n    map $remote_addr $fonu_client_ip {\n")
	b.WriteString("        default $remote_addr;\n")
	b.WriteString("    }\n\n")
}

// Validate before emitting directives, including values loaded from old DBs
// or imported backups. Never silently interpolate invalid trusted proxies.
func validateGenerateSecurity(opts GenerateOptions)error{
	for _,list:=range [][]string{opts.GlobalIPBlacklist,opts.GlobalIPWhitelist,opts.TrustedProxy.effectiveCIDRs()}{
		if err:=validate.IPOrCIDRList(list);err!=nil{return fmt.Errorf("全局 IP 安全配置无效：%w",err)}
	}
	return opts.TrustedProxy.Validate()
}
// Validate protects both saved settings and generated Nginx configuration.
func(c TrustedProxyConfig) Validate()error{
 if !c.Enabled{return nil}
 if c.Preset!="cloudflare"&&c.Preset!="custom"&&c.Preset!=""{return fmt.Errorf("不支持的信任代理预设")}
 switch c.effectiveHeader(){case "CF-Connecting-IP","X-Real-IP","X-Forwarded-For":default:return fmt.Errorf("不支持的真实 IP 请求头")}
 cidrs:=c.effectiveCIDRs()
 if len(cidrs)==0||len(cidrs)>256{return fmt.Errorf("请配置 1–256 个可信代理 IP 或 CIDR")}
 for _,raw:=range cidrs{
  normalized,err:=validate.IPOrCIDR(raw);if err!=nil{return fmt.Errorf("无效的可信代理地址")}
  if prefix,err:=netip.ParsePrefix(normalized);err==nil&&prefix.Bits()==0{return fmt.Errorf("信任代理不能信任全部公网地址")}
 }
 return nil
}
