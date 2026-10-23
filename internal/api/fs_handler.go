package api

import (
	"net/http"

	"github.com/fonu/fonu/internal/fsbrowse"
)

type FSHandler struct{}

func NewFSHandler() *FSHandler {
	return &FSHandler{}
}

func (h *FSHandler) ListDirectories(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	result, err := fsbrowse.ListDirectories(path)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
