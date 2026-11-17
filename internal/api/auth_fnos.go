package api

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/fonu/fonu/internal/auth"
	"github.com/fonu/fonu/internal/fnos"
)

const (
	codeFnosNotAvailable    = "FNOS_NOT_AVAILABLE"
	codeFnosGatewayRequired = "FNOS_GATEWAY_REQUIRED"
	codeFnosAuthDisabled    = "FNOS_AUTH_DISABLED"
	codeFnosUserNotBound    = "FNOS_USER_NOT_BOUND"
	codeFnosIdentityInvalid = "FNOS_IDENTITY_INVALID"
	codeFnosAuthFailed      = "FNOS_AUTH_FAILED"
)

type fnosStatusResponse struct {
	Available   bool   `json:"available"`
	Enabled     bool   `json:"enabled"`
	Gateway     bool   `json:"gateway"`
	Bound       bool   `json:"bound"`
	LoginMethod string `json:"login_method"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

type fnosSettingsResponse struct {
	Enabled         bool               `json:"enabled"`
	Gateway         bool               `json:"gateway"`
	GatewayURL      string             `json:"gateway_url"`
	SuggestedURL    string             `json:"suggested_url,omitempty"`
	Bindings        []auth.FnosBinding `json:"bindings"`
	CurrentUID      string             `json:"current_uid,omitempty"`
	CurrentUsername string             `json:"current_username,omitempty"`
}

func (h *AuthHandler) FnosStatus(w http.ResponseWriter, r *http.Request) {
	resp := fnosStatusResponse{LoginMethod: "none", Enabled: true}
	if ident, ok := fnos.IdentityFromContext(r.Context()); ok && strings.TrimSpace(ident.UID) != "" {
		resp.Gateway = true
		resp.Available = true
		resp.LoginMethod = "gateway"
		if _, err := h.auth.BoundAdmin(r.Context(), ident.UID, ""); err == nil {
			resp.Bound = true
		} else if !errors.Is(err, auth.ErrFnosNotBound) {
			writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "读取飞牛登录状态失败", err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	if target := fnos.DetectLoginURL(r); target != "" {
		resp.Available = true
		resp.LoginMethod = "redirect"
		resp.RedirectURL = target
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) FnosLogin(w http.ResponseWriter, r *http.Request) {
	ident, ok := fnos.IdentityFromContext(r.Context())
	if !ok {
		writeCodedError(r, w, http.StatusForbidden, codeFnosGatewayRequired, "请从飞牛桌面或飞牛网关入口打开后再使用 NAS 登录")
		return
	}
	if strings.TrimSpace(ident.UID) == "" {
		writeCodedError(r, w, http.StatusForbidden, codeFnosIdentityInvalid, "无法验证当前飞牛用户，请重新登录飞牛 fnOS")
		return
	}
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil && !errors.Is(err, io.EOF) {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	adminID, err := h.auth.BoundAdmin(r.Context(), ident.UID, ident.Username)
	if errors.Is(err, auth.ErrFnosNotBound) {
		if strings.TrimSpace(creds.Username) == "" || creds.Password == "" {
			writeCodedError(r, w, http.StatusForbidden, codeFnosUserNotBound, "这个飞牛账号还没绑定。请填写 Fonu 账号和密码后再点一次。")
			return
		}
		adminID, err = h.auth.Authenticate(r.Context(), strings.TrimSpace(creds.Username), creds.Password)
		if err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				writeError(r, w, http.StatusUnauthorized, "用户名或密码错误")
				return
			}
			writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "登录失败", err)
			return
		}
		if err := h.auth.BindFnos(r.Context(), adminID, ident.UID, ident.Username); err != nil {
			writeCodedError(r, w, http.StatusBadRequest, codeFnosAuthFailed, err.Error())
			return
		}
	} else if err != nil {
		writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "登录失败", err)
		return
	}
	sessionID, err := h.auth.OpenSession(r.Context(), adminID, auth.SourceFnos, ident.UID)
	if err != nil {
		writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "登录失败", err)
		return
	}
	setSessionCookie(w, r, sessionID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "登录成功"})
}

func (h *AuthHandler) FnosSettings(w http.ResponseWriter, r *http.Request) {
	resp, err := h.fnosSettings(r)
	if err != nil {
		writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "读取飞牛登录设置失败", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) FnosSaveSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled    *bool   `json:"enabled"`
		GatewayURL *string `json:"gateway_url"`
		Password   string  `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if req.Enabled == nil && req.GatewayURL == nil {
		writeError(r, w, http.StatusBadRequest, "没有要保存的设置")
		return
	}
	adminID, ok := h.confirmAdmin(w, r, req.Password)
	if !ok {
		return
	}
	if req.GatewayURL != nil {
		raw := strings.TrimSpace(*req.GatewayURL)
		if raw == "" {
			if err := h.auth.SetFnosGatewayURL(r.Context(), ""); err != nil {
				writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "保存飞牛网关地址失败", err)
				return
			}
		} else {
			verified, err := fnos.VerifyGateway(r.Context(), raw, requestHost(r))
			if err != nil {
				writeCodedError(r, w, http.StatusBadRequest, codeFnosNotAvailable, err.Error())
				return
			}
			if err := h.auth.SetFnosGatewayURL(r.Context(), verified); err != nil {
				writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "保存飞牛网关地址失败", err)
				return
			}
		}
	}
	if req.Enabled != nil {
		if err := h.auth.SetFnosEnabled(r.Context(), *req.Enabled); err != nil {
			writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "保存飞牛登录开关失败", err)
			return
		}
	}
	_ = adminID
	resp, err := h.fnosSettings(r)
	if err != nil {
		writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "读取飞牛登录设置失败", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) FnosBind(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	adminID, ok := h.confirmAdmin(w, r, req.Password)
	if !ok {
		return
	}
	ident, gateway := fnos.IdentityFromContext(r.Context())
	if !gateway || strings.TrimSpace(ident.UID) == "" {
		writeCodedError(r, w, http.StatusForbidden, codeFnosGatewayRequired, "请从飞牛桌面打开 Fonu 后再绑定当前账号")
		return
	}
	if err := h.auth.BindFnos(r.Context(), adminID, ident.UID, ident.Username); err != nil {
		writeCodedError(r, w, http.StatusBadRequest, codeFnosAuthFailed, err.Error())
		return
	}
	resp, err := h.fnosSettings(r)
	if err != nil {
		writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "读取飞牛登录设置失败", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) FnosUnbind(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
		UID      string `json:"nas_uid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	adminID, ok := h.confirmAdmin(w, r, req.Password)
	if !ok {
		return
	}
	if err := h.auth.UnbindFnos(r.Context(), adminID, req.UID); err != nil {
		if errors.Is(err, auth.ErrFnosNotBound) {
			writeCodedError(r, w, http.StatusNotFound, codeFnosUserNotBound, "没有找到这个飞牛账号绑定")
			return
		}
		writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "解除绑定失败", err)
		return
	}
	resp, err := h.fnosSettings(r)
	if err != nil {
		writeCodedError(r, w, http.StatusInternalServerError, codeFnosAuthFailed, "读取飞牛登录设置失败", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) confirmAdmin(w http.ResponseWriter, r *http.Request, password string) (int, bool) {
	if strings.TrimSpace(password) == "" {
		writeError(r, w, http.StatusBadRequest, "请填写管理员密码")
		return 0, false
	}
	sessionID := sessionIDFromContext(r.Context())
	adminID, err := h.auth.AdminIDForSession(r.Context(), sessionID)
	if err != nil {
		writeError(r, w, http.StatusUnauthorized, "未登录或会话已过期")
		return 0, false
	}
	if err := h.auth.ConfirmPassword(r.Context(), adminID, password); err != nil {
		writeError(r, w, http.StatusUnauthorized, "当前密码不正确")
		return 0, false
	}
	return adminID, true
}

func (h *AuthHandler) fnosSettings(r *http.Request) (fnosSettingsResponse, error) {
	enabled, err := h.auth.FnosEnabled(r.Context())
	if err != nil {
		return fnosSettingsResponse{}, err
	}
	saved, err := h.auth.FnosGatewayURL(r.Context())
	if err != nil {
		return fnosSettingsResponse{}, err
	}
	bindings, err := h.auth.ListFnosBindings(r.Context())
	if err != nil {
		return fnosSettingsResponse{}, err
	}
	resp := fnosSettingsResponse{
		Enabled:    enabled,
		GatewayURL: saved,
		Bindings:   bindings,
	}
	if ident, ok := fnos.IdentityFromContext(r.Context()); ok {
		resp.Gateway = true
		resp.CurrentUID = ident.UID
		resp.CurrentUsername = ident.Username
	}
	if saved == "" {
		resp.SuggestedURL = fnos.LoginRedirect(r, "")
	}
	return resp, nil
}

func requestHost(r *http.Request) string {
	host := strings.TrimSpace(r.Host)
	if host == "" && r.URL != nil {
		host = r.URL.Host
	}
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	return strings.Trim(host, "[]")
}
