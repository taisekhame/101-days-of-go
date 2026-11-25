package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"github.com/taisekhame/101-days-of-go/day-33-production-ready/internal/model"
	"github.com/taisekhame/101-days-of-go/day-33-production-ready/internal/store"
)

// newTestStore gives each test its own empty copy of the tables, inside a
// temporary schema, so these tests never touch your real data. When the test
// ends, the schema is dropped.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping the database tests in -short mode")
	}

	_ = godotenv.Load("../../.env") // the day's .env file, if there is one
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set, skipping the database tests")
	}

	// One connection with no special settings, used only to create and drop the schema
	admin, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	schema := fmt.Sprintf("it_%d_%d", time.Now().UnixNano(), rand.Intn(1000))
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("could not create the test schema: %v", err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("could not drop the test schema %s: %v", schema, err)
		}
	})

	// A second connection pool whose tables all live in that schema
	u, err := url.Parse(dbURL)
	if err != nil {
		t.Fatal("DATABASE_URL could not be read")
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	s := store.New(db)
	if err := s.Setup(ctx); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	return s
}

func TestSetup(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	products, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 3 || products[0].Name != "Keyboard" || products[1].Name != "Mouse" || products[2].Name != "Monitor" {
		t.Fatalf("expected the three sample products, got %+v", products)
	}

	// Running Setup again must not add the samples twice
	if err := s.Setup(ctx); err != nil {
		t.Fatal(err)
	}
	if again, _ := s.List(ctx); len(again) != 3 {
		t.Errorf("Setup is not safe to repeat: %d products after the second run", len(again))
	}
}

func TestProductLifecycle(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, model.ProductInput{Name: "  Webcam  ", Price: 59.99, Stock: 12})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID != 4 || created.Name != "Webcam" || created.Price != 59.99 || created.Stock != 12 {
		t.Errorf("unexpected product: %+v", created)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil || got != created {
		t.Errorf("Get = %+v, %v; want %+v", got, err, created)
	}

	updated, err := s.Update(ctx, created.ID, model.ProductInput{Name: "HD Webcam", Price: 79.5, Stock: 3})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ID != created.ID || updated.Name != "HD Webcam" || updated.Price != 79.5 || updated.Stock != 3 {
		t.Errorf("unexpected update result: %+v", updated)
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	t.Run("after deleting, everything reports not found", func(t *testing.T) {
		if _, err := s.Get(ctx, created.ID); !errors.Is(err, model.ErrNotFound) {
			t.Errorf("Get error = %v, want ErrNotFound", err)
		}
		if _, err := s.Update(ctx, created.ID, model.ProductInput{Name: "X", Price: 1, Stock: 1}); !errors.Is(err, model.ErrNotFound) {
			t.Errorf("Update error = %v, want ErrNotFound", err)
		}
		if err := s.Delete(ctx, created.ID); !errors.Is(err, model.ErrNotFound) {
			t.Errorf("Delete error = %v, want ErrNotFound", err)
		}
	})
}

func TestPriceIsStoredWithTwoDecimals(t *testing.T) {
	s := newTestStore(t)

	p, err := s.Create(context.Background(), model.ProductInput{Name: "Odd price", Price: 59.999, Stock: 1})
	if err != nil {
		t.Fatal(err)
	}
	if p.Price != 60 {
		t.Errorf("price = %v, want 60, because the column keeps two decimals", p.Price)
	}
}

// The API checks input first, but the database is the last line of defence
func TestDatabaseRejectsInvalidProducts(t *testing.T) {
	s := newTestStore(t)

	tests := []struct {
		name string
		in   model.ProductInput
	}{
		{"price of zero", model.ProductInput{Name: "X", Price: 0, Stock: 1}},
		{"negative price", model.ProductInput{Name: "X", Price: -5, Stock: 1}},
		{"negative stock", model.ProductInput{Name: "X", Price: 5, Stock: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Create(context.Background(), tt.in)
			if err == nil {
				t.Fatal("the database accepted an invalid product")
			}
			if errors.Is(err, model.ErrNotFound) {
				t.Errorf("a rule violation should not look like a missing row: %v", err)
			}
		})
	}
}

func TestUsers(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	user, err := s.CreateUser(ctx, "mtb@example.com", "a-hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.ID == 0 || user.Email != "mtb@example.com" || user.PasswordHash != "a-hash" || user.CreatedAt.IsZero() {
		t.Errorf("unexpected user: %+v", user)
	}

	byEmail, err := s.UserByEmail(ctx, "mtb@example.com")
	if err != nil || byEmail.ID != user.ID {
		t.Errorf("UserByEmail = %+v, %v", byEmail, err)
	}
	byID, err := s.UserByID(ctx, user.ID)
	if err != nil || byID.Email != "mtb@example.com" {
		t.Errorf("UserByID = %+v, %v", byID, err)
	}

	t.Run("the same email twice", func(t *testing.T) {
		if _, err := s.CreateUser(ctx, "mtb@example.com", "another-hash"); !errors.Is(err, model.ErrEmailTaken) {
			t.Errorf("error = %v, want ErrEmailTaken", err)
		}
	})

	t.Run("unknown users", func(t *testing.T) {
		if _, err := s.UserByEmail(ctx, "nobody@example.com"); !errors.Is(err, model.ErrNotFound) {
			t.Errorf("UserByEmail error = %v, want ErrNotFound", err)
		}
		if _, err := s.UserByID(ctx, 9999); !errors.Is(err, model.ErrNotFound) {
			t.Errorf("UserByID error = %v, want ErrNotFound", err)
		}
	})
}

// Goroutines (Days 17 to 19) put the unique-email rule under pressure
func TestSameEmailRegisteredAtOnce(t *testing.T) {
	s := newTestStore(t)

	const attempts = 10
	results := make(chan error, attempts)

	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.CreateUser(context.Background(), "race@example.com", "a-hash")
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	created, taken := 0, 0
	for err := range results {
		switch {
		case err == nil:
			created++
		case errors.Is(err, model.ErrEmailTaken):
			taken++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if created != 1 || taken != attempts-1 {
		t.Errorf("%d created and %d refused, want 1 created and %d refused", created, taken, attempts-1)
	}
}