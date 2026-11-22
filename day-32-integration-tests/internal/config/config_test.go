package config

import (
	"strings"
	"testing"
	"time"
)

const goodSecret = "test-secret-that-is-at-least-32-characters-long"

// setEnv gives every setting a known value for one test. An empty string
// counts as "not set", so a real value on your machine can't leak into the test.
func setEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for _, key := range []string{"DATABASE_URL", "ADDR", "GIN_MODE", "DB_MAX_CONNS", "JWT_SECRET", "JWT_TTL_MINUTES"} {
		t.Setenv(key, values[key])
	}
}

func TestLoadDefaults(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://user:pass@localhost:5432/store",
		"JWT_SECRET":   goodSecret,
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != "localhost:8080" {
		t.Errorf("Addr = %q, want the default localhost:8080", cfg.Addr)
	}
	if cfg.GinMode != "debug" {
		t.Errorf("GinMode = %q, want debug", cfg.GinMode)
	}
	if cfg.MaxConns != 10 {
		t.Errorf("MaxConns = %d, want 10", cfg.MaxConns)
	}
	if cfg.JWTTTL != time.Hour {
		t.Errorf("JWTTTL = %v, want 1h", cfg.JWTTTL)
	}
}

func TestLoadOverrides(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL":    "postgres://user:pass@localhost:5432/store",
		"JWT_SECRET":      goodSecret,
		"ADDR":            "localhost:9090",
		"GIN_MODE":        "release",
		"DB_MAX_CONNS":    "5",
		"JWT_TTL_MINUTES": "15",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != "localhost:9090" || cfg.GinMode != "release" || cfg.MaxConns != 5 || cfg.JWTTTL != 15*time.Minute {
		t.Errorf("settings were not read correctly: %+v", cfg)
	}
}

func TestLoadProblems(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want []string // every one of these must appear in the error
	}{
		{
			name: "missing database url",
			env:  map[string]string{"JWT_SECRET": goodSecret},
			want: []string{"DATABASE_URL is missing"},
		},
		{
			name: "secret too short",
			env:  map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": "change-me"},
			want: []string{"JWT_SECRET must be at least 32 characters"},
		},
		{
			name: "unknown gin mode",
			env:  map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": goodSecret, "GIN_MODE": "staging"},
			want: []string{"GIN_MODE must be debug or release"},
		},
		{
			name: "max connections is not a number",
			env:  map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": goodSecret, "DB_MAX_CONNS": "abc"},
			want: []string{"DB_MAX_CONNS must be a whole number"},
		},
		{
			name: "max connections is zero",
			env:  map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": goodSecret, "DB_MAX_CONNS": "0"},
			want: []string{"DB_MAX_CONNS must be a whole number"},
		},
		{
			name: "token lifetime is negative",
			env:  map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": goodSecret, "JWT_TTL_MINUTES": "-5"},
			want: []string{"JWT_TTL_MINUTES must be a whole number"},
		},
		{
			name: "everything wrong at once lists every problem",
			env:  map[string]string{"GIN_MODE": "staging", "DB_MAX_CONNS": "abc", "JWT_TTL_MINUTES": "x"},
			want: []string{"DATABASE_URL", "JWT_SECRET", "GIN_MODE", "DB_MAX_CONNS", "JWT_TTL_MINUTES"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.env)

			_, err := Load()
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			for _, fragment := range tt.want {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("error should mention %q, got:\n%v", fragment, err)
				}
			}
		})
	}
}

func TestSafeURL(t *testing.T) {
	t.Run("hides the password", func(t *testing.T) {
		got := SafeURL("postgres://neondb_owner:supersecret@host.example.com/neondb?sslmode=require")
		if strings.Contains(got, "supersecret") {
			t.Fatalf("the password leaked: %s", got)
		}
		if !strings.Contains(got, "xxxxx") || !strings.Contains(got, "host.example.com") {
			t.Errorf("unexpected output: %s", got)
		}
	})

	t.Run("an unreadable URL is never echoed back", func(t *testing.T) {
		got := SafeURL("://missing-scheme:password@host")
		if strings.Contains(got, "password") {
			t.Fatalf("the unreadable URL leaked: %s", got)
		}
	})
}
