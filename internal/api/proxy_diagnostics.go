package api
import("net/http")
var proxyDiagnosticSlots=make(chan struct{},4)
func(h *ProxyHandler) Diagnostics(w http.ResponseWriter,r *http.Request){
 id,err:=parseID(r.PathValue("id"));if err!=nil{writeError(r,w,http.StatusBadRequest,"无效规则 ID");return}
 out,err:=h.svc.ProxyDiagnostics(r.Context(),id);if err!=nil{writeError(r,w,http.StatusBadRequest,"读取诊断记录失败");return}
 writeJSON(w,http.StatusOK,out)
}
func(h *ProxyHandler) RunDiagnostics(w http.ResponseWriter,r *http.Request){
 id,err:=parseID(r.PathValue("id"));if err!=nil{writeError(r,w,http.StatusBadRequest,"无效规则 ID");return}
 select{case proxyDiagnosticSlots<-struct{}{}:defer func(){<-proxyDiagnosticSlots}();default:writeError(r,w,http.StatusTooManyRequests,"诊断任务繁忙，请稍后重试");return}
 out,err:=h.svc.RunProxyDiagnostic(r.Context(),id);if err!=nil{writeError(r,w,http.StatusBadRequest,"诊断未保存：规则已变化或数据库不可用");return}
 writeJSON(w,http.StatusOK,out)
}
