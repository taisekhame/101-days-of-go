package api_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/taisekhame/101-days-of-go/day-32-integration-tests/internal/api"
	"github.com/taisekhame/101-days-of-go/day-32-integration-tests/internal/auth"
	"github.com/taisekhame/101-days-of-go/day-32-integration-tests/internal/model"
)

const testSecret = "test-secret-that-is-at-least-32-characters-long"

// fakeStore keeps everything in memory. It does the same job as the real
// PostgreSQL store, which is why the API can't tell the difference.
type fakeStore struct {
	mu       sync.Mutex
	products map[int]model.Product
	users    map[int]model.User
	nextProd int
	nextUser int
}

// If either interface changes, this stops compiling until the fake catches up
var (
	_ api.ProductStore = (*fakeStore)(nil)
	_ api.UserStore    = (*fakeStore)(nil)
)

func newFakeStore() *fakeStore {
	return &fakeStore{
		products: map[int]model.Product{
			1: {ID: 1, Name: "Keyboard", Price: 49.99, Stock: 25},
			2: {ID: 2, Name: "Mouse", Price: 19.99, Stock: 4},
			3: {ID: 3, Name: "Monitor", Price: 189.50, Stock: 7},
		},
		users:    map[int]model.User{},
		nextProd: 4,
		nextUser: 1,
	}
}

func (f *fakeStore) List(ctx context.Context) ([]model.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]model.Product, 0, len(f.products))
	for _, p := range f.products {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeStore) Get(ctx context.Context, id int) (model.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p, ok := f.products[id]
	if !ok {
		return model.Product{}, model.ErrNotFound
	}
	return p, nil
}

func (f *fakeStore) Create(ctx context.Context, in model.ProductInput) (model.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p := model.Product{ID: f.nextProd, Name: strings.TrimSpace(in.Name), Price: in.Price, Stock: in.Stock}
	f.nextProd++
	f.products[p.ID] = p
	return p, nil
}

func (f *fakeStore) Update(ctx context.Context, id int, in model.ProductInput) (model.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.products[id]; !ok {
		return model.Product{}, model.ErrNotFound
	}
	p := model.Product{ID: id, Name: strings.TrimSpace(in.Name), Price: in.Price, Stock: in.Stock}
	f.products[id] = p
	return p, nil
}

func (f *fakeStore) Delete(ctx context.Context, id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.products[id]; !ok {
		return model.ErrNotFound
	}
	delete(f.products, id)
	return nil
}

func (f *fakeStore) CreateUser(ctx context.Context, email, hash string) (model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, u := range f.users {
		if u.Email == email {
			return model.User{}, model.ErrEmailTaken
		}
	}
	u := model.User{ID: f.nextUser, Email: email, PasswordHash: hash, CreatedAt: time.Now()}
	f.nextUser++
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeStore) UserByEmail(ctx context.Context, email string) (model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return model.User{}, model.ErrNotFound
}

func (f *fakeStore) UserByID(ctx context.Context, id int) (model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.users[id]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return u, nil
}

// ---------- Helpers shared by the tests ----------

// newRouter builds the real router, wired to a fresh fake store
func newRouter(t *testing.T) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)

	gin.DefaultWriter = io.Discard // keep the request log out of the test output
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	authService, err := auth.New(testSecret, time.Hour)
	if err != nil {
		t.Fatalf("auth.New: %v", err)
	}
	store := newFakeStore()
	router, err := api.NewRouter(store, store, authService)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	return router
}

// send makes one request straight into the router, with no network involved
func send(t *testing.T, h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
