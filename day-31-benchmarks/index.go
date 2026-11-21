package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver for database/sql

	"github.com/taisekhame/101-days-of-go/day-31-benchmarks/internal/api"
	"github.com/taisekhame/101-days-of-go/day-31-benchmarks/internal/auth"
	"github.com/taisekhame/101-days-of-go/day-31-benchmarks/internal/config"
	"github.com/taisekhame/101-days-of-go/day-31-benchmarks/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run only connects the pieces together. The real work lives in the packages.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	gin.SetMode(cfg.GinMode)
	setupLogger(cfg.GinMode)

	fmt.Printf("Config: addr=%s mode=%s max_conns=%d token_lifetime=%v\n",
		cfg.Addr, cfg.GinMode, cfg.MaxConns, cfg.JWTTTL)
	fmt.Println("Database:", config.SafeURL(cfg.DatabaseURL))

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("cannot reach the database: %w", err)
	}

	st := store.New(db)
	if err := st.Setup(ctx); err != nil {
		return fmt.Errorf("setting up the database: %w", err)
	}

	authService, err := auth.New(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return err
	}

	// The store satisfies both interfaces the API asks for
	router, err := api.NewRouter(st, st, authService)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Debug("database ready", "max_conns", cfg.MaxConns)
	slog.Info("MTB's Store API v11 (benchmarks) started", "addr", cfg.Addr, "mode", cfg.GinMode)

	return srv.ListenAndServe()
}

// Readable text while developing, JSON in production
func setupLogger(mode string) {
	var handler slog.Handler
	if mode == "release" {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(handler))
}
