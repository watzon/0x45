package server

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/models"
	"go.uber.org/zap"
	"golang.org/x/net/html"
	"gorm.io/gorm"
)

// PasteOptions contains configuration options for creating a new paste
type PasteOptions struct {
	Extension string         // File extension (optional)
	ExpiresIn string         // Duration string for paste expiry (e.g. "24h")
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

// createPasteFromMultipart creates a new paste from a multipart file upload
// It handles file reading, MIME type detection, and storage
func (s *Server) createPasteFromMultipart(c *fiber.Ctx, file *multipart.FileHeader, opts *PasteOptions) (*models.Paste, error) {
	// Add API key from context if available
	if apiKey, ok := c.Locals("apiKey").(*models.APIKey); ok {
		opts.APIKey = apiKey
	}

	f, err := file.Open()
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to read uploaded file")
	}
	defer f.Close()

	// Read all content first
	content, err := io.ReadAll(f)
	if err != nil {
		s.logger.Error("failed to read multipart content",
			zap.String("filename", file.Filename),
			zap.Error(err),
		)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to read upload")
	}

	// Detect mime type from the byte slice
	mime := mimetype.Detect(content)
	mimeType := mime.String()
	if mimeType == "" {
		mimeType = file.Header.Get("Content-Type")
	}

	s.logger.Debug("processing multipart upload",
		zap.String("filename", file.Filename),
		zap.Int("content_size", len(content)),
		zap.String("mime_type", mimeType),
	)

	return s.createPaste(bytes.NewReader(content), int64(len(content)), mimeType, opts)
}

// createPasteFromRaw creates a new paste from raw content bytes
// It handles MIME type detection and storage of the raw content
func (s *Server) createPasteFromRaw(c *fiber.Ctx, content []byte, opts *PasteOptions) (*models.Paste, error) {
	// Add API key from context if available
	if apiKey, ok := c.Locals("apiKey").(*models.APIKey); ok {
		opts.APIKey = apiKey
	}

	// Log the incoming content size
	s.logger.Debug("received raw content",
		zap.Int("content_length", len(content)),
	)

	// Detect mime type from the byte slice directly
	mime := mimetype.Detect(content)
	mimeType := mime.String()

	// If mime detection failed, fallback to Content-Type header
	if mimeType == "" {
		mimeType = c.Get("Content-Type")
	}

	s.logger.Debug("creating paste",
		zap.String("mime_type", mimeType),
		zap.Int("content_size", len(content)),
	)

	return s.createPaste(bytes.NewReader(content), int64(len(content)), mimeType, opts)
}

// createPasteFromURL creates a new paste by downloading content from a URL
// It handles HTTP fetching, MIME type detection, and storage
func (s *Server) createPasteFromURL(c *fiber.Ctx, url string, opts *PasteOptions) (*models.Paste, error) {
	// Add API key from context if available
	if apiKey, ok := c.Locals("apiKey").(*models.APIKey); ok {
		opts.APIKey = apiKey
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Failed to fetch URL")
	}
	defer resp.Body.Close()

	// Read all content first
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("failed to read URL content",
			zap.String("url", url),
			zap.Error(err),
		)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to read URL content")
	}

	// Detect mime type from the byte slice
	mime := mimetype.Detect(content)
	mimeType := mime.String()
	if mimeType == "" {
		mimeType = resp.Header.Get("Content-Type")
	}

	s.logger.Debug("processing URL upload",
		zap.String("url", url),
		zap.Int("content_size", len(content)),
		zap.String("mime_type", mimeType),
	)

	return s.createPaste(bytes.NewReader(content), int64(len(content)), mimeType, opts)
}

