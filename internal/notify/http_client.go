package notify

import (
	"net"
	"net/http"
	"time"
)

func cloneHTTPTransport(client *http.Client) *http.Transport {
	if client != nil && client.Transport != nil {
		if transport, ok := client.Transport.(*http.Transport); ok && transport != nil {
			return transport.Clone()
		}
	}
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
