package acme

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestZeroSSLEABCredentialsUsesAuthorizationHeader(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_key") != "" {
			t.Error("expected access_key query param to be omitted")
		}
		if r.Header.Get("Authorization") != "ApiKey secret-key" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"success":true,"eab_kid":"kid","eab_hmac_key":"hmac"}`))
	}))
	t.Cleanup(srv.Close)

	kid, hmac, err := zerosslEABCredentialsAt(context.Background(), "secret-key", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if kid != "kid" || hmac != "hmac" {
		t.Fatalf("kid=%q hmac=%q", kid, hmac)
	}
}

func TestZeroSSLAPISuccess(t *testing.T) {
	t.Parallel()
	if !zeroSSLAPISuccess(json.RawMessage(`true`)) {
		t.Fatal("want true")
	}
	if !zeroSSLAPISuccess(json.RawMessage(`1`)) {
		t.Fatal("want 1")
	}
	if zeroSSLAPISuccess(json.RawMessage(`false`)) {
		t.Fatal("want false")
	}
}

func TestZeroSSLAPIErrorMessage(t *testing.T) {
	t.Parallel()
	var payload zeroSSLEABResponse
	payload.Error.Type = "test_user_only"
	payload.Error.Info = "beta only"
	if msg := zeroSSLAPIErrorMessage(payload, 422); msg != "beta only" {
		t.Fatalf("msg=%q", msg)
	}
	payload.Error.Info = ""
	if msg := zeroSSLAPIErrorMessage(payload, 422); !strings.Contains(msg, "test_user_only") {
		t.Fatalf("msg=%q", msg)
	}
}

func TestZeroSSLEABCredentialsLegacySuccessInt(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":1,"eab_kid":"k","eab_hmac_key":"h"}`))
	}))
	t.Cleanup(srv.Close)

	kid, hmac, err := zerosslEABCredentialsAt(context.Background(), "key", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if kid != "k" || hmac != "h" {
		t.Fatalf("kid=%q hmac=%q", kid, hmac)
	}
}
