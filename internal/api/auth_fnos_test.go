package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/auth"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/fnos"
)

func TestFnosBindAndLogin(t *testing.T) {
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

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password123"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("password login: %d %s", loginRec.Code, loginRec.Body.String())
	}
	passwordCookie := loginRec.Result().Cookies()[0]

	bindBody, _ := json.Marshal(map[string]string{"password": "password123", "nas_uid": "9999"})
	bindSpoof := httptest.NewRequest(http.MethodPost, "/api/auth/fnos/bind", bytes.NewReader(bindBody))
	bindSpoof.Header.Set("X-Trim-Userid", "1000")
	bindSpoof.Header.Set("X-Trim-Username", "nas-admin")
	bindSpoof.Header.Set("X-Trim-Isadmin", "true")
	bindSpoof.AddCookie(passwordCookie)
	bindSpoofRec := httptest.NewRecorder()
	handler.ServeHTTP(bindSpoofRec, bindSpoof)
	if bindSpoofRec.Code != http.StatusForbidden || !bytes.Contains(bindSpoofRec.Body.Bytes(), []byte("FNOS_GATEWAY_REQUIRED")) {
		t.Fatalf("tcp bind status=%d body=%s", bindSpoofRec.Code, bindSpoofRec.Body.String())
	}

	noPassword := httptest.NewRequest(http.MethodPost, "/app/fonu/api/auth/fnos/bind", bytes.NewReader(bindBody))
	noPassword.Header.Set("X-Trim-Userid", "1000")
	noPassword.Header.Set("X-Trim-Username", "nas-admin")
	noPassword.Header.Set("X-Trim-Isadmin", "true")
	noPasswordRec := httptest.NewRecorder()
	gateway.ServeHTTP(noPasswordRec, noPassword)
	if noPasswordRec.Code != http.StatusUnauthorized {
		t.Fatalf("bind without session: %d %s", noPasswordRec.Code, noPasswordRec.Body.String())
	}

	enableBody, _ := json.Marshal(map[string]any{"enabled": true, "password": "password123"})
	enableReq := httptest.NewRequest(http.MethodPut, "/api/settings/fnos-auth", bytes.NewReader(enableBody))
	enableReq.AddCookie(passwordCookie)
	enableRec := httptest.NewRecorder()
	handler.ServeHTTP(enableRec, enableReq)
	if enableRec.Code != http.StatusOK || !bytes.Contains(enableRec.Body.Bytes(), []byte(`"enabled":true`)) {
		t.Fatalf("enable: %d %s", enableRec.Code, enableRec.Body.String())
	}

	realBindBody, _ := json.Marshal(map[string]string{"password": "password123"})
	realBind := httptest.NewRequest(http.MethodPost, "/app/fonu/api/auth/fnos/bind", bytes.NewReader(realBindBody))
	realBind.Header.Set("X-Trim-Userid", "1000")
	realBind.Header.Set("X-Trim-Username", "nas-admin")
	realBind.Header.Set("X-Trim-Isadmin", "false")
	realBind.AddCookie(passwordCookie)
	realBindRec := httptest.NewRecorder()
	gateway.ServeHTTP(realBindRec, realBind)
	if realBindRec.Code != http.StatusOK || !bytes.Contains(realBindRec.Body.Bytes(), []byte(`"nas_uid":"1000"`)) {
		t.Fatalf("bind: %d %s", realBindRec.Code, realBindRec.Body.String())
	}

	public := httptest.NewRequest(http.MethodGet, "/api/auth/fnos/status", nil)
	publicRec := httptest.NewRecorder()
	handler.ServeHTTP(publicRec, public)
	if bytes.Contains(publicRec.Body.Bytes(), []byte("1000")) || bytes.Contains(publicRec.Body.Bytes(), []byte("nas-admin")) {
		t.Fatalf("public status leaked identity: %s", publicRec.Body.String())
	}

	other := httptest.NewRequest(http.MethodPost, "/app/fonu/api/auth/fnos/login", nil)
	other.Header.Set("X-Trim-Userid", "1001")
	other.Header.Set("X-Trim-Username", "nas-admin")
	other.Header.Set("X-Trim-Isadmin", "true")
	otherRec := httptest.NewRecorder()
	gateway.ServeHTTP(otherRec, other)
	if otherRec.Code != http.StatusForbidden || !bytes.Contains(otherRec.Body.Bytes(), []byte("FNOS_USER_NOT_BOUND")) {
		t.Fatalf("other uid: %d %s", otherRec.Code, otherRec.Body.String())
	}

	sameName := httptest.NewRequest(http.MethodPost, "/app/fonu/api/auth/fnos/login", nil)
	sameName.Header.Set("X-Trim-Username", "admin")
	sameName.Header.Set("X-Trim-Isadmin", "true")
	sameNameRec := httptest.NewRecorder()
	gateway.ServeHTTP(sameNameRec, sameName)
	if sameNameRec.Code != http.StatusForbidden || !bytes.Contains(sameNameRec.Body.Bytes(), []byte("FNOS_IDENTITY_INVALID")) {
		t.Fatalf("username only: %d %s", sameNameRec.Code, sameNameRec.Body.String())
	}

	okLogin := httptest.NewRequest(http.MethodPost, "/app/fonu/api/auth/fnos/login", nil)
	okLogin.Header.Set("X-Trim-Userid", "1000")
	okLogin.Header.Set("X-Trim-Username", "renamed")
	okLogin.Header.Set("X-Trim-Isadmin", "false")
	okLoginRec := httptest.NewRecorder()
	gateway.ServeHTTP(okLoginRec, okLogin)
	if okLoginRec.Code != http.StatusOK {
		t.Fatalf("bound login: %d %s", okLoginRec.Code, okLoginRec.Body.String())
	}
	fnosCookie := okLoginRec.Result().Cookies()[0]
	if fnosCookie.Path != "/app/fonu" || fnosCookie.Value == "" || fnosCookie.Value == passwordCookie.Value {
		t.Fatalf("fnos cookie %+v", fnosCookie)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/app/fonu/api/auth/status", nil)
	statusReq.AddCookie(fnosCookie)
	statusRec := httptest.NewRecorder()
	gateway.ServeHTTP(statusRec, statusReq)
	if !bytes.Contains(statusRec.Body.Bytes(), []byte(`"authenticated":true`)) {
		t.Fatalf("fnos session: %s", statusRec.Body.String())
	}

	unbindBody, _ := json.Marshal(map[string]string{"password": "password123", "nas_uid": "1000"})
	unbind := httptest.NewRequest(http.MethodDelete, "/api/auth/fnos/bind", bytes.NewReader(unbindBody))
	unbind.AddCookie(passwordCookie)
	unbindRec := httptest.NewRecorder()
	handler.ServeHTTP(unbindRec, unbind)
	if unbindRec.Code != http.StatusOK {
		t.Fatalf("unbind: %d %s", unbindRec.Code, unbindRec.Body.String())
	}
	revoked := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	revoked.AddCookie(fnosCookie)
	revokedRec := httptest.NewRecorder()
	handler.ServeHTTP(revokedRec, revoked)
	if bytes.Contains(revokedRec.Body.Bytes(), []byte(`"authenticated":true`)) {
		t.Fatalf("fnos session survived unbind: %s", revokedRec.Body.String())
	}
	kept := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	kept.AddCookie(passwordCookie)
	keptRec := httptest.NewRecorder()
	handler.ServeHTTP(keptRec, kept)
	if !bytes.Contains(keptRec.Body.Bytes(), []byte(`"authenticated":true`)) {
		t.Fatalf("password session was revoked: %s", keptRec.Body.String())
	}

	again := httptest.NewRequest(http.MethodPost, "/app/fonu/api/auth/fnos/login", nil)
	again.Header.Set("X-Trim-Userid", "1000")
	again.Header.Set("X-Trim-Username", "nas-admin")
	againRec := httptest.NewRecorder()
	gateway.ServeHTTP(againRec, again)
	if againRec.Code != http.StatusForbidden || !bytes.Contains(againRec.Body.Bytes(), []byte("FNOS_USER_NOT_BOUND")) {
		t.Fatalf("login after unbind: %d %s", againRec.Code, againRec.Body.String())
	}
}
