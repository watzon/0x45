package server

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/dustin/go-humanize"
	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/models"
	"github.com/watzon/0x45/internal/utils"
	"go.uber.org/zap"
)

// Web Interface Handlers

// handleIndex serves the main web interface page
func (s *Server) handleIndex(c *fiber.Ctx) error {
	// Generate retention data with config
	retentionStats, err := utils.GenerateRetentionData(int64(s.config.Server.MaxUploadSize), s.config)
	if err != nil {
		s.logger.Error("failed to generate retention data", zap.Error(err))
		// Continue with empty retention data
	}

	noKeyHistory, _ := json.Marshal(retentionStats.Data["noKey"])
	withKeyHistory, _ := json.Marshal(retentionStats.Data["withKey"])

	return c.Render("index", fiber.Map{
		"retention": fiber.Map{
			"noKey":          retentionStats.NoKeyRange,
			"withKey":        retentionStats.WithKeyRange,
			"minAge":         s.config.Retention.NoKey.MinAge,
			"maxAge":         s.config.Retention.WithKey.MaxAge,
			"maxSize":        s.config.Server.MaxUploadSize / (1024 * 1024),
			"noKeyHistory":   string(noKeyHistory),
			"withKeyHistory": string(withKeyHistory),
		},
		"baseUrl": s.config.Server.BaseURL,
	}, "layouts/main")
}

// handleStats serves the statistics page
// Displays current statistics and historical data for pastes and URLs
func (s *Server) handleStats(c *fiber.Ctx) error {
	// Get current stats
	var totalPastes, totalUrls int64
	s.db.Model(&models.Paste{}).Count(&totalPastes)
	s.db.Model(&models.Shortlink{}).Count(&totalUrls)

	// Get historical data with empty defaults
	history := &StatsHistory{
		Pastes:  make([]ChartDataPoint, 7),
		URLs:    make([]ChartDataPoint, 7),
		Storage: make([]ChartDataPoint, 7),
	}

	if histData, err := s.getStatsHistory(7); err == nil {
		history = histData
	} else {
		s.logger.Error("failed to get stats history", zap.Error(err))
	}

	// Convert data to JSON strings with empty array fallbacks
	pastesHistory, _ := json.Marshal(history.Pastes)
	urlsHistory, _ := json.Marshal(history.URLs)
	storageHistory, _ := json.Marshal(history.Storage)

	// Get storage by file type data with empty map fallback
	storageByType := make(map[string]int64)
	if typeData, err := s.getStorageByFileType(); err == nil {
		storageByType = typeData
	} else {
		s.logger.Error("failed to get storage by file type", zap.Error(err))
	}

	// Convert storageByType to JSON
	storageByTypeJSON, _ := json.Marshal(storageByType)

	// Get average paste size with zero default
	var avgSize float64
	if err := s.db.Model(&models.Paste{}).
		Select("COALESCE(AVG(NULLIF(size, 0)), 0)").
		Row().
		Scan(&avgSize); err != nil {
		s.logger.Error("failed to get average size", zap.Error(err))
		avgSize = 0
	}

	// Get active API keys count
	var activeApiKeys int64
	s.db.Model(&models.APIKey{}).Where("verified = ?", true).Count(&activeApiKeys)

	// Get popular extensions with empty map fallback
	extensionStats := make(map[string]int64)
	rows, err := s.db.Model(&models.Paste{}).
		Select("extension, COUNT(*) as count").
		Where("extension != ''").
		Group("extension").
		Order("count DESC").
		Limit(10).
		Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ext string
			var count int64
			if err := rows.Scan(&ext, &count); err == nil {
				extensionStats[ext] = count
			}
		}
	} else {
		s.logger.Error("failed to get extension stats", zap.Error(err))
	}

	// Get expiring content counts
	var expiringPastes, expiringUrls int64
	twentyFourHours := time.Now().Add(24 * time.Hour)
	s.db.Model(&models.Paste{}).
		Where("expires_at < ? AND expires_at > ?", twentyFourHours, time.Now()).
		Count(&expiringPastes)
	s.db.Model(&models.Shortlink{}).
		Where("expires_at < ? AND expires_at > ?", twentyFourHours, time.Now()).
		Count(&expiringUrls)

	// Get private vs public paste ratio with zero defaults
	var privatePastes int64
	s.db.Model(&models.Paste{}).Where("private = ?", true).Count(&privatePastes)
	publicPastes := totalPastes - privatePastes

	// Calculate private ratio safely
	var privateRatio float64
	if totalPastes > 0 {
		privateRatio = float64(privatePastes) / float64(totalPastes) * 100
	}

	// Get average paste views safely
	var avgViews float64
	if err := s.db.Model(&models.Paste{}).
		Select("COALESCE(AVG(NULLIF(views, 0)), 0)").
		Row().
		Scan(&avgViews); err != nil {
		s.logger.Error("failed to get average views", zap.Error(err))
		avgViews = 0
	}

	// Get total storage used with zero default
	totalStorage, err := s.getStorageSize()
	if err != nil {
		return fmt.Errorf("failed getting storage size: %w", err)
	}

	return c.Render("stats", fiber.Map{
		"stats": fiber.Map{
			// Current totals (these are safe as Count returns 0 if no rows)
			"pastes":        totalPastes,
			"urls":          totalUrls,
			"storage":       humanize.IBytes(totalStorage),
			"activeApiKeys": activeApiKeys,

			// Historical data (already has empty defaults)
			"pastesHistory":  string(pastesHistory),
			"urlsHistory":    string(urlsHistory),
			"storageHistory": string(storageHistory),

			// File type statistics (already has empty defaults)
			"storageByType":  string(storageByTypeJSON),
			"extensionStats": extensionStats,
			"avgSize":        humanize.IBytes(uint64(avgSize)),

			// Expiring content (safe as Count returns 0 if no rows)
			"expiringPastes24h": expiringPastes,
			"expiringUrls24h":   expiringUrls,

			// Additional metrics (with safe defaults)
			"privatePastes": privatePastes,
			"publicPastes":  publicPastes,
			"privateRatio":  privateRatio,
			"avgViews":      avgViews,
		},
		"baseUrl": s.config.Server.BaseURL,
	}, "layouts/main")
}

