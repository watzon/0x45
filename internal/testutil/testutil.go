package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/watzon/paste69/internal/config"
	"github.com/watzon/paste69/internal/storage"
)

// TestConfig returns a config suitable for testing
func TestConfig(t *testing.T) *config.Config {
	t.Helper()

	tempDir := t.TempDir()

	cfg := &config.Config{}

	// Storage config
	cfg.Storage.Type = "local"
	cfg.Storage.Path = filepath.Join(tempDir, "uploads")

	// Server config
	cfg.Server.Address = ":0" // Random port
	cfg.Server.BaseURL = "http://localhost"
	cfg.Server.MaxUploadSize = 1024 * 1024 // 1MB
	cfg.Server.Cleanup.Enabled = true
	cfg.Server.Cleanup.Interval = 3600
	cfg.Server.Cleanup.MaxAge = "24h"

	return cfg
}

// NewTestStorage creates a new test storage
func NewTestStorage(t *testing.T) storage.Store {
	t.Helper()

	cfg := TestConfig(t)
	store, err := storage.NewStore(cfg)
	if err != nil {
		t.Fatalf("failed to create test storage: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(cfg.Storage.Path)
	})

	return store
}