// createPaste is the core paste creation function used by all paste creation methods
// It handles storage and database operations for creating a new paste
func (s *Server) createPaste(content io.Reader, size int64, contentType string, opts *PasteOptions) (*models.Paste, error) {
	// Get default store if no specific store is requested
	store, storeName, err := s.storage.GetDefaultStore()
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "No storage configuration available")
	}

	// Store the file
	storagePath, err := store.Save(content, opts.Filename)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to save file")
	}

	// Create paste record
	paste := &models.Paste{
		Filename:    opts.Filename,
		MimeType:    contentType,
		Size:        size,
		Extension:   opts.Extension,
		StoragePath: storagePath,
		StorageType: store.Type(),
		StorageName: storeName,
		Private:     opts.Private,
	}

	if opts.APIKey != nil {
		paste.APIKey = opts.APIKey.Key
	}

	if opts.ExpiresIn != "" {
		expiry, err := time.ParseDuration(opts.ExpiresIn)
		if err == nil {
			expiryTime := time.Now().Add(expiry)
			paste.ExpiresAt = &expiryTime
		}
	}

	// Save to database
	if err := s.db.Create(paste).Error; err != nil {
		// Try to cleanup stored file
		_ = store.Delete(storagePath)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to save paste")
	}

	return paste, nil
}

// createShortlink creates a new URL shortlink with the given options
// It handles database operations and expiry time calculation
func (s *Server) createShortlink(_url string, opts *ShortlinkOptions) (*models.Shortlink, error) {
	// Check if the URL is empty before we do anything else
	if _url == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "URL cannot be empty")
	}

	// Validate URL
	parsedURL, err := url.Parse(_url)
	if err != nil || !parsedURL.IsAbs() || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid URL. Must be a valid absolute HTTP(S) URL")
	}

	if opts.Title == "" {
		title, err := fetchURLTitle(_url)
		if err == nil {
			opts.Title = title
		}
	}

	// Sanitize title - trim spaces and limit length
	opts.Title = strings.TrimSpace(opts.Title)
	if len(opts.Title) > 255 { // Common DB VARCHAR limit
		opts.Title = opts.Title[:255]
	}

	shortlink := &models.Shortlink{
		TargetURL: _url,
		Title:     opts.Title,
		APIKey:    opts.APIKey.Key,
	}

	if opts.ExpiresIn != "" {
		expiry, err := time.ParseDuration(opts.ExpiresIn)
		if err == nil {
			expiryTime := time.Now().Add(expiry)
			shortlink.ExpiresAt = &expiryTime
		}
	}

	if err := s.db.Create(shortlink).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create shortlink")
	}

	return shortlink, nil
}

func fetchURLTitle(url string) (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Only attempt to parse HTML content
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return "", nil
	}

	// Parse HTML and look for title
	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			// End of document or error
			return "", tokenizer.Err()
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == "title" {
				// Next token should be the title text
				tokenType = tokenizer.Next()
				if tokenType == html.TextToken {
					return strings.TrimSpace(tokenizer.Token().Data), nil
				}
				return "", nil
			}
		}
	}
}

// findPaste retrieves a paste by ID with expiry checking
// It performs two queries: one to check existence and another to verify expiry
func (s *Server) findPaste(id string) (*models.Paste, error) {
	var paste models.Paste

	// First try without expiry check to see if paste exists at all
	err := s.db.Where("id = ?", id).First(&paste).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("paste not found",
				zap.String("id", id),
			)
			return nil, fiber.NewError(fiber.StatusNotFound, "Paste not found")
		}
		// Log the actual error if it's something else
		s.logger.Error("database error while finding paste",
			zap.String("id", id),
			zap.Error(err),
		)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	// Now check with expiry
	err = s.db.Where("id = ? AND (expires_at IS NULL OR expires_at > ?)", id, time.Now()).First(&paste).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("paste has expired",
				zap.String("id", id),
				zap.Time("expires_at", *paste.ExpiresAt),
			)
			return nil, fiber.NewError(fiber.StatusNotFound, "Paste has expired")
		}
		s.logger.Error("database error while checking paste expiry",
			zap.String("id", id),
			zap.Error(err),
		)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Database error")
	}

	s.logger.Debug("paste found successfully",
		zap.String("id", id),
		zap.String("filename", paste.Filename),
		zap.Int64("size", paste.Size),
	)

	return &paste, nil
}

