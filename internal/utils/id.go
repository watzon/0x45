package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateID creates a URL-safe random string of specified length
func GenerateID(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}
