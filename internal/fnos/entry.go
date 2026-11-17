package fnos

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OnNAS reports whether this process is the fnOS package.
// LAN versus ZeroTier does not matter: the browser's host is used as-is.
func OnNAS() bool {
	return strings.TrimSpace(os.Getenv("TRIM_APPDEST")) != "" || strings.TrimSpace(os.Getenv("TRIM_APPNAME")) != ""
}

// DetectLoginURL is the fnOS page the browser should open for NAS login.
// On the NAS package it stays on the host already in the address bar.
// Otherwise it uses a Feiniu service found on a local subnet, such as ZeroTier.
func DetectLoginURL(r *http.Request) string {
	if r == nil {
		return ""
	}
	if OnNAS() {
		if target := LoginRedirect(r, ""); target != "" {
			return strings.TrimRight(target, "/") + "/login"
		}
	}
	origin, ok := Find(r.Context(), requestHostname(r))
	if !ok {
		return ""
	}
	return strings.TrimRight(origin, "/") + Prefix() + "/login"
}

// LoginRedirect is where the browser should open the fnOS gateway.
// FONU_FNOS_URL is an operator setting. On the NAS package, the host is the
// one the browser already used, so a ZeroTier address is not rewritten to a
// LAN address. Nothing in here scans the local network.
func LoginRedirect(r *http.Request, saved string) string {
	if base := configuredBase(); base != "" {
		return base + Prefix() + "/"
	}
	reqHost := ""
	if r != nil {
		reqHost = requestHostname(r)
	}
	if saved = strings.TrimSpace(saved); saved != "" {
		if norm, err := NormalizeGatewayURL(saved, reqHost); err == nil {
			if !OnNAS() || (reqHost != "" && sameHostname(norm, reqHost)) {
				return norm
			}
		}
	}
	if !OnNAS() || r == nil || reqHost == "" {
		return ""
	}
	port, https, ok := localGateway()
	if !ok {
		return ""
	}
	scheme := "http"
	if https {
		scheme = "https"
	}
	return scheme + "://" + net.JoinHostPort(reqHost, strconv.Itoa(port)) + Prefix() + "/"
}

func localGateway() (int, bool, bool) {
	if dialLocal(5666) {
		return 5666, false, true
	}
	if dialLocal(5667) {
		return 5667, true, true
	}
	return 0, false, false
}

// BrowserEntry is the fnOS desktop URL for this app.
// FONU_FNOS_URL overrides discovery. Otherwise, on a NAS install, the host
// comes from the request (a ZeroTier address stays a ZeroTier address).
func BrowserEntry(r *http.Request) string {
	if base := configuredBase(); base != "" {
		return base + "/app/fonu/"
	}
	if !OnNAS() || r == nil {
		return ""
	}
	host := requestHostname(r)
	if host == "" {
		return ""
	}
	port, https := webPort()
	scheme := "http"
	if https {
		scheme = "https"
	}
	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/app/fonu/"
}

func configuredBase() string {
	raw := strings.TrimSpace(os.Getenv("FONU_FNOS_URL"))
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func requestHostname(r *http.Request) string {
	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = strings.TrimSpace(r.URL.Host)
	}
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	return strings.Trim(host, "[]")
}

var (
	portOnce  sync.Once
	httpPort  = 5666
	httpsPort bool
)

func webPort() (int, bool) {
	portOnce.Do(func() {
		if dialLocal(5667) {
			httpPort = 5667
			httpsPort = true
			return
		}
		if dialLocal(5666) {
			httpPort = 5666
		}
	})
	return httpPort, httpsPort
}

func dialLocal(port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
