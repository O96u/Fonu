package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
 "strings"
 "time"
 "net/url"
 "sync"

	"github.com/fonu/fonu/internal/notify"
 "github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/service"
	"github.com/fonu/fonu/internal/settings"
	"github.com/fonu/fonu/internal/validate"
)

type SettingsHandler struct {
 mu sync.Mutex
	store  *settings.Store
	proxy  *service.ProxyService
	notify *notify.Service
}

func NewSettingsHandler(store *settings.Store, proxySvc *service.ProxyService, notifySvc *notify.Service) *SettingsHandler {
	return &SettingsHandler{store: store, proxy: proxySvc, notify: notifySvc}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Cache-Control","no-store")
	values, err := h.store.List(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取设置失败", err)
		return
	}
	sanitizeSettingsResponse(values)
	writeJSON(w, http.StatusOK, values)
}

func (h *SettingsHandler) Put(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Cache-Control","no-store")
 h.mu.Lock();defer h.mu.Unlock()
 r.Body=http.MaxBytesReader(w,r.Body,1<<20)
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	allowed := map[string]bool{
		settings.KeyTheme:                        true,
		settings.KeyTimezone:                     true,
		settings.KeyLogRetentionDays:             true,
		settings.KeyDDNSCheckInterval:            true,
		settings.KeyCertRenewThreshold:           true,
		settings.KeyRootDomain:                   true,
		settings.KeyACMEEmail:                    true,
		settings.KeyACMECA:                       true,
		settings.KeyZeroSSLAPIKey:                true,
		settings.KeyZeroSSLEABKid:                true,
		settings.KeyZeroSSLEABHmac:               true,
		settings.KeyGoogleEABKid:                 true,
		settings.KeyGoogleEABHmac:                true,
		settings.KeySSLcomEABKid:                 true,
		settings.KeySSLcomEABHmac:                true,
		settings.KeyFreeSSLEABKid:                true,
		settings.KeyFreeSSLEABHmac:               true,
		settings.KeyFreeSSLDirectoryURL:          true,
		settings.KeyFreeSSLAutomationToken:       true,
		settings.KeyACMEDNSPropagationTimeout:    true,
		settings.KeyACMEDNSDisableAuthNS:         true,
		settings.KeyACMEDNSIgnorePropagation:     true,
		settings.KeyACMEDNSRecursiveNS:           true,
		settings.KeyActalisEABKid:                true,
		settings.KeyActalisEABHmac:               true,
		settings.KeyCustomACMEDirectoryURL:       true,
		settings.KeyCustomACMEEABKid:             true,
		settings.KeyCustomACMEEABHmac:            true,
		settings.KeyNotifyWebhookURL:             true,
		settings.KeyNotifyType:                   true,
		settings.KeyNotifyEmailJSON:              true,
		settings.KeyNotifySMTPPassword:           true,
		settings.KeyNotifyWebhookJSON:            true,
		settings.KeyNotifyWebhookSecret:          true,
		settings.KeyNotifyTelegramJSON:           true,
		settings.KeyNotifyTelegramToken:          true,
		settings.KeyNotifyOnDDNSIPChange:         true,
		settings.KeyNotifyOnDDNSFailure:          true,
		settings.KeyNotifyOnCertExpiry:           true,
		settings.KeyNotifyOnCertRenewSuccess:     true,
		settings.KeyNotifyOnIPFrequentAccess:     true,
 settings.KeyNotifyOnScan:true,
 settings.KeyNotifyOnCC:true,
		settings.KeyNotifyOnCertRenewFailure:     true,
		settings.KeyNotifyOnLoginFailure:         true,
		settings.KeyNotifyOnNginxReloadFailure:   true,
		settings.KeyNotifyIPFrequentThreshold:    true,
		settings.KeyNotifyIPFrequentWindowSec:    true,
		settings.KeyNotifyLoginFailureThreshold:  true,
		settings.KeyNotifyLoginFailureWindowSec:  true,
		settings.KeyTrustedProxy:                 true,
		settings.KeyGlobalIPBlacklist:            true,
		settings.KeyGlobalIPWhitelist:            true,
		settings.KeyChinaCIDRUpdateHours:         true,
		settings.KeyChinaCIDRSourceV4:            true,
		settings.KeyChinaCIDRSourceV6:            true,
		settings.KeyAnalyticsRetentionDays:       true,
		settings.KeyAnalyticsExcludePrivateIPs:   true,
		settings.KeyAnalyticsGeoRemoteEnabled:    true,
		settings.KeyAnalyticsGeoRemoteURL:        true,
		settings.KeyAnalyticsSkipHistoryBackfill: true,
	}
	filtered := map[string]string{}
	for key, value := range req {
		if !allowed[key] {
			continue
		}
        if isMaskedSetting(key) && value==notify.MaskedSecret {continue}
        if len(value)>128*1024 {writeError(r,w,http.StatusBadRequest,"设置值过长");return}
		if err := h.validateSetting(key, value); err != nil {
			writeError(r, w, http.StatusBadRequest, err.Error())
			return
		}
		filtered[key] = value
	}
	for key:=range filtered {if isNotifySettingKey(key)&&(h.notify==nil||h.notify.Store()==nil){writeError(r,w,http.StatusServiceUnavailable,"通知服务尚未初始化");return}}
	if h.notify != nil && h.notify.Store() != nil {
		existingNotify, loadErr := h.notify.Store().Load(r.Context())
        if loadErr!=nil {writeError(r,w,http.StatusInternalServerError,"读取通知设置失败");return}
		if in, ok := notify.ParseSaveInputFromMap(filtered, existingNotify); ok {
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
	_, trustedChanged := filtered[settings.KeyTrustedProxy]
	_, blacklistChanged := filtered[settings.KeyGlobalIPBlacklist]
	_, whitelistChanged := filtered[settings.KeyGlobalIPWhitelist]
	var saveErr error
	if h.proxy != nil && (trustedChanged || blacklistChanged || whitelistChanged) {
		saveErr = h.proxy.SaveSettingsWithPublication(r.Context(), filtered)
	} else if len(filtered)>0 {
		saveErr = h.store.SetMany(r.Context(), filtered)
	}
	if saveErr != nil {
		writeError(r,w,http.StatusBadRequest,"保存设置失败："+saveErr.Error(),saveErr)
		return
	}
	values, err := h.store.List(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取设置失败", err)
		return
	}
	sanitizeSettingsResponse(values)
	writeJSON(w, http.StatusOK, values)
}

func isMaskedSetting(key string) bool {
 switch key {
 case settings.KeyZeroSSLAPIKey,settings.KeyZeroSSLEABKid,settings.KeyZeroSSLEABHmac,settings.KeyGoogleEABKid,settings.KeyGoogleEABHmac,settings.KeySSLcomEABKid,settings.KeySSLcomEABHmac,settings.KeyFreeSSLEABKid,settings.KeyFreeSSLEABHmac,settings.KeyFreeSSLAutomationToken,settings.KeyActalisEABKid,settings.KeyActalisEABHmac,settings.KeyCustomACMEEABKid,settings.KeyCustomACMEEABHmac,settings.KeyFRPAuthToken,settings.KeyDNSHEAPIKey,settings.KeyDNSHEAPISecret:
 return true
 }
 return false
}
func sanitizeSettingsResponse(values map[string]string){
 notify.SanitizeSettings(values)
 for key,value:=range values {if isMaskedSetting(key)&&strings.TrimSpace(value)!=""{values[key]=notify.MaskedSecret}}
 // DNSHE multi-account storage belongs to the dedicated account API.
 delete(values,settings.KeyDNSHEAccounts)
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
 settings.KeyNotifyOnScan,
 settings.KeyNotifyOnCC,
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
 case settings.KeyTheme:
  if value!="system"&&value!="light"&&value!="dark"{return fmt.Errorf("无效的主题模式")}
 case settings.KeyTimezone:
  if strings.TrimSpace(value)==""{return fmt.Errorf("请选择时区")};if _,err:=time.LoadLocation(value);err!=nil{return fmt.Errorf("无效的时区")}
 case settings.KeyDDNSCheckInterval,settings.KeyCertRenewThreshold,settings.KeyChinaCIDRUpdateHours,settings.KeyACMEDNSPropagationTimeout:
  min,max:=1,1440
  switch key{case settings.KeyCertRenewThreshold:max=90;case settings.KeyChinaCIDRUpdateHours:max=168;case settings.KeyACMEDNSPropagationTimeout:min,max=30,900}
  n,err:=strconv.Atoi(value);if err!=nil||n<min||n>max{return fmt.Errorf("设置值需在 %d–%d 之间",min,max)}
 case settings.KeyFreeSSLDirectoryURL,settings.KeyCustomACMEDirectoryURL,settings.KeyAnalyticsGeoRemoteURL:
  if value!=""{u,err:=url.Parse(value);if err!=nil||u.Hostname()==""||(u.Scheme!="https"&&u.Scheme!="http")||u.User!=nil{return fmt.Errorf("请输入有效的 HTTP/HTTPS 服务地址")}}
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
	case settings.KeyLogRetentionDays, settings.KeyAnalyticsRetentionDays:
		if value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > 365 {
				return fmt.Errorf("保留天数需在 1–365 之间")
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
		if value!="" {var cfg nginx.TrustedProxyConfig;if err:=json.Unmarshal([]byte(value),&cfg);err!=nil{return fmt.Errorf("信任代理配置无效")};return cfg.Validate()}
	}
	return nil
}