// handleDocs serves the API documentation page
// Shows API endpoints, usage examples, and system limits
func (s *Server) handleDocs(c *fiber.Ctx) error {
	// Generate retention data with config
	retentionStats, err := utils.GenerateRetentionData(int64(s.config.Server.MaxUploadSize), s.config)
	if err != nil {
		s.logger.Error("failed to generate retention data", zap.Error(err))
		// Continue with empty retention data
	}

	return c.Render("docs", fiber.Map{
		"baseUrl": s.config.Server.BaseURL,
		"maxSize": humanize.IBytes(uint64(s.config.Server.MaxUploadSize)),
		"retention": fiber.Map{
			"noKey":   retentionStats.NoKeyRange,
			"withKey": retentionStats.WithKeyRange,
		},
		"apiKeyEnabled": s.hasMailer(),
	}, "layouts/main")
}

// Paste Creation Handlers

// handleUpload is a unified entry point for all upload types
// Automatically routes to the appropriate handler based on Content-Type and request format
// Supports:
// - multipart/form-data (file uploads)
// - application/json (JSON payload with content or URL)
// - any other Content-Type (treated as raw content)
func (s *Server) handleUpload(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	// Get content type, removing any charset suffix
	contentType := strings.Split(c.Get("Content-Type"), ";")[0]

	switch contentType {
	case "multipart/form-data":
		// Check if we have a file in the form
		if _, err := c.FormFile("file"); err == nil {
			return s.handleMultipartUpload(c)
		}
		return fiber.NewError(fiber.StatusBadRequest, "No file provided in multipart form")

	case "application/json":
		// Verify we have a JSON body
		if len(c.Body()) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Empty JSON body")
		}
		return s.handleJSONUpload(c)

	default:
		// Treat everything else as raw content
		if len(c.Body()) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Empty content")
		}
		return s.handleRawUpload(c)
	}
}

