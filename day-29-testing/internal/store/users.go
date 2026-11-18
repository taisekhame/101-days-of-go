package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/taisekhame/101-days-of-go/day-29-testing/internal/model"
)

func (s *Store) CreateUser(ctx context.Context, email, hash string) (model.User, error) {
	var u model.User
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at`,
		email, hash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	// 23505 is PostgreSQL's code for "this value breaks a UNIQUE rule"
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return model.User{}, model.ErrEmailTaken
	}
	return u, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrNotFound
	}
	return u, err
}

func (s *Store) UserByID(ctx context.Context, id int) (model.User, error) {
	var u model.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrNotFound
	}
	return u, err
}
