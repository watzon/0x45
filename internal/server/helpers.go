package server

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/paste69/internal/models"
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

func (s *Server) createPasteFromMultipart(c *fiber.Ctx, file *multipart.FileHeader, opts *PasteOptions) (*models.Paste, error) {
	f, err := file.Open()
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to read uploaded file")
	}
	defer f.Close()

	return s.createPaste(f, file.Size, file.Header.Get("Content-Type"), opts)
}

func (s *Server) createPasteFromRaw(c *fiber.Ctx, content []byte, opts *PasteOptions) (*models.Paste, error) {
	return s.createPaste(bytes.NewReader(content), int64(len(content)), c.Get("Content-Type"), opts)
}

func (s *Server) createPasteFromURL(c *fiber.Ctx, url string, opts *PasteOptions) (*models.Paste, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Failed to fetch URL")
	}
	defer resp.Body.Close()

	return s.createPaste(resp.Body, resp.ContentLength, resp.Header.Get("Content-Type"), opts)
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
	err := s.db.Where("id = ? AND (expires_at IS NULL OR expires_at > ?)", id, time.Now()).First(&paste).Error
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Paste not found")
	}
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
	switch mimeType {
	case "text/plain", "text/html", "text/css", "text/javascript",
		"application/json", "application/xml", "application/javascript":
		return true
	default:
		return false
	}
}
