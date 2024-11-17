package services

import (
	"fmt"
	"time"

	"github.com/watzon/0x45/internal/models"
)

// APIKeyRequest represents the request structure for creating an API key
type APIKeyRequest struct {
	Email string `json:"email" xml:"email" form:"email"`
	Name  string `json:"name" xml:"name" form:"name"`
}

// APIKeyResponse represents the response sent after an API key is requested
type APIKeyResponse struct {
	Message string `json:"message" xml:"message" form:"message"`
}

// PasteOptions contains configuration options for creating a new paste
type PasteOptions struct {
	Content   string         `json:"content" xml:"content" form:"content"`          // Content to be pasted
	Extension string         `json:"extension" xml:"extension" form:"extension"`    // File extension (optional)
	Private   bool           `json:"private" xml:"private" form:"private"`          // Whether the paste is private
	Filename  string         `json:"filename" xml:"filename" form:"filename"`       // Original filename
	APIKey    *models.APIKey `json:"api_key" xml:"api_key" form:"api_key"`          // Associated API key for authentication
	URL       string         `json:"url" xml:"url" form:"url"`                      // URL to be pasted
	ExpiresIn *time.Duration `json:"expires_in" xml:"expires_in" form:"expires_in"` // Duration string for paste expiry (e.g. "24h")
	ExpiresAt *time.Time     `json:"expires_at" xml:"expires_at" form:"expires_at"` // Expiration time for the paste
}

// PasteResponse represents the response structure for creating a new paste
type PasteResponse struct {
	ID          string     `json:"id" xml:"id" form:"id"`
	Filename    string     `json:"filename" xml:"filename" form:"filename"`
	URL         string     `json:"url" xml:"url" form:"url"`
	RawURL      string     `json:"raw_url" xml:"raw_url" form:"raw_url"`
	DownloadURL string     `json:"download_url" xml:"download_url" form:"download_url"`
	DeleteURL   string     `json:"delete_url" xml:"delete_url" form:"delete_url"`
	MimeType    string     `json:"mime_type" xml:"mime_type" form:"mime_type"`
	Size        int64      `json:"size" xml:"size" form:"size"`
	ExpiresAt   *time.Time `json:"expires_at" xml:"expires_at" form:"expires_at"`
	Private     bool       `json:"private" xml:"private" form:"private"`
}

// UpdatePasteExpirationRequest represents the request structure for updating a paste's expiration time
type UpdatePasteExpirationRequest struct {
	ExpiresIn *time.Duration `json:"expires_in" xml:"expires_in" form:"expires_in"` // Duration string for paste expiry (e.g. "24h")
	ExpiresAt *time.Time     `json:"expires_at" xml:"expires_at" form:"expires_at"` // Expiration time for the paste
}

// NewPasteResponse creates a new PasteResponse from a paste
func NewPasteResponse(paste *models.Paste, baseURL string) PasteResponse {
	urlSuffix := paste.ID
	if paste.Extension != "" {
		urlSuffix = urlSuffix + "." + paste.Extension
	}

	return PasteResponse{
		ID:          paste.ID,
		Filename:    paste.Filename,
		URL:         fmt.Sprintf("%s/p/%s", baseURL, urlSuffix),
		RawURL:      fmt.Sprintf("%s/p/%s/raw", baseURL, urlSuffix),
		DownloadURL: fmt.Sprintf("%s/p/%s/download", baseURL, urlSuffix),
		DeleteURL:   fmt.Sprintf("%s/p/%s/%s", baseURL, paste.ID, paste.DeleteKey),
		Private:     paste.Private,
		MimeType:    paste.MimeType,
		Size:        paste.Size,
		ExpiresAt:   paste.ExpiresAt,
	}
}

// ListPastesResponse represents the response structure for listing pastes
type ListPastesResponse struct {
	Pastes []PasteResponse `json:"pastes"`
	Total  int64           `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

// NewListPastesResponse creates a new ListPastesResponse from a list of pastes
func NewListPastesResponse(pastes []models.Paste, baseURL string) ListPastesResponse {
	respose := ListPastesResponse{
		Pastes: make([]PasteResponse, len(pastes)),
		Total:  int64(len(pastes)),
	}

	for i, paste := range pastes {
		respose.Pastes[i] = NewPasteResponse(&paste, baseURL)
	}

	return respose
}

// ShortlinkOptions contains configuration options for creating a new shortlink
type ShortlinkOptions struct {
	URL       string         `json:"url" xml:"url" form:"url"`                      // URL to be shortened
	Title     string         `json:"title" xml:"title" form:"title"`                // Display title for the shortlink
	ExpiresIn *time.Duration `json:"expires_in" xml:"expires_in" form:"expires_in"` // Duration string for shortlink expiry (e.g. "24h")
}

// ShortlinkResponse represents the response structure for creating a new shortlink
type ShortlinkResponse struct {
	ID        string `json:"id" xml:"id" form:"id"`
	URL       string `json:"url" xml:"url" form:"url"`
	Title     string `json:"title" xml:"title" form:"title"`
	ShortURL  string `json:"short_url" xml:"short_url" form:"short_url"`
	StatsURL  string `json:"stats_url" xml:"stats_url" form:"stats_url"`
	DeleteURL string `json:"delete_url" xml:"delete_url" form:"delete_url"`
}

// ChartDataPoint represents a single point of data in time-series statistics
type ChartDataPoint struct {
	Value any       `json:"value" xml:"value" form:"value"` // The value at this point (can be number or string)
	Date  time.Time `json:"date" xml:"date" form:"date"`    // The timestamp for this data point
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
	Size      int64
	HasAPIKey bool
	ExpiresAt *time.Time
	ExpiresIn *time.Duration
}
