package api

import (
	"encoding/json"
	"net/http"

	"github.com/fonu/fonu/internal/notify"
)

type NotifyHandler struct {
	notify *notify.Service
}

func NewNotifyHandler(notifySvc *notify.Service) *NotifyHandler {
	return &NotifyHandler{notify: notifySvc}
}

func (h *NotifyHandler) Test(w http.ResponseWriter, r *http.Request) {
	if h.notify == nil {
		writeError(r, w, http.StatusInternalServerError, "通知服务未初始化")
		return
	}
	var req notify.TestInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if err := h.notify.SendTest(r.Context(), req); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error(), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "测试通知已发送"})
}
