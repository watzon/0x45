package utils

import (
	"fmt"
	"math"

	"github.com/watzon/0x45/internal/config"
)

type RetentionPoint struct {
	Value float64 `json:"value"`
	Date  string  `json:"date"` // We'll use this for the file size
}

type RetentionStats struct {
	NoKeyRange   string                      `json:"noKeyRange"`
	WithKeyRange string                      `json:"withKeyRange"`
	Data         map[string][]RetentionPoint `json:"data"`
}

// GenerateRetentionData creates data points for the retention curve
func GenerateRetentionData(maxSize int64, cfg *config.Config) (*RetentionStats, error) {
	points := cfg.Retention.Points

	// No key retention settings
	minAgeNoKey := cfg.Retention.NoKey.MinAge
	maxAgeNoKey := cfg.Retention.NoKey.MaxAge

	// With key retention settings
	minAgeWithKey := cfg.Retention.WithKey.MinAge
	maxAgeWithKey := cfg.Retention.WithKey.MaxAge

	data := make(map[string][]RetentionPoint)
	data["noKey"] = make([]RetentionPoint, points+1)
	data["withKey"] = make([]RetentionPoint, points+1)

	for i := 0; i <= points; i++ {
		fileSize := float64(i) / float64(points) * float64(maxSize)
		sizeRatio := fileSize / float64(maxSize)

		// Calculate retention for no key using a sigmoid-like curve
		noKeyRetention := minAgeNoKey
		if sizeRatio <= 1 {
			// Use exponential decay based on file size ratio
			noKeyRetention += (maxAgeNoKey - minAgeNoKey) * math.Pow(1-sizeRatio, 2)
		}
		noKeyRetention = math.Max(minAgeNoKey, math.Min(maxAgeNoKey, noKeyRetention))

		// Calculate retention for with key (more generous curve)
		withKeyRetention := minAgeWithKey
		if sizeRatio <= 1 {
			// Use a gentler exponential decay for authenticated uploads
			withKeyRetention += (maxAgeWithKey - minAgeWithKey) * math.Pow(1-sizeRatio, 1.5)
		}
		withKeyRetention = math.Max(minAgeWithKey, math.Min(maxAgeWithKey, withKeyRetention))

		// Store points
		data["noKey"][i] = RetentionPoint{
			Value: noKeyRetention,
			Date:  fmt.Sprintf("%.1f", fileSize/(1024*1024)), // Convert to MiB
		}
		data["withKey"][i] = RetentionPoint{
			Value: withKeyRetention,
			Date:  fmt.Sprintf("%.1f", fileSize/(1024*1024)), // Convert to MiB
		}
	}

	return &RetentionStats{
		NoKeyRange:   fmt.Sprintf("%.0f-%.0f days", minAgeNoKey, maxAgeNoKey),
		WithKeyRange: fmt.Sprintf("%.0f-%.0f days", minAgeWithKey, maxAgeWithKey),
		Data:         data,
	}, nil
}
