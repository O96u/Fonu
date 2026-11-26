package api

import (
 "encoding/json"
 "net/http"
 "github.com/fonu/fonu/internal/service"
)
func(h *ProxyHandler) BusinessGroups(w http.ResponseWriter,r *http.Request){
 out,err:=h.svc.ProxyBusiness(r.Context());if err!=nil{writeError(r,w,http.StatusInternalServerError,"读取业务分组失败");return};writeJSON(w,http.StatusOK,out)
}
func(h *ProxyHandler) ChangeBusinessGroups(w http.ResponseWriter,r *http.Request){
 var in service.ProxyBusinessChange
 r.Body=http.MaxBytesReader(w,r.Body,64<<10)
 if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{writeError(r,w,http.StatusBadRequest,"无效的分组请求");return}
 out,err:=h.svc.ChangeProxyBusiness(r.Context(),in);if err!=nil{writeError(r,w,http.StatusBadRequest,err.Error());return};writeJSON(w,http.StatusOK,out)
}
