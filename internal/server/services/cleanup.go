package services

import (
	"time"

	"github.com/watzon/0x45/internal/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CleanupService struct {
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config
	paste  *PasteService
	url    *URLService
	apiKey *APIKeyService
}

func NewCleanupService(db *gorm.DB, logger *zap.Logger, config *config.Config, services *Services) *CleanupService {
	return &CleanupService{
		db:     db,
		logger: logger,
		config: config,
		paste:  services.Paste,
		url:    services.URL,
		apiKey: services.APIKey,
	}
}

// RunCleanup performs all cleanup tasks
func (s *CleanupService) RunCleanup() {
	s.logger.Info("starting cleanup tasks")

	// Cleanup expired pastes
	if count, err := s.paste.CleanupExpired(); err != nil {
		s.logger.Error("failed to cleanup expired pastes", zap.Error(err))
	} else {
		s.logger.Info("cleaned up expired pastes", zap.Int64("count", count))
	}

	// Cleanup expired shortlinks
	if count, err := s.url.CleanupExpired(); err != nil {
		s.logger.Error("failed to cleanup expired shortlinks", zap.Error(err))
	} else {
		s.logger.Info("cleaned up expired shortlinks", zap.Int64("count", count))
	}

	// Cleanup unverified API keys
	if count := s.apiKey.CleanupUnverifiedKeys(); count > 0 {
		s.logger.Info("cleaned up unverified API keys", zap.Int64("count", count))
	}

	s.logger.Info("cleanup tasks completed")
}

// StartCleanupScheduler starts a periodic cleanup task
func (s *CleanupService) StartCleanupScheduler(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.RunCleanup()
		}
	}()

	s.logger.Info("cleanup scheduler started", zap.Duration("interval", interval))
}
