package api

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/auth"
	"github.com/fonu/fonu/internal/notify"
)

type AuthHandler struct {
	auth   *auth.Service
	notify *notify.Service
}

func NewAuthHandler(authSvc *auth.Service, notifySvc *notify.Service) *AuthHandler {
	return &AuthHandler{auth: authSvc, notify: notifySvc}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authStatusResponse struct {
	Initialized bool `json:"initialized"`
	Authenticated bool `json:"authenticated"`
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	initialized, err := h.auth.IsInitialized(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取初始化状态失败")
		return
	}
	authenticated := false
	if sessionID := sessionIDFromContext(r.Context()); sessionID != "" {
		if err := h.auth.ValidateSession(r.Context(), sessionID); err == nil {
			authenticated = true
		}
	}
	writeJSON(w, http.StatusOK, authStatusResponse{
		Initialized:   initialized,
		Authenticated: authenticated,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	sessionID, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			if h.notify != nil {
				h.notify.RecordLoginFailure(r.Context(), clientIP(r))
			}
			writeError(r, w, http.StatusUnauthorized, "用户名或密码错误")
			return
		}
		writeError(r, w, http.StatusInternalServerError, "登录失败")
		return
	}
	setSessionCookie(w, sessionID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "登录成功"})
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	sessionID := sessionIDFromContext(r.Context())
	adminID, err := h.auth.AdminIDForSession(r.Context(), sessionID)
	if err != nil {
		writeError(r, w, http.StatusUnauthorized, "未登录或会话已过期")
		return
	}
	if err := h.auth.ChangePassword(r.Context(), adminID, req.OldPassword, req.NewPassword); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "密码已更新"})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if sessionID := sessionIDFromContext(r.Context()); sessionID != "" {
		_ = h.auth.Logout(r.Context(), sessionID)
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "已退出登录"})
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "fonu_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "fonu_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}
