package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/taisekhame/101-days-of-go/day-33-production-ready/internal/model"
)

type Credentials struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"` // bcrypt only uses 72 bytes
}

func (s *Server) register(c *gin.Context) {
	cred, ok := bindJSON[Credentials](c)
	if !ok {
		return
	}
	email := strings.ToLower(strings.TrimSpace(cred.Email))

	// Only the hash is ever stored, never the password itself
	hash, err := s.auth.HashPassword(cred.Password)
	if err != nil {
		respondError(c, err)
		return
	}

	user, err := s.users.CreateUser(c.Request.Context(), email, hash)
	if errors.Is(err, model.ErrEmailTaken) {
		c.IndentedJSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusCreated, user)
}

func (s *Server) login(c *gin.Context) {
	cred, ok := bindJSON[Credentials](c)
	if !ok {
		return
	}
	email := strings.ToLower(strings.TrimSpace(cred.Email))

	user, err := s.users.UserByEmail(c.Request.Context(), email)
	if errors.Is(err, model.ErrNotFound) {
		s.auth.CheckDummy(cred.Password)
		unauthorized(c, "invalid email or password")
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}

	if !s.auth.CheckPassword(user.PasswordHash, cred.Password) {
		unauthorized(c, "invalid email or password") // same message as an unknown email
		return
	}

	token, err := s.auth.IssueToken(user.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"token":      token,
		"token_type": "Bearer",
		"expires_in": int(s.auth.TTL().Seconds()),
	})
}

// Middleware: lets a request through only if it carries a valid token
func (s *Server) requireAuth(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		unauthorized(c, "missing or malformed Authorization header")
		return
	}

	userID, err := s.auth.ParseToken(strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		unauthorized(c, "invalid or expired token")
		return
	}

	c.Set("userID", userID)
	c.Next()
}

// A protected route: shows who the token belongs to
func (s *Server) me(c *gin.Context) {
	user, err := s.users.UserByID(c.Request.Context(), c.GetInt("userID"))
	if errors.Is(err, model.ErrNotFound) {
		unauthorized(c, "this account no longer exists")
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.IndentedJSON(http.StatusOK, user)
}
