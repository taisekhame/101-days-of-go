package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// The store keeps products in memory. Every request runs in its own
// goroutine, so the mutex stops two requests from changing the data at once.
type Store struct {
	mu       sync.RWMutex
	products []Product
	nextID   int
}

func NewStore() *Store {
	return &Store{
		products: []Product{
			{1, "Keyboard", 49.99, 25},
			{2, "Mouse", 19.99, 4},
			{3, "Monitor", 189.50, 7},
		},
		nextID: 4,
	}
}

func (s *Store) All() []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Product, len(s.products))
	copy(out, s.products)
	return out
}

func (s *Store) Add(p Product) Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID
	s.nextID++
	s.products = append(s.products, p)
	return p
}

type API struct {
	store *Store
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		log.Println("could not write response:", err)
	}
}

func errBody(message string) map[string]string {
	return map[string]string{"error": message}
}

// GET /products lists products, POST /products creates one
func (a *API) productsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.store.All())
	case http.MethodPost:
		a.createProduct(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, errBody("method not allowed"))
	}
}

func (a *API) createProduct(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
		Stock int     `json:"stock"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // refuse bodies over 1 MB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("invalid JSON: "+err.Error()))
		return
	}

	// Never trust what the client sends, check it first
	name := strings.TrimSpace(input.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, errBody("name is required"))
		return
	}
	if input.Price <= 0 {
		writeJSON(w, http.StatusBadRequest, errBody("price must be greater than 0"))
		return
	}
	if input.Stock < 0 {
		writeJSON(w, http.StatusBadRequest, errBody("stock cannot be negative"))
		return
	}

	created := a.store.Add(Product{Name: name, Price: input.Price, Stock: input.Stock})
	writeJSON(w, http.StatusCreated, created)
}

// Demo only: a route that crashes, to show the recovery middleware working
func boomHandler(w http.ResponseWriter, r *http.Request) {
	panic("something went very wrong")
}

// Wraps the response writer so we can remember the status code it sent
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

// Middleware 1: logs every request after it finishes
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		ms := float64(time.Since(start).Microseconds()) / 1000
		log.Printf("%s %s -> %d (%.2fms)", r.Method, r.URL.Path, rec.status, ms)
	})
}

// Middleware 2: catches a panic so one bad request can't take the server down
func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println("recovered from panic:", err)
				writeJSON(w, http.StatusInternalServerError, errBody("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func main() {
	log.SetFlags(log.Ltime)

	api := &API{store: NewStore()}

	mux := http.NewServeMux()
	mux.HandleFunc("/products", api.productsHandler)
	mux.HandleFunc("/boom", boomHandler)

	// Requests pass through logging first, then recovery, then the mux
	handler := logging(recovery(mux))

	srv := &http.Server{
		Addr:              "localhost:8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("MTB's Store API v2 running on http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}