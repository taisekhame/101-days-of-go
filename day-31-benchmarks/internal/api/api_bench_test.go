package api_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/taisekhame/101-days-of-go/day-31-benchmarks/internal/api"
	"github.com/taisekhame/101-days-of-go/day-31-benchmarks/internal/auth"
	"github.com/taisekhame/101-days-of-go/day-31-benchmarks/internal/model"
)

// benchRouter builds the real router, with extra products added to the fake store
func benchRouter(b *testing.B, extraProducts int) (http.Handler, *auth.Service) {
	b.Helper()
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	// Only log errors, so these benchmarks measure the handlers and not the logger
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))

	authService, err := auth.New(testSecret, time.Hour)
	if err != nil {
		b.Fatal(err)
	}

	store := newFakeStore()
	for i := 0; i < extraProducts; i++ {
		_, _ = store.Create(context.Background(), model.ProductInput{
			Name: fmt.Sprintf("Product %d", i), Price: 9.99, Stock: i,
		})
	}
	if _, err := store.CreateUser(context.Background(), "mtb@example.com", "hash"); err != nil {
		b.Fatal(err)
	}

	router, err := api.NewRouter(store, store, authService)
	if err != nil {
		b.Fatal(err)
	}
	return router, authService
}

// benchGet sends the same GET request again and again and checks it succeeds
func benchGet(b *testing.B, h http.Handler, path, token string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d, want 200", rec.Code)
		}
	}
}

func BenchmarkListProducts3(b *testing.B) {
	h, _ := benchRouter(b, 0)
	benchGet(b, h, "/products", "")
}

func BenchmarkListProducts100(b *testing.B) {
	h, _ := benchRouter(b, 97)
	benchGet(b, h, "/products", "")
}

func BenchmarkPublicRoute(b *testing.B) {
	h, _ := benchRouter(b, 0)
	benchGet(b, h, "/products/1", "")
}

// A protected route: every request also checks the token and looks up the user
func BenchmarkProtectedRoute(b *testing.B) {
	h, svc := benchRouter(b, 0)
	token, err := svc.IssueToken(1)
	if err != nil {
		b.Fatal(err)
	}
	benchGet(b, h, "/me", token)
}

// Sub-benchmarks: the same 100 products, written as compact JSON and as indented JSON
func BenchmarkJSONvsIndentedJSON(b *testing.B) {
	products := make([]model.Product, 100)
	for i := range products {
		products[i] = model.Product{ID: i + 1, Name: fmt.Sprintf("Product %d", i), Price: 9.99, Stock: i}
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/compact", func(c *gin.Context) { c.JSON(http.StatusOK, products) })
	r.GET("/indented", func(c *gin.Context) { c.IndentedJSON(http.StatusOK, products) })

	for _, path := range []string{"/compact", "/indented"} {
		b.Run(path[1:], func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			}
		})
	}
}
