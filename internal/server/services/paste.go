package services

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/models"
	"github.com/watzon/0x45/internal/storage"
	"github.com/watzon/0x45/internal/utils"
	"github.com/watzon/hdur"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PasteService struct {
	db        *gorm.DB
	logger    *zap.Logger
	config    *config.Config
	storage   storage.Provider
	analytics *AnalyticsService
}

func NewPasteService(db *gorm.DB, logger *zap.Logger, config *config.Config) *PasteService {
	return &PasteService{
		db:        db,
		logger:    logger,
		config:    config,
		storage:   storage.NewProvider(config),
		analytics: NewAnalyticsService(db, logger, config),
	}
}

// CreatePaste handles the creation of a new paste
func (s *PasteService) UploadPaste(c *fiber.Ctx) error {
	s.logger.Debug("Received upload request",
		zap.String("content-type", c.Get("Content-Type")),
		zap.String("body", string(c.Body())))

	p := new(PasteOptions)
	if err := c.BodyParser(p); err != nil {
		s.logger.Error("Failed to parse request body",
			zap.Error(err),
			zap.String("content-type", c.Get("Content-Type")),
			zap.String("body", string(c.Body())))
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	s.logger.Debug("Parsed paste options",
		zap.Any("options", p))

	// Get file content
	var content []byte
	var filename string
	if file, err := c.FormFile("file"); err == nil {
		// Read file content
		f, err := file.Open()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to open uploaded file")
		}
		defer f.Close()

		content, err = io.ReadAll(f)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to read file content")
		}

		// Get filename from form field if available
		if file.Filename != "" {
			filename = file.Filename
		}
	} else if p.URL != "" {
		// Read content from the given URL
		content, err = utils.GetContentFromURL(p.URL)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Failed to fetch URL")
		}

		// Try to get filename from URL if not explicitly provided
		if p.Filename == "" {
			filename = utils.GetFilenameFromURL(p.URL)
		}
	} else if p.Content != "" {
		// Use content from the request body
		content = []byte(p.Content)
	} else {
		return fiber.NewError(fiber.StatusBadRequest, "No file provided")
	}

	// Check for empty content
	if len(content) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Empty file")
	}

	// If we found a filename and none was provided in the request, use it
	if filename != "" && p.Filename == "" {
		p.Filename = filename
	}

	var apiKey *models.APIKey
	if key := c.Locals("apiKey"); key != nil {
		apiKey = key.(*models.APIKey)
	}

	// Check if the user is attempting to do something they're not allowed to do
	if p.Private && apiKey == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Private pastes can only be created with an API key")
	}

	// Create the paste
	paste, err := s.createPaste(bytes.NewReader(content), apiKey, int64(len(content)), p)
	if err != nil {
		return err
	}

	baseURL := s.config.Server.BaseURL
	response := &PasteResponse{
		ID:        paste.ID,
		Filename:  paste.Filename,
		URL:       fmt.Sprintf("%s/p/%s.%s", baseURL, paste.ID, paste.Extension),
		DeleteURL: fmt.Sprintf("%s/p/%s.%s/%s", baseURL, paste.ID, paste.Extension, paste.DeleteKey),
		Private:   paste.Private,
		MimeType:  paste.MimeType,
		Size:      paste.Size,
		ExpiresAt: paste.ExpiresAt,
	}

	// If this is a browser form submission (application/x-www-form-urlencoded), redirect to the paste view
	if strings.Contains(c.Get("Content-Type"), "application/x-www-form-urlencoded") {
		// Store the deletion URL in the session for display after redirect
		c.Cookie(&fiber.Cookie{
			Name:     "deletion_url",
			Value:    response.DeleteURL,
			Path:     "/",
			Expires:  time.Now().Add(5 * time.Minute),
			HTTPOnly: true,
		})
		return c.Redirect(response.URL)
	}

	// For API requests, return JSON response
	return c.JSON(response)
}

// GetPaste retrieves a paste by ID with expiry checking
func (s *PasteService) GetPaste(id string) (*models.Paste, error) {
	// Strip any extension from the ID
	if idx := strings.LastIndex(id, "."); idx != -1 {
		id = id[:idx]
	}

	var paste models.Paste
	err := s.db.Where("id = ? AND (expires_at IS NULL OR expires_at > ?)", id, time.Now()).First(&paste).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "Paste not found or expired")
		}
		return nil, err
	}
	return &paste, nil
}

