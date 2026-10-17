package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
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

// What a client is allowed to send. The ID is never taken from the client.
type ProductInput struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// Returns an error message, or an empty string if the input is fine
func (in ProductInput) Validate() string {
	if strings.TrimSpace(in.Name) == "" {
		return "name is required"
	}
	if in.Price <= 0 {
		return "price must be greater than 0"
	}
	if in.Stock < 0 {
		return "stock cannot be negative"
	}
	return ""
}

// ---------- Store: the data layer ----------

type Store struct {
	mu       sync.RWMutex
	products map[int]Product
	nextID   int
}

func NewStore() *Store {
	s := &Store{products: make(map[int]Product), nextID: 1}
	s.Create(ProductInput{"Keyboard", 49.99, 25})
	s.Create(ProductInput{"Mouse", 19.99, 4})
	s.Create(ProductInput{"Monitor", 189.50, 7})
	return s
}

func (s *Store) List() []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Product, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Store) Get(id int) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.products[id]
	return p, ok
}

func (s *Store) Create(in ProductInput) Product {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := Product{ID: s.nextID, Name: strings.TrimSpace(in.Name), Price: in.Price, Stock: in.Stock}
	s.nextID++
	s.products[p.ID] = p
	return p
}

func (s *Store) Update(id int, in ProductInput) (Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.products[id]; !ok {
		return Product{}, false
	}
	p := Product{ID: id, Name: strings.TrimSpace(in.Name), Price: in.Price, Stock: in.Stock}
	s.products[id] = p
	return p, true
}

func (s *Store) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.products[id]; !ok {
		return false
	}
	delete(s.products, id)
	return true
}

// ---------- Helpers ----------

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		log.Println("could not write response:", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// Reads and validates the JSON body. If anything is wrong it writes the
// error response itself and returns false.
func decodeInput(w http.ResponseWriter, r *http.Request) (ProductInput, bool) {
	var in ProductInput

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return in, false
	}
	if msg := in.Validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return in, false
	}
	return in, true
}

// Pulls the id out of a path like /products/3
func idFromPath(path string) (int, bool) {
	id, err := strconv.Atoi(strings.TrimPrefix(path, "/products/"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// ---------- Handlers ----------

type API struct {
	store *Store
}

// /products: the collection
func (a *API) collection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.store.List())

	case http.MethodPost:
		in, ok := decodeInput(w, r)
		if !ok {
			return
		}
		created := a.store.Create(in)
		w.Header().Set("Location", fmt.Sprintf("/products/%d", created.ID))
		writeJSON(w, http.StatusCreated, created)

	default:
		methodNotAllowed(w, "GET, POST")
	}
}

// /products/{id}: one item in the collection
func (a *API) item(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		p, found := a.store.Get(id)
		if !found {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeJSON(w, http.StatusOK, p)

	case http.MethodPut:
		in, ok := decodeInput(w, r)
		if !ok {
			return
		}
		p, found := a.store.Update(id, in)
		if !found {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeJSON(w, http.StatusOK, p)

	case http.MethodDelete:
		if !a.store.Delete(id) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		w.WriteHeader(http.StatusNoContent) // success, nothing to send back

	default:
		methodNotAllowed(w, "GET, PUT, DELETE")
	}
}

// ---------- Middleware (from Day 22) ----------

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		ms := float64(time.Since(start).Microseconds()) / 1000
		log.Printf("%s %s -> %d (%.2fms)", r.Method, r.URL.Path, rec.status, ms)
	})
}

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println("recovered from panic:", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func main() {
	log.SetFlags(log.Ltime)

	api := &API{store: NewStore()}

	mux := http.NewServeMux()
	mux.HandleFunc("/products", api.collection)
	mux.HandleFunc("/products/", api.item)

	srv := &http.Server{
		Addr:              "localhost:8080",
		Handler:           logging(recovery(mux)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("MTB's Store API v3 running on http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}
