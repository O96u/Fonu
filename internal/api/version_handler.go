package api

import (
	"net/http"

	"github.com/fonu/fonu/internal/version"
)

func Version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": version.Version,
	})
}
