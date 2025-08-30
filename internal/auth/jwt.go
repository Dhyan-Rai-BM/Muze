package auth

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token
func GenerateToken(userID, name string) (string, error) {
	return GenerateTokenWithExpiry(userID, name, time.Now().Add(24*time.Hour))
}

// GenerateTokenWithExpiry creates a new JWT token with custom expiry
func GenerateTokenWithExpiry(userID, name string, expiry time.Time) (string, error) {
	claims := Claims{
		UserID: userID,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// ValidateToken validates and extracts user info from JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ExtractUserFromContext extracts user info from context
func ExtractUserFromContext(ctx context.Context) (*Claims, error) {
	user := ctx.Value("user")
	if user == nil {
		return nil, errors.New("user not found in context")
	}

	claims, ok := user.(*Claims)
	if !ok {
		return nil, errors.New("invalid user claims")
	}

	return claims, nil
}
