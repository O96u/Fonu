package api

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	authRequestBodyLimit = 16 << 10
	authConcurrentLimit = 4
	authWindowDuration = time.Minute
	authMaxTrackedIPs = 4096
	authMaxAttempts = 10
)

type loginWindow struct {
	count int
	until time.Time
}

func protectRequests(next http.Handler) http.Handler {
	var mu sync.Mutex
	attempts := make(map[string]loginWindow)
	var nextSweep time.Time
	authSlots := make(chan struct{}, authConcurrentLimit)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-store")
			if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
				origin := r.Header.Get("Origin")
				if origin != "" {
					u, err := url.Parse(origin)
					if err != nil || !strings.EqualFold(u.Host, r.Host) || (u.Scheme != "http" && u.Scheme != "https") {
						writeError(r, w, http.StatusForbidden, "不允许跨来源请求")
						return
					}
				}
				if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
					writeError(r, w, http.StatusForbidden, "不允许跨站请求")
					return
				}
				if r.URL.Path != "/api/backup/restore" {
					r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
				}
			}
			if r.Method == http.MethodPost && (r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/setup") {
				now := time.Now()
				ip := clientIP(r)
				mu.Lock()
				if !now.Before(nextSweep) {
					for k, v := range attempts {
						if !now.Before(v.until) {
							delete(attempts, k)
						}
					}
					nextSweep = now.Add(authWindowDuration)
				}
				window, exists := attempts[ip]
				if exists && !now.Before(window.until) {
					delete(attempts, ip)
					window, exists = loginWindow{}, false
				}
				if window.count >= authMaxAttempts || (!exists && len(attempts) >= authMaxTrackedIPs) {
					mu.Unlock()
					w.Header().Set("Retry-After", "60")
					writeError(r, w, http.StatusTooManyRequests, "请求过于频繁，请稍后重试")
					return
				}
				if !exists {
					window.until = now.Add(authWindowDuration)
				}
				window.count++
				attempts[ip] = window
				mu.Unlock()

				// Bound concurrent password hashing across all source addresses.
				// Do not queue work or block unrelated management requests.
				select {
				case authSlots <- struct{}{}:
					defer func() { <-authSlots }()
				default:
					w.Header().Set("Retry-After", "1")
					writeError(r, w, http.StatusTooManyRequests, "登录请求过多，请稍后重试")
					return
				}
				r.Body = http.MaxBytesReader(w, r.Body, authRequestBodyLimit)
				// Bound slow login bodies without disrupting SSE or backup uploads.
				controller := http.NewResponseController(w)
				if err := controller.SetReadDeadline(now.Add(15 * time.Second)); err == nil {
					defer controller.SetReadDeadline(time.Time{})
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