// GetPasteImage returns an image of the paste suitable for Open Graph
func (s *PasteService) GetPasteImage(c *fiber.Ctx, paste *models.Paste) error {
	// First check if the paste is even text, if not we won't generate an image
	if !s.isTextContent(paste.MimeType) {
		s.logger.Debug("Cannot generate image for non-text content",
			zap.String("mime_type", paste.MimeType),
			zap.String("id", paste.ID))
		return fiber.NewError(fiber.StatusBadRequest, "Cannot generate image for non-text content")
	}

	// Get the content
	content, err := s.storage.Get(paste.StoragePath)
	if err != nil {
		s.logger.Error("Failed to get paste content for image generation",
			zap.Error(err),
			zap.String("id", paste.ID),
			zap.String("storage_path", paste.StoragePath))
		return err
	}

	// Generate the image
	image, err := GenerateCodeImage(string(content))
	if err != nil {
		s.logger.Error("Failed to generate paste image",
			zap.Error(err),
			zap.String("id", paste.ID))
		return err
	}

	c.Set("Cache-Control", "max-age=31536000, immutable")
	c.Set("Content-Type", "image/png")
	return c.Send(image)
}

// RenderPaste renders the paste view for text content
func (s *PasteService) RenderPaste(c *fiber.Ctx, paste *models.Paste) error {
	if s.isTextContent(paste.MimeType) {
		return s.renderPasteView(c, paste)
	}
	if s.isImageContent(paste.MimeType) {
		return s.RenderPasteRaw(c, paste)
	}
	return c.Redirect("/p/" + paste.ID + "/download")
}

// RenderPasteRaw serves the raw content with proper content type
func (s *PasteService) RenderPasteRaw(c *fiber.Ctx, paste *models.Paste) error {
	content, err := s.storage.Get(paste.StoragePath)
	if err != nil {
		return err
	}
	c.Set("Content-Type", paste.MimeType)
	// Add permanent cache headers since content is immutable
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Set("ETag", paste.ID)
	return c.Send(content)
}

// RenderPasteJSON serves the paste as JSON. If the paste is text, the content will be included
// in the response. Otherwise only the URL will be included for downloading purposes.
func (s *PasteService) RenderPasteJSON(c *fiber.Ctx, paste *models.Paste) error {
	pasteJson := struct {
		ID       string `json:"id"`
		Filename string `json:"filename"`
		MimeType string `json:"mimeType"`
		URL      string `json:"url"`
		Content  string `json:"content"`
	}{
		ID:       paste.ID,
		Filename: paste.Filename,
		MimeType: paste.MimeType,
		URL:      fmt.Sprintf("%s/p/%s.%s", s.config.Server.BaseURL, paste.ID, paste.Extension),
	}

	if s.isTextContent(paste.MimeType) {
		content, err := s.storage.Get(paste.StoragePath)
		if err != nil {
			return err
		}
		pasteJson.Content = string(content)
	}

	return c.JSON(pasteJson)
}

// RenderDownload serves the content as a downloadable file
func (s *PasteService) RenderDownload(c *fiber.Ctx, paste *models.Paste) error {
	content, err := s.storage.Get(paste.StoragePath)
	if err != nil {
		return err
	}

	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, paste.Filename))
	// Add permanent cache headers since content is immutable
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Set("ETag", paste.ID)
	return c.Send(content)
}

// DeleteWithKey deletes a paste using its deletion key
func (s *PasteService) DeleteWithKey(c *fiber.Ctx, id string) error {
	key := c.Params("key") // Get key from URL path instead of query
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Deletion key is required")
	}

	// Strip any extension from the ID
	if idx := strings.LastIndex(id, "."); idx != -1 {
		id = id[:idx]
	}

	paste, err := s.GetPaste(id)
	if err != nil {
		// Pass through the 404 error from GetPaste
		return err
	}

	if paste.DeleteKey != key {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid deletion key")
	}

	// For DELETE requests, delete the paste
	if c.Method() == fiber.MethodDelete {
		if err := s.Delete(c, id); err != nil {
			return err
		}

		// Return appropriate response based on Accept header
		if strings.Contains(c.Get("Accept"), "application/json") {
			return c.JSON(fiber.Map{
				"message": "Paste deleted successfully",
				"id":      id,
			})
		}

		// For HTML requests, render the success page
		return c.Render("delete_success", fiber.Map{
			"isDeleteSuccess": true,
			"baseUrl":         s.config.Server.BaseURL,
		}, "layouts/main")
	}

	// For GET requests, show a confirmation page
	if strings.Contains(c.Get("Accept"), "application/json") {
		return c.JSON(fiber.Map{
			"message": "Paste found and will be deleted",
			"id":      id,
		})
	}

	// For HTML requests, render the confirmation page
	return c.Render("delete_confirm", fiber.Map{
		"isDeleteConfirm": true,
		"baseUrl":         s.config.Server.BaseURL,
		"pasteId":         id,
		"deleteKey":       key,
	}, "layouts/main")
}

