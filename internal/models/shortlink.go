package models

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/paste69/internal/utils"
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
		s.ID = utils.GenerateID(6) // Shorter IDs for URLs
	}
	if s.DeleteKey == "" {
		s.DeleteKey = utils.GenerateID(32)
	}
	return nil
}

func (s *Shortlink) ToResponse() fiber.Map {
	return fiber.Map{
		"id":         s.ID,
		"url":        s.TargetURL,
		"title":      s.Title,
		"created_at": s.CreatedAt,
		"expires_at": s.ExpiresAt,
		"clicks":     s.Clicks,
		"last_click": s.LastClick,
		"delete_url": fmt.Sprintf("/delete/%s.%s", s.ID, s.DeleteKey),
	}
}
