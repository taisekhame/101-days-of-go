package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver for database/sql
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
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

// New today: users and what they send to register or log in
type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // the "-" keeps it out of every JSON response
	CreatedAt    time.Time `json:"created_at"`
}

type Credentials struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"` // bcrypt only uses 72 bytes
}

var (
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("email already registered")
)

const jwtIssuer = "mtb-store-api"

// ---------- Config ----------

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

func loadConfig() (Config, error) {
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

	// New today: the secret that signs every token must be long
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

func safeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(could not read the URL)"
	}
	return u.Redacted()
}

// ---------- Store ----------

type Store struct {
	db *sql.DB
}

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

// New today: user queries

func (s *Store) CreateUser(ctx context.Context, email, hash string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at`,
		email, hash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	// 23505 is PostgreSQL's code for "this value breaks a UNIQUE rule"
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return User{}, ErrEmailTaken
	}
	return u, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByID(ctx context.Context, id int) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// ---------- Helpers ----------

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
		case "Email":
			return "a valid email is required"
		case "Password":
			return "password must be between 8 and 72 characters"
		}
	}
	return "invalid JSON: " + err.Error()
}

// Generic (Day 16): reads and validates a JSON body into any struct type
func bindJSON[T any](c *gin.Context) (T, bool) {
	var in T

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(&in); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": friendly(err)})
		return in, false
	}
	return in, true
}

func bindProduct(c *gin.Context) (ProductInput, bool) {
	in, ok := bindJSON[ProductInput](c)
	if !ok {
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

func respondError(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	log.Println("database error:", err)
	c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
}

// ---------- Product handlers ----------

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

// ---------- Auth: new today ----------

type Auth struct {
	store     *Store
	secret    []byte
	ttl       time.Duration
	dummyHash []byte // used to keep login timing the same for unknown emails
}

func unauthorized(c *gin.Context, message string) {
	c.Header("WWW-Authenticate", "Bearer")
	c.Abort()
	c.IndentedJSON(http.StatusUnauthorized, gin.H{"error": message})
}

func (a *Auth) register(c *gin.Context) {
	cred, ok := bindJSON[Credentials](c)
	if !ok {
		return
	}
	email := strings.ToLower(strings.TrimSpace(cred.Email))

	// Only the hash is ever stored, never the password itself
	hash, err := bcrypt.GenerateFromPassword([]byte(cred.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, err)
		return
	}

	user, err := a.store.CreateUser(c.Request.Context(), email, string(hash))
	if errors.Is(err, ErrEmailTaken) {
		c.IndentedJSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusCreated, user)
}

func (a *Auth) login(c *gin.Context) {
	cred, ok := bindJSON[Credentials](c)
	if !ok {
		return
	}
	email := strings.ToLower(strings.TrimSpace(cred.Email))

	user, err := a.store.UserByEmail(c.Request.Context(), email)
	if errors.Is(err, ErrNotFound) {
		// Do the same slow work as a real check, so timing doesn't reveal which emails exist
		_ = bcrypt.CompareHashAndPassword(a.dummyHash, []byte(cred.Password))
		unauthorized(c, "invalid email or password")
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cred.Password)) != nil {
		unauthorized(c, "invalid email or password") // same message as an unknown email
		return
	}

	token, err := a.issueToken(user.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"token":      token,
		"token_type": "Bearer",
		"expires_in": int(a.ttl.Seconds()),
	})
}

// Signs a token that says who the user is and when it stops working
func (a *Auth) issueToken(userID int) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    jwtIssuer,
		Subject:   strconv.Itoa(userID),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(a.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
}

// Middleware: lets a request through only if it carries a valid token
func (a *Auth) require(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		unauthorized(c, "missing or malformed Authorization header")
		return
	}
	raw := strings.TrimPrefix(header, "Bearer ")

	token, err := jwt.ParseWithClaims(raw, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) { return a.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), // refuse any other signing method
		jwt.WithIssuer(jwtIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		unauthorized(c, "invalid or expired token")
		return
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		unauthorized(c, "invalid or expired token")
		return
	}
	id, err := strconv.Atoi(claims.Subject)
	if err != nil {
		unauthorized(c, "invalid or expired token")
		return
	}

	c.Set("userID", id)
	c.Next()
}

// A protected route: shows who the token belongs to
func (a *Auth) me(c *gin.Context) {
	user, err := a.store.UserByID(c.Request.Context(), c.GetInt("userID"))
	if errors.Is(err, ErrNotFound) {
		unauthorized(c, "this account no longer exists")
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, user)
}

// ---------- Startup ----------

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	binding.EnableDecoderDisallowUnknownFields = true

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	gin.SetMode(cfg.GinMode)

	fmt.Printf("Config: addr=%s mode=%s max_conns=%d token_lifetime=%v\n",
		cfg.Addr, cfg.GinMode, cfg.MaxConns, cfg.JWTTTL)
	fmt.Println("Database:", safeURL(cfg.DatabaseURL))

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

	store := &Store{db: db}
	if err := store.Setup(ctx); err != nil {
		return fmt.Errorf("setting up the database: %w", err)
	}

	dummyHash, err := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	api := &API{store: store}
	auth := &Auth{
		store:     store,
		secret:    []byte(cfg.JWTSecret),
		ttl:       cfg.JWTTTL,
		dummyHash: dummyHash,
	}

	r := gin.Default()
	r.HandleMethodNotAllowed = true
	if err := r.SetTrustedProxies(nil); err != nil {
		return err
	}

	r.POST("/auth/register", auth.register)
	r.POST("/auth/login", auth.login)
	r.GET("/me", auth.require, auth.me)

	products := r.Group("/products")
	{
		// Anyone can read
		products.GET("", api.list)
		products.GET("/:id", api.get)

		// Only logged-in users can change things
		write := products.Group("", auth.require)
		write.POST("", api.create)
		write.PUT("/:id", api.update)
		write.DELETE("/:id", api.remove)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("MTB's Store API v7 (JWT auth) running on http://" + cfg.Addr)
	return srv.ListenAndServe()
}
