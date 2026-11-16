// Package api is the HTTP layer: routes, handlers and middleware.
package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/taisekhame/101-days-of-go/day-28-project-structure/internal/auth"
	"github.com/taisekhame/101-days-of-go/day-28-project-structure/internal/model"
)

// The API only needs these abilities from the database. Any type that has
// these methods will do, which makes the handlers easy to test later.
type ProductStore interface {
	List(ctx context.Context) ([]model.Product, error)
	Get(ctx context.Context, id int) (model.Product, error)
	Create(ctx context.Context, in model.ProductInput) (model.Product, error)
	Update(ctx context.Context, id int, in model.ProductInput) (model.Product, error)
	Delete(ctx context.Context, id int) error
}

type UserStore interface {
	CreateUser(ctx context.Context, email, hash string) (model.User, error)
	UserByEmail(ctx context.Context, email string) (model.User, error)
	UserByID(ctx context.Context, id int) (model.User, error)
}

type Server struct {
	products ProductStore
	users    UserStore
	auth     *auth.Service
}

// NewRouter builds the Gin router with every route attached
func NewRouter(products ProductStore, users UserStore, authSvc *auth.Service) (*gin.Engine, error) {
	binding.EnableDecoderDisallowUnknownFields = true

	s := &Server{products: products, users: users, auth: authSvc}

	r := gin.Default()
	r.HandleMethodNotAllowed = true
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}

	r.POST("/auth/register", s.register)
	r.POST("/auth/login", s.login)
	r.GET("/me", s.requireAuth, s.me)

	group := r.Group("/products")
	{
		// Anyone can read
		group.GET("", s.listProducts)
		group.GET("/:id", s.getProduct)

		// Only logged-in users can change things
		write := group.Group("", s.requireAuth)
		write.POST("", s.createProduct)
		write.PUT("/:id", s.updateProduct)
		write.DELETE("/:id", s.deleteProduct)
	}

	return r, nil
}
