package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// health lets a hosting platform (or you) check that the app is up
func (s *Server) health(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"status": "ok"})
}
