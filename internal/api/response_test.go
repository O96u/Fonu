package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
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

func TestWriteErrorLogsClientError(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(prev)

	req := httptest.NewRequest(http.MethodPost, "/api/settings/notify/test", nil)
	rec := httptest.NewRecorder()
	writeError(req, rec, http.StatusBadRequest, "请填写 SMTP 主机")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !bytes.Contains(buf.Bytes(), []byte("请填写 SMTP 主机")) {
		t.Fatalf("expected log message, got %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("/api/settings/notify/test")) {
		t.Fatalf("expected log path, got %s", buf.String())
	}
}
