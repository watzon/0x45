package storage

import (
	"io"

	"github.com/watzon/0x45/internal/config"
)

// Provider defines the interface for storage implementations
type Provider interface {
	// Put stores content with the given path and returns the full storage path
	Put(path string, content io.Reader) (string, error)
	// Get retrieves content at the given path
	Get(path string) ([]byte, error)
	// Delete removes content at the given path
	Delete(path string) error
}

// StoreProvider wraps a Store to implement the Provider interface
type StoreProvider struct {
	store Store
}

// NewProvider creates a new storage provider based on configuration
func NewProvider(cfg *config.Config) Provider {
	manager, err := NewStorageManager(cfg)
	if err != nil {
		panic(err) // TODO: Better error handling
	}

	store, _, err := manager.GetDefaultStore()
	if err != nil {
		panic(err) // TODO: Better error handling
	}

	return &StoreProvider{store: store}
}

func (p *StoreProvider) Put(path string, content io.Reader) (string, error) {
	return p.store.Save(content, path)
}

func (p *StoreProvider) Get(path string) ([]byte, error) {
	reader, err := p.store.Get(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func (p *StoreProvider) Delete(path string) error {
	return p.store.Delete(path)
}
