package handlers

import (
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/server/services"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handlers holds all handler instances
type Handlers struct {
	Web    *WebHandlers
	APIKey *APIKeyHandlers
	Paste  *PasteHandlers
	URL    *URLHandlers
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config
}

// NewHandlers creates a new Handlers instance with all handler dependencies
func NewHandlers(db *gorm.DB, logger *zap.Logger, config *config.Config, services *services.Services) *Handlers {
	h := &Handlers{
		db:     db,
		logger: logger,
		config: config,
	}

	// Initialize handlers with services first
	h.Web = NewWebHandlers(services, logger, config)
	h.APIKey = NewAPIKeyHandlers(services, logger, config)
	h.Paste = NewPasteHandlers(services, logger, config)
	h.URL = NewURLHandlers(services, logger, config)

	return h
}
