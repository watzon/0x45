package services

import (
	"time"

	"github.com/watzon/0x45/internal/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Services holds all service instances
type Services struct {
	Paste     *PasteService
	URL       *URLService
	APIKey    *APIKeyService
	Analytics *AnalyticsService
	Stats     *StatsService
	Cleanup   *CleanupService
}

// NewServices creates a new Services instance with all service dependencies
func NewServices(db *gorm.DB, logger *zap.Logger, config *config.Config) *Services {
	services := &Services{
		Paste:     NewPasteService(db, logger, config),
		URL:       NewURLService(db, logger, config),
		APIKey:    NewAPIKeyService(db, logger, config),
		Analytics: NewAnalyticsService(db, logger, config),
		Stats:     NewStatsService(db, logger, config),
	}

	// Create cleanup service last since it depends on other services
	services.Cleanup = NewCleanupService(db, logger, config, services)

	return services
}

// StartCleanupScheduler starts the cleanup scheduler with the configured interval
func (s *Services) StartCleanupScheduler(interval string) error {
	duration, err := time.ParseDuration(interval)
	if err != nil {
		return err
	}

	s.Cleanup.StartCleanupScheduler(duration)
	return nil
}
