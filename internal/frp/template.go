package frp

import "fmt"

// BuildFRPSConfig generates frps.toml for the user's VPS, aligned with Fonu Nginx ports and auth.
func BuildFRPSConfig(bindPort, httpPort, httpsPort int, token string) string {
	if bindPort <= 0 {
		bindPort = 7000
	}
	if httpPort <= 0 {
		httpPort = 80
	}
	if httpsPort <= 0 {
		httpsPort = 443
	}
	if token == "" {
		token = "your-secret-token"
	}
	return fmt.Sprintf(`bindAddr = "0.0.0.0"
bindPort = %d

vhostHTTPPort = %d
vhostHTTPSPort = %d

[auth]
method = "token"
token = "%s"
`, bindPort, httpPort, httpsPort, token)
}
