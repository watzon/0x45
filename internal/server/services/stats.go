package services

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type StatsService struct {
	db        *gorm.DB
	logger    *zap.Logger
	config    *config.Config
	analytics *AnalyticsService
}

func NewStatsService(db *gorm.DB, logger *zap.Logger, config *config.Config) *StatsService {
	return &StatsService{
		db:        db,
		logger:    logger,
		config:    config,
		analytics: NewAnalyticsService(db, logger, config),
	}
}

// GetSystemStats returns current system statistics and historical data
func (s *StatsService) GetSystemStats() (fiber.Map, error) {
	// Get current stats
	var totalPastes, totalUrls int64
	s.db.Model(&models.Paste{}).Count(&totalPastes)
	s.db.Model(&models.Shortlink{}).Count(&totalUrls)

	// Get historical data
	history, err := s.analytics.GetStatsHistory(7)
	if err != nil {
		s.logger.Error("failed to get stats history", zap.Error(err))
		history = &StatsHistory{
			Pastes:  make([]ChartDataPoint, 7),
			URLs:    make([]ChartDataPoint, 7),
			Storage: make([]ChartDataPoint, 7),
		}
	}

	// Convert data to JSON strings
	pastesHistory, _ := json.Marshal(history.Pastes)
	urlsHistory, _ := json.Marshal(history.URLs)
	storageHistory, _ := json.Marshal(history.Storage)

	// Get storage by file type
	storageByType, err := s.getStorageByFileType()
	if err != nil {
		s.logger.Error("failed to get storage by file type", zap.Error(err))
		storageByType = make(map[string]int64)
	}

	// Convert to JSON
	storageByTypeJSON, _ := json.Marshal(storageByType)

	// Get average paste size
	var avgSize float64
	if err := s.db.Model(&models.Paste{}).
		Select("COALESCE(AVG(NULLIF(size, 0)), 0)").
		Row().
		Scan(&avgSize); err != nil {
		s.logger.Error("failed to get average size", zap.Error(err))
		avgSize = 0
	}

	// Get active API keys count
	var activeApiKeys int64
	s.db.Model(&models.APIKey{}).Where("verified = ?", true).Count(&activeApiKeys)

	// Get popular extensions
	extensionStats := make(map[string]int64)
	rows, err := s.db.Model(&models.Paste{}).
		Select("extension, COUNT(*) as count").
		Where("extension != ''").
		Group("extension").
		Order("count DESC").
		Limit(10).
		Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ext string
			var count int64
			if err := rows.Scan(&ext, &count); err == nil {
				extensionStats[ext] = count
			}
		}
	} else {
		s.logger.Error("failed to get extension stats", zap.Error(err))
	}

	// Get expiring content counts
	var expiringPastes, expiringUrls int64
	twentyFourHours := time.Now().Add(24 * time.Hour)
	s.db.Model(&models.Paste{}).
		Where("expires_at < ? AND expires_at > ?", twentyFourHours, time.Now()).
		Count(&expiringPastes)
	s.db.Model(&models.Shortlink{}).
		Where("expires_at < ? AND expires_at > ?", twentyFourHours, time.Now()).
		Count(&expiringUrls)

	// Get private vs public paste ratio
	var privatePastes int64
	s.db.Model(&models.Paste{}).Where("private = ?", true).Count(&privatePastes)
	publicPastes := totalPastes - privatePastes

	// Calculate private ratio
	var privateRatio float64
	if totalPastes > 0 {
		privateRatio = float64(privatePastes) / float64(totalPastes) * 100
	}

	// Get total storage used
	totalStorage, err := s.getStorageSize()
	if err != nil {
		s.logger.Error("failed to get storage size", zap.Error(err))
		totalStorage = 0
	}

	return fiber.Map{
		"current": fiber.Map{
			"pastes":        totalPastes,
			"urls":          totalUrls,
			"storage":       totalStorage,
			"storageByType": string(storageByTypeJSON),
			"avgSize":       avgSize,
			"activeApiKeys": activeApiKeys,
			"extensionStats": extensionStats,
			"expiringPastes": expiringPastes,
			"expiringUrls":   expiringUrls,
		},
		"history": fiber.Map{
			"pastes":  string(pastesHistory),
			"urls":    string(urlsHistory),
			"storage": string(storageHistory),
		},
		"storage": fiber.Map{
			"byType":  string(storageByTypeJSON),
			"avgSize": avgSize,
		},
		"extensions": extensionStats,
		"expiring": fiber.Map{
			"pastes": expiringPastes,
			"urls":   expiringUrls,
		},
		"privacy": fiber.Map{
			"private":      privatePastes,
			"public":       publicPastes,
			"privateRatio": privateRatio,
		},
	}, nil
}

// Helper functions

func (s *StatsService) getStorageByFileType() (map[string]int64, error) {
	result := make(map[string]int64)

	rows, err := s.db.Model(&models.Paste{}).
		Select("mime_type, SUM(size) as total_size").
		Where("mime_type != ''").
		Group("mime_type").
		Rows()

	if err != nil {
		s.logger.Error("failed to query storage by file type", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mimeType string
		var size int64
		if err := rows.Scan(&mimeType, &size); err != nil {
			s.logger.Error("failed to scan row", zap.Error(err))
			continue
		}
		category := s.categorizeMimeType(mimeType)
		result[category] += size
	}

	return result, nil
}

func (s *StatsService) categorizeMimeType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "text/"):
		return "text"
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.Contains(mimeType, "pdf"):
		return "pdf"
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "tar") || strings.Contains(mimeType, "compress"):
		return "archive"
	default:
		return "other"
	}
}

func (s *StatsService) getStorageSize() (uint64, error) {
	var totalSize uint64
	err := s.db.Model(&models.Paste{}).Select("COALESCE(SUM(size), 0)").Row().Scan(&totalSize)
	return totalSize, err
}
