package services

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/models"
	"github.com/watzon/0x45/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/net/html"
	"gorm.io/gorm"
)

type URLService struct {
	db        *gorm.DB
	logger    *zap.Logger
	config    *config.Config
	analytics *AnalyticsService
}

func NewURLService(db *gorm.DB, logger *zap.Logger, config *config.Config) *URLService {
	return &URLService{
		db:        db,
		logger:    logger,
		config:    config,
		analytics: NewAnalyticsService(db, logger, config),
	}
}

// CreateShortlink creates a new URL shortlink
func (s *URLService) CreateShortlink(c *fiber.Ctx) error {
	u := new(ShortlinkOptions)
	if err := c.BodyParser(u); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	apiKey := c.Locals("apiKey").(*models.APIKey)

	shortlink, err := s.createShortlink(apiKey, &ShortlinkOptions{
		URL:       u.URL,
		Title:     u.Title,
		ExpiresIn: u.ExpiresIn,
	})
	if err != nil {
		return err
	}

	return c.JSON(shortlink.ToResponse(s.config.Server.BaseURL))
}

// GetStats returns statistics for a shortened URL
func (s *URLService) GetStats(c *fiber.Ctx) error {
	shortlinkID := c.Params("id")
	if shortlinkID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Shortlink ID is required")
	}

	shortlink, err := s.FindShortlink(shortlinkID)
	if err != nil {
		return err
	}

	// Parse timeframe from query parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var timeframe AnalyticsTimeframe
	if startDate != "" {
		start, err := time.Parse("2006-01-02", startDate)
		if err == nil {
			timeframe.StartTime = &start
		}
	}
	if endDate != "" {
		end, err := time.Parse("2006-01-02", endDate)
		if err == nil {
			timeframe.EndTime = &end
		}
	}

	stats, err := s.analytics.GetResourceStats("shortlink", shortlink.ID, timeframe)
	if err != nil {
		return err
	}

	return c.JSON(stats)
}

// ListURLs returns a paginated list of URLs for the API key
func (s *URLService) ListURLs(c *fiber.Ctx) error {
	apiKey := c.Locals("apiKey").(*models.APIKey)

	var shortlinks []models.Shortlink
	query := s.db.Where("api_key = ?", apiKey.Key)

	// Add pagination
	page := utils.QueryInt(c, "page", 1)
	limit := utils.QueryInt(c, "limit", 20)
	offset := (page - 1) * limit

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return err
	}

	if err := query.Offset(offset).Limit(limit).Find(&shortlinks).Error; err != nil {
		return err
	}

	// Convert shortlinks to response format
	shortlinkResponses := make([]fiber.Map, len(shortlinks))
	for i, shortlink := range shortlinks {
		shortlinkResponses[i] = shortlink.ToResponse(s.config.Server.BaseURL)
	}

	return c.JSON(fiber.Map{
		"shortlinks": shortlinkResponses,
		"total":      total,
		"page":       page,
		"limit":      limit,
	})
}

// UpdateExpiration updates a URL's expiration time
func (s *URLService) UpdateExpiration(c *fiber.Ctx) error {
	var req struct {
		ExpiresIn string `json:"expires_in"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	shortlinkID := c.Params("id")
	shortlink, err := s.FindShortlink(shortlinkID)
	if err != nil {
		return err
	}

	// Parse and validate expiration time
	expiry, err := time.ParseDuration(req.ExpiresIn)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid expiration format")
	}

	expiryTime := time.Now().Add(expiry)
	shortlink.ExpiresAt = &expiryTime

	if err := s.db.Save(shortlink).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update expiration")
	}

	return c.JSON(shortlink.ToResponse(s.config.Server.BaseURL))
}

// Delete deletes a URL (requires API key ownership)
func (s *URLService) Delete(c *fiber.Ctx) error {
	shortlinkID := c.Params("id")
	shortlink, err := s.FindShortlink(shortlinkID)
	if err != nil {
		return err
	}

	apiKey := c.Locals("apiKey").(*models.APIKey)
	if shortlink.APIKey != apiKey.Key {
		return fiber.NewError(fiber.StatusUnauthorized, "Not authorized to delete this shortlink")
	}

	if err := s.db.Delete(shortlink).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete shortlink")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// CleanupExpired removes expired shortlinks
func (s *URLService) CleanupExpired() (int64, error) {
	result := s.db.Where("expires_at < ? AND expires_at IS NOT NULL", time.Now()).Delete(&models.Shortlink{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// Helper functions

func (s *URLService) createShortlink(apiKey *models.APIKey, opts *ShortlinkOptions) (*models.Shortlink, error) {
	// Check if the URL is empty
	if opts.URL == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "URL cannot be empty")
	}

	// Validate URL
	parsedURL, err := url.Parse(opts.URL)
	if err != nil || !parsedURL.IsAbs() || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid URL. Must be a valid absolute HTTP(S) URL")
	}

	if opts.Title == "" {
		title, err := s.fetchURLTitle(opts.URL)
		if err == nil {
			opts.Title = title
		}
	}

	// Sanitize title
	opts.Title = strings.TrimSpace(opts.Title)
	if len(opts.Title) > 255 {
		opts.Title = opts.Title[:255]
	}

	shortlink := &models.Shortlink{
		TargetURL: opts.URL,
		Title:     opts.Title,
		APIKey:    apiKey.Key,
	}

	if opts.ExpiresIn != nil {
		expiryTime := opts.ExpiresIn.Add(time.Now())
		shortlink.ExpiresAt = &expiryTime
	}

	if err := s.db.Create(shortlink).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create shortlink")
	}

	return shortlink, nil
}

// FindShortlink retrieves a shortlink by ID with expiry checking
func (s *URLService) FindShortlink(id string) (*models.Shortlink, error) {
	var shortlink models.Shortlink
	err := s.db.Where("id = ? AND (expires_at IS NULL OR expires_at > ?)", id, time.Now()).First(&shortlink).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "Shortlink not found or expired")
		}
		return nil, err
	}
	return &shortlink, nil
}

func (s *URLService) fetchURLTitle(url string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return "", nil
	}

	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return "", tokenizer.Err()
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == "title" {
				tokenType = tokenizer.Next()
				if tokenType == html.TextToken {
					return strings.TrimSpace(tokenizer.Token().Data), nil
				}
				return "", nil
			}
		}
	}
}
