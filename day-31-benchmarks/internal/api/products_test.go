package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/taisekhame/101-days-of-go/day-31-benchmarks/internal/model"
)

// loginToken registers a user and logs in through the real handlers
func loginToken(t *testing.T, h http.Handler) string {
	t.Helper()

	body := `{"email":"mtb@example.com","password":"mtb-password-123"}`
	if rec := send(t, h, http.MethodPost, "/auth/register", body, ""); rec.Code != http.StatusCreated {
		t.Fatalf("register: status %d, body %s", rec.Code, rec.Body)
	}

	rec := send(t, h, http.MethodPost, "/auth/login", body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status %d, body %s", rec.Code, rec.Body)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Token == "" {
		t.Fatalf("login did not return a token: %s", rec.Body)
	}
	return out.Token
}

func productCount(t *testing.T, h http.Handler) int {
	t.Helper()
	rec := send(t, h, http.MethodGet, "/products", "", "")
	var products []model.Product
	if err := json.Unmarshal(rec.Body.Bytes(), &products); err != nil {
		t.Fatalf("could not read the product list: %v", err)
	}
	return len(products)
}

func TestListProducts(t *testing.T) {
	h := newRouter(t)

	rec := send(t, h, http.MethodGet, "/products", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := productCount(t, h); got != 3 {
		t.Errorf("got %d products, want 3", got)
	}
}

func TestGetProduct(t *testing.T) {
	h := newRouter(t)

	tests := []struct {
		name string
		path string
		want int
	}{
		{"existing product", "/products/1", http.StatusOK},
		{"product that does not exist", "/products/99", http.StatusNotFound},
		{"id is not a number", "/products/abc", http.StatusBadRequest},
		{"id is zero", "/products/0", http.StatusBadRequest},
		{"id is negative", "/products/-4", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := send(t, h, http.MethodGet, tt.path, "", ""); rec.Code != tt.want {
				t.Errorf("GET %s = %d, want %d", tt.path, rec.Code, tt.want)
			}
		})
	}
}

func TestWritesRequireAToken(t *testing.T) {
	h := newRouter(t)
	body := `{"name":"Webcam","price":59.99,"stock":12}`

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"create", http.MethodPost, "/products"},
		{"update", http.MethodPut, "/products/1"},
		{"delete", http.MethodDelete, "/products/1"},
	}

	for _, tt := range tests {
		t.Run(tt.name+" without a token", func(t *testing.T) {
			rec := send(t, h, tt.method, tt.path, body, "")
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
			if rec.Header().Get("WWW-Authenticate") == "" {
				t.Error("a 401 should include a WWW-Authenticate header")
			}
		})

		t.Run(tt.name+" with a made-up token", func(t *testing.T) {
			rec := send(t, h, tt.method, tt.path, body, "not.a.real-token")
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
		})
	}

	// The important part: a refused request must not change anything
	if got := productCount(t, h); got != 3 {
		t.Errorf("refused requests changed the data: %d products, want 3", got)
	}
	if rec := send(t, h, http.MethodGet, "/products/1", "", ""); rec.Code != http.StatusOK {
		t.Errorf("product 1 should still exist, got status %d", rec.Code)
	}
}

func TestCreateProduct(t *testing.T) {
	h := newRouter(t)
	token := loginToken(t, h)

	rec := send(t, h, http.MethodPost, "/products", `{"name":"  Webcam  ","price":59.99,"stock":12}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body %s", rec.Code, rec.Body)
	}
	if loc := rec.Header().Get("Location"); loc != "/products/4" {
		t.Errorf("Location = %q, want /products/4", loc)
	}

	var created model.Product
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID != 4 || created.Name != "Webcam" || created.Price != 59.99 || created.Stock != 12 {
		t.Errorf("unexpected product: %+v", created)
	}

	// It really was saved
	if rec := send(t, h, http.MethodGet, "/products/4", "", ""); rec.Code != http.StatusOK {
		t.Errorf("GET /products/4 = %d, want 200", rec.Code)
	}
	if got := productCount(t, h); got != 4 {
		t.Errorf("got %d products, want 4", got)
	}
}

func TestCreateProductValidation(t *testing.T) {
	h := newRouter(t)
	token := loginToken(t, h)

	tests := []struct {
		name string
		body string
		want string // part of the error message
	}{
		{"missing name", `{"price":10,"stock":1}`, "name is required"},
		{"empty name", `{"name":"","price":10,"stock":1}`, "name is required"},
		{"name is only spaces", `{"name":"   ","price":10,"stock":1}`, "name is required"},
		{"price is zero", `{"name":"X","price":0,"stock":1}`, "price must be greater than 0"},
		{"price is negative", `{"name":"X","price":-5,"stock":1}`, "price must be greater than 0"},
		{"stock is negative", `{"name":"X","price":5,"stock":-2}`, "stock cannot be negative"},
		{"unknown field", `{"name":"X","price":5,"stock":1,"colour":"red"}`, "unknown field"},
		{"not JSON at all", `this is not json`, "invalid JSON"},
		{"wrong type", `{"name":"X","price":"cheap","stock":1}`, "invalid JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := send(t, h, http.MethodPost, "/products", tt.body, token)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body %s", rec.Code, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Errorf("error should mention %q, got %s", tt.want, rec.Body)
			}
		})
	}

	if got := productCount(t, h); got != 3 {
		t.Errorf("rejected products were saved anyway: %d products, want 3", got)
	}
}

func TestUpdateProduct(t *testing.T) {
	h := newRouter(t)
	token := loginToken(t, h)

	rec := send(t, h, http.MethodPut, "/products/2", `{"name":"Gaming Mouse","price":39.99,"stock":15}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body %s", rec.Code, rec.Body)
	}

	var got model.Product
	_ = json.Unmarshal(send(t, h, http.MethodGet, "/products/2", "", "").Body.Bytes(), &got)
	if got.Name != "Gaming Mouse" || got.Price != 39.99 || got.Stock != 15 {
		t.Errorf("the update was not saved: %+v", got)
	}

	t.Run("product that does not exist", func(t *testing.T) {
		rec := send(t, h, http.MethodPut, "/products/99", `{"name":"X","price":5,"stock":1}`, token)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestDeleteProduct(t *testing.T) {
	h := newRouter(t)
	token := loginToken(t, h)

	if rec := send(t, h, http.MethodDelete, "/products/1", "", token); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if rec := send(t, h, http.MethodGet, "/products/1", "", ""); rec.Code != http.StatusNotFound {
		t.Errorf("a deleted product should be gone, got status %d", rec.Code)
	}
	if rec := send(t, h, http.MethodDelete, "/products/1", "", token); rec.Code != http.StatusNotFound {
		t.Errorf("deleting twice should give 404, got %d", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := newRouter(t)
	token := loginToken(t, h)

	rec := send(t, h, http.MethodPatch, "/products/1", `{}`, token)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
