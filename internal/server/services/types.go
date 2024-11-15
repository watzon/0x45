package services

import (
	"time"

	"github.com/watzon/0x45/internal/models"
)

// PasteOptions contains configuration options for creating a new paste
type PasteOptions struct {
	Extension string         // File extension (optional)
	ExpiresAt *time.Time     // Expiration time for the paste
	Private   bool           // Whether the paste is private
	Filename  string         // Original filename
	APIKey    *models.APIKey // Associated API key for authentication
}

// ShortlinkOptions contains configuration options for creating a new shortlink
type ShortlinkOptions struct {
	Title     string         // Display title for the shortlink
	ExpiresIn string         // Duration string for shortlink expiry (e.g. "24h")
	APIKey    *models.APIKey // Required API key for authentication
}

// ChartDataPoint represents a single point of data in time-series statistics
type ChartDataPoint struct {
	Value any       `json:"value"` // The value at this point (can be number or string)
	Date  time.Time `json:"date"`  // The timestamp for this data point
}

// StatsHistory contains time-series data for system statistics
type StatsHistory struct {
	Pastes     []ChartDataPoint
	URLs       []ChartDataPoint
	Storage    []ChartDataPoint
	AvgSize    []ChartDataPoint
	APIKeys    []ChartDataPoint
	Extensions []ChartDataPoint // Top extensions per day
	ErrorRates []ChartDataPoint // If we add error tracking
}

// UploadRequest represents a unified structure for all upload types
type UploadRequest struct {
	Content     []byte // Raw content bytes
	Filename    string // Original filename
	Extension   string // File extension
	ExpiresIn   string // Expiration duration
	Private     bool   // Privacy flag
	ContentType string // MIME type
	URL         string // Optional URL for URL-based uploads
}

// AnalyticsTimeframe represents a time period for analytics queries
type AnalyticsTimeframe struct {
	StartTime *time.Time
	EndTime   *time.Time
}

// AnalyticsStats contains aggregated statistics for a resource
type AnalyticsStats struct {
	TotalViews   int64            `json:"total_views"`
	UniqueViews  int64            `json:"unique_views"`
	ViewsByDay   []ChartDataPoint `json:"views_by_day"`
	TopReferrers map[string]int64 `json:"top_referrers"`
	TopCountries map[string]int64 `json:"top_countries"`
	TopBrowsers  map[string]int64 `json:"top_browsers"`
}

// ExpiryOptions contains parameters for calculating paste expiration
type ExpiryOptions struct {
	Size            int64
	HasAPIKey       bool
	RequestedExpiry string
}
