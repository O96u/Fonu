package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
)

type ErrorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		_, _ = w.Write([]byte("null"))
		return
	}
	// Go nil slices/maps encode as JSON null; clients expect [] / {}.
	v := reflect.ValueOf(payload)
	switch v.Kind() {
	case reflect.Slice:
		if v.IsNil() {
			_, _ = w.Write([]byte("[]"))
			return
		}
	case reflect.Map:
		if v.IsNil() {
			_, _ = w.Write([]byte("{}"))
			return
		}
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(r *http.Request, w http.ResponseWriter, status int, message string, cause ...error) {
	logAPIError(r, status, message, cause)
	writeJSON(w, status, ErrorBody{Error: message})
}

func logAPIError(r *http.Request, status int, message string, cause []error) {
	if status < 400 {
		return
	}
	attrs := []any{
		"module", "API",
		"status", status,
		"message", message,
	}
	if r != nil {
		attrs = append(attrs, "method", r.Method, "path", r.URL.Path)
	}
	if len(cause) > 0 && cause[0] != nil {
		attrs = append(attrs, "error", cause[0].Error())
	}
	switch {
	case status >= 500:
		slog.Error("request failed", attrs...)
	default:
		slog.Warn("request rejected", attrs...)
	}
}
