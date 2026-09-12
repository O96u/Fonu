package api

import (
	"encoding/json"
	"net/http"

	"github.com/fonu/fonu/internal/settings"
)

type SettingsHandler struct {
	store *settings.Store
}

func NewSettingsHandler(store *settings.Store) *SettingsHandler {
	return &SettingsHandler{store: store}
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
		settings.KeyTheme:              true,
		settings.KeyTimezone:           true,
		settings.KeyLogRetentionDays:   true,
		settings.KeyDDNSCheckInterval:  true,
		settings.KeyCertRenewThreshold: true,
		settings.KeyRootDomain:         true,
		settings.KeyACMEEmail:          true,
		settings.KeyACMECA:             true,
		settings.KeyNotifyWebhookURL:   true,
		settings.KeyNotifyOnDDNSError:    true,
		settings.KeyNotifyOnCertError:    true,
		settings.KeyNotifyOnNginxError:   true,
	}
	filtered := map[string]string{}
	for key, value := range req {
		if allowed[key] {
			filtered[key] = value
		}
	}
	if err := h.store.SetMany(r.Context(), filtered); err != nil {
		writeError(w, http.StatusInternalServerError, "保存设置失败")
		return
	}
	values, err := h.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取设置失败")
		return
	}
	writeJSON(w, http.StatusOK, values)
}
