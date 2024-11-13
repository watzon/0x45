package models

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite" // pure go sqlite driver
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestAPIKey_BeforeCreate(t *testing.T) {
	t.Run("Generate Key", func(t *testing.T) {
		apiKey := &APIKey{
			Email: "test@example.com",
			Name:  "Test User",
		}

		err := apiKey.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Len(t, apiKey.Key, 64)
	})

	t.Run("Preserve Existing Key", func(t *testing.T) {
		existingKey := "custom-key-123"
		apiKey := &APIKey{
			Key:   existingKey,
			Email: "test@example.com",
		}

		err := apiKey.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingKey, apiKey.Key)
	})
}

func TestAPIKey_Defaults(t *testing.T) {
	db := newTestDB(t)

	// Create new API key through the database
	apiKey := &APIKey{
		Email: "test@example.com",
	}

	err := db.Create(apiKey).Error
	assert.NoError(t, err)

	// Fetch the key back from the database
	var retrieved APIKey
	err = db.First(&retrieved, "key = ?", apiKey.Key).Error
	assert.NoError(t, err)

	// Test default values
	assert.Equal(t, int64(10485760), retrieved.MaxFileSize)
	assert.Equal(t, "24h", retrieved.MaxExpiration)
	assert.Equal(t, 100, retrieved.RateLimit)
	assert.True(t, retrieved.AllowPrivate)
	assert.False(t, retrieved.AllowUpdates)
	assert.False(t, retrieved.AllowShortlinks)
	assert.Equal(t, 0, retrieved.ShortlinkQuota)
}

func TestAPIKey_UsageTracking(t *testing.T) {
	db := newTestDB(t)

	apiKey := &APIKey{
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Create API key
	err := db.Create(apiKey).Error
	assert.NoError(t, err)
	assert.NotEmpty(t, apiKey.Key)

	// Update usage
	now := time.Now()
	err = db.Model(apiKey).Updates(map[string]any{
		"last_used_at": now,
		"usage_count":  1,
	}).Error
	assert.NoError(t, err)

	// Retrieve and verify
	var retrieved APIKey
	err = db.First(&retrieved, "key = ?", apiKey.Key).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), retrieved.UsageCount)
	assert.NotNil(t, retrieved.LastUsedAt)
}

func newTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	err = db.AutoMigrate(&APIKey{})
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}
