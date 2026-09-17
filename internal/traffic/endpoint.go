package traffic

import (
	"fmt"
	"strings"
)

func EndpointKey(host string, port int) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if i := strings.Index(host, ":"); i > 0 {
		host = host[:i]
	}
	if port <= 0 {
		return host
	}
	return fmt.Sprintf("%s:%d", host, port)
}
