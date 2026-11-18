// Package store is the only place that talks to PostgreSQL.
package store

import (
	"context"
	"database/sql"

	"github.com/taisekhame/101-days-of-go/day-29-testing/internal/model"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Setup creates the tables the first time, and adds sample products if there are none
func (s *Store) Setup(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS products (
			id    SERIAL PRIMARY KEY,
			name  TEXT NOT NULL,
			price NUMERIC(10,2) NOT NULL CHECK (price > 0),
			stock INTEGER NOT NULL CHECK (stock >= 0)
		)`)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id            SERIAL PRIMARY KEY,
			email         TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return err
	}

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	samples := []model.ProductInput{
		{Name: "Keyboard", Price: 49.99, Stock: 25},
		{Name: "Mouse", Price: 19.99, Stock: 4},
		{Name: "Monitor", Price: 189.50, Stock: 7},
	}
	for _, in := range samples {
		if _, err := s.Create(ctx, in); err != nil {
			return err
		}
	}
	return nil
}
