package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// tokenTTL is how long an issued token stays valid.
const tokenTTL = 24 * time.Hour

// IssueToken creates a signed JWT whose subject is the user id.
func IssueToken(userID string, secret []byte) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,                                // who the token is given to
		IssuedAt:  jwt.NewNumericDate(now),               // when it was issued
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)), // when it stops being valid
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret) // signs header.payload with the secret (HMAC-SHA256)
}

// ParseToken verifies a token's signature and expiry, returning the user id (subject).
func ParseToken(tokenStr string, secret []byte) (string, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		// Only accept HMAC-signed tokens
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	return claims.Subject, nil
}