// Delete removes a paste and its associated files
func (s *PasteService) Delete(c *fiber.Ctx, id string) error {
	// Strip any extension from the ID
	if idx := strings.LastIndex(id, "."); idx != -1 {
		id = id[:idx]
	}

	paste, err := s.GetPaste(id)
	if err != nil {
		return err
	}

	if err := s.storage.Delete(paste.StoragePath); err != nil {
		s.logger.Error("failed to delete paste content", zap.Error(err))
	}

	return s.db.Delete(paste).Error
}

// ListPastes returns a paginated list of pastes for the API key
func (s *PasteService) ListPastes(c *fiber.Ctx) error {
	apiKey := c.Locals("apiKey").(*models.APIKey)

	var pastes []models.Paste
	query := s.db.Where("api_key = ?", apiKey.Key)

	// Add pagination
	page := utils.QueryInt(c, "page", 1)
	limit := utils.QueryInt(c, "limit", 20)
	offset := (page - 1) * limit

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return err
	}

	if err := query.Offset(offset).Limit(limit).Find(&pastes).Error; err != nil {
		return err
	}

	// Convert pastes to response format
	respose := NewListPastesResponse(pastes, s.config.Server.BaseURL)
	return c.JSON(respose)
}

// UpdateExpiration updates a paste's expiration time
func (s *PasteService) UpdateExpiration(c *fiber.Ctx, id string) error {
	// Strip any extension from the ID
	if idx := strings.LastIndex(id, "."); idx != -1 {
		id = id[:idx]
	}

	paste, err := s.GetPaste(id)
	if err != nil {
		return err
	}

	req := new(UpdatePasteExpirationRequest)
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	expiryTime, err := s.calculateExpiry(ExpiryOptions{
		Size:      paste.Size,
		HasAPIKey: paste.APIKey != "",
		ExpiresAt: req.ExpiresAt,
		ExpiresIn: req.ExpiresIn,
	})
	if err != nil {
		return err
	}

	paste.ExpiresAt = expiryTime
	if err := s.db.Save(paste).Error; err != nil {
		return err
	}

	// Build response
	response := NewPasteResponse(paste, s.config.Server.BaseURL)
	return c.JSON(response)
}

// CleanupExpired removes expired pastes and their associated files
func (s *PasteService) CleanupExpired() (int64, error) {
	var totalDeleted int64

	// Use a transaction to ensure consistency
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var pastes []models.Paste
		if err := tx.Where("expires_at < ? AND expires_at IS NOT NULL", time.Now()).Find(&pastes).Error; err != nil {
			return err
		}

		for _, paste := range pastes {
			// Delete storage content first
			if err := s.storage.Delete(paste.StoragePath); err != nil {
				s.logger.Error("failed to delete paste content",
					zap.String("id", paste.ID),
					zap.String("path", paste.StoragePath),
					zap.Error(err),
				)
				// Skip this paste if we can't delete the storage
				continue
			}

			// Delete the database record only if storage deletion was successful
			if err := tx.Delete(&paste).Error; err != nil {
				s.logger.Error("failed to delete paste record",
					zap.String("id", paste.ID),
					zap.Error(err),
				)
				// Try to recover the storage file since we couldn't delete the record
				if _, err := s.storage.Put(paste.StoragePath, bytes.NewReader([]byte{})); err != nil {
					s.logger.Error("failed to recover storage after failed deletion",
						zap.String("id", paste.ID),
						zap.String("path", paste.StoragePath),
						zap.Error(err),
					)
				}
				continue
			}

			totalDeleted++
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return totalDeleted, nil
}

// Helper functions

// validateFileSize checks if the file size is within the allowed limits
func (s *PasteService) validateFileSize(size int64, apiKey *models.APIKey) error {
	// First check against absolute maximum size for security
	if size > int64(s.config.Server.MaxUploadSize) {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("File exceeds maximum allowed size of %d bytes", s.config.Server.MaxUploadSize))
	}

	// Then check against the appropriate tier limit
	if apiKey != nil {
		if size > int64(s.config.Server.APIUploadSize) {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("File exceeds API upload limit of %d bytes", s.config.Server.APIUploadSize))
		}
	} else {
		if size > int64(s.config.Server.DefaultUploadSize) {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("File exceeds default upload limit of %d bytes", s.config.Server.DefaultUploadSize))
		}
	}

	return nil
}

