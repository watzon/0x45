package handlers

import (
	"encoding/json"

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
		"retention": retentionStats,
		"baseUrl":   h.config.Server.BaseURL,
	}, "layouts/main")
}
