package server

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/handlebars/v2"
	"github.com/redis/go-redis/v9"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/database"
	"github.com/watzon/0x45/internal/server/handlers"
	"github.com/watzon/0x45/internal/server/middleware"
	"github.com/watzon/0x45/internal/server/services"
	"github.com/watzon/0x45/internal/storage"
	"go.uber.org/zap"
)

type Server struct {
	app        *fiber.App
	db         *database.Database
	storage    *storage.StorageManager
	config     *config.Config
	logger     *zap.Logger
	services   *services.Services
	handlers   *handlers.Handlers
	middleware *middleware.Middleware
}

func New(db *database.Database, storageManager *storage.StorageManager, config *config.Config) *Server {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil
	}

	// Initialize template engine
	engine := handlebars.New("./views", ".hbs")

	// Initialize services
	svc := services.NewServices(db.DB, logger, config)

	// Initialize middleware
	mw := middleware.NewMiddleware(db.DB, logger, config, svc)

	// Initialize handlers
	hdl := handlers.NewHandlers(db.DB, logger, config, svc)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
		BodyLimit:    config.Server.MaxUploadSize,
		Views:        engine,
		Prefork:      config.Server.Prefork,
		ServerHeader: config.Server.ServerHeader,
		AppName:      config.Server.AppName,
	})

	// Add all middleware in the correct order
	for _, middleware := range mw.GetMiddleware() {
		app.Use(middleware)
	}

	// Serve static files
	app.Static("/public", "./public")

	// Initialize Redis if enabled
	if config.Redis.Enabled {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     config.Redis.Address,
			Password: config.Redis.Password,
			DB:       config.Redis.DB,
		})

		// Test Redis connection
		if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
			logger.Error("failed to connect to Redis", zap.Error(err))
			return nil
		}

		// Set Redis client in rate limiter if using Redis
		// if config.Server.Prefork {
		// 	// TODO: Set Redis client in rate limiter
		// }
	}

	return &Server{
		app:        app,
		db:         db,
		storage:    storageManager,
		config:     config,
		logger:     logger,
		services:   svc,
		handlers:   hdl,
		middleware: mw,
	}
}

// SetupRoutes configures all the routes for the server
func (s *Server) SetupRoutes() {
	// Web interface routes
	s.app.Get("/", s.handlers.Web.HandleIndex)
	s.app.Get("/stats", s.handlers.Web.HandleStats)
	s.app.Get("/docs", s.handlers.Web.HandleDocs)

	// API Key routes
	apiKeys := s.app.Group("/api/keys")
	apiKeys.Post("/request", s.handlers.APIKey.HandleRequestAPIKey)
	apiKeys.Get("/verify", s.handlers.APIKey.HandleVerifyAPIKey)

	// Paste routes
	pastes := s.app.Group("/api/pastes")
	pastes.Use(s.middleware.Auth.Auth(true))
	pastes.Post("/", s.handlers.Paste.HandleUpload)
	pastes.Get("/", s.handlers.Paste.HandleListPastes)
	pastes.Delete("/:id", s.handlers.Paste.HandleDeletePaste)
	pastes.Put("/:id/expiry", s.handlers.Paste.HandleUpdateExpiration)

	// URL routes
	urls := s.app.Group("/api/urls")
	urls.Use(s.middleware.Auth.Auth(true))
	urls.Post("/", s.handlers.URL.HandleURLShorten)
	urls.Get("/", s.handlers.URL.HandleListURLs)
	urls.Get("/:id/stats", s.handlers.URL.HandleURLStats)
	urls.Delete("/:id", s.handlers.URL.HandleDeleteURL)
	urls.Put("/:id/expiry", s.handlers.URL.HandleUpdateURLExpiration)

	// Public routes
	// Handle paste routes with extensions
	s.app.Get("/:id.:ext", func(c *fiber.Ctx) error {
		// Set the extension in locals for the paste handler to use
		c.Locals("extension", c.Params("ext"))
		return s.handlers.Paste.HandleView(c)
	})
	s.app.Get("/:id/raw.:ext", func(c *fiber.Ctx) error {
		c.Locals("extension", c.Params("ext"))
		return s.handlers.Paste.HandleRawView(c)
	})
	s.app.Get("/:id/download.:ext", func(c *fiber.Ctx) error {
		c.Locals("extension", c.Params("ext"))
		return s.handlers.Paste.HandleDownload(c)
	})

	// Handle paste routes without extensions
	s.app.Get("/:id/raw", s.handlers.Paste.HandleRawView)
	s.app.Get("/:id/download", s.handlers.Paste.HandleDownload)
	s.app.Delete("/:id/:key", s.handlers.Paste.HandleDeleteWithKey)

	// Handle base /:id route - try paste first, fallback to URL redirect
	s.app.Get("/:id", func(c *fiber.Ctx) error {
		// Try to get paste first
		if paste, err := s.services.Paste.GetPaste(c.Params("id")); err == nil {
			// Log the view
			if err := s.services.Analytics.LogPasteView(c, paste.ID); err != nil {
				s.logger.Error("failed to log paste view", zap.Error(err))
			}
			return s.handlers.Paste.HandleView(c)
		}

		// If paste not found, try URL redirect
		return s.handlers.URL.HandleRedirect(c)
	})
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
		"error": message,
	})
}

func (s *Server) Start(addr string) error {
	// Start cleanup scheduler
	if s.config.Server.Cleanup.Enabled {
		interval := fmt.Sprintf("%ds", s.config.Server.Cleanup.Interval)
		if err := s.services.StartCleanupScheduler(interval); err != nil {
			s.logger.Error("failed to start cleanup scheduler", zap.Error(err))
		}
	}

	// Setup routes
	s.SetupRoutes()

	// Start server
	return s.app.Listen(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

func (s *Server) Cleanup() error {
	if s.db != nil && s.db.DB != nil {
		return s.db.DB.Error
	}
	return nil
}
