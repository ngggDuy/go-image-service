package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns bcrypt hash of password string (safe to store)
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword returns whether password matches the stored bcrypt hash
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
