package server

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofiber/fiber/v2"
	"github.com/watzon/paste69/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PasteOptions struct {
	Extension string
	ExpiresIn string
	Private   bool
	Filename  string
	APIKey    *models.APIKey // Optional
}

type ShortlinkOptions struct {
	Title     string
	ExpiresIn string
	APIKey    *models.APIKey // Required
}

type ChartDataPoint struct {
	Value interface{} `json:"value"`
	Date  time.Time   `json:"date"`
}

type StatsHistory struct {
	Pastes  []ChartDataPoint
	URLs    []ChartDataPoint
	Storage []ChartDataPoint
}

func (s *Server) createPasteFromMultipart(c *fiber.Ctx, file *multipart.FileHeader, opts *PasteOptions) (*models.Paste, error) {
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

func (s *Server) createPasteFromRaw(c *fiber.Ctx, content []byte, opts *PasteOptions) (*models.Paste, error) {
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

func (s *Server) createPasteFromURL(c *fiber.Ctx, url string, opts *PasteOptions) (*models.Paste, error) {
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

func (s *Server) createPaste(content io.Reader, size int64, contentType string, opts *PasteOptions) (*models.Paste, error) {
	// Store the file
	storagePath, err := s.store.Save(content, opts.Filename)
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
		s.store.Delete(storagePath)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to save paste")
	}

	return paste, nil
}

func (s *Server) createShortlink(url string, opts *ShortlinkOptions) (*models.Shortlink, error) {
	shortlink := &models.Shortlink{
		TargetURL: url,
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

func (s *Server) findShortlink(id string) (*models.Shortlink, error) {
	var shortlink models.Shortlink
	err := s.db.Where("id = ? AND (expires_at IS NULL OR expires_at > ?)", id, time.Now()).First(&shortlink).Error
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Shortlink not found")
	}
	return &shortlink, nil
}

func (s *Server) updateShortlinkStats(shortlink *models.Shortlink, c *fiber.Ctx) {
	now := time.Now()
	s.db.Model(shortlink).Updates(map[string]interface{}{
		"clicks":     gorm.Expr("clicks + 1"),
		"last_click": now,
	})
}

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

func isImageContent(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

func isBinaryContent(mimeType string) bool {
	return !isTextContent(mimeType) && !isImageContent(mimeType)
}

func (s *Server) getStatsHistory(days int) (*StatsHistory, error) {
	history := &StatsHistory{
		Pastes:  make([]ChartDataPoint, days),
		URLs:    make([]ChartDataPoint, days),
		Storage: make([]ChartDataPoint, days),
	}

	// Get data for each day
	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.AddDate(0, 0, 1)

		// Count pastes for this day
		var pasteCount int64
		s.db.Model(&models.Paste{}).
			Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
			Count(&pasteCount)

		// Count URLs for this day
		var urlCount int64
		s.db.Model(&models.Shortlink{}).
			Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
			Count(&urlCount)

		// Get storage size for this day
		var storageSize int64
		s.db.Model(&models.Paste{}).
			Where("created_at <= ?", endOfDay).
			Select("COALESCE(SUM(size), 0)").
			Row().
			Scan(&storageSize)

		// Format the values appropriately
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
	}

	return history, nil
}
