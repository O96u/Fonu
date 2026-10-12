package nginx

import (
	"fmt"
	"strings"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

type listenEndpoint struct {
	Port     int
	SSL      bool
	IPv4     bool
	IPv6     bool
	Hostnames []string
	SSLCert  *certFiles
}

func collectListenEndpoints(rules []proxy.Rule, certsDir string, certs []CertSource) []listenEndpoint {
	merged := map[string]*listenEndpoint{}
	order := make([]string, 0)

	for _, rule := range rules {
		if !rule.Enabled || rule.NginxMode == "custom" {
			continue
		}
		for _, group := range rule.PortGroups() {
			if len(group.Hostnames) == 0 {
				continue
			}
			cert := findCertificateForHosts(certsDir, group.Hostnames, certs)
			useHTTPS := rule.HTTPSEnabled && certUsable(cert)

			key := listenEndpointKey(group.Port, useHTTPS, rule.ListenIPv4, rule.ListenIPv6)
			ep := merged[key]
			if ep == nil {
				ep = &listenEndpoint{
					Port:  group.Port,
					SSL:   useHTTPS,
					IPv4:  rule.ListenIPv4,
					IPv6:  rule.ListenIPv6,
				}
				merged[key] = ep
				order = append(order, key)
			} else {
				ep.IPv4 = ep.IPv4 || rule.ListenIPv4
				ep.IPv6 = ep.IPv6 || rule.ListenIPv6
			}
			ep.Hostnames = append(ep.Hostnames, group.Hostnames...)
			if useHTTPS && certUsable(cert) {
				ep.SSLCert = preferDefaultSSLCert(ep.SSLCert, cert)
			}
		}
	}

	out := make([]listenEndpoint, 0, len(order))
	for _, key := range order {
		out = append(out, *merged[key])
	}
	return out
}

func listenEndpointKey(port int, ssl bool, ipv4 bool, ipv6 bool) string {
	return fmt.Sprintf("%d|%t|%t|%t", port, ssl, ipv4, ipv6)
}

func preferDefaultSSLCert(current, candidate *certFiles) *certFiles {
	if !certUsable(candidate) {
		return current
	}
	if !certUsable(current) {
		return candidate
	}
	if strings.Contains(current.CertPath, "wildcard") {
		return current
	}
	if strings.Contains(candidate.CertPath, "wildcard") {
		return candidate
	}
	return current
}

func writeUnmatchedHostCatchAllServers(b *strings.Builder, cfg config.Config, endpoints []listenEndpoint) {
	for _, ep := range endpoints {
		if ep.Port <= 0 {
			continue
		}
		if !ep.IPv4 && !ep.IPv6 {
			continue
		}
		if ep.SSL && !certUsable(ep.SSLCert) {
			writeSSLRejectCatchAll(b, ep)
			continue
		}
		writeCommentLine(b, 0, fmt.Sprintf("未配置域名（监听 %d%s）：返回 404", ep.Port, sslLabel(ep.SSL)))
		b.WriteString("server {\n")
		writeListenDirectives(b, ep.Port, ep.SSL, ep.IPv4, ep.IPv6, true)
		b.WriteString("    server_name _;\n\n")
		if ep.SSL {
			writeCommentLine(b, 4, "默认证书（仅用于未匹配域名的 TLS 握手）")
			b.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", ep.SSLCert.CertPath))
			b.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", ep.SSLCert.KeyPath))
			b.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n\n")
		}
		writeErrorPages(b, cfg)
		b.WriteString("    location / {\n        return 404;\n    }\n")
		b.WriteString("}\n\n")
	}
}

func writeSSLRejectCatchAll(b *strings.Builder, ep listenEndpoint) {
	writeCommentLine(b, 0, fmt.Sprintf("未配置域名（监听 %d HTTPS）：拒绝 TLS 握手", ep.Port))
	b.WriteString("server {\n")
	writeListenDirectives(b, ep.Port, true, ep.IPv4, ep.IPv6, true)
	b.WriteString("    server_name _;\n")
	b.WriteString("    ssl_reject_handshake on;\n")
	b.WriteString("}\n\n")
}

func sslLabel(ssl bool) string {
	if ssl {
		return " HTTPS"
	}
	return ""
}
