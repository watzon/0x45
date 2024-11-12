package storage

import (
	"io"
	"time"
)

// Store defines the interface for storage backends
type Store interface {
	// Save stores content and returns the storage path
	Save(content io.Reader, filename string) (string, error)

	// Get retrieves content by storage path
	Get(path string) (io.ReadCloser, error)

	// Delete removes content by storage path
	Delete(path string) error

	// GetURL returns a public URL for the content (if supported)
	GetURL(path string) string

	// GetSize returns the size of the content
	GetSize(path string) (int64, error)

	// SetExpiry sets an expiration time for the content
	SetExpiry(path string, expiry time.Time) error

	// Type returns the type of the storage backend
	Type() string

	// SetDefault sets the storage backend as the default
	SetDefault() error

	// IsDefault returns whether the storage backend is the default
	IsDefault() bool
}
