package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Addr        string
	GinMode     string
	MaxConns    int
	JWTSecret   string
	JWTTTL      time.Duration
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Load reads settings from a .env file (if there is one) and from the environment
func Load() (Config, error) {
	err := godotenv.Load()
	switch {
	case err == nil:
		fmt.Println("Loaded settings from .env")
	case errors.Is(err, os.ErrNotExist):
		fmt.Println("No .env file found, using environment variables only")
	default:
		return Config{}, fmt.Errorf("reading .env: %w", err)
	}

	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Addr:        getEnv("ADDR", "localhost:8080"),
		GinMode:     getEnv("GIN_MODE", "debug"),
		MaxConns:    10,
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTTTL:      60 * time.Minute,
	}

	var problems []string

	// Hosting platforms tell the app which port to use through PORT, and the app
	// must listen on every network interface there. ADDR still wins if it is set.
	if os.Getenv("ADDR") == "" {
		if port := os.Getenv("PORT"); port != "" {
			if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
				problems = append(problems, fmt.Sprintf("PORT must be a number between 1 and 65535, got %q", port))
			} else {
				cfg.Addr = ":" + port
			}
		}
	}

	if cfg.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is missing (copy .env.example to .env and fill it in)")
	}
	if cfg.GinMode != "debug" && cfg.GinMode != "release" {
		problems = append(problems, fmt.Sprintf("GIN_MODE must be debug or release, got %q", cfg.GinMode))
	}
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			problems = append(problems, fmt.Sprintf("DB_MAX_CONNS must be a whole number of at least 1, got %q", v))
		} else {
			cfg.MaxConns = n
		}
	}
	if len(cfg.JWTSecret) < 32 {
		problems = append(problems, "JWT_SECRET must be at least 32 characters long")
	}
	if v := os.Getenv("JWT_TTL_MINUTES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			problems = append(problems, fmt.Sprintf("JWT_TTL_MINUTES must be a whole number of at least 1, got %q", v))
		} else {
			cfg.JWTTTL = time.Duration(n) * time.Minute
		}
	}

	if len(problems) > 0 {
		return Config{}, errors.New("config problems:\n  - " + strings.Join(problems, "\n  - "))
	}
	return cfg, nil
}

// SafeURL hides the password so the URL is safe to print or screenshot
func SafeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(could not read the URL)"
	}
	return u.Redacted()
}
