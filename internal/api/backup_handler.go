package api

import (
	"io"
 "errors"
 "context"
 "time"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fonu/fonu/internal/backup"
)

func backupBusy(w http.ResponseWriter,r *http.Request,err error)bool{
 if !errors.Is(err,backup.ErrBusy){return false};writeError(r,w,http.StatusConflict,"备份或恢复任务正在执行，请稍后重试");return true
}

type BackupHandler struct {
	svc *backup.Service
}

func NewBackupHandler(svc *backup.Service) *BackupHandler {
	return &BackupHandler{svc: svc}
}

func (h *BackupHandler) Export(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Cache-Control","no-store")
	ctx,cancel:=context.WithTimeout(r.Context(),10*time.Minute);defer cancel()
 path, err := h.svc.Export(ctx)
	if err != nil {
  if backupBusy(w,r,err){return}
		writeError(r, w, http.StatusInternalServerError, "创建备份失败")
		return
	}
	defer os.Remove(path)

	file, err := os.Open(path)
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取备份失败")
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(path))
	_, _ = io.Copy(w, file)
}

func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Cache-Control","no-store")
	r.Body = http.MaxBytesReader(w, r.Body, 512 << 20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(r, w, http.StatusBadRequest, "上传文件无效")
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(r, w, http.StatusBadRequest, "请选择备份文件")
		return
	}
	defer file.Close()

	out, err := os.CreateTemp("", "fonu-restore-*.tar.gz")
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "保存上传文件失败")
		return
	}
	tmpPath := out.Name()
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		_ = os.Remove(tmpPath)
		writeError(r, w, http.StatusInternalServerError, "保存上传文件失败")
		return
	}
	if err:=out.Close();err!=nil{_ = os.Remove(tmpPath);writeError(r,w,http.StatusInternalServerError,"保存上传文件失败");return}

	if err := h.svc.Restore(r.Context(), tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		if backupBusy(w,r,err){return}
  writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	_ = os.Remove(tmpPath)
 h.restored(w)
}

func(h *BackupHandler)restored(w http.ResponseWriter){
 clearSessionCookie(w)
 writeJSON(w,http.StatusOK,map[string]string{
		"message": "备份已恢复，请重新登录。",
	})
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	h.svc.NotifyRestored()
}

func(h *BackupHandler)ListLocal(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Cache-Control","no-store")
 rows,err:=h.svc.ListLocal();if err!=nil{if backupBusy(w,r,err){return};writeError(r,w,500,"读取本地备份失败");return};writeJSON(w,200,rows)
}
func(h *BackupHandler)CreateLocal(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Cache-Control","no-store")
 ctx,cancel:=context.WithTimeout(r.Context(),10*time.Minute);defer cancel()
 row,err:=h.svc.CreateLocal(ctx);if err!=nil{if backupBusy(w,r,err){return};writeError(r,w,400,"创建备份失败，请检查磁盘空间或备份数量");return};writeJSON(w,201,row)
}
func(h *BackupHandler)DownloadLocal(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Cache-Control","no-store")
 f,err:=h.svc.OpenLocal(r.PathValue("name"));if err!=nil{if backupBusy(w,r,err){return};writeError(r,w,404,"备份不存在");return};defer f.Close()
 info,err:=f.Stat();if err!=nil{writeError(r,w,500,"读取失败");return}
 w.Header().Set("Content-Type","application/gzip")
 w.Header().Set("Content-Disposition","attachment; filename="+info.Name())
 http.ServeContent(w,r,info.Name(),info.ModTime(),f)
}
func(h *BackupHandler)DeleteLocal(w http.ResponseWriter,r *http.Request){
 if err:=h.svc.DeleteLocal(r.PathValue("name"));err!=nil{if backupBusy(w,r,err){return};writeError(r,w,400,"删除备份失败");return};writeJSON(w,200,map[string]string{"message":"已删除"})
}
func(h *BackupHandler)RestoreLocal(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Cache-Control","no-store")
 ctx,cancel:=context.WithTimeout(r.Context(),10*time.Minute);defer cancel()
 if err:=h.svc.RestoreLocal(ctx,r.PathValue("name"));err!=nil{if backupBusy(w,r,err){return};writeError(r,w,400,"恢复失败：备份无效、密钥不匹配或已有待恢复任务");return};h.restored(w)
}
