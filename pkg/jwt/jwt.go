package jwt

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// minJWTSecretLength is the minimum acceptable length for the JWT signing secret.
	minJWTSecretLength = 32

	// TokenTypeAccess identifies an access token.
	TokenTypeAccess = "access"

	// TokenTypeRefresh identifies a refresh token.
	TokenTypeRefresh = "refresh"
)

var (
	jwtSecret         []byte
	defaultAccessTTL  time.Duration
	defaultRefreshTTL time.Duration
)

// Init configures the JWT package with the signing secret and default TTLs.
// Must be called once at application startup before any token operations.
func Init(secret string, accessTTL, refreshTTL time.Duration) {
	if len(secret) == 0 {
		log.Fatalf("FATAL: JWT secret is not set. An auth API cannot run without a signing secret.")
	}
	if len(secret) < minJWTSecretLength {
		log.Fatalf("FATAL: JWT secret is too short (%d bytes). Minimum required: %d bytes.", len(secret), minJWTSecretLength)
	}
	jwtSecret = []byte(secret)
	defaultAccessTTL = accessTTL
	defaultRefreshTTL = refreshTTL
}

// Claims struct that will be embedded in JWT
type Claims struct {
	UserID    string   `json:"user_id"`
	AppID     string   `json:"app_id"`
	SessionID string   `json:"session_id,omitempty"` // Session identifier for multi-device session management
	TokenType string   `json:"token_type,omitempty"` // "access" or "refresh"; empty for legacy tokens
	Roles     []string `json:"roles,omitempty"`      // User's role names in the application
	jwt.RegisteredClaims
}

// DefaultAccessTokenTTL returns the configured global access token TTL.
func DefaultAccessTokenTTL() time.Duration { return defaultAccessTTL }

// DefaultRefreshTokenTTL returns the configured global refresh token TTL.
func DefaultRefreshTokenTTL() time.Duration { return defaultRefreshTTL }

// GenerateAccessToken generates a new access token with an explicit TTL.
// Pass 0 (or DefaultAccessTokenTTL()) to use the global configured value.
func GenerateAccessToken(appID, userID, sessionID string, roles []string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = DefaultAccessTokenTTL()
	}
	expirationTime := time.Now().Add(ttl)
	claims := &Claims{
		UserID:    userID,
		AppID:     appID,
		SessionID: sessionID,
		TokenType: TokenTypeAccess,
		Roles:     roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GenerateRefreshToken generates a new refresh token with an explicit TTL.
// Pass 0 (or DefaultRefreshTokenTTL()) to use the global configured value.
func GenerateRefreshToken(appID, userID, sessionID string, roles []string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = DefaultRefreshTokenTTL()
	}
	expirationTime := time.Now().Add(ttl)
	claims := &Claims{
		UserID:    userID,
		AppID:     appID,
		SessionID: sessionID,
		TokenType: TokenTypeRefresh,
		Roles:     roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken parses and validates a JWT token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}
