package api

import (
	"encoding/json"
	"net/http"

	"github.com/fonu/fonu/internal/service"
	"github.com/fonu/fonu/internal/settings"
	"github.com/fonu/fonu/internal/validate"
)

type SettingsHandler struct {
	store *settings.Store
	proxy *service.ProxyService
}

func NewSettingsHandler(store *settings.Store, proxySvc *service.ProxyService) *SettingsHandler {
	return &SettingsHandler{store: store, proxy: proxySvc}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	values, err := h.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取设置失败")
		return
	}
	writeJSON(w, http.StatusOK, values)
}

func (h *SettingsHandler) Put(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	allowed := map[string]bool{
		settings.KeyTheme:                 true,
		settings.KeyTimezone:              true,
		settings.KeyLogRetentionDays:      true,
		settings.KeyDDNSCheckInterval:     true,
		settings.KeyCertRenewThreshold:    true,
		settings.KeyRootDomain:            true,
		settings.KeyACMEEmail:             true,
		settings.KeyACMECA:                true,
		settings.KeyZeroSSLAPIKey:         true,
		settings.KeyNotifyWebhookURL:      true,
		settings.KeyNotifyOnDDNSError:     true,
		settings.KeyNotifyOnCertError:     true,
		settings.KeyNotifyOnNginxError:    true,
		settings.KeyTrustedProxy:          true,
		settings.KeyGlobalIPBlacklist:     true,
		settings.KeyChinaCIDRUpdateHours:  true,
		settings.KeyChinaCIDRSourceV4:     true,
		settings.KeyChinaCIDRSourceV6:     true,
	}
	filtered := map[string]string{}
	for key, value := range req {
		if !allowed[key] {
			continue
		}
		if err := h.validateSetting(key, value); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		filtered[key] = value
	}
	if err := h.store.SetMany(r.Context(), filtered); err != nil {
		writeError(w, http.StatusInternalServerError, "保存设置失败")
		return
	}
	if h.proxy != nil && (filtered[settings.KeyTrustedProxy] != "" || filtered[settings.KeyGlobalIPBlacklist] != "") {
		if err := h.proxy.ReloadAll(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "设置已保存，但 Nginx 重载失败："+err.Error())
			return
		}
	}
	values, err := h.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取设置失败")
		return
	}
	writeJSON(w, http.StatusOK, values)
}

func (h *SettingsHandler) validateSetting(key, value string) error {
	switch key {
	case settings.KeyGlobalIPBlacklist:
		var list []string
		if value != "" && value != "[]" {
			if err := json.Unmarshal([]byte(value), &list); err != nil {
				return err
			}
			if err := validate.IPOrCIDRList(list); err != nil {
				return err
			}
		}
	case settings.KeyTrustedProxy:
		if value != "" && value != "{}" {
			var raw map[string]any
			if err := json.Unmarshal([]byte(value), &raw); err != nil {
				return err
			}
		}
	}
	return nil
}
