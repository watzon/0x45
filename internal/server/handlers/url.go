package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/server/services"
	"go.uber.org/zap"
)

type URLHandlers struct {
	services *services.Services
	logger   *zap.Logger
	config   *config.Config
}

func NewURLHandlers(services *services.Services, logger *zap.Logger, config *config.Config) *URLHandlers {
	return &URLHandlers{
		services: services,
		logger:   logger,
		config:   config,
	}
}

// HandleURLShorten creates a new shortened URL
func (h *URLHandlers) HandleURLShorten(c *fiber.Ctx) error {
	// debug the incoming json body
	return h.services.URL.CreateShortlink(c)
}

// HandleURLStats returns statistics for a shortened URL
func (h *URLHandlers) HandleURLStats(c *fiber.Ctx) error {
	return h.services.URL.GetStats(c)
}

// HandleListURLs returns a paginated list of URLs for the API key
func (h *URLHandlers) HandleListURLs(c *fiber.Ctx) error {
	return h.services.URL.ListURLs(c)
}

// HandleUpdateURLExpiration updates a URL's expiration time
func (h *URLHandlers) HandleUpdateURLExpiration(c *fiber.Ctx) error {
	return h.services.URL.UpdateExpiration(c)
}

// HandleDeleteURL deletes a URL (requires API key ownership)
func (h *URLHandlers) HandleDeleteURL(c *fiber.Ctx) error {
	return h.services.URL.Delete(c)
}

// HandleRedirect redirects to the target URL
func (h *URLHandlers) HandleRedirect(c *fiber.Ctx) error {
	id := c.Params("id")
	shortlink, err := h.services.URL.FindShortlink(id)
	if err != nil {
		return err
	}

	// Log the click
	if err := h.services.Analytics.LogShortlinkClick(c, shortlink.ID); err != nil {
		h.logger.Error("failed to log shortlink click", zap.Error(err))
	}

	return c.Redirect(shortlink.TargetURL, fiber.StatusTemporaryRedirect)
}
