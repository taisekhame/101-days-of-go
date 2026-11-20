package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/taisekhame/101-days-of-go/day-30-logging/internal/model"
)

func bindProduct(c *gin.Context) (model.ProductInput, bool) {
	in, ok := bindJSON[model.ProductInput](c)
	if !ok {
		return in, false
	}
	if strings.TrimSpace(in.Name) == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return in, false
	}
	return in, true
}

func (s *Server) listProducts(c *gin.Context) {
	products, err := s.products.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, products)
}

func (s *Server) getProduct(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := s.products.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, p)
}

func (s *Server) createProduct(c *gin.Context) {
	in, ok := bindProduct(c)
	if !ok {
		return
	}
	created, err := s.products.Create(c.Request.Context(), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.Header("Location", fmt.Sprintf("/products/%d", created.ID))
	c.IndentedJSON(http.StatusCreated, created)
}

func (s *Server) updateProduct(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	in, ok := bindProduct(c)
	if !ok {
		return
	}
	p, err := s.products.Update(c.Request.Context(), id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, p)
}

func (s *Server) deleteProduct(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := s.products.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
