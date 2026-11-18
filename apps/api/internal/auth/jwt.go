package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload.
type Claims struct {
	UserID string `json:"uid"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Manager issues and parses JWTs.
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// NewManager creates a JWT manager.
func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret), ttl: 7 * 24 * time.Hour}
}

// Issue creates a signed token for a user.
func (m *Manager) Issue(userID, name, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Name:   name,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ningpm",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(m.secret)
}

// Parse validates a token and returns claims.
func (m *Manager) Parse(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
