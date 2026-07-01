package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT generation and parsing.
type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewJWTManager loads RSA keys from PEM bytes.
func NewJWTManager(privatePEM, publicPEM []byte) (*JWTManager, error) {
	privateKey, err := parseRSAPrivateKey(privatePEM)
	if err != nil {
		return nil, err
	}

	publicKey, err := parseRSAPublicKey(publicPEM)
	if err != nil {
		return nil, err
	}

	return &JWTManager{privateKey: privateKey, publicKey: publicKey}, nil
}

func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return key, nil
}

func parseRSAPublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid public key PEM")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	public, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}
	return public, nil
}

// GenerateToken creates a signed JWT with a short lifetime.
func (m *JWTManager) GenerateToken(subject, username, subjectType, actorType string, permissions []string) (string, error) {
	if m.privateKey == nil {
		return "", fmt.Errorf("private key is not configured")
	}

	claims := jwt.MapClaims{
		"sub":          subject,
		"username":     username,
		"subject_type": subjectType,
		"actor_type":   actorType,
		"permissions":  permissions,
		"iat":          time.Now().Unix(),
		"exp":          time.Now().Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ComparePassword checks whether a password matches the stored hash.
func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// HashContent returns a SHA-256 hash of the given input.
func hashContent(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum)
}

// GenerateSecureToken creates a random token string.
func GenerateSecureToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
