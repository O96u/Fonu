package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthRequestProtectionLimitsBodyAndAttempts(t *testing.T) {
	calls := 0
	h := protectRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, err := io.ReadAll(r.Body)
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < authMaxAttempts; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(strings.Repeat("a", authRequestBodyLimit+1)))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("oversized body: %d", w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/setup", nil))
	if w.Code != http.StatusTooManyRequests || calls != authMaxAttempts {
		t.Fatalf("rate limit: status=%d calls=%d", w.Code, calls)
	}
}

func TestAuthRequestProtectionBoundsGlobalConcurrency(t *testing.T) {
	entered := make(chan struct{}, authConcurrentLimit)
	release := make(chan struct{})
	done := make(chan struct{}, authConcurrentLimit)
	h := protectRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/login" {
			entered <- struct{}{}
			<-release
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer func() {
		close(release)
		for i := 0; i < authConcurrentLimit; i++ {
			<-done
		}
	}()
	for i := 0; i < authConcurrentLimit; i++ {
		go func() {
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/auth/login", nil))
			done <- struct{}{}
		}()
	}
	for i := 0; i < authConcurrentLimit; i++ {
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("authentication requests did not enter")
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	r.RemoteAddr = "192.0.2.2:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("concurrency limit: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("unrelated request: %d", w.Code)
	}
}