// findShortlink retrieves an active shortlink by ID
// It includes expiry checking in the query
func (s *Server) findShortlink(id string) (*models.Shortlink, error) {
	var shortlink models.Shortlink
	err := s.db.Where("id = ? AND (expires_at IS NULL OR expires_at > ?)", id, time.Now()).First(&shortlink).Error
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Shortlink not found")
	}
	return &shortlink, nil
}

// updateShortlinkStats increments the click count and updates last click time
// for a given shortlink
func (s *Server) updateShortlinkStats(shortlink *models.Shortlink) {
	now := time.Now()
	s.db.Model(shortlink).Updates(map[string]any{
		"clicks":     gorm.Expr("clicks + 1"),
		"last_click": now,
	})
}

// isTextContent determines if a MIME type represents textual content
// This includes plain text, JSON, XML, and JavaScript
func isTextContent(mimeType string) bool {
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	if strings.HasPrefix(mimeType, "application/") {
		subtype := strings.SplitN(mimeType, "/", 2)[1]
		if subtype == "json" || subtype == "xml" || subtype == "javascript" {
			return true
		}
	}
	return false
}

// isImageContent determines if a MIME type represents an image
func isImageContent(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

// getStatsHistory generates usage statistics for the specified number of days
func (s *Server) getStatsHistory(days int) (*StatsHistory, error) {
	history := &StatsHistory{
		Pastes:     make([]ChartDataPoint, days),
		URLs:       make([]ChartDataPoint, days),
		Storage:    make([]ChartDataPoint, days),
		AvgSize:    make([]ChartDataPoint, days),
		APIKeys:    make([]ChartDataPoint, days),
		Extensions: make([]ChartDataPoint, days),
		ErrorRates: make([]ChartDataPoint, days),
	}

	// Get data for each day
	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.AddDate(0, 0, 1)

		// Existing metrics
		var pasteCount, urlCount, storageSize int64
		s.db.Model(&models.Paste{}).
			Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
			Count(&pasteCount)

		s.db.Model(&models.Shortlink{}).
			Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
			Count(&urlCount)

		err := s.db.Model(&models.Paste{}).
			Where("created_at <= ?", endOfDay).
			Select("COALESCE(SUM(size), 0)").
			Row().
			Scan(&storageSize)
		if err != nil {
			return nil, fmt.Errorf("getting storage size: %w", err)
		}

		// New metrics
		var avgSize float64
		err = s.db.Model(&models.Paste{}).
			Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
			Select("COALESCE(AVG(size), 0)").
			Row().
			Scan(&avgSize)
		if err != nil {
			return nil, fmt.Errorf("getting avg size: %w", err)
		}

		var activeAPIKeys int64
		s.db.Model(&models.APIKey{}).
			Where("created_at <= ? AND verified = ?", endOfDay, true).
			Count(&activeAPIKeys)

		// Get top extension for the day
		var topExtension struct {
			Extension string
			Count     int64
		}
		s.db.Model(&models.Paste{}).
			Select("extension, COUNT(*) as count").
			Where("created_at BETWEEN ? AND ? AND extension != ''", startOfDay, endOfDay).
			Group("extension").
			Order("count DESC").
			Limit(1).
			Scan(&topExtension)

		// Store all values
		history.Pastes[i] = ChartDataPoint{
			Value: pasteCount,
			Date:  startOfDay,
		}
		history.URLs[i] = ChartDataPoint{
			Value: urlCount,
			Date:  startOfDay,
		}
		history.Storage[i] = ChartDataPoint{
			Value: humanize.IBytes(uint64(storageSize)),
			Date:  startOfDay,
		}
		history.AvgSize[i] = ChartDataPoint{
			Value: humanize.IBytes(uint64(avgSize)),
			Date:  startOfDay,
		}
		history.APIKeys[i] = ChartDataPoint{
			Value: activeAPIKeys,
			Date:  startOfDay,
		}
		history.Extensions[i] = ChartDataPoint{
			Value: fmt.Sprintf("%s (%d)", topExtension.Extension, topExtension.Count),
			Date:  startOfDay,
		}
		// Error rates would need to be tracked elsewhere in the application
		history.ErrorRates[i] = ChartDataPoint{
			Value: 0, // Placeholder until we implement error tracking
			Date:  startOfDay,
		}
	}

	return history, nil
}

