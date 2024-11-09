package server

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/handlebars/v2"
	"github.com/watzon/paste69/internal/config"
	"github.com/watzon/paste69/internal/database"
	"github.com/watzon/paste69/internal/middleware"
	"github.com/watzon/paste69/internal/storage"
	"go.uber.org/zap"
)

type Server struct {
	app         *fiber.App
	db          *database.Database
	auth        *middleware.AuthMiddleware
	store       storage.Store
	config      *config.Config
	rateLimiter *RateLimiter
	logger      *zap.Logger
}

// RateLimiter handles rate limiting for API endpoints
type RateLimiter struct {
	limits map[string]time.Time
	mu     sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]time.Time),
	}
}

func (r *RateLimiter) Allow(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if lastTime, exists := r.limits[key]; exists {
		if time.Since(lastTime) < time.Minute {
			return fiber.NewError(fiber.StatusTooManyRequests, "Rate limit exceeded")
		}
	}
	r.limits[key] = time.Now()
	return nil
}

func New(db *database.Database, store storage.Store, config *config.Config) *Server {
	// Initialize template engine
	engine := handlebars.New("./views", ".hbs")

	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
		BodyLimit:    config.Server.MaxUploadSize,
		Views:        engine,
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

	return &Server{
		app:         app,
		db:          db,
		auth:        middleware.NewAuthMiddleware(db.DB),
		store:       store,
		config:      config,
		rateLimiter: NewRateLimiter(),
		logger:      logger,
	}
}

func (s *Server) SetupRoutes() {
	// Public routes
	s.app.Get("/", s.handleIndex)
	s.app.Get("/docs", s.handleDocs)
	s.app.Get("/:id", s.handleView)
	s.app.Get("/raw/:id", s.handleRawView)
	s.app.Get("/download/:id", s.handleDownload)
	s.app.Delete("/delete/:id.:key", s.handleDeleteWithKey)

	// Paste creation routes
	s.app.Post("/", s.auth.Auth(false), s.handleUpload)

	// URL shortener routes (requires API key)
	s.app.Post("/url", s.auth.Auth(true), s.handleURLShorten)
	s.app.Get("/url/:id/stats", s.auth.Auth(true), s.handleURLStats)

	// Management routes (requires API key)
	s.app.Get("/pastes", s.auth.Auth(true), s.handleListPastes)
	s.app.Delete("/pastes/:id", s.auth.Auth(true), s.handleDeletePaste)
	s.app.Put("/pastes/:id/expire", s.auth.Auth(true), s.handleUpdateExpiration)
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
	return s.app.Listen(addr)
}

func (s *Server) Cleanup() {
	if s.logger != nil {
		s.logger.Sync() // flush any buffered log entries
	}
}
