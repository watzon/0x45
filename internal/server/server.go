package server

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/handlebars/v2"
	"github.com/redis/go-redis/v9"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/database"
	"github.com/watzon/0x45/internal/mailer"
	"github.com/watzon/0x45/internal/middleware"
	"github.com/watzon/0x45/internal/ratelimit"
	"github.com/watzon/0x45/internal/storage"
	"go.uber.org/zap"
)

type Server struct {
	app         *fiber.App
	db          *database.Database
	auth        *middleware.AuthMiddleware
	storage     *storage.StorageManager
	config      *config.Config
	rateLimiter *ratelimit.RateLimiter
	logger      *zap.Logger
	mailer      *mailer.Mailer
}

func New(db *database.Database, storageManager *storage.StorageManager, config *config.Config) *Server {
	// Initialize template engine
	engine := handlebars.New("./views", ".hbs")

	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
		BodyLimit:    config.Server.MaxUploadSize,
		Views:        engine,
		Prefork:      config.Server.Prefork,
		ServerHeader: config.Server.ServerHeader,
		AppName:      config.Server.AppName,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Serve static files
	app.Static("/public", "./public")

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil
	}

	// Initialize mailer if enabled
	var mailClient *mailer.Mailer
	if config.SMTP.Enabled {
		mailClient, err = mailer.New(config)
		if err != nil {
			logger.Error("failed to initialize mailer", zap.Error(err))
			// Continue without mailer
		}
	}

	// Initialize Redis if enabled
	var redisClient *redis.Client
	if config.Redis.Enabled {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     config.Redis.Address,
			Password: config.Redis.Password,
			DB:       config.Redis.DB,
		})

		// Test Redis connection
		if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
			logger.Error("failed to connect to Redis", zap.Error(err))
			return nil
		}
	}

	// Initialize rate limiter
	rateLimiterConfig := ratelimit.Config{
		Global: struct {
			Enabled bool
			Rate    float64
			Burst   int
		}{
			Enabled: config.Server.RateLimit.Global.Enabled,
			Rate:    config.Server.RateLimit.Global.Rate,
			Burst:   config.Server.RateLimit.Global.Burst,
		},
		PerIP: struct {
			Enabled bool
			Rate    float64
			Burst   int
		}{
			Enabled: config.Server.RateLimit.PerIP.Enabled,
			Rate:    config.Server.RateLimit.PerIP.Rate,
			Burst:   config.Server.RateLimit.PerIP.Burst,
		},
		Redis:    redisClient, // Will be nil if Redis is not enabled
		UseRedis: config.Server.RateLimit.UseRedis,
	}

	server := &Server{
		app:         app,
		db:          db,
		auth:        middleware.NewAuthMiddleware(db.DB),
		storage:     storageManager,
		config:      config,
		rateLimiter: ratelimit.New(rateLimiterConfig),
		logger:      logger,
		mailer:      mailClient,
	}

	return server
}

// Add a helper method to check if email features are available
func (s *Server) hasMailer() bool {
	return s.config.SMTP.Enabled && s.mailer != nil
}

func (s *Server) SetupRoutes() {
	// URL shortener routes (requires API key)
	s.app.Post("/url", s.auth.Auth(true), s.handleURLShorten)
	s.app.Get("/url/:id/stats", s.auth.Auth(true), s.handleURLStats)

	// Management routes (requires API key)
	s.app.Get("/pastes", s.auth.Auth(true), s.handleListPastes)
	s.app.Delete("/pastes/:id", s.auth.Auth(true), s.handleDeletePaste)
	s.app.Put("/pastes/:id/expire", s.auth.Auth(true), s.handleUpdateExpiration)

	// API Key management
	s.app.Post("/api-key", s.handleRequestAPIKey)
	s.app.Get("/verify/:token", s.handleVerifyAPIKey)

	// Public routes
	s.app.Get("/", s.handleIndex)
	s.app.Get("/docs", s.handleDocs)
	s.app.Get("/stats", s.handleStats)
	s.app.Post("/", s.auth.Auth(false), s.handleUpload)
	s.app.All("/delete/:id/:key", s.handleDeleteWithKey)
	s.app.Get("/download/:id", s.handleDownload)
	s.app.Get("/raw/:id", s.handleRawView)
	s.app.Get("/:id", s.handleView)
}

// Error handler
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   message,
	})
}

func (s *Server) Start(addr string) error {
	// Start cleanup goroutine if enabled
	if s.config.Server.Cleanup.Enabled {
		go func() {
			ticker := time.NewTicker(time.Duration(s.config.Server.Cleanup.Interval) * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				s.cleanupUnverifiedKeys()
			}
		}()
	}

	return s.app.Listen(addr)
}

func (s *Server) Cleanup() {
	if s.logger != nil {
		s.logger.Sync() // flush any buffered log entries
	}
}
