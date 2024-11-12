package storage

import (
	"fmt"

	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/storage/local"
	"github.com/watzon/0x45/internal/storage/s3"
)

type StorageManager struct {
	stores map[string]Store
}

func NewStorageManager(cfg *config.Config) (*StorageManager, error) {
	manager := &StorageManager{
		stores: make(map[string]Store),
	}

	for _, storageCfg := range cfg.Storage {
		var store Store
		var err error

		switch storageCfg.Type {
		case "local":
			store, err = local.New(storageCfg.Path, cfg.Server.BaseURL, storageCfg.IsDefault)
		case "s3":
			store, err = s3.New(
				storageCfg.S3Bucket,
				storageCfg.S3Region,
				storageCfg.S3Key,
				storageCfg.S3Secret,
				storageCfg.S3Endpoint,
				storageCfg.IsDefault,
			)
		default:
			return nil, fmt.Errorf("unsupported storage type: %s", storageCfg.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to initialize storage %s: %w", storageCfg.Name, err)
		}

		manager.stores[storageCfg.Name] = store
	}

	return manager, nil
}

func (m *StorageManager) GetStore(name string) (Store, error) {
	store, ok := m.stores[name]
	if !ok {
		return nil, fmt.Errorf("storage not found: %s", name)
	}
	return store, nil
}

func (m *StorageManager) GetDefaultStore() (Store, string, error) {
	for name, store := range m.stores {
		if store.IsDefault() {
			return store, name, nil
		}
	}

	// If no default store is found, return the first store
	for name, store := range m.stores {
		return store, name, nil
	}

	return nil, "", fmt.Errorf("no storage configurations available")
}
