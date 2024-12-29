package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/server/services"
	"github.com/watzon/0x45/internal/utils"
	"go.uber.org/zap"
)

type WebHandlers struct {
	services *services.Services
	logger   *zap.Logger
	config   *config.Config
}

func NewWebHandlers(services *services.Services, logger *zap.Logger, config *config.Config) *WebHandlers {
	return &WebHandlers{
		services: services,
		logger:   logger,
		config:   config,
	}
}

// HandleIndex serves the main web interface page
func (h *WebHandlers) HandleIndex(c *fiber.Ctx) error {
	retentionStats, err := utils.GenerateRetentionData(int64(h.config.Server.MaxUploadSize), h.config)
	if err != nil {
		h.logger.Error("failed to generate retention data", zap.Error(err))
	}

	noKeyHistory, _ := json.Marshal(retentionStats.Data["noKey"])
	withKeyHistory, _ := json.Marshal(retentionStats.Data["withKey"])

	return c.Render("index", fiber.Map{
		"retention": fiber.Map{
			"noKey":          retentionStats.NoKeyRange,
			"withKey":        retentionStats.WithKeyRange,
			"minAge":         h.config.Retention.NoKey.MinAge,
			"maxAge":         h.config.Retention.WithKey.MaxAge,
			"maxSize":        h.config.Server.MaxUploadSize / (1024 * 1024),
			"maxSizeMiB":     humanize.IBytes(uint64(h.config.Server.MaxUploadSize)),
			"noKeyHistory":   string(noKeyHistory),
			"withKeyHistory": string(withKeyHistory),
		},
		"baseUrl": h.config.Server.BaseURL,
	}, "layouts/main")
}

// HandleStats serves the statistics page
func (h *WebHandlers) HandleStats(c *fiber.Ctx) error {
	stats, err := h.services.Stats.GetSystemStats()
	if err != nil {
		return err
	}

	return c.Render("stats", fiber.Map{
		"stats":   stats,
		"baseUrl": h.config.Server.BaseURL,
	}, "layouts/main")
}

// HandleDocs serves the API documentation page
func (h *WebHandlers) HandleDocs(c *fiber.Ctx) error {
	retentionStats, err := utils.GenerateRetentionData(int64(h.config.Server.MaxUploadSize), h.config)
	if err != nil {
		h.logger.Error("failed to generate retention data", zap.Error(err))
	}

	return c.Render("docs", fiber.Map{
		"baseUrl":        h.config.Server.BaseURL,
		"apiKeysEnabled": h.services.APIKey.HasMailer(),
		"retention": fiber.Map{
			"noKey":   retentionStats.NoKeyRange,
			"withKey": retentionStats.WithKeyRange,
			"minAge":  h.config.Retention.NoKey.MinAge,
			"maxAge":  h.config.Retention.WithKey.MaxAge,
		},
		"rateLimits": fiber.Map{
			"global": fmt.Sprintf("%.0f/s", h.config.Server.RateLimit.Global.Rate),
			"perIP":  fmt.Sprintf("%.0f/s", h.config.Server.RateLimit.PerIP.Rate),
		},
		"maxSize": h.config.Server.MaxUploadSize / (1024 * 1024),
	}, "layouts/main")
}

// HandleSubmit serves the paste submission page
func (h *WebHandlers) HandleSubmit(c *fiber.Ctx) error {
	return c.Render("submit", fiber.Map{
		"baseUrl": h.config.Server.BaseURL,
	}, "layouts/main")
}