func (s *PasteService) createPaste(content io.Reader, apiKey *models.APIKey, size int64, opts *PasteOptions) (*models.Paste, error) {
	// Read content for MIME type detection
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to read content")
	}

	// Check file size against limit either globally or per API key
	if err := s.validateFileSize(size, apiKey); err != nil {
		return nil, err
	}

	// Detect MIME type if not provided
	mime := mimetype.Detect(contentBytes)
	contentType := mime.String()

	// Create paste record
	paste := &models.Paste{
		Filename:  opts.Filename,
		MimeType:  contentType,
		Size:      size,
		Extension: opts.Extension,
		Private:   opts.Private,
	}

	// Set extension in order of precedence
	if paste.Extension == "" {
		if paste.Filename != "" {
			parts := strings.Split(paste.Filename, ".")
			if len(parts) > 1 {
				paste.Extension = parts[len(parts)-1]
			}
		}

		if paste.Extension == "" {
			mime := mimetype.Detect(contentBytes)
			paste.Extension = strings.TrimPrefix(mime.Extension(), ".")

			if paste.Extension == "" && strings.HasPrefix(contentType, "text/") {
				paste.Extension = "txt"
			}
		}
	}

	// Calculate expiry time if provided
	if (opts.ExpiresIn != nil && opts.ExpiresIn.String() != "") || opts.ExpiresAt != nil {
		expiry, err := s.calculateExpiry(ExpiryOptions{
			Size:      int64(len(contentBytes)),
			HasAPIKey: apiKey != nil,
			ExpiresIn: opts.ExpiresIn,
			ExpiresAt: opts.ExpiresAt,
		})
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		paste.ExpiresAt = expiry
	}

	// Set API key if provided
	if apiKey != nil {
		paste.APIKey = apiKey.Key
	}

	// Use a transaction for the entire creation process
	var storagePath string
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Set the default storage configuration
		for _, storage := range s.config.Storage {
			if storage.IsDefault {
				paste.StorageName = storage.Name
				paste.StorageType = storage.Type
				break
			}
		}

		if paste.StorageName == "" {
			return fiber.NewError(fiber.StatusInternalServerError, "No default storage configuration found")
		}

		// Create the initial database record
		if err := tx.Create(paste).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to save paste")
		}

		// Generate filename
		filename := paste.ID
		if paste.Extension != "" {
			filename = paste.ID + "." + paste.Extension
		}

		// Store the content and get the storage path
		var err error
		storagePath, err = s.storage.Put(filename, bytes.NewReader(contentBytes))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to store content")
		}

		// Update the paste with the storage path
		paste.StoragePath = storagePath
		if err := tx.Save(paste).Error; err != nil {
			// Try to cleanup the stored content since we couldn't update the record
			_ = s.storage.Delete(storagePath)
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update paste")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return paste, nil
}

func (s *PasteService) isTextContent(mimeType string) bool {
	switch {
	case strings.HasPrefix(mimeType, "text/"):
		return true
	case strings.Contains(mimeType, "json"):
		return true
	case strings.Contains(mimeType, "xml"):
		return true
	case strings.Contains(mimeType, "javascript"):
		return true
	case strings.Contains(mimeType, "yaml"):
		return true
	case strings.Contains(mimeType, "x-www-form-urlencoded"):
		return true
	default:
		return false
	}
}