// getStorageByFileType retrieves and categorizes storage usage by file type
func (s *Server) getStorageByFileType() (map[string]int64, error) {
	var results []struct {
		MimeType  string
		TotalSize int64
	}

	err := s.db.Model(&models.Paste{}).
		Select("mime_type, SUM(size) as total_size").
		Group("mime_type").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	categories := map[string]int64{
		"Text":      0,
		"Images":    0,
		"Archives":  0,
		"Documents": 0,
		"Other":     0,
	}

	for _, result := range results {
		category := categorizeMimeType(result.MimeType)
		categories[category] += result.TotalSize
	}

	// Create a new map with only non-zero values
	nonZeroCategories := make(map[string]int64)
	for category, size := range categories {
		if size > 0 {
			nonZeroCategories[category] = size
		}
	}

	return nonZeroCategories, nil
}

// categorizeMimeType categorizes a MIME type into one of the predefined categories
func categorizeMimeType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "text/"):
		return "Text"
	case strings.HasPrefix(mimeType, "image/"):
		return "Images"
	case strings.Contains(mimeType, "compressed") || strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "tar"):
		return "Archives"
	case strings.Contains(mimeType, "pdf") || strings.Contains(mimeType, "document") || strings.Contains(mimeType, "msword"):
		return "Documents"
	default:
		return "Other"
	}
}

func (s *Server) cleanupUnverifiedKeys() int64 {
	count := s.db.Where("verified = ? AND verify_expiry < ?",
		false, time.Now()).Delete(&models.APIKey{}).RowsAffected
	return count
}

// cleanupExpiredContent removes expired pastes and their associated files
func (s *Server) cleanupExpiredContent() int64 {
	// Parse max age duration
	maxAge, err := time.ParseDuration(s.config.Server.Cleanup.MaxAge)
	if err != nil {
		s.logger.Error("failed to parse cleanup max age", zap.Error(err))
		return 0
	}

	// Find expired pastes
	var expiredPastes []models.Paste
	if err := s.db.Where("expires_at < ? OR (expires_at IS NULL AND created_at < ?)",
		time.Now(), time.Now().Add(-maxAge)).Find(&expiredPastes).Error; err != nil {
		s.logger.Error("failed to find expired pastes", zap.Error(err))
		return 0
	}

	// Delete each expired paste
	for _, paste := range expiredPastes {
		// Get the storage backend
		store, err := s.storage.GetStore(paste.StorageName)
		if err != nil {
			s.logger.Error("failed to get storage for cleanup",
				zap.String("storage", paste.StorageName),
				zap.Error(err))
			continue
		}

		// Delete the file from storage
		if err := store.Delete(paste.StoragePath); err != nil {
			s.logger.Error("failed to delete file during cleanup",
				zap.String("id", paste.ID),
				zap.Error(err))
			continue
		}

		// Delete the database record
		if err := s.db.Delete(&paste).Error; err != nil {
			s.logger.Error("failed to delete paste record during cleanup",
				zap.String("id", paste.ID),
				zap.Error(err))
			continue
		}

		s.logger.Info("cleaned up expired paste",
			zap.String("id", paste.ID),
			zap.String("filename", paste.Filename))
	}

	s.logger.Info("cleanup completed",
		zap.Int("pastes_cleaned", len(expiredPastes)))

	return int64(len(expiredPastes))
}
