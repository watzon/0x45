package models

import (
	"time"

	"gorm.io/gorm"
)

// EventType represents different types of events that can be tracked
type EventType string

const (
	EventShortlinkClick EventType = "shortlink_click"
	EventPasteView      EventType = "paste_view"
)

// AnalyticsEvent represents a single analytics event
type AnalyticsEvent struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Event information
	EventType EventType `gorm:"type:varchar(32);index;not null"`

	// Resource information (what the event is about)
	ResourceID   string `gorm:"type:varchar(16);index;not null"` // ID of the shortlink or paste
	ResourceType string `gorm:"type:varchar(32);index;not null"` // "shortlink" or "paste"

	// Request information
	UserAgent   string `gorm:"type:text"`
	IPAddress   string `gorm:"type:varchar(45)"` // IPv6 addresses can be up to 45 chars
	RefererURL  string `gorm:"type:text"`
	CountryCode string `gorm:"type:varchar(2)"`

	// Additional data
	Metadata JSON `gorm:"type:jsonb"`
}

// CreateEvent is a helper function to create a new analytics event
func CreateEvent(db *gorm.DB, eventType EventType, resourceType string, resourceID string, userAgent string, ipAddress string, refererURL string) error {
	event := &AnalyticsEvent{
		EventType:    eventType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		RefererURL:   refererURL,
	}

	return db.Create(event).Error
}
