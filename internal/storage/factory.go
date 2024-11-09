package storage

import (
	"fmt"

	"github.com/watzon/paste69/internal/config"
	"github.com/watzon/paste69/internal/storage/local"
	// "github.com/watzon/paste69/internal/storage/s3" // We'll implement this later
)

func NewStore(cfg *config.Config) (Store, error) {
	switch cfg.Storage.Type {
	case "local":
		return local.New(cfg.Storage.Path, cfg.Server.BaseURL)
	case "s3":
		// return s3.New(cfg.Storage)
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}
