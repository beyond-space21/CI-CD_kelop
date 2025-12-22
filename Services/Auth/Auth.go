package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UID string `json:"uid"`
	jwt.RegisteredClaims
}

var (
	JWTSecret     []byte
	TokenValidity = 24 * time.Hour // Token expires in 24 hours
)

// InitAuth initializes the JWT authentication system
func Initauth() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a random secret if not provided (for development only)
		// In production, JWT_SECRET should be set in environment
		log.Println("Warning: JWT_SECRET not set, generating random secret (not recommended for production)")
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			log.Fatalf("Failed to generate JWT secret: %v", err)
		}
		secret = base64.URLEncoding.EncodeToString(secretBytes)
		log.Printf("Generated JWT_SECRET: %s (save this for production)", secret)
	}
	JWTSecret = []byte(secret)

	// Set token validity from env if provided
	if validityStr := os.Getenv("JWT_TOKEN_VALIDITY_HOURS"); validityStr != "" {
		if hours, err := time.ParseDuration(validityStr + "h"); err == nil {
			TokenValidity = hours
		}
	}
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(uid string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UID: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenValidity)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "hifi-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(JWTSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// VerifyToken verifies and parses a JWT token
func VerifyToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return JWTSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// GetClaims extracts and verifies JWT token from request Authorization header
// Returns a Token-like struct for compatibility with existing code
func GetClaims(r *http.Request) (*Token, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, false
	}

	// Remove "Bearer " prefix if present
	const bearerPrefix = "Bearer "
	var tokenString string
	if len(authHeader) > len(bearerPrefix) && authHeader[:len(bearerPrefix)] == bearerPrefix {
		tokenString = strings.TrimSpace(authHeader[len(bearerPrefix):])
	} else {
		tokenString = strings.TrimSpace(authHeader)
	}

	if tokenString == "" {
		return nil, false
	}

	claims, err := VerifyToken(tokenString)
	if err != nil {
		return nil, false
	}

	// Return a Token struct compatible with the old Firebase auth.Token interface
	return &Token{UID: claims.UID}, true
}

// Token represents a user token (compatible with Firebase auth.Token interface)
type Token struct {
	UID string
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPasswordHash compares a password with a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// DeleteUser is a no-op for custom JWT auth (user deletion is handled in Users package)
// Kept for compatibility with existing code
func DeleteUser(ctx context.Context, uid string) error {
	// User deletion is handled in the Users package by removing from database
	// No separate auth service deletion needed for JWT
	return nil
}
