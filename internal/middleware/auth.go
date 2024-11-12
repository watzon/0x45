package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/models"
	"gorm.io/gorm"
)

type AuthMiddleware struct {
	db *gorm.DB
}

func NewAuthMiddleware(db *gorm.DB) *AuthMiddleware {
	return &AuthMiddleware{db: db}
}

// Auth returns a middleware that validates API keys
func (m *AuthMiddleware) Auth(required bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			if required {
				return fiber.NewError(fiber.StatusUnauthorized, "API key required")
			}
			return c.Next()
		}

		apiKey := strings.TrimPrefix(auth, "Bearer ")

		// Validate API key and set rate limits
		key, err := m.validateAPIKey(apiKey)
		if err != nil {
			if required {
				return fiber.NewError(fiber.StatusUnauthorized, "Invalid API key")
			}
			return c.Next()
		}

		// Check rate limit
		if err := m.checkRateLimit(key); err != nil {
			return fiber.NewError(fiber.StatusTooManyRequests, "Rate limit exceeded")
		}

		// Store API key in context
		c.Locals("apiKey", key)
		return c.Next()
	}
}

func (m *AuthMiddleware) validateAPIKey(key string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := m.db.Where("key = ?", key).First(&apiKey).Error
	if err != nil {
		return nil, err
	}

	// Update last used timestamp
	m.db.Model(&apiKey).Updates(map[string]interface{}{
		"last_used_at": time.Now(),
		"usage_count":  gorm.Expr("usage_count + 1"),
	})

	return &apiKey, nil
}

func (m *AuthMiddleware) checkRateLimit(key *models.APIKey) error {
	// Get usage count in the last hour
	var count int64
	err := m.db.Model(&models.APIKey{}).
		Where("key = ? AND last_used_at > ?", key.Key, time.Now().Add(-time.Hour)).
		Count(&count).Error
	if err != nil {
		return err
	}

	if count >= int64(key.RateLimit) {
		return fiber.NewError(fiber.StatusTooManyRequests, "Rate limit exceeded")
	}

	return nil
}
