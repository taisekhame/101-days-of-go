package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// In-memory data for now, a real database comes later
var products = []Product{
	{1, "Keyboard", 49.99, 25},
	{2, "Mouse", 19.99, 4},
	{3, "Monitor", 189.50, 7},
}

// Sends any value back to the client as JSON with a status code
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		log.Println("could not write response:", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Reads a value from the query string: /hello?name=Sam
func helloHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "MTB"
	}
	fmt.Fprintf(w, "Hello, %s!\n", name)
}

// GET /products          returns every product
// GET /products?id=2     returns one product
func productsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "only GET is allowed"})
		return
	}

	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		writeJSON(w, http.StatusOK, products)
		return
	}

	id, err := strconv.Atoi(idParam)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id must be a number"})
		return
	}

	for _, p := range products {
		if p.ID == id {
			writeJSON(w, http.StatusOK, p)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
}

func main() {
	// The mux decides which handler runs for which path
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/hello", helloHandler)
	mux.HandleFunc("/products", productsHandler)

	srv := &http.Server{
		Addr:              "localhost:8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("MTB's Store API running on http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}
