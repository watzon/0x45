package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/models"
	"github.com/watzon/0x45/internal/server/services"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthMiddleware struct {
	db       *gorm.DB
	logger   *zap.Logger
	config   *config.Config
	services *services.Services
}

func NewAuthMiddleware(db *gorm.DB, logger *zap.Logger, config *config.Config, services *services.Services) *AuthMiddleware {
	return &AuthMiddleware{
		db:       db,
		logger:   logger,
		config:   config,
		services: services,
	}
}

// Auth returns a middleware that validates API keys
func (m *AuthMiddleware) Auth(required bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// First try to get API key from Authorization header
		auth := c.Get("Authorization")
		apiKey := ""

		if strings.HasPrefix(auth, "Bearer ") {
			apiKey = strings.TrimPrefix(auth, "Bearer ")
		} else {
			// If not in header, try to get from query parameter
			apiKey = c.Query("api_key")
		}

		// If no API key found in either place
		if apiKey == "" {
			if required {
				return fiber.NewError(fiber.StatusUnauthorized, "API key required")
			}
			return c.Next()
		}

		// Validate API key and set rate limits
		key, err := m.validateAPIKey(apiKey)
		if err != nil {
			if required {
				return fiber.NewError(fiber.StatusUnauthorized, "Invalid API key")
			}
			return c.Next()
		}

		// Store API key in context
		c.Locals("apiKey", key)
		return c.Next()
	}
}

func (m *AuthMiddleware) validateAPIKey(key string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := m.db.Where("key = ? AND verified = ?", key, true).First(&apiKey).Error
	if err != nil {
		return nil, err
	}

	// if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
	// 	return nil, fiber.NewError(fiber.StatusUnauthorized, "API key has expired")
	// }

	// Update last used timestamp and usage count
	if err := m.db.Model(&apiKey).Updates(map[string]any{
		"last_used_at": time.Now(),
		"usage_count":  gorm.Expr("usage_count + 1"),
	}).Error; err != nil {
		m.logger.Error("failed to update API key usage",
			zap.String("key", key),
			zap.Error(err),
		)
	}

	return &apiKey, nil
}
