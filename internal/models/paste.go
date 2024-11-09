package models

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/paste69/internal/utils"
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
		p.ID = utils.GenerateID(8) // We'll implement this in utils
	}
	if p.DeleteKey == "" {
		p.DeleteKey = utils.GenerateID(32)
	}
	return nil
}

// ToResponse returns a map of the paste data for API responses
func (p *Paste) ToResponse() fiber.Map {
	response := fiber.Map{
		"id":         p.ID,
		"filename":   p.Filename,
		"size":       p.Size,
		"mime_type":  p.MimeType,
		"created_at": p.CreatedAt,
		"expires_at": p.ExpiresAt,
		"private":    p.Private,
	}

	// Add URL paths
	response["url"] = fmt.Sprintf("/view/%s", p.ID)
	response["raw_url"] = fmt.Sprintf("/raw/%s", p.ID)
	response["download_url"] = fmt.Sprintf("/download/%s", p.ID)

	// Only include delete_url if there's a delete key
	if p.DeleteKey != "" {
		response["delete_url"] = fmt.Sprintf("/delete/%s/%s", p.ID, p.DeleteKey)
	}

	return response
}
