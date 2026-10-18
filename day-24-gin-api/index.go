package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// The validation rules now live in the struct tags
type ProductInput struct {
	Name  string  `json:"name" binding:"required"`
	Price float64 `json:"price" binding:"required,gt=0"`
	Stock int     `json:"stock" binding:"gte=0"`
}

// ---------- Store: exactly the same as Day 23 ----------

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

// Turns Gin's validation errors into short messages
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

// Reads and validates the JSON body, writing the error response if it fails
func bindProduct(c *gin.Context) (ProductInput, bool) {
	var in ProductInput

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(&in); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": friendly(err)})
		return in, false
	}
	// A name made only of spaces passes "required", so check it ourselves
	if strings.TrimSpace(in.Name) == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return in, false
	}
	return in, true
}

// Reads :id from the URL, writing the error response if it isn't valid
func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return 0, false
	}
	return id, true
}

// ---------- Handlers ----------

type API struct {
	store *Store
}

func (a *API) list(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, a.store.List())
}

func (a *API) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, found := a.store.Get(id)
	if !found {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.IndentedJSON(http.StatusOK, p)
}

func (a *API) create(c *gin.Context) {
	in, ok := bindProduct(c)
	if !ok {
		return
	}
	created := a.store.Create(in)
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
	p, found := a.store.Update(id, in)
	if !found {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.IndentedJSON(http.StatusOK, p)
}

func (a *API) remove(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !a.store.Delete(id) {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func main() {
	// Reject JSON fields we don't know about, like we did by hand on Day 22
	binding.EnableDecoderDisallowUnknownFields = true

	api := &API{store: NewStore()}

	// gin.Default() already includes a logger and panic recovery,
	// the two middleware we wrote ourselves on Day 22
	r := gin.Default()
	r.HandleMethodNotAllowed = true // wrong method gives 405 instead of 404
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
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
		Addr:              "localhost:8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("MTB's Store API v4 (Gin) running on http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}
