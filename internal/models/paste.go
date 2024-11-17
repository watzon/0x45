package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/utils"
	"gorm.io/gorm"
)

type Paste struct {
	ID        string `gorm:"primarykey;type:varchar(16)"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Content information
	Filename  string `gorm:"type:varchar(255)"`
	MimeType  string `gorm:"type:varchar(255)"`
	Size      int64
	Extension string `gorm:"type:varchar(32)"`

	// Storage information
	StoragePath string `gorm:"type:varchar(512)"`
	StorageType string `gorm:"type:varchar(32)"` // "local" or "s3"
	StorageName string `gorm:"type:varchar(64)"` // Name of the storage config

	// Access control
	Private   bool
	DeleteKey string `gorm:"type:varchar(32)"`
	APIKey    string `gorm:"type:varchar(64);index"` // If created with an API key

	// Expiration
	ExpiresAt *time.Time `gorm:"index"`

	// Optional metadata
	Metadata JSON `gorm:"type:jsonb"` // For PostgreSQL, will fallback to JSON string for SQLite
}

// BeforeCreate generates ID and DeleteKey if not set
func (p *Paste) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = utils.MustGenerateID(8)
	}

	if p.DeleteKey == "" {
		p.DeleteKey = utils.MustGenerateID(32)
	}

	// Set default filename if not provided
	if p.Filename == "" {
		p.Filename = "untitled"
	}

	// Handle file extension from filename if not already set
	if p.Extension == "" && p.Filename != "" {
		parts := strings.Split(p.Filename, ".")
		if len(parts) > 1 {
			p.Extension = parts[len(parts)-1]
		}
	}

	// Clean the extension (remove any leading dots and whitespace)
	p.Extension = strings.TrimSpace(strings.TrimPrefix(p.Extension, "."))

	// Storage configuration handling
	if p.StorageName == "" {
		cfg, ok := tx.Statement.Context.Value("config").(*config.Config)
		if !ok {
			return fmt.Errorf("config not found in context")
		}

		for _, storage := range cfg.Storage {
			if storage.IsDefault {
				p.StorageName = storage.Name
				p.StorageType = storage.Type
				break
			}
		}

		if p.StorageName == "" {
			return fmt.Errorf("no default storage configuration found")
		}
	}

	return nil
}
