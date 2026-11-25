package api

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type loginWindow struct { count int; until time.Time }

func protectRequests(next http.Handler) http.Handler {
	var mu sync.Mutex
	attempts := make(map[string]loginWindow)
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
				for k, v := range attempts { if !now.Before(v.until) { delete(attempts, k) } }
				window, exists := attempts[ip]
				if window.count >= 10 || (!exists && len(attempts) >= 4096) {
					mu.Unlock()
					w.Header().Set("Retry-After", "60")
					writeError(r, w, http.StatusTooManyRequests, "请求过于频繁，请稍后重试")
					return
				}
				if !exists { window.until = now.Add(time.Minute) }
				window.count++
				attempts[ip] = window
				mu.Unlock()
			}
		}
		next.ServeHTTP(w, r)
	})
}
