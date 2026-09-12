package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONNilSlice(t *testing.T) {
	var items []string
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, items)

	if rec.Body.String() != "[]" {
		t.Fatalf("expected [], got %q", rec.Body.String())
	}
}

func TestWriteJSONNilMap(t *testing.T) {
	var values map[string]string
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, values)

	if rec.Body.String() != "{}" {
		t.Fatalf("expected {}, got %q", rec.Body.String())
	}
}

func TestWriteJSONSlice(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, []int{1, 2})

	var out []int
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 items, got %v", out)
	}
}
