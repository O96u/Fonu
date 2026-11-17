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
	"github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/discovery"
	"github.com/fonu/fonu/internal/fnos"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/service"
	"github.com/fonu/fonu/internal/settings"
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

	tmpDir := t.TempDir()
	authSvc := auth.New(conn)
	proxySvc := service.NewProxyService(config.Config{DataDir: tmpDir, NginxPIDFile: tmpDir + "/nginx.pid"}, conn, proxy.NewStore(conn), certificate.NewStore(conn), settings.NewStore(conn), nil)
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
	if setupRec.Code != http.StatusOK {
		t.Fatalf("setup status: %d body=%s", setupRec.Code, setupRec.Body.String())
	}

	setupAgain := httptest.NewRequest(http.MethodPost, "/api/auth/setup", bytes.NewReader(setupBody))
	setupAgainRec := httptest.NewRecorder()
	handler.ServeHTTP(setupAgainRec, setupAgain)
	if setupAgainRec.Code != http.StatusBadRequest {
		t.Fatalf("second setup status: %d body=%s", setupAgainRec.Code, setupAgainRec.Body.String())
	}

	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "password123",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status: %d body=%s", loginRec.Code, loginRec.Body.String())
	}

	cookie := loginRec.Result().Cookies()[0]
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
	if authStatus.Username == nil || *authStatus.Username != "admin" {
		t.Fatalf("expected username admin in status, got %+v", authStatus.Username)
	}

	usernameBody, _ := json.Marshal(map[string]string{
		"password":     "password123",
		"new_username": "superuser",
	})
	usernameReq := httptest.NewRequest(http.MethodPost, "/api/auth/username", bytes.NewReader(usernameBody))
	usernameReq.AddCookie(cookie)
	usernameRec := httptest.NewRecorder()
	handler.ServeHTTP(usernameRec, usernameReq)
	if usernameRec.Code != http.StatusOK {
		t.Fatalf("change username status: %d body=%s", usernameRec.Code, usernameRec.Body.String())
	}

	authStatusReq2 := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	authStatusReq2.AddCookie(cookie)
	authStatusRec2 := httptest.NewRecorder()
	handler.ServeHTTP(authStatusRec2, authStatusReq2)
	var authStatus2 authStatusResponse
	if err := json.Unmarshal(authStatusRec2.Body.Bytes(), &authStatus2); err != nil {
		t.Fatalf("decode auth status: %v", err)
	}
	if authStatus2.Username == nil || *authStatus2.Username != "superuser" {
		t.Fatalf("expected updated username in status, got %+v", authStatus2.Username)
	}

	proxiesReq := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
	proxiesReq.AddCookie(cookie)
	proxiesRec := httptest.NewRecorder()
	handler.ServeHTTP(proxiesRec, proxiesReq)
	if proxiesRec.Code != http.StatusOK {
		t.Fatalf("proxies status: %d", proxiesRec.Code)
	}
}

func TestFnosLoginOnlyTrustsGateway(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "fonu.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	authSvc := auth.New(conn)
	if err := authSvc.Setup(context.Background(), "admin", "password123"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	handler := NewRouter(Deps{Auth: authSvc})
	gateway := fnos.Handler("/app/fonu", handler)

	spoof := httptest.NewRequest(http.MethodGet, "/api/auth/fnos", nil)
	spoof.Header.Set("X-Trim-Userid", "1000")
	spoof.Header.Set("X-Trim-Username", "admin")
	spoof.Header.Set("X-Trim-Isadmin", "true")
	spoofRec := httptest.NewRecorder()
	handler.ServeHTTP(spoofRec, spoof)
	if spoofRec.Code != http.StatusOK || !bytes.Contains(spoofRec.Body.Bytes(), []byte(`"available":false`)) {
		t.Fatalf("tcp spoof status=%d body=%s", spoofRec.Code, spoofRec.Body.String())
	}

	spoofLogin := httptest.NewRequest(http.MethodPost, "/api/auth/fnos-login", nil)
	spoofLogin.Header.Set("X-Trim-Userid", "1000")
	spoofLogin.Header.Set("X-Trim-Username", "admin")
	spoofLogin.Header.Set("X-Trim-Isadmin", "true")
	spoofLoginRec := httptest.NewRecorder()
	handler.ServeHTTP(spoofLoginRec, spoofLogin)
	if spoofLoginRec.Code != http.StatusForbidden {
		t.Fatalf("tcp spoof login status=%d body=%s", spoofLoginRec.Code, spoofLoginRec.Body.String())
	}

	member := httptest.NewRequest(http.MethodPost, "/app/fonu/api/auth/fnos-login", nil)
	member.Header.Set("X-Trim-Userid", "1001")
	member.Header.Set("X-Trim-Username", "family")
	member.Header.Set("X-Trim-Isadmin", "false")
	memberRec := httptest.NewRecorder()
	gateway.ServeHTTP(memberRec, member)
	if memberRec.Code != http.StatusForbidden {
		t.Fatalf("member login status=%d body=%s", memberRec.Code, memberRec.Body.String())
	}

	login := httptest.NewRequest(http.MethodPost, "/app/fonu/api/auth/fnos/login", nil)
	login.Header.Set("X-Trim-Userid", "1000")
	login.Header.Set("X-Trim-Username", "nas-admin")
	login.Header.Set("X-Trim-Isadmin", "true")
	loginRec := httptest.NewRecorder()
	gateway.ServeHTTP(loginRec, login)
	if loginRec.Code != http.StatusForbidden || !bytes.Contains(loginRec.Body.Bytes(), []byte(`FNOS_USER_NOT_BOUND`)) {
		t.Fatalf("unbound admin login status=%d body=%s", loginRec.Code, loginRec.Body.String())
	}

	bindBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	bind := httptest.NewRequest(http.MethodPost, "/app/fonu/api/auth/fnos/login", bytes.NewReader(bindBody))
	bind.Header.Set("X-Trim-Userid", "1000")
	bind.Header.Set("X-Trim-Username", "nas-admin")
	bind.Header.Set("X-Trim-Isadmin", "true")
	bindRec := httptest.NewRecorder()
	gateway.ServeHTTP(bindRec, bind)
	if bindRec.Code != http.StatusOK {
		t.Fatalf("bind on login status=%d body=%s", bindRec.Code, bindRec.Body.String())
	}
	cookie := bindRec.Result().Cookies()[0]
	if cookie.Name != "fonu_session" || cookie.Path != "/app/fonu" || cookie.Value == "" {
		t.Fatalf("cookie %+v", cookie)
	}
}
