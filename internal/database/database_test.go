package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/models"
)

// newTestConfig returns a config suitable for testing
func newTestConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg := &config.Config{}

	// Database config - use in-memory SQLite
	cfg.Database.Driver = "sqlite"
	cfg.Database.Name = ":memory:" // This tells SQLite to use an in-memory database

	return cfg
}

// newTestDB creates a new test database
func newTestDB(t *testing.T) *Database {
	t.Helper()

	cfg := newTestConfig(t)
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Auto-migrate the test database
	err = db.AutoMigrate(
		&models.Paste{},
		&models.Shortlink{},
		&models.APIKey{},
	)
	if err != nil {
		t.Fatalf("failed to auto-migrate test database: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	})

	return db
}

func TestDatabase_Paste(t *testing.T) {
	db := newTestDB(t)

	t.Run("Create and Retrieve Paste", func(t *testing.T) {
		paste := &models.Paste{
			Filename: "test.txt",
			MimeType: "text/plain",
			Size:     100,
		}

		err := db.Create(paste).Error
		assert.NoError(t, err)
		assert.NotEmpty(t, paste.ID)

		var retrieved models.Paste
		err = db.First(&retrieved, "id = ?", paste.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, paste.Filename, retrieved.Filename)
	})

	t.Run("Expire Paste", func(t *testing.T) {
		expiry := time.Now().Add(-1 * time.Hour)
		paste := &models.Paste{
			Filename:  "expired.txt",
			ExpiresAt: &expiry,
		}

		err := db.Create(paste).Error
		assert.NoError(t, err)

		var retrieved models.Paste
		err = db.Where("id = ? AND (expires_at IS NULL OR expires_at > ?)",
			paste.ID, time.Now()).First(&retrieved).Error
		assert.Error(t, err) // Should not find expired paste
	})
}
