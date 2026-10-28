package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver for database/sql
)

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type ProductInput struct {
	Name  string  `json:"name" binding:"required"`
	Price float64 `json:"price" binding:"required,gt=0"`
	Stock int     `json:"stock" binding:"gte=0"`
}

var ErrNotFound = errors.New("product not found")

// ---------- Store: now backed by PostgreSQL ----------

type Store struct {
	db *sql.DB
}

// Creates the table the first time, and adds sample data if it's empty
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

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	samples := []ProductInput{
		{"Keyboard", 49.99, 25},
		{"Mouse", 19.99, 4},
		{"Monitor", 189.50, 7},
	}
	for _, in := range samples {
		if _, err := s.Create(ctx, in); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]Product, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, price, stock FROM products ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (s *Store) Get(ctx context.Context, id int) (Product, error) {
	var p Product
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, price, stock FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)

	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}

func (s *Store) Create(ctx context.Context, in ProductInput) (Product, error) {
	var p Product
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO products (name, price, stock) VALUES ($1, $2, $3)
		 RETURNING id, name, price, stock`,
		strings.TrimSpace(in.Name), in.Price, in.Stock,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)
	return p, err
}

func (s *Store) Update(ctx context.Context, id int, in ProductInput) (Product, error) {
	var p Product
	err := s.db.QueryRowContext(ctx,
		`UPDATE products SET name = $1, price = $2, stock = $3 WHERE id = $4
		 RETURNING id, name, price, stock`,
		strings.TrimSpace(in.Name), in.Price, in.Stock, id,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)

	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
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
		return ErrNotFound
	}
	return nil
}

// ---------- Helpers ----------

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func friendly(err error) string {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		switch verrs[0].Field() {
		case "Name":
			return "name is required"
		case "Price":
			return "price must be greater than 0"
		case "Stock":
			return "stock cannot be negative"
		}
	}
	return "invalid JSON: " + err.Error()
}

func bindProduct(c *gin.Context) (ProductInput, bool) {
	var in ProductInput

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(&in); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": friendly(err)})
		return in, false
	}
	if strings.TrimSpace(in.Name) == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return in, false
	}
	return in, true
}

func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return 0, false
	}
	return id, true
}

// Not found is the client's problem. Anything else is ours: log the real
// reason, but never show database details to the client.
func respondError(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	log.Println("database error:", err)
	c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
}

// ---------- Handlers ----------

type API struct {
	store *Store
}

func (a *API) list(c *gin.Context) {
	products, err := a.store.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, products)
}

func (a *API) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := a.store.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, p)
}

func (a *API) create(c *gin.Context) {
	in, ok := bindProduct(c)
	if !ok {
		return
	}
	created, err := a.store.Create(c.Request.Context(), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.Header("Location", fmt.Sprintf("/products/%d", created.ID))
	c.IndentedJSON(http.StatusCreated, created)
}

func (a *API) update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	in, ok := bindProduct(c)
	if !ok {
		return
	}
	p, err := a.store.Update(c.Request.Context(), id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, p)
}

func (a *API) remove(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := a.store.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- Startup ----------

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	binding.EnableDecoderDisallowUnknownFields = true

	// Config comes from environment variables, with a default for local dev
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:secret@localhost:5432/store?sslmode=disable")
	addr := getEnv("ADDR", "localhost:8080")

	db, err := sql.Open("pgx", dbURL) // this only prepares the pool, it doesn't connect yet
	if err != nil {
		return err
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Fail fast with a clear message if the database can't be reached
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("cannot reach the database: %w", err)
	}

	store := &Store{db: db}
	if err := store.Setup(ctx); err != nil {
		return fmt.Errorf("setting up the database: %w", err)
	}

	api := &API{store: store}

	r := gin.Default()
	r.HandleMethodNotAllowed = true
	if err := r.SetTrustedProxies(nil); err != nil {
		return err
	}

	products := r.Group("/products")
	{
		products.GET("", api.list)
		products.POST("", api.create)
		products.GET("/:id", api.get)
		products.PUT("/:id", api.update)
		products.DELETE("/:id", api.remove)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("MTB's Store API v5 (PostgreSQL) running on http://" + addr)
	return srv.ListenAndServe()
}
