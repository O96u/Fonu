package validate

import (
	"fmt"
	"net"
	"strings"
)

func IPOrCIDR(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("IP 或 CIDR 不能为空")
	}
	if strings.Contains(raw, "/") {
		_, ipNet, err := net.ParseCIDR(raw)
		if err != nil {
			return "", fmt.Errorf("CIDR 格式无效")
		}
		return ipNet.String(), nil
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return "", fmt.Errorf("IP 格式无效")
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String(), nil
	}
	return ip.String(), nil
}

func IPOrCIDRList(list []string) error {
	for _, item := range list {
		if _, err := IPOrCIDR(item); err != nil {
			return err
		}
	}
	return nil
}
