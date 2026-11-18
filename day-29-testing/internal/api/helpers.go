package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/taisekhame/101-days-of-go/day-29-testing/internal/model"
)

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
		case "Email":
			return "a valid email is required"
		case "Password":
			return "password must be between 8 and 72 characters"
		}
	}
	return "invalid JSON: " + err.Error()
}

// Generic (Day 16): reads and validates a JSON body into any struct type
func bindJSON[T any](c *gin.Context) (T, bool) {
	var in T

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(&in); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": friendly(err)})
		return in, false
	}
	return in, true
}

func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return 0, false
	}
	return id, true
}

func respondError(c *gin.Context, err error) {
	if errors.Is(err, model.ErrNotFound) {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	log.Println("database error:", err)
	c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
}

func unauthorized(c *gin.Context, message string) {
	c.Header("WWW-Authenticate", "Bearer")
	c.Abort()
	c.IndentedJSON(http.StatusUnauthorized, gin.H{"error": message})
}
