package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver for database/sql

	"github.com/taisekhame/101-days-of-go/day-28-project-structure/internal/api"
	"github.com/taisekhame/101-days-of-go/day-28-project-structure/internal/auth"
	"github.com/taisekhame/101-days-of-go/day-28-project-structure/internal/config"
	"github.com/taisekhame/101-days-of-go/day-28-project-structure/internal/store"
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

	fmt.Println("MTB's Store API v8 (project structure) running on http://" + cfg.Addr)
	return srv.ListenAndServe()
}
