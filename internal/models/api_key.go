package models

import (
	"time"

	"github.com/watzon/0x45/internal/utils"
	"gorm.io/gorm"
)

type APIKey struct {
	Key       string `gorm:"primarykey;type:varchar(64)"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Paste-related limits and permissions
	MaxFileSize   int64  `gorm:"default:10485760"` // 10MB default
	MaxExpiration string `gorm:"default:24h"`      // Duration string
	RateLimit     int    `gorm:"default:100"`      // Requests per hour
	AllowPrivate  bool   `gorm:"default:true"`
	AllowUpdates  bool   `gorm:"default:false"`

	// URL shortening permissions
	AllowShortlinks bool   `gorm:"default:false"`    // Whether this key can create shortlinks
	ShortlinkQuota  int    `gorm:"default:0"`        // 0 = unlimited
	ShortlinkPrefix string `gorm:"type:varchar(16)"` // Optional custom prefix for shortened URLs

	// Optional user information
	Email string `gorm:"type:varchar(255)"`
	Name  string `gorm:"type:varchar(255)"`

	// Usage tracking
	LastUsedAt *time.Time
	UsageCount int64

	// Verification
	Verified     bool   `gorm:"default:false"`
	VerifyToken  string `gorm:"type:varchar(64)"`
	VerifyExpiry time.Time

	IsReset bool `json:"is_reset" gorm:"default:false"`
}

// GenerateKey generates a new API key string
func GenerateAPIKey() string {
	return utils.GenerateID(64)
}

// BeforeCreate sets defaults and generates the API key if not set
func (k *APIKey) BeforeCreate(tx *gorm.DB) error {
	if k.Key == "" {
		k.Key = GenerateAPIKey()
	}

	// Set defaults if not specified
	if k.MaxFileSize == 0 {
		k.MaxFileSize = 10485760 // 10MB
	}
	if k.MaxExpiration == "" {
		k.MaxExpiration = "24h"
	}
	if k.RateLimit == 0 {
		k.RateLimit = 100
	}
	if !k.AllowPrivate && !k.AllowUpdates && !k.AllowShortlinks {
		k.AllowPrivate = true // Default to allowing private pastes
	}

	return nil
}

// NewAPIKey creates a new APIKey with default values
func NewAPIKey() *APIKey {
	key := &APIKey{}
	key.BeforeCreate(nil) // Set defaults
	return key
}