// handleMultipartUpload processes multipart form file uploads
func (s *Server) handleMultipartUpload(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "No file provided")
	}

	// Read file content
	f, err := file.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read uploaded file")
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read upload")
	}

	req := &UploadRequest{
		Content:   content,
		Filename:  c.Query("filename", file.Filename),
		Extension: c.Query("ext"),
		ExpiresIn: c.Query("expires"),
		Private:   c.QueryBool("private", false),
	}

	paste, err := s.processUpload(c, req)
	if err != nil {
		return err
	}

	response := paste.ToResponse()
	s.addBaseURLToPasteResponse(response)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// handleRawUpload handles raw body uploads
func (s *Server) handleRawUpload(c *fiber.Ctx) error {
	content := c.Body()
	if len(content) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Empty content")
	}

	req := &UploadRequest{
		Content:     content,
		Filename:    c.Query("filename", "paste"),
		Extension:   c.Query("ext"),
		ExpiresIn:   c.Query("expires"),
		Private:     c.QueryBool("private", false),
		ContentType: c.Get("Content-Type"),
	}

	paste, err := s.processUpload(c, req)
	if err != nil {
		return err
	}

	response := paste.ToResponse()
	s.addBaseURLToPasteResponse(response)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// handleJSONUpload handles JSON payload uploads
func (s *Server) handleJSONUpload(c *fiber.Ctx) error {
	var jsonReq struct {
		Content   string `json:"content"`
		URL       string `json:"url"`
		Filename  string `json:"filename"`
		Extension string `json:"extension"`
		ExpiresIn string `json:"expires_in"`
		Private   bool   `json:"private"`
	}

	if err := c.BodyParser(&jsonReq); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid JSON")
	}

	req := &UploadRequest{
		Content:   []byte(jsonReq.Content),
		URL:       jsonReq.URL,
		Filename:  jsonReq.Filename,
		Extension: jsonReq.Extension,
		ExpiresIn: jsonReq.ExpiresIn,
		Private:   jsonReq.Private,
	}

	paste, err := s.processUpload(c, req)
	if err != nil {
		return err
	}

	response := paste.ToResponse()
	s.addBaseURLToPasteResponse(response)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// URL Shortener Handlers

