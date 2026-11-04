package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fonu/fonu/internal/backup"
)

type BackupHandler struct {
	svc *backup.Service
}

func NewBackupHandler(svc *backup.Service) *BackupHandler {
	return &BackupHandler{svc: svc}
}

func (h *BackupHandler) Export(w http.ResponseWriter, r *http.Request) {
	path, err := h.svc.Export(r.Context())
	if err != nil {
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
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(r, w, http.StatusBadRequest, "上传文件无效")
		return
	}
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
	out.Close()

	if err := h.svc.Restore(r.Context(), tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	_ = os.Remove(tmpPath)
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "备份已恢复，请重新登录。",
	})
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	h.svc.NotifyRestored()
}
