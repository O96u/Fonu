package api

import (
	"encoding/json"
	"net/http"

	"github.com/fonu/fonu/internal/service"
)

type NginxHandler struct {
	proxy *service.ProxyService
}

func NewNginxHandler(proxy *service.ProxyService) *NginxHandler {
	return &NginxHandler{proxy: proxy}
}

type saveRuleNginxRequest struct {
	Mode    string  `json:"mode"`
	Content *string `json:"content"`
}

type rollbackNginxRequest struct {
	Backup string `json:"backup"`
}

func (h *NginxHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(r, w, http.StatusBadRequest, "无效的规则 ID")
		return
	}
	view, err := h.proxy.GetRuleNginx(r.Context(), id)
	if err != nil {
		writeServiceError(r, w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *NginxHandler) PutRule(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(r, w, http.StatusBadRequest, "无效的规则 ID")
		return
	}
	var req saveRuleNginxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	view, err := h.proxy.SaveRuleNginx(r.Context(), id, service.SaveRuleNginxInput{
		Mode:    req.Mode,
		Content: req.Content,
	})
	if err != nil {
		writeServiceError(r, w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *NginxHandler) RollbackRule(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(r, w, http.StatusBadRequest, "无效的规则 ID")
		return
	}
	var req rollbackNginxRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(r, w, http.StatusBadRequest, "请求格式无效")
			return
		}
	}
	view, err := h.proxy.RollbackRuleNginx(r.Context(), id, req.Backup)
	if err != nil {
		writeServiceError(r, w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *NginxHandler) GetGlobal(w http.ResponseWriter, r *http.Request) {
	view, err := h.proxy.GetGlobalNginx(r.Context())
	if err != nil {
		writeServiceError(r, w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *NginxHandler) PutGlobal(w http.ResponseWriter, r *http.Request) {
	var req saveRuleNginxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	view, err := h.proxy.SaveGlobalNginx(r.Context(), service.SaveGlobalNginxInput{
		Mode:    req.Mode,
		Content: req.Content,
	})
	if err != nil {
		writeServiceError(r, w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *NginxHandler) RollbackGlobal(w http.ResponseWriter, r *http.Request) {
	var req rollbackNginxRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(r, w, http.StatusBadRequest, "请求格式无效")
			return
		}
	}
	view, err := h.proxy.RollbackGlobalNginx(r.Context(), req.Backup)
	if err != nil {
		writeServiceError(r, w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func writeServiceError(r *http.Request, w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if msg == "规则不存在" {
		writeError(r, w, http.StatusNotFound, msg, err)
		return
	}
	writeError(r, w, http.StatusBadRequest, msg, err)
}