// handleURLShorten creates a new shortened URL (requires API key)
// Accepts: application/json with structure:
//
//	{
//	  "url": "string",       // Required
//	  "title": "string",     // Optional
//	  "expires_in": "string" // Optional
//	}
func (s *Server) handleURLShorten(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	apiKey := c.Locals("apiKey").(*models.APIKey)
	if !apiKey.AllowShortlinks {
		return fiber.NewError(fiber.StatusForbidden, "API key does not allow URL shortening")
	}

	// Get content type, removing any charset suffix
	contentType := strings.Split(c.Get("Content-Type"), ";")[0]
	if contentType != "application/json" {
		return fiber.NewError(
			fiber.StatusBadRequest,
			"Content-Type must be application/json",
		)
	}

	var req struct {
		URL       string `json:"url"`
		Title     string `json:"title"`
		ExpiresIn string `json:"expires_in"`
	}

	if err := c.BodyParser(&req); err != nil {
		s.logger.Error("failed to parse request body",
			zap.Error(err),
			zap.String("body", string(c.Body())))
		return fiber.NewError(fiber.StatusBadRequest, "Invalid JSON")
	}

	s.logger.Info("received shortlink request",
		zap.String("url", req.URL),
		zap.String("title", req.Title),
		zap.String("expires_in", req.ExpiresIn))

	shortlink, err := s.createShortlink(req.URL, &ShortlinkOptions{
		Title:     req.Title,
		ExpiresIn: req.ExpiresIn,
		APIKey:    apiKey,
	})
	if err != nil {
		s.logger.Error("failed to create shortlink",
			zap.Error(err),
			zap.String("url", req.URL))
		return err
	}

	s.logger.Info("created shortlink",
		zap.String("id", shortlink.ID),
		zap.String("target_url", shortlink.TargetURL))

	response := shortlink.ToResponse()
	s.addBaseURLToPasteResponse(response)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// handleURLStats returns statistics for a shortened URL (requires API key)
// Returns: view count, last viewed time, and other metadata
func (s *Server) handleURLStats(c *fiber.Ctx) error {
	apiKey := c.Locals("apiKey").(*models.APIKey)
	id := c.Params("id")

	shortlink, err := s.findShortlink(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Shortlink not found")
	}

	// Check if API key owns this shortlink
	if shortlink.APIKey != apiKey.Key {
		return fiber.NewError(fiber.StatusForbidden, "Not authorized to view these stats")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":         shortlink.ID,
			"url":        shortlink.TargetURL,
			"title":      shortlink.Title,
			"clicks":     shortlink.Clicks,
			"created_at": shortlink.CreatedAt,
			"last_click": shortlink.LastClick,
			"expires_at": shortlink.ExpiresAt,
		},
	})
}

// Paste Management Handlers

// handleListPastes returns a paginated list of pastes for the API key
// Optional query params:
//   - page: page number (default: 1)
//   - limit: items per page (default: 20)
//   - sort: sort order (default: "created_at desc")
func (s *Server) handleListPastes(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	apiKey := c.Locals("apiKey").(*models.APIKey)

	// Get pagination params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	sort := c.Query("sort", "created_at desc")

	var pastes []models.Paste
	var total int64

	// Build query
	query := s.db.Model(&models.Paste{}).Where("api_key = ?", apiKey.Key)

	// Get total count
	query.Count(&total)

	// Get paginated results
	err := query.Order(sort).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&pastes).Error

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch pastes")
	}

	// Convert to response format
	items := []fiber.Map{}
	for _, paste := range pastes {
		response := paste.ToResponse()
		s.addBaseURLToPasteResponse(response)
		items = append(items, response)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"items": items,
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// handleDeletePaste deletes a paste (requires API key ownership)
// Verifies API key ownership before deletion
// Removes both storage content and database record
func (s *Server) handleDeletePaste(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	id := c.Params("id")
	apiKey := c.Locals("apiKey").(*models.APIKey)

	// Find paste
	paste, err := s.findPaste(id)
	if err != nil {
		return err
	}

	// Check ownership
	if paste.APIKey != apiKey.Key {
		return fiber.NewError(fiber.StatusForbidden, "Not authorized to delete this paste")
	}

	// Delete from storage first
	store, err := s.storage.GetStore(paste.StorageName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get storage")
	}

	if err := store.Delete(paste.StoragePath); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete paste content")
	}

	// Delete from database
	if err := s.db.Delete(paste).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete paste record")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Paste deleted successfully",
	})
}