func (s *PasteService) isImageContent(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

func (s *PasteService) renderPasteView(c *fiber.Ctx, paste *models.Paste) error {
	content, err := s.storage.Get(paste.StoragePath)
	if err != nil {
		return err
	}

	// Determine lexer based on extension or content
	var lexer chroma.Lexer
	if paste.Extension != "" {
		lexer = lexers.Get(paste.Extension)
	}
	if lexer == nil {
		lexer = lexers.Get(paste.MimeType)
	}
	if lexer == nil {
		lexer = lexers.Analyse(string(content))
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Create formatter
	formatter := html.New(
		html.WithLineNumbers(true),
		html.WithLinkableLineNumbers(true, ""),
		html.TabWidth(4),
		html.WithClasses(false), // Use inline styles
	)

	// Create buffer for highlighted code
	var codeBuffer bytes.Buffer

	// Write highlighted code
	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return err
	}

	// Use GitHub Dark style
	style := styles.Get("github-dark")
	if style == nil {
		style = styles.Fallback
	}

	if err := formatter.Format(&codeBuffer, style, iterator); err != nil {
		return err
	}

	// Build paste ID with extension if available
	pasteID := paste.ID
	if paste.Extension != "" {
		pasteID = paste.ID + "." + paste.Extension
	}

	// Check for deletion URL cookie
	var deletionUrl string
	if cookie := c.Cookies("deletion_url"); cookie != "" {
		// Clear the cookie before reading it to ensure one-time use
		c.Cookie(&fiber.Cookie{
			Name:     "deletion_url",
			Value:    "",
			Path:     "/",
			Expires:  time.Now().Add(-24 * time.Hour),
			HTTPOnly: true,
		})
		deletionUrl = cookie
	}

	return c.Render("paste", fiber.Map{
		"isPaste":     true,
		"id":          pasteID,
		"filename":    paste.Filename,
		"extension":   paste.Extension,
		"created":     paste.CreatedAt.Format("2006-01-02 15:04:05"),
		"expires":     formatExpiryTime(paste.ExpiresAt),
		"language":    lexer.Config().Name,
		"content":     codeBuffer.String(),
		"rawContent":  string(content),
		"baseUrl":     s.config.Server.BaseURL,
		"deletionUrl": deletionUrl,
		"metadata": fiber.Map{
			"size":      formatSize(paste.Size),
			"mimeType":  paste.MimeType,
			"createdAt": paste.CreatedAt,
			"expiresAt": paste.ExpiresAt,
		},
	}, "layouts/main")
}

func formatExpiryTime(t *time.Time) string {
	if t == nil {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func (s *PasteService) calculateExpiry(opts ExpiryOptions) (*time.Time, error) {
	// Calculate maximum allowed retention based on file size
	maxRetention := s.calculateMaxRetention(opts.Size, opts.HasAPIKey)
	maxDuration := hdur.Hours(maxRetention * 24)

	// Handle explicit expiry requests
	if opts.ExpiresAt != nil {
		now := time.Now()
		if opts.ExpiresAt.Before(now) {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Expiration time must be in the future")
		}
		requestedDuration := hdur.Sub(*opts.ExpiresAt, now)
		if requestedDuration.Days > maxDuration.Days {
			return nil, fiber.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("Maximum allowed expiry for this file size is %.1f days", float64(maxDuration.Days)))
		}
		return opts.ExpiresAt, nil
	}

	if opts.ExpiresIn != nil {
		if opts.ExpiresIn.Days > maxDuration.Days {
			return nil, fiber.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("Maximum allowed expiry for this file size is %.1f days", float64(maxDuration.Days)))
		}
		expiryTime := opts.ExpiresIn.Add(time.Now())
		return &expiryTime, nil
	}

	// If no explicit expiry is set, use maximum retention
	expiryTime := maxDuration.Add(time.Now())
	return &expiryTime, nil
}

func (s *PasteService) calculateMaxRetention(size int64, hasAPIKey bool) float64 {
	// Get retention limits based on API key status
	var retention config.RetentionLimitConfig
	if hasAPIKey {
		retention = s.config.Retention.WithKey
	} else {
		retention = s.config.Retention.NoKey
	}

	// Calculate retention based on file size ratio
	sizeRatio := float64(size) / float64(s.config.Server.MaxUploadSize)
	if sizeRatio > 1 {
		sizeRatio = 1
	}

	// Linear interpolation between min and max age based on size ratio
	return retention.MinAge + (retention.MaxAge-retention.MinAge)*(1-sizeRatio)
}
