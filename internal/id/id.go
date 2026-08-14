package id

import (
	"crypto/rand"
	"encoding/hex"
)

// Generates a new 32 character unique ID from 16 random bytes
func New() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
