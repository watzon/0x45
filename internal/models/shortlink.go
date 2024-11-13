package models

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/utils"
	"gorm.io/gorm"
)

type Shortlink struct {
	ID        string `gorm:"primarykey;type:varchar(8)"` // Shorter IDs for URLs
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// URL information
	TargetURL string `gorm:"type:text;not null"`
	Title     string `gorm:"type:varchar(255)"` // Optional, can be fetched from target

	// Access control
	APIKey    string     `gorm:"type:varchar(64);not null;index"` // Required for creation
	DeleteKey string     `gorm:"type:varchar(32);not null"`
	ExpiresAt *time.Time `gorm:"index"`

	// Analytics (optional)
	Clicks    int64
	LastClick *time.Time

	// Optional metadata (referrer stats, etc.)
	Metadata JSON `gorm:"type:jsonb"`
}

func (s *Shortlink) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = utils.MustGenerateID(6) // Shorter IDs for URLs
	}
	if s.DeleteKey == "" {
		s.DeleteKey = utils.MustGenerateID(32)
	}
	return nil
}

func (s *Shortlink) ToResponse() fiber.Map {
	response := fiber.Map{
		"id":         s.ID,
		"url":        s.TargetURL,
		"title":      s.Title,
		"created_at": s.CreatedAt,
		"expires_at": s.ExpiresAt,
		"clicks":     s.Clicks,
		"last_click": s.LastClick,
	}

	// Add URL paths (similar to how Paste does it)
	response["short_url"] = fmt.Sprintf("/%s", s.ID)

	// Only include delete_url if there's a delete key
	if s.DeleteKey != "" {
		response["delete_url"] = fmt.Sprintf("/delete/%s/%s", s.ID, s.DeleteKey)
	}

	return response
}
