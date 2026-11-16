package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/taisekhame/101-days-of-go/day-28-project-structure/internal/model"
)

func (s *Store) List(ctx context.Context) ([]model.Product, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, price, stock FROM products ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (s *Store) Get(ctx context.Context, id int) (model.Product, error) {
	var p model.Product
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, price, stock FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)

	if errors.Is(err, sql.ErrNoRows) {
		return model.Product{}, model.ErrNotFound
	}
	return p, err
}

func (s *Store) Create(ctx context.Context, in model.ProductInput) (model.Product, error) {
	var p model.Product
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO products (name, price, stock) VALUES ($1, $2, $3)
		 RETURNING id, name, price, stock`,
		strings.TrimSpace(in.Name), in.Price, in.Stock,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)
	return p, err
}

func (s *Store) Update(ctx context.Context, id int, in model.ProductInput) (model.Product, error) {
	var p model.Product
	err := s.db.QueryRowContext(ctx,
		`UPDATE products SET name = $1, price = $2, stock = $3 WHERE id = $4
		 RETURNING id, name, price, stock`,
		strings.TrimSpace(in.Name), in.Price, in.Stock, id,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)

	if errors.Is(err, sql.ErrNoRows) {
		return model.Product{}, model.ErrNotFound
	}
	return p, err
}

func (s *Store) Delete(ctx context.Context, id int) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}
