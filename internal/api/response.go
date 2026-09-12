package api

import (
	"encoding/json"
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

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorBody{Error: message})
}
