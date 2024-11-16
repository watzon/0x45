package middleware

import (
	"runtime/debug"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/server/services"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Middleware holds all middleware instances
type Middleware struct {
	Auth      *AuthMiddleware
	RateLimit *RateLimiter
	db        *gorm.DB
	logger    *zap.Logger
	config    *config.Config
	services  *services.Services
}

// NewMiddleware creates a new Middleware instance with all middleware dependencies
func NewMiddleware(db *gorm.DB, logger *zap.Logger, config *config.Config, services *services.Services) *Middleware {
	return &Middleware{
		Auth:      NewAuthMiddleware(db, logger, config, services),
		RateLimit: NewRateLimiter(logger, config),
		db:        db,
		logger:    logger,
		config:    config,
		services:  services,
	}
}

// Common middleware functions

// Logger returns a middleware that logs request information
func (m *Middleware) Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		status := c.Response().StatusCode()
		m.logger.Info("request completed",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", status),
			zap.Duration("duration", duration),
			zap.String("ip", c.IP()),
		)

		return err
	}
}

// Recover returns a middleware that recovers from panics
func (m *Middleware) Recover() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Error("recovered from panic",
					zap.Any("error", r),
					zap.String("stack", string(debug.Stack())),
				)
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal Server Error",
				})
			}
		}()
		return c.Next()
	}
}

// CORS returns a middleware that handles CORS
func (m *Middleware) CORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     strings.Join(m.config.Server.CORSOrigins, ","),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: false,
		MaxAge:           300,
	})
}

// Compression returns a middleware that compresses responses
func (m *Middleware) Compression() fiber.Handler {
	return compress.New(compress.Config{
		Level: compress.LevelDefault,
	})
}

// RequestID returns a middleware that adds a request ID to each request
func (m *Middleware) RequestID() fiber.Handler {
	return requestid.New()
}

// ETag returns a middleware that adds ETag headers
func (m *Middleware) ETag() fiber.Handler {
	return etag.New()
}

// GetMiddleware returns all middleware handlers in the recommended order
func (m *Middleware) GetMiddleware() []fiber.Handler {
	return []fiber.Handler{
		m.RequestID(),
		m.Logger(),
		m.Recover(),
		m.CORS(),
		m.Compression(),
		m.ETag(),
	}
}
