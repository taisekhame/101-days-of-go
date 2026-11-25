package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-that-is-at-least-32-characters-long"

func newService(t *testing.T, ttl time.Duration) *Service {
	t.Helper()
	svc, err := New(testSecret, ttl)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return svc
}

func TestHashPassword(t *testing.T) {
	svc := newService(t, time.Hour)

	hash, err := svc.HashPassword("mtb-password-123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "mtb-password-123" {
		t.Fatal("the hash must not be the password itself")
	}
	if !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("expected a bcrypt hash, got %q", hash[:4])
	}

	// bcrypt adds a random salt, so hashing the same password twice gives different results
	again, _ := svc.HashPassword("mtb-password-123")
	if hash == again {
		t.Error("two hashes of the same password should differ")
	}
}

func TestCheckPassword(t *testing.T) {
	svc := newService(t, time.Hour)
	hash, err := svc.HashPassword("mtb-password-123")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"correct password", "mtb-password-123", true},
		{"wrong password", "mtb-password-124", false},
		{"different case", "MTB-PASSWORD-123", false},
		{"empty password", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.CheckPassword(hash, tt.password); got != tt.want {
				t.Errorf("CheckPassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

func TestIssueAndParseToken(t *testing.T) {
	svc := newService(t, time.Hour)

	token, err := svc.IssueToken(42)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if parts := strings.Split(token, "."); len(parts) != 3 {
		t.Fatalf("a JWT has three parts, got %d", len(parts))
	}

	id, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if id != 42 {
		t.Errorf("user id = %d, want 42", id)
	}
}

// signed builds a token with any claims and signing method, for the bad-token tests
func signed(t *testing.T, method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("signing test token: %v", err)
	}
	return token
}

func TestParseTokenRejectsBadTokens(t *testing.T) {
	future := jwt.NewNumericDate(time.Now().Add(time.Hour))
	good := jwt.RegisteredClaims{Issuer: issuer, Subject: "1", ExpiresAt: future}

	tests := []struct {
		name  string
		token func(t *testing.T) string
	}{
		{"empty string", func(t *testing.T) string { return "" }},
		{"garbage", func(t *testing.T) string { return "not.a.token" }},
		{"expired", func(t *testing.T) string {
			token, _ := newService(t, -time.Minute).IssueToken(1)
			return token
		}},
		{"signed with a different secret", func(t *testing.T) string {
			other, _ := New("a-completely-different-secret-value-0123456789", time.Hour)
			token, _ := other.IssueToken(1)
			return token
		}},
		{"no expiry claim", func(t *testing.T) string {
			c := good
			c.ExpiresAt = nil
			return signed(t, jwt.SigningMethodHS256, []byte(testSecret), c)
		}},
		{"wrong issuer", func(t *testing.T) string {
			c := good
			c.Issuer = "someone-else"
			return signed(t, jwt.SigningMethodHS256, []byte(testSecret), c)
		}},
		{"alg none (no signature)", func(t *testing.T) string {
			return signed(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, good)
		}},
		{"HS512 signed with the right secret", func(t *testing.T) string {
			return signed(t, jwt.SigningMethodHS512, []byte(testSecret), good)
		}},
		{"subject is not a number", func(t *testing.T) string {
			c := good
			c.Subject = "abc"
			return signed(t, jwt.SigningMethodHS256, []byte(testSecret), c)
		}},
		{"payload edited, old signature kept", func(t *testing.T) string {
			token, _ := newService(t, time.Hour).IssueToken(1)
			parts := strings.Split(token, ".")
			payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
			var claims map[string]any
			_ = json.Unmarshal(payload, &claims)
			claims["sub"] = "2" // pretend to be someone else
			edited, _ := json.Marshal(claims)
			parts[1] = base64.RawURLEncoding.EncodeToString(edited)
			return strings.Join(parts, ".")
		}},
	}

	svc := newService(t, time.Hour)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if id, err := svc.ParseToken(tt.token(t)); err == nil {
				t.Errorf("expected an error, but the token was accepted (user id %d)", id)
			}
		})
	}
}
