package models

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestPaste_BeforeCreate(t *testing.T) {
	paste := &Paste{}
	err := paste.BeforeCreate(nil)

	assert.NoError(t, err)
	assert.Len(t, paste.ID, 8)
	assert.Len(t, paste.DeleteKey, 32)
}

func TestPaste_ToResponse(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	paste := &Paste{
		ID:        "test123",
		Filename:  "test.txt",
		Size:      1024,
		MimeType:  "text/plain",
		CreatedAt: now,
		ExpiresAt: &expiresAt,
		Private:   true,
		DeleteKey: "deletekey123",
	}

	response := paste.ToResponse()
	expected := fiber.Map{
		"id":           "test123",
		"filename":     "test.txt",
		"size":         int64(1024),
		"mime_type":    "text/plain",
		"created_at":   now,
		"expires_at":   &expiresAt,
		"private":      true,
		"url":          "/view/test123",
		"raw_url":      "/raw/test123",
		"download_url": "/download/test123",
		"delete_url":   "/delete/test123/deletekey123",
	}

	assert.Equal(t, expected, response)
}
