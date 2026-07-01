package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTValidator validates JWT tokens from identity-service
type JWTValidator struct {
	publicKey interface{}
}

// Claims represents JWT claims structure
type Claims struct {
	Subject     string   `json:"sub"`
	Username    string   `json:"username"`
	SubjectType string   `json:"subject_type"`
	ActorType   string   `json:"actor_type"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// ValidateToken validates a JWT token
func (v *JWTValidator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("empty token")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(r *http.Request) string {
	bearerToken := r.Header.Get("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		return strings.Split(bearerToken, " ")[1]
	}
	return ""
}

// InternalAPIKey validates internal service-to-service API key
func ValidateInternalAPIKey(r *http.Request, expectedKey string) bool {
	return r.Header.Get("X-Internal-Api-Key") == expectedKey
}
