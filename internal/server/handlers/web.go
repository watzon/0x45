package handlers

import (
	"encoding/json"
	"fmt"
	"regexp"

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

var httpRe = regexp.MustCompile(`^https?://`)

func (h *WebHandlers) getBaseURL() string {
	return httpRe.ReplaceAllString(h.config.Server.BaseURL, "")
}

// HandleIndex serves the main web interface page
func (h *WebHandlers) HandleIndex(c *fiber.Ctx) error {
	h.logger.Debug("generating retention data for index page")
	retentionStats, err := utils.GenerateRetentionData(int64(h.config.Server.MaxUploadSize), h.config)
	if err != nil {
		h.logger.Error("failed to generate retention data", zap.Error(err))
	}

	h.logger.Debug("marshaling retention history data")
	noKeyHistory, err := json.Marshal(retentionStats.Data["noKey"])
	if err != nil {
		h.logger.Error("failed to marshal noKey history", zap.Error(err))
		return err
	}

	withKeyHistory, err := json.Marshal(retentionStats.Data["withKey"])
	if err != nil {
		h.logger.Error("failed to marshal withKey history", zap.Error(err))
		return err
	}

	h.logger.Debug("preparing template data",
		zap.String("baseUrl", h.getBaseURL()),
		zap.Any("retention", retentionStats))

	err = c.Render("index", fiber.Map{
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
		"baseUrl": h.getBaseURL(),
	}, "layouts/main")

	if err != nil {
		h.logger.Error("failed to render index template",
			zap.Error(err),
			zap.String("template", "index"),
			zap.String("layout", "layouts/main"))
		return err
	}

	return nil
}

// HandleStats serves the statistics page
func (h *WebHandlers) HandleStats(c *fiber.Ctx) error {
	stats, err := h.services.Stats.GetSystemStats()
	if err != nil {
		return err
	}

	return c.Render("stats", fiber.Map{
		"stats":   stats,
		"baseUrl": h.getBaseURL(),
	}, "layouts/main")
}

// HandleDocs serves the API documentation page
func (h *WebHandlers) HandleDocs(c *fiber.Ctx) error {
	retentionStats, err := utils.GenerateRetentionData(int64(h.config.Server.MaxUploadSize), h.config)
	if err != nil {
		h.logger.Error("failed to generate retention data", zap.Error(err))
	}

	return c.Render("docs", fiber.Map{
		"baseUrl":        h.getBaseURL(),
		"apiKeysEnabled": h.services.APIKey.IsEnabled(),
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
		"baseUrl": h.getBaseURL(),
	}, "layouts/main")
}
