package api

import (
	"encoding/json"
	"net/http"

	"github.com/fonu/fonu/internal/service"
)

func (h *ProxyHandler) LogBlacklistStatus(w http.ResponseWriter,r *http.Request){
	r.Body=http.MaxBytesReader(w,r.Body,65536)
	var req struct{Rows []service.LogBlacklistStatusInput `json:"rows"`}
	if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{writeError(r,w,http.StatusBadRequest,"请求格式无效");return}
	result,err:=h.svc.LogBlacklistStatuses(r.Context(),req.Rows)
	if err!=nil{writeError(r,w,http.StatusBadRequest,err.Error());return}
	w.Header().Set("Cache-Control","no-store")
	writeJSON(w,http.StatusOK,result)
}
