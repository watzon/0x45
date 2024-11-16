package services

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AnalyticsService struct {
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config
}

func NewAnalyticsService(db *gorm.DB, logger *zap.Logger, config *config.Config) *AnalyticsService {
	return &AnalyticsService{
		db:     db,
		logger: logger,
		config: config,
	}
}

// GetResourceStats retrieves analytics statistics for a given resource
func (s *AnalyticsService) GetResourceStats(resourceType string, resourceID string, timeframe AnalyticsTimeframe) (*AnalyticsStats, error) {
	stats := &AnalyticsStats{
		TopReferrers: make(map[string]int64),
		TopCountries: make(map[string]int64),
		TopBrowsers:  make(map[string]int64),
	}

	// Base query
	query := s.db.Model(&models.AnalyticsEvent{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID)

	// Apply timeframe filters if provided
	if timeframe.StartTime != nil {
		query = query.Where("created_at >= ?", timeframe.StartTime)
	}
	if timeframe.EndTime != nil {
		query = query.Where("created_at <= ?", timeframe.EndTime)
	}

	// Get total views
	query.Count(&stats.TotalViews)

	// Get unique views (by IP)
	s.db.Model(&models.AnalyticsEvent{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Distinct("ip_address").
		Count(&stats.UniqueViews)

	// Get views by day
	type DailyViews struct {
		Date  time.Time
		Count int64
	}
	var dailyViews []DailyViews

	viewsQuery := s.db.Model(&models.AnalyticsEvent{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Group("DATE(created_at)").
		Order("date ASC")

	if timeframe.StartTime != nil {
		viewsQuery = viewsQuery.Where("created_at >= ?", timeframe.StartTime)
	}
	if timeframe.EndTime != nil {
		viewsQuery = viewsQuery.Where("created_at <= ?", timeframe.EndTime)
	}

	viewsQuery.Find(&dailyViews)

	stats.ViewsByDay = make([]ChartDataPoint, len(dailyViews))
	for i, dv := range dailyViews {
		stats.ViewsByDay[i] = ChartDataPoint{
			Date:  dv.Date,
			Value: dv.Count,
		}
	}

	// Get top referrers (excluding empty ones)
	s.db.Model(&models.AnalyticsEvent{}).
		Select("referer_url, COUNT(*) as count").
		Where("resource_type = ? AND resource_id = ? AND referer_url != ''", resourceType, resourceID).
		Group("referer_url").
		Order("count DESC").
		Limit(10).
		Find(&map[string]int64{}).
		Scan(&stats.TopReferrers)

	// Get top countries
	s.db.Model(&models.AnalyticsEvent{}).
		Select("country_code, COUNT(*) as count").
		Where("resource_type = ? AND resource_id = ? AND country_code != ''", resourceType, resourceID).
		Group("country_code").
		Order("count DESC").
		Limit(10).
		Find(&map[string]int64{}).
		Scan(&stats.TopCountries)

	// Get top browsers (parsed from user agent)
	s.db.Model(&models.AnalyticsEvent{}).
		Select("SUBSTRING_INDEX(user_agent, '/', 1) as browser, COUNT(*) as count").
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Group("browser").
		Order("count DESC").
		Limit(10).
		Find(&map[string]int64{}).
		Scan(&stats.TopBrowsers)

	return stats, nil
}

// LogEvent creates a new analytics event with common request information
func (s *AnalyticsService) LogEvent(c *fiber.Ctx, eventType models.EventType, resourceType string, resourceID string) error {
	// Get request information
	userAgent := c.Get("User-Agent")
	ipAddress := c.IP()
	refererURL := c.Get("Referer")

	// Create event with request context
	event := &models.AnalyticsEvent{
		EventType:    eventType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		RefererURL:   refererURL,
	}

	return s.db.Create(event).Error
}

// LogPasteView creates an analytics event for paste views
func (s *AnalyticsService) LogPasteView(c *fiber.Ctx, pasteID string) error {
	return s.LogEvent(c, models.EventPasteView, "paste", pasteID)
}

// LogShortlinkClick creates an analytics event for shortlink clicks
func (s *AnalyticsService) LogShortlinkClick(c *fiber.Ctx, shortlinkID string) error {
	return s.LogEvent(c, models.EventShortlinkClick, "shortlink", shortlinkID)
}

// GetStatsHistory generates usage statistics for the specified number of days
func (s *AnalyticsService) GetStatsHistory(days int) (*StatsHistory, error) {
	history := &StatsHistory{
		Pastes:     make([]ChartDataPoint, days),
		URLs:       make([]ChartDataPoint, days),
		Storage:    make([]ChartDataPoint, days),
		AvgSize:    make([]ChartDataPoint, days),
		APIKeys:    make([]ChartDataPoint, days),
		Extensions: make([]ChartDataPoint, days),
	}

	// Calculate date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get paste counts by day
	type DailyCount struct {
		Date  time.Time
		Count int64
	}

	// Query paste counts
	var pasteCounts []DailyCount
	s.db.Model(&models.Paste{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&pasteCounts)

	// Query URL counts
	var urlCounts []DailyCount
	s.db.Model(&models.Shortlink{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&urlCounts)

	// Query storage usage
	var storageCounts []struct {
		Date  time.Time
		Size  int64
		Count int64
	}
	s.db.Model(&models.Paste{}).
		Select("DATE(created_at) as date, SUM(size) as size, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&storageCounts)

	// Query API key counts
	var apiKeyCounts []DailyCount
	s.db.Model(&models.APIKey{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ? AND verified = ?", startDate, endDate, true).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&apiKeyCounts)

	// Convert to time series data
	for i := 0; i < days; i++ {
		date := endDate.AddDate(0, 0, -i)
		dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

		// Initialize with zero values
		history.Pastes[i] = ChartDataPoint{Date: dateOnly, Value: int64(0)}
		history.URLs[i] = ChartDataPoint{Date: dateOnly, Value: int64(0)}
		history.Storage[i] = ChartDataPoint{Date: dateOnly, Value: int64(0)}
		history.AvgSize[i] = ChartDataPoint{Date: dateOnly, Value: float64(0)}
		history.APIKeys[i] = ChartDataPoint{Date: dateOnly, Value: int64(0)}

		// Update with actual values if available
		for _, pc := range pasteCounts {
			if pc.Date.Equal(dateOnly) {
				history.Pastes[i].Value = pc.Count
				break
			}
		}

		for _, uc := range urlCounts {
			if uc.Date.Equal(dateOnly) {
				history.URLs[i].Value = uc.Count
				break
			}
		}

		for _, sc := range storageCounts {
			if sc.Date.Equal(dateOnly) {
				history.Storage[i].Value = sc.Size
				if sc.Count > 0 {
					history.AvgSize[i].Value = float64(sc.Size) / float64(sc.Count)
				}
				break
			}
		}

		for _, ac := range apiKeyCounts {
			if ac.Date.Equal(dateOnly) {
				history.APIKeys[i].Value = ac.Count
				break
			}
		}
	}

	return history, nil
}