// handleUpdateExpiration updates a paste's expiration time
// Accepts: application/json with structure:
//
//	{
//	  "expires_in": "string" // Required (e.g., "24h", or "never")
//	}
func (s *Server) handleUpdateExpiration(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	id := c.Params("id")
	apiKey := c.Locals("apiKey").(*models.APIKey)

	var req struct {
		ExpiresIn string `json:"expires_in"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid JSON")
	}

	// Find paste
	paste, err := s.findPaste(id)
	if err != nil {
		return err
	}

	// Check ownership
	if paste.APIKey != apiKey.Key {
		return fiber.NewError(fiber.StatusForbidden, "Not authorized to modify this paste")
	}

	// Update expiration
	if req.ExpiresIn == "never" {
		paste.ExpiresAt = nil
	} else {
		expiry, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid expiration format")
		}
		expiryTime := time.Now().Add(expiry)
		paste.ExpiresAt = &expiryTime
	}

	// Save changes
	if err := s.db.Save(paste).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update paste")
	}

	response := paste.ToResponse()
	s.addBaseURLToPasteResponse(response)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// handleRequestAPIKey handles the initial API key request
func (s *Server) handleRequestAPIKey(c *fiber.Ctx) error {
	if !s.hasMailer() {
		return fiber.NewError(
			fiber.StatusServiceUnavailable,
			"Email verification is not available. Please contact the administrator.",
		)
	}

	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	var req struct {
		Email string `json:"email" validate:"required,email"`
		Name  string `json:"name" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Check if email already has a verified key
	var existingKey models.APIKey
	err := s.db.Where("email = ? AND verified = ?", req.Email, true).
		First(&existingKey).Error

	// If user exists, create a temporary verification for key reset
	if err == nil {
		// Create temporary verification record
		tempKey := models.NewAPIKey()
		tempKey.Email = req.Email
		tempKey.Name = req.Name
		tempKey.VerifyToken = utils.MustGenerateID(64)
		tempKey.VerifyExpiry = time.Now().Add(24 * time.Hour)
		tempKey.IsReset = true

		if err := s.db.Create(tempKey).Error; err != nil {
			s.logger.Error("failed to create temporary verification",
				zap.String("email", req.Email),
				zap.Error(err))
			return fiber.NewError(
				fiber.StatusInternalServerError,
				"Failed to process request",
			)
		}

		// Send verification email
		if err := s.mailer.SendVerification(req.Email, tempKey.VerifyToken); err != nil {
			s.logger.Error("failed to send verification email",
				zap.String("email", req.Email),
				zap.Error(err))
			s.db.Delete(tempKey)
			return fiber.NewError(
				fiber.StatusInternalServerError,
				"Failed to send verification email",
			)
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Please check your email to verify your key reset request",
		})
	}

	// Create API key with verification token
	apiKey := models.NewAPIKey()
	apiKey.Email = req.Email
	apiKey.Name = req.Name
	apiKey.VerifyToken = utils.MustGenerateID(64)
	apiKey.VerifyExpiry = time.Now().Add(24 * time.Hour)

	if err := s.db.Create(apiKey).Error; err != nil {
		s.logger.Error("failed to create API key",
			zap.String("email", req.Email),
			zap.Error(err))
		return fiber.NewError(
			fiber.StatusInternalServerError,
			"Failed to create API key",
		)
	}

	// Send verification email
	if err := s.mailer.SendVerification(req.Email, apiKey.VerifyToken); err != nil {
		s.logger.Error("failed to send verification email",
			zap.String("email", req.Email),
			zap.Error(err))

		// Delete the API key if we couldn't send the email
		s.db.Delete(apiKey)

		return fiber.NewError(
			fiber.StatusInternalServerError,
			"Failed to send verification email",
		)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Please check your email to verify your API key",
	})
}

// handleVerifyAPIKey verifies the email and activates the API key
func (s *Server) handleVerifyAPIKey(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	token := c.Params("token")

	var tempKey models.APIKey
	err := s.db.Where("verify_token = ? AND verify_expiry > ? AND verified = ?",
		token, time.Now(), false).First(&tempKey).Error

	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Invalid or expired verification token")
	}

	// Handle key reset case
	if tempKey.IsReset {
		// Find the existing verified key
		var existingKey models.APIKey
		err := s.db.Where("email = ? AND verified = ? AND is_reset = ?",
			tempKey.Email, true, false).First(&existingKey).Error
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to process key reset")
		}

		// Update existing key with new credentials
		existingKey.Key = models.GenerateAPIKey()
		if err := s.db.Save(&existingKey).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to reset API key")
		}

		// Delete the temporary verification record
		s.db.Delete(&tempKey)

		return c.Render("verify_success", fiber.Map{
			"apiKey":  existingKey.Key,
			"baseUrl": s.config.Server.BaseURL,
			"reset":   true,
		}, "layouts/main")
	}

	// Activate the key
	tempKey.Verified = true
	tempKey.VerifyToken = ""
	if err := s.db.Save(&tempKey).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify API key")
	}

	return c.Render("verify_success", fiber.Map{
		"apiKey":  tempKey.Key,
		"baseUrl": s.config.Server.BaseURL,
	}, "layouts/main")
}

