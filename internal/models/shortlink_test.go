package models

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestShortlink_BeforeCreate(t *testing.T) {
	shortlink := &Shortlink{
		TargetURL: "https://example.com",
		APIKey:    "test-key",
	}

	err := shortlink.BeforeCreate(nil)
	assert.NoError(t, err)

	// Check ID generation
	assert.Len(t, shortlink.ID, 6)

	// Check DeleteKey generation
	assert.Len(t, shortlink.DeleteKey, 32)

	// Test with pre-set values
	presetShortlink := &Shortlink{
		ID:        "custom",
		DeleteKey: "preset-key",
		TargetURL: "https://example.com",
		APIKey:    "test-key",
	}

	err = presetShortlink.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.Equal(t, "custom", presetShortlink.ID)
	assert.Equal(t, "preset-key", presetShortlink.DeleteKey)
}

func TestShortlink_ToResponse(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	lastClick := now.Add(-1 * time.Hour)

	shortlink := &Shortlink{
		ID:        "abc123",
		TargetURL: "https://example.com",
		Title:     "Example Site",
		CreatedAt: now,
		ExpiresAt: &expiresAt,
		Clicks:    42,
		LastClick: &lastClick,
		DeleteKey: "delete-key-123",
	}

	response := shortlink.ToResponse()
	expected := fiber.Map{
		"id":         "abc123",
		"url":        "https://example.com",
		"title":      "Example Site",
		"created_at": now,
		"expires_at": &expiresAt,
		"clicks":     int64(42),
		"last_click": &lastClick,
		"delete_url": "/delete/abc123.delete-key-123",
	}

	assert.Equal(t, expected, response)
}
