package fnos

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

// Find looks for an fnOS web UI on this machine or on the local subnets
// attached to it, including a ZeroTier adapter. Tests may replace it.
var Find = findCached

var (
	findMu     sync.Mutex
	findAt     time.Time
	findOrigin string
	findOK     bool
)

func findCached(ctx context.Context, preferHost string) (string, bool) {
	findMu.Lock()
	if !findAt.IsZero() && time.Since(findAt) < 20*time.Second {
		origin, ok := findOrigin, findOK
		findMu.Unlock()
		return origin, ok
	}
	findMu.Unlock()

	origin, ok := scan(ctx, preferHost)
	findMu.Lock()
	findAt = time.Now()
	findOrigin, findOK = origin, ok
	findMu.Unlock()
	return origin, ok
}

func scan(ctx context.Context, preferHost string) (string, bool) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if origin, ok := probe(ctx, "127.0.0.1"); ok {
		return rewriteLoopback(origin, preferHost), true
	}
	hosts := subnetHosts()
	if len(hosts) == 0 {
		return "", false
	}
	ctx, cancelScan := context.WithCancel(ctx)
	defer cancelScan()
	var (
		mu    sync.Mutex
		found []string
	)
	sem := make(chan struct{}, 64)
	var wg sync.WaitGroup
	for _, host := range hosts {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()
			origin, ok := probe(ctx, ip)
			if !ok {
				return
			}
			mu.Lock()
			found = append(found, origin)
			if ip == preferHost {
				cancelScan()
			}
			mu.Unlock()
		}(host)
	}
	wg.Wait()
	return pickOrigin(found, preferHost)
}

func rewriteLoopback(origin, preferHost string) string {
	if preferHost == "" || preferHost == "localhost" || preferHost == "127.0.0.1" || preferHost == "::1" {
		return origin
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Port() == "" {
		return origin
	}
	parsed.Host = net.JoinHostPort(preferHost, parsed.Port())
	return parsed.String()
}

func pickOrigin(found []string, preferHost string) (string, bool) {
	if len(found) == 0 {
		return "", false
	}
	sort.Strings(found)
	for _, origin := range found {
		if host, _, err := net.SplitHostPort(trimScheme(origin)); err == nil && host == preferHost {
			return origin, true
		}
	}
	return found[0], true
}

func trimScheme(origin string) string {
	if len(origin) > 8 && origin[:8] == "https://" {
		return origin[8:]
	}
	if len(origin) > 7 && origin[:7] == "http://" {
		return origin[7:]
	}
	return origin
}

func probe(ctx context.Context, ip string) (string, bool) {
	if origin, ok := probePort(ctx, ip, "5666", false); ok {
		return origin, true
	}
	return probePort(ctx, ip, "5667", true)
}

func probePort(ctx context.Context, ip, port string, https bool) (string, bool) {
	dialer := net.Dialer{Timeout: 120 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
	if err != nil {
		return "", false
	}
	_ = conn.Close()
	scheme := "http"
	if https {
		scheme = "https"
	}
	origin := scheme + "://" + net.JoinHostPort(ip, port)
	reqCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, origin+"/", nil)
	if err != nil {
		return "", false
	}
	client := &http.Client{
		Timeout: 500 * time.Millisecond,
		Transport: &http.Transport{
			Proxy: nil,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: https,
				MinVersion:         tls.VersionTLS12,
			},
			DialContext:           (&net.Dialer{Timeout: 200 * time.Millisecond}).DialContext,
			ResponseHeaderTimeout: 400 * time.Millisecond,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if !looksLikeFNOS(body) {
		return "", false
	}
	return origin, true
}

func looksLikeFNOS(body []byte) bool {
	if bytes.Contains(bytes.ToLower(body), []byte("fnos")) {
		return true
	}
	return bytes.Contains(body, []byte("飞牛"))
}

func subnetHosts() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var hosts []string
	seen := map[string]bool{}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}
			for _, host := range hostsOn(ipnet.IP, ipnet.Mask) {
				text := host.String()
				if seen[text] {
					continue
				}
				seen[text] = true
				hosts = append(hosts, text)
				if len(hosts) >= 512 {
					return hosts
				}
			}
		}
	}
	return hosts
}

func hostsOn(ip net.IP, mask net.IPMask) []net.IP {
	ip = ip.To4()
	if ip == nil {
		return nil
	}
	ones, bits := mask.Size()
	if bits != 32 || ones <= 0 || ones > 30 {
		return nil
	}
	if ones < 24 {
		mask = net.CIDRMask(24, 32)
		ones = 24
	}
	network := ip.Mask(mask).To4()
	base := binary.BigEndian.Uint32(network)
	size := 1 << (32 - ones)
	out := make([]net.IP, 0, size)
	for i := 1; i < size-1; i++ {
		raw := base + uint32(i)
		host := make(net.IP, 4)
		binary.BigEndian.PutUint32(host, raw)
		if host.Equal(ip) {
			continue
		}
		out = append(out, host)
	}
	return out
}