// Public Access Handlers

// handleView serves the content with syntax highlighting if applicable
// For URLs, redirects to target URL and increments view counter
// For text content, renders with syntax highlighting
// For other content types, redirects to download handler
func (s *Server) handleView(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	id := getPasteID(c)
	hasExtension := c.Params("ext") != ""

	// Only check for shortlink if there's no extension
	if !hasExtension {
		if shortlink, err := s.findShortlink(id); err == nil {
			// Log initial state
			s.logger.Info("shortlink found",
				zap.String("id", id),
				zap.String("original_url", shortlink.TargetURL))

			// Update click stats asynchronously
			go s.updateShortlinkStats(shortlink)

			// Clean and validate the URL
			targetURL := strings.TrimSpace(shortlink.TargetURL)
			s.logger.Info("cleaned url",
				zap.String("id", id),
				zap.String("cleaned_url", targetURL))

			// Ensure URL has a protocol
			if !strings.Contains(targetURL, "://") {
				targetURL = "https://" + targetURL
				s.logger.Info("added protocol",
					zap.String("id", id),
					zap.String("final_url", targetURL))
			}

			// Log final redirect attempt
			s.logger.Info("attempting redirect",
				zap.String("id", id),
				zap.String("redirect_url", targetURL))

			return c.Redirect(targetURL, fiber.StatusFound)
		}
	}

	// Try paste
	paste, err := s.findPaste(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Not found")
	}

	c.Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	// Handle view based on content type
	switch {
	case isTextContent(paste.MimeType):
		return s.renderPasteView(c, paste)
	case isImageContent(paste.MimeType):
		return s.renderRawContent(c, paste)
	default:
		return c.Redirect("/download/"+id, fiber.StatusTemporaryRedirect)
	}
}

// renderPasteView renders the paste view for text content
// Includes syntax highlighting using Chroma
// Supports language detection and line numbering
func (s *Server) renderPasteView(c *fiber.Ctx, paste *models.Paste) error {
	// Get content from storage
	store, err := s.storage.GetStore(paste.StorageName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Storage not found")
	}

	content, err := store.Get(paste.StoragePath)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read content")
	}
	defer content.Close()

	// Read all content
	data, err := io.ReadAll(content)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read content")
	}

	// Get lexer based on extension or mime type
	lexer := lexers.Get(paste.Extension)
	if lexer == nil {
		// Try to match by filename
		lexer = lexers.Match(paste.Filename)
		if lexer == nil {
			// Try to analyze content
			lexer = lexers.Analyse(string(data))
			if lexer == nil {
				lexer = lexers.Fallback
			}
		}
	}
	lexer = chroma.Coalesce(lexer)

	// Create formatter without classes (will use inline styles)
	formatter := html.New(
		html.WithLineNumbers(true),
		html.WithLinkableLineNumbers(true, ""),
		html.TabWidth(4),
	)

	// Use gruvbox style (dark theme that matches our UI)
	style := styles.Get("gruvbox")
	if style == nil {
		style = styles.Fallback
	}

	// Generate highlighted HTML
	var highlightedContent strings.Builder
	iterator, err := lexer.Tokenise(nil, string(data))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to tokenize content")
	}

	err = formatter.Format(&highlightedContent, style, iterator)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to format content")
	}

	return c.Render("paste", fiber.Map{
		"id":       paste.ID,
		"filename": paste.Filename,
		"content":  highlightedContent.String(),
		"language": lexer.Config().Name,
		"created":  paste.CreatedAt.Format(time.RFC3339),
		"expires":  paste.ExpiresAt,
		"baseUrl":  s.config.Server.BaseURL,
	}, "layouts/main")
}

