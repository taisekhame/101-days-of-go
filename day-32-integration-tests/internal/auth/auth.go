// Package auth handles passwords and tokens. It knows nothing about HTTP or Gin.
package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const issuer = "mtb-store-api"

type Service struct {
	secret    []byte
	ttl       time.Duration
	dummyHash []byte // used to keep login timing the same for unknown emails
}

func New(secret string, ttl time.Duration) (*Service, error) {
	dummy, err := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Service{secret: []byte(secret), ttl: ttl, dummyHash: dummy}, nil
}

// TTL is how long a token stays valid
func (s *Service) TTL() time.Duration {
	return s.ttl
}

// HashPassword turns a password into a hash that is safe to store
func (s *Service) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// CheckPassword reports whether the password matches the stored hash
func (s *Service) CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CheckDummy does the same slow work as a real check, so timing doesn't reveal which emails exist
func (s *Service) CheckDummy(password string) {
	_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
}

// IssueToken signs a token that says who the user is and when it stops working
func (s *Service) IssueToken(userID int) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   strconv.Itoa(userID),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// ParseToken checks a token and returns the user id inside it
func (s *Service) ParseToken(raw string) (int, error) {
	token, err := jwt.ParseWithClaims(raw, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), // refuse any other signing method
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token")
	}
	id, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return 0, errors.New("invalid token subject")
	}
	return id, nil
}
