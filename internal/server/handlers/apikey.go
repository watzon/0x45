package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/server/services"
	"go.uber.org/zap"
)

type APIKeyHandlers struct {
	services *services.Services
	logger   *zap.Logger
	config   *config.Config
}

func NewAPIKeyHandlers(services *services.Services, logger *zap.Logger, config *config.Config) *APIKeyHandlers {
	return &APIKeyHandlers{
		services: services,
		logger:   logger,
		config:   config,
	}
}

// HandleRequestAPIKey handles the initial API key request
func (h *APIKeyHandlers) HandleRequestAPIKey(c *fiber.Ctx) error {
	return h.services.APIKey.RequestKey(c)
}

// HandleVerifyAPIKey verifies the email and activates the API key
func (h *APIKeyHandlers) HandleVerifyAPIKey(c *fiber.Ctx) error {
	return h.services.APIKey.VerifyKey(c)
}
