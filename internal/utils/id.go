package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// MustGenerateID creates a URL-safe random string of specified length
func MustGenerateID(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)[:length]
}