// renderRawContent serves the raw content with proper content type
// Used for displaying images and other browser-viewable content
func (s *Server) renderRawContent(c *fiber.Ctx, paste *models.Paste) error {
	// Get the correct store for this paste
	store, err := s.storage.GetStore(paste.StorageName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Storage not found")
	}

	// Get content from storage
	content, err := store.Get(paste.StoragePath)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read content")
	}
	defer content.Close()

	// Set appropriate headers
	c.Set("Content-Type", paste.MimeType)
	c.Set("Content-Length", fmt.Sprintf("%d", paste.Size))

	// Read and send the content
	data, err := io.ReadAll(content)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read content")
	}

	return c.Send(data)
}

// handleRawView serves the raw content of a paste
// Sets appropriate content type and cache headers
// For text content, forces text/plain content type
func (s *Server) handleRawView(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	id := getPasteID(c)

	paste, err := s.findPaste(id)
	if err != nil {
		return err
	}

	// Get the correct store for this paste
	store, err := s.storage.GetStore(paste.StorageName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Storage not found")
	}

	// Get content from storage
	content, err := store.Get(paste.StoragePath)
	if err != nil {
		s.logger.Error("failed to read content from storage",
			zap.String("id", id),
			zap.String("storage", paste.StorageName),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read content")
	}
	defer content.Close()

	// For text content, use text/plain to display in browser
	contentType := paste.MimeType
	if isTextContent(paste.MimeType) {
		contentType = "text/plain; charset=utf-8"
	}

	// Set content type and cache headers
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", fmt.Sprintf("%d", paste.Size))
	c.Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	// Read all content first
	data, err := io.ReadAll(content)
	if err != nil {
		s.logger.Error("failed to read content",
			zap.String("id", id),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read content")
	}

	return c.Send(data)
}

// handleDownload serves the content as a downloadable file
// Sets Content-Disposition header for download
// Includes original filename in download prompt
func (s *Server) handleDownload(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	id := getPasteID(c)

	paste, err := s.findPaste(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Not found")
	}

	// Get the correct store for this paste
	store, err := s.storage.GetStore(paste.StorageName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Storage not found")
	}

	// Get content from storage
	content, err := store.Get(paste.StoragePath)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read content")
	}
	defer content.Close()

	// Read all content first
	data, err := io.ReadAll(content)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to read content")
	}

	// Set download headers
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Length", fmt.Sprintf("%d", paste.Size))
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, paste.Filename))
	c.Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	return c.Send(data)
}

// handleDeleteWithKey deletes a paste using its deletion key
// No authentication required, but deletion key must match
// Removes both storage content and database record
func (s *Server) handleDeleteWithKey(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	id := c.Params("id")
	key := c.Params("key")

	paste, err := s.findPaste(id)
	if err != nil {
		return err
	}

	if paste.DeleteKey != key {
		return fiber.NewError(fiber.StatusForbidden, "Invalid delete key")
	}

	// Get the correct store for this paste
	store, err := s.storage.GetStore(paste.StorageName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Storage not found")
	}

	// Delete from storage first
	if err := store.Delete(paste.StoragePath); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete paste content")
	}

	// Delete from database
	if err := s.db.Delete(paste).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete paste record")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Paste deleted successfully",
	})
}

