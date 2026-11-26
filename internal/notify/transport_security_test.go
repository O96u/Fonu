package notify

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebhookDoesNotFollowRedirectOrExposeCredentials(t *testing.T) {
	hits := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	req, _ := http.NewRequest(http.MethodPost, source.URL+"/secret-token", strings.NewReader("secret payload"))
	if err := doRequest(source.Client(), req); err == nil || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("unexpected redirect result: %v", err)
	}
	if hits != 0 { t.Fatal("notification forwarded to redirect") }
	reject := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "secret-token", http.StatusBadRequest)
	}))
	defer reject.Close()
	req, _ = http.NewRequest(http.MethodPost, reject.URL, nil)
	if err := doRequest(reject.Client(), req); err == nil || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSMTPContextCancelsStalledGreeting(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer ln.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil { return }
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(2*time.Second))
		var buf [1]byte
		_, _ = conn.Read(buf[:])
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := sendSMTPContext(ctx, ln.Addr().String(), "localhost", nil, "a@example.com", []string{"b@example.com"}, nil, false, false); err == nil {
		t.Fatal("expected timeout")
	}
	select {
	case <-done:
	case <-time.After(time.Second): t.Fatal("SMTP connection not closed")
	}
}
