package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/taisekhame/101-days-of-go/day-29-testing/internal/auth"
)

func TestRegister(t *testing.T) {
	h := newRouter(t)

	rec := send(t, h, http.MethodPost, "/auth/register", `{"email":"MTB@Example.com","password":"mtb-password-123"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body %s", rec.Code, rec.Body)
	}

	body := rec.Body.String()
	if strings.Contains(body, "password") || strings.Contains(body, "$2a$") {
		t.Errorf("the response must never include the password or its hash: %s", body)
	}
	if !strings.Contains(body, `"email": "mtb@example.com"`) {
		t.Errorf("the email should be stored in lower case: %s", body)
	}

	t.Run("the same email again, in a different case", func(t *testing.T) {
		rec := send(t, h, http.MethodPost, "/auth/register", `{"email":"mtb@example.com","password":"another-password-1"}`, "")
		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409", rec.Code)
		}
	})
}

func TestRegisterValidation(t *testing.T) {
	h := newRouter(t)

	tests := []struct {
		name string
		body string
		want string
	}{
		{"not an email", `{"email":"nope","password":"mtb-password-123"}`, "a valid email is required"},
		{"missing email", `{"password":"mtb-password-123"}`, "a valid email is required"},
		{"password too short", `{"email":"a@example.com","password":"short"}`, "between 8 and 72"},
		{"password too long", `{"email":"a@example.com","password":"` + strings.Repeat("x", 73) + `"}`, "between 8 and 72"},
		{"missing password", `{"email":"a@example.com"}`, "between 8 and 72"},
		{"unknown field", `{"email":"a@example.com","password":"long-enough-1","role":"admin"}`, "unknown field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := send(t, h, http.MethodPost, "/auth/register", tt.body, "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body %s", rec.Code, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Errorf("error should mention %q, got %s", tt.want, rec.Body)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	h := newRouter(t)
	send(t, h, http.MethodPost, "/auth/register", `{"email":"mtb@example.com","password":"mtb-password-123"}`, "")

	t.Run("correct password returns a token that works", func(t *testing.T) {
		rec := send(t, h, http.MethodPost, "/auth/login", `{"email":"mtb@example.com","password":"mtb-password-123"}`, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var out struct {
			Token     string `json:"token"`
			TokenType string `json:"token_type"`
			ExpiresIn int    `json:"expires_in"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.Token == "" || out.TokenType != "Bearer" || out.ExpiresIn != 3600 {
			t.Errorf("unexpected login response: %+v", out)
		}

		created := send(t, h, http.MethodPost, "/products", `{"name":"Webcam","price":59.99,"stock":12}`, out.Token)
		if created.Code != http.StatusCreated {
			t.Errorf("the token should allow creating a product, got status %d", created.Code)
		}
	})

	t.Run("email is not case sensitive", func(t *testing.T) {
		rec := send(t, h, http.MethodPost, "/auth/login", `{"email":"MTB@EXAMPLE.COM","password":"mtb-password-123"}`, "")
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("wrong password and unknown email look identical", func(t *testing.T) {
		wrong := send(t, h, http.MethodPost, "/auth/login", `{"email":"mtb@example.com","password":"wrong-password-1"}`, "")
		unknown := send(t, h, http.MethodPost, "/auth/login", `{"email":"nobody@example.com","password":"wrong-password-1"}`, "")

		if wrong.Code != http.StatusUnauthorized || unknown.Code != http.StatusUnauthorized {
			t.Fatalf("both should be 401, got %d and %d", wrong.Code, unknown.Code)
		}
		if wrong.Body.String() != unknown.Body.String() {
			t.Errorf("the messages differ, which reveals which emails exist:\n%s\n%s", wrong.Body, unknown.Body)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		rec := send(t, h, http.MethodPost, "/auth/login", `{"email":"mtb@example.com"}`, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestMe(t *testing.T) {
	h := newRouter(t)
	token := loginToken(t, h)

	rec := send(t, h, http.MethodGet, "/me", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "mtb@example.com") {
		t.Errorf("/me should show the logged-in user: %s", rec.Body)
	}

	if rec := send(t, h, http.MethodGet, "/me", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("/me without a token = %d, want 401", rec.Code)
	}
}

func TestExpiredTokenIsRefused(t *testing.T) {
	h := newRouter(t)

	// A service that signs with the same secret, but whose tokens are already expired
	old, err := auth.New(testSecret, -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	token, err := old.IssueToken(1)
	if err != nil {
		t.Fatal(err)
	}

	rec := send(t, h, http.MethodPost, "/products", `{"name":"Webcam","price":59.99,"stock":12}`, token)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
