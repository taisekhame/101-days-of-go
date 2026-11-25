package api_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	h := newRouter(t)

	rec := send(t, h, http.MethodGet, "/healthz", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status": "ok"`) {
		t.Errorf("unexpected body: %s", rec.Body)
	}

	t.Run("it needs no token", func(t *testing.T) {
		if rec := send(t, h, http.MethodGet, "/healthz", "", ""); rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 without a token", rec.Code)
		}
	})

	t.Run("only GET is allowed", func(t *testing.T) {
		if rec := send(t, h, http.MethodPost, "/healthz", `{}`, ""); rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", rec.Code)
		}
	})
}
