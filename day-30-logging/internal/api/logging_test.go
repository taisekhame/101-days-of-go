package api_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
)

func TestRequestLogging(t *testing.T) {
	h := newRouter(t)

	// newRouter silences the logger, so send this test's logs into a buffer instead
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	tests := []struct {
		name   string
		path   string
		status int
		level  string
	}{
		{"a success is logged as INFO", "/products", http.StatusOK, "INFO"},
		{"a missing product is logged as WARN", "/products/99", http.StatusNotFound, "WARN"},
		{"bad input is logged as WARN", "/products/abc", http.StatusBadRequest, "WARN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			send(t, h, http.MethodGet, tt.path, "", "")

			var entry struct {
				Level      string  `json:"level"`
				Msg        string  `json:"msg"`
				Method     string  `json:"method"`
				Path       string  `json:"path"`
				Status     int     `json:"status"`
				DurationMS float64 `json:"duration_ms"`
			}
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("expected one JSON log line, got %q: %v", buf.String(), err)
			}

			if entry.Level != tt.level || entry.Status != tt.status {
				t.Errorf("logged level %s and status %d, want %s and %d", entry.Level, entry.Status, tt.level, tt.status)
			}
			if entry.Msg != "request" || entry.Method != http.MethodGet || entry.Path != tt.path {
				t.Errorf("unexpected log entry: %+v", entry)
			}
		})
	}
}