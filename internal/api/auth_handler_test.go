package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/auth"
	"github.com/fonu/fonu/internal/backup"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/discovery"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/service"
)

func TestAuthSetupAndLoginFlow(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "fonu.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	authSvc := auth.New(conn)
	proxySvc := service.NewProxyService(conn, proxy.NewStore(conn), nil)
	tmpDir := t.TempDir()
	handler := NewRouter(Deps{
		Config:    config.Config{DataDir: tmpDir, NginxPIDFile: tmpDir + "/nginx.pid"},
		Auth:      authSvc,
		Proxy:     proxySvc,
		Backup:    backup.New(tmpDir),
		Discovery: discovery.New(),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	})

	statusReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	statusRec := httptest.NewRecorder()
	handler.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status code: %d", statusRec.Code)
	}

	setupBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "password123",
	})
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", bytes.NewReader(setupBody))
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status: %d body=%s", setupRec.Code, setupRec.Body.String())
	}

	cookie := setupRec.Result().Cookies()[0]
	if cookie.Name != "fonu_session" || cookie.HttpOnly == false {
		t.Fatal("expected httponly session cookie")
	}

	authStatusReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	authStatusReq.AddCookie(cookie)
	authStatusRec := httptest.NewRecorder()
	handler.ServeHTTP(authStatusRec, authStatusReq)
	if authStatusRec.Code != http.StatusOK {
		t.Fatalf("auth status code: %d", authStatusRec.Code)
	}
	var authStatus authStatusResponse
	if err := json.Unmarshal(authStatusRec.Body.Bytes(), &authStatus); err != nil {
		t.Fatalf("decode auth status: %v", err)
	}
	if !authStatus.Initialized || !authStatus.Authenticated {
		t.Fatalf("expected authenticated status, got %+v", authStatus)
	}

	proxiesReq := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
	proxiesReq.AddCookie(cookie)
	proxiesRec := httptest.NewRecorder()
	handler.ServeHTTP(proxiesRec, proxiesReq)
	if proxiesRec.Code != http.StatusOK {
		t.Fatalf("proxies status: %d", proxiesRec.Code)
	}
}
