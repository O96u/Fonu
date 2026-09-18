package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fonu/fonu/internal/notify"
	"github.com/fonu/fonu/internal/service"
	"github.com/fonu/fonu/internal/settings"
	"github.com/fonu/fonu/internal/validate"
)

type SettingsHandler struct {
	store  *settings.Store
	proxy  *service.ProxyService
	notify *notify.Service
}

func NewSettingsHandler(store *settings.Store, proxySvc *service.ProxyService, notifySvc *notify.Service) *SettingsHandler {
	return &SettingsHandler{store: store, proxy: proxySvc, notify: notifySvc}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	values, err := h.store.List(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取设置失败", err)
		return
	}
	notify.SanitizeSettings(values)
	writeJSON(w, http.StatusOK, values)
}

func (h *SettingsHandler) Put(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
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
		settings.KeyNotifyType:            true,
		settings.KeyNotifyEmailJSON:       true,
		settings.KeyNotifySMTPPassword:    true,
		settings.KeyNotifyWebhookJSON:     true,
		settings.KeyNotifyWebhookSecret:   true,
		settings.KeyNotifyTelegramJSON:    true,
		settings.KeyNotifyTelegramToken:   true,
		settings.KeyNotifyOnDDNSIPChange:      true,
		settings.KeyNotifyOnDDNSFailure:       true,
		settings.KeyNotifyOnCertExpiry:          true,
		settings.KeyNotifyOnCertRenewSuccess:    true,
		settings.KeyNotifyOnIPFrequentAccess:    true,
		settings.KeyNotifyOnCertRenewFailure:    true,
		settings.KeyNotifyOnLoginFailure:        true,
		settings.KeyNotifyOnNginxReloadFailure:  true,
		settings.KeyNotifyIPFrequentThreshold:   true,
		settings.KeyNotifyIPFrequentWindowSec:   true,
		settings.KeyNotifyLoginFailureThreshold: true,
		settings.KeyNotifyLoginFailureWindowSec: true,
		settings.KeyTrustedProxy:          true,
		settings.KeyGlobalIPBlacklist:     true,
		settings.KeyGlobalIPWhitelist:     true,
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
			writeError(r, w, http.StatusBadRequest, err.Error())
			return
		}
		filtered[key] = value
	}
	if h.notify != nil && h.notify.Store() != nil {
		if in, ok := notify.ParseSaveInputFromMap(filtered); ok {
			if err := h.notify.Store().Save(r.Context(), in); err != nil {
				writeError(r, w, http.StatusBadRequest, err.Error())
				return
			}
			for key := range filtered {
				if isNotifySettingKey(key) {
					delete(filtered, key)
				}
			}
		}
	}
	if err := h.store.SetMany(r.Context(), filtered); err != nil {
		writeError(r, w, http.StatusInternalServerError, "保存设置失败", err)
		return
	}
	if h.proxy != nil && (filtered[settings.KeyTrustedProxy] != "" || filtered[settings.KeyGlobalIPBlacklist] != "" || filtered[settings.KeyGlobalIPWhitelist] != "") {
		if err := h.proxy.ReloadAll(r.Context()); err != nil {
			writeError(r, w, http.StatusInternalServerError, "设置已保存，但 Nginx 重载失败："+err.Error(), err)
			return
		}
	}
	values, err := h.store.List(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取设置失败", err)
		return
	}
	notify.SanitizeSettings(values)
	writeJSON(w, http.StatusOK, values)
}

func isNotifySettingKey(key string) bool {
	switch key {
	case settings.KeyNotifyType,
		settings.KeyNotifyEmailJSON,
		settings.KeyNotifySMTPPassword,
		settings.KeyNotifyWebhookJSON,
		settings.KeyNotifyWebhookSecret,
		settings.KeyNotifyTelegramJSON,
		settings.KeyNotifyTelegramToken,
		settings.KeyNotifyOnDDNSIPChange,
		settings.KeyNotifyOnDDNSFailure,
		settings.KeyNotifyOnCertExpiry,
		settings.KeyNotifyOnCertRenewSuccess,
		settings.KeyNotifyOnIPFrequentAccess,
		settings.KeyNotifyOnCertRenewFailure,
		settings.KeyNotifyOnLoginFailure,
		settings.KeyNotifyOnNginxReloadFailure,
		settings.KeyNotifyIPFrequentThreshold,
		settings.KeyNotifyIPFrequentWindowSec,
		settings.KeyNotifyLoginFailureThreshold,
		settings.KeyNotifyLoginFailureWindowSec,
		settings.KeyNotifyWebhookURL:
		return true
	default:
		return false
	}
}

func (h *SettingsHandler) validateSetting(key, value string) error {
	switch key {
	case settings.KeyNotifyEmailJSON, settings.KeyNotifyWebhookJSON, settings.KeyNotifyTelegramJSON:
		if value != "" {
			var raw map[string]any
			if err := json.Unmarshal([]byte(value), &raw); err != nil {
				return err
			}
		}
	case settings.KeyNotifyType:
		switch value {
		case "", "email", "webhook", "telegram":
		default:
			return fmt.Errorf("无效的通知类型")
		}
	case settings.KeyNotifyIPFrequentThreshold:
		if value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 10 || n > 10000 {
				return fmt.Errorf("频繁访问阈值需在 10–10000 之间")
			}
		}
	case settings.KeyNotifyIPFrequentWindowSec, settings.KeyNotifyLoginFailureWindowSec:
		if value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 10 || n > 3600 {
				return fmt.Errorf("统计窗口需在 10–3600 秒之间")
			}
		}
	case settings.KeyNotifyLoginFailureThreshold:
		if value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 3 || n > 100 {
				return fmt.Errorf("登录失败阈值需在 3–100 之间")
			}
		}
	case settings.KeyGlobalIPBlacklist, settings.KeyGlobalIPWhitelist:
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
