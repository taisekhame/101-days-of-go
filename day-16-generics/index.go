package main

import (
	"errors"
	"fmt"
	"sort"
)

// Anything stored in the repository needs an ID
type Entity interface {
	GetID() int
}

type User struct {
	ID     int
	Name   string
	Email  string
	Active bool
}

func (u User) GetID() int { return u.ID }

type Product struct {
	ID    int
	Name  string
	Price float64
	Stock int
}

func (p Product) GetID() int { return p.ID }

// One repository that works for users, products, orders, anything with an ID
type Repository[T Entity] struct {
	items map[int]T
}

func NewRepository[T Entity]() *Repository[T] {
	return &Repository[T]{items: make(map[int]T)}
}

func (r *Repository[T]) Save(item T) {
	r.items[item.GetID()] = item
}

func (r *Repository[T]) FindByID(id int) (T, error) {
	item, ok := r.items[id]
	if !ok {
		var zero T
		return zero, errors.New("not found")
	}
	return item, nil
}

func (r *Repository[T]) All() []T {
	result := make([]T, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].GetID() < result[j].GetID()
	})
	return result
}

// Generic helpers that work on a slice of anything
func Filter[T any](items []T, keep func(T) bool) []T {
	var result []T
	for _, item := range items {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}

func Map[T any, R any](items []T, transform func(T) R) []R {
	result := make([]R, 0, len(items))
	for _, item := range items {
		result = append(result, transform(item))
	}
	return result
}

// Generic API response wrapper, one shape for every endpoint
type APIResponse[T any] struct {
	Success bool
	Data    T
	Message string
}

func Ok[T any](data T) APIResponse[T] {
	return APIResponse[T]{Success: true, Data: data, Message: "ok"}
}

func Fail[T any](message string) APIResponse[T] {
	var zero T
	return APIResponse[T]{Success: false, Data: zero, Message: message}
}

func getUser(repo *Repository[User], id int) APIResponse[User] {
	user, err := repo.FindByID(id)
	if err != nil {
		return Fail[User](fmt.Sprintf("user %d not found", id))
	}
	return Ok(user)
}

func main() {
	fmt.Println("MTB's Store Backend")

	users := NewRepository[User]()
	users.Save(User{ID: 1, Name: "MTB", Email: "mtb@example.com", Active: true})
	users.Save(User{ID: 2, Name: "Alex", Email: "alex@example.com", Active: false})
	users.Save(User{ID: 3, Name: "Sam", Email: "sam@example.com", Active: true})

	products := NewRepository[Product]()
	products.Save(Product{ID: 1, Name: "Keyboard", Price: 49.99, Stock: 25})
	products.Save(Product{ID: 2, Name: "Mouse", Price: 19.99, Stock: 4})
	products.Save(Product{ID: 3, Name: "Monitor", Price: 189.50, Stock: 7})

	// The same Filter and Map work on users and on products
	activeUsers := Filter(users.All(), func(u User) bool { return u.Active })
	emails := Map(activeUsers, func(u User) string { return u.Email })
	fmt.Println("Emails to notify:", emails)

	lowStock := Filter(products.All(), func(p Product) bool { return p.Stock < 10 })
	names := Map(lowStock, func(p Product) string { return p.Name })
	fmt.Println("Low stock items:", names)

	// Generic API response in action
	found := getUser(users, 1)
	fmt.Println(found.Success, found.Message, found.Data.Name)

	missing := getUser(users, 99)
	fmt.Println(missing.Success, missing.Message)
}