// handleListURLs returns a paginated list of URLs for the API key
// Optional query params:
//   - page: page number (default: 1)
//   - limit: items per page (default: 20)
//   - sort: sort order (default: "created_at desc")
func (s *Server) handleListURLs(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	apiKey := c.Locals("apiKey").(*models.APIKey)

	// Get pagination params
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	sort := c.Query("sort", "created_at desc")

	var urls []models.Shortlink
	var total int64

	// Build query
	query := s.db.Model(&models.Shortlink{}).Where("api_key = ?", apiKey.Key)

	// Get total count
	query.Count(&total)

	// Get paginated results
	err := query.Order(sort).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&urls).Error

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch URLs")
	}

	// Convert to response format
	items := []fiber.Map{}
	for _, url := range urls {
		response := url.ToResponse()
		s.addBaseURLToPasteResponse(response)
		items = append(items, response)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"items": items,
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// handleUpdateURLExpiration updates a URL's expiration time
// Accepts: application/json with structure:
//
//	{
//	  "expires_in": "string" // Required (e.g., "24h", or "never")
//	}
func (s *Server) handleUpdateURLExpiration(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	id := c.Params("id")
	apiKey := c.Locals("apiKey").(*models.APIKey)

	var req struct {
		ExpiresIn string `json:"expires_in"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid JSON")
	}

	// Find URL
	var shortlink models.Shortlink
	if err := s.db.First(&shortlink, "id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "URL not found")
	}

	// Check ownership
	if shortlink.APIKey != apiKey.Key {
		return fiber.NewError(fiber.StatusForbidden, "Not authorized to modify this URL")
	}

	// Update expiration
	if req.ExpiresIn == "never" {
		shortlink.ExpiresAt = nil
	} else {
		expiry, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid expiration format")
		}
		expiryTime := time.Now().Add(expiry)
		shortlink.ExpiresAt = &expiryTime
	}

	// Save changes
	if err := s.db.Save(&shortlink).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update URL")
	}

	response := shortlink.ToResponse()
	s.addBaseURLToPasteResponse(response)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// handleDeleteURL deletes a URL (requires API key ownership)
func (s *Server) handleDeleteURL(c *fiber.Ctx) error {
	if err := s.rateLimiter.Check(c.IP()); err != nil {
		return err
	}

	id := c.Params("id")
	apiKey := c.Locals("apiKey").(*models.APIKey)

	// Find URL
	var shortlink models.Shortlink
	if err := s.db.First(&shortlink, "id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "URL not found")
	}

	// Check ownership
	if shortlink.APIKey != apiKey.Key {
		return fiber.NewError(fiber.StatusForbidden, "Not authorized to delete this URL")
	}

	// Delete from database
	if err := s.db.Delete(&shortlink).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete URL")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "URL deleted successfully",
	})
}

// Helper Functions

// getStorageSize returns total size of stored files in bytes
// Calculated as sum of all paste sizes in database
func (s *Server) getStorageSize() (uint64, error) {
	var total uint64
	err := s.db.Model(&models.Paste{}).
		Select("COALESCE(SUM(size), 0)").
		Row().
		Scan(&total)
	return total, err
}

// addBaseURLToPasteResponse adds the configured base URL to all URL fields
// Modifies the response map in place, appending base URL to *_url fields
func (s *Server) addBaseURLToPasteResponse(response fiber.Map) {
	baseURL := strings.TrimSuffix(s.config.Server.BaseURL, "/")
	for key, value := range response {
		if strValue, ok := value.(string); ok {
			if strings.HasSuffix(key, "url") {
				// Skip if the URL already has a protocol
				if !strings.Contains(strValue, "://") {
					response[key] = baseURL + strValue
				}
			}
		}
	}
}

// Helper function to extract paste ID from params
func getPasteID(c *fiber.Ctx) string {
	id := c.Params("id")
	// If the ID includes an extension, remove it
	if ext := c.Params("ext"); ext != "" {
		return strings.TrimSuffix(id, "."+ext)
	}
	return id
}
