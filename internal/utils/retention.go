package utils

import (
	"fmt"
	"math"
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
func GenerateRetentionData(maxSize int64) (*RetentionStats, error) {
	const (
		points   = 50
		minAge   = 30.0  // 30 days minimum
		maxNoKey = 365.0 // 1 year without key
		maxKey   = 730.0 // 2 years with key
		midNoKey = 197.5 // Midpoint for no key curve
		midKey   = 365.0 // Midpoint for key curve
	)

	data := make(map[string][]RetentionPoint)
	data["noKey"] = make([]RetentionPoint, points+1)
	data["withKey"] = make([]RetentionPoint, points+1)

	for i := 0; i <= points; i++ {
		fileSize := float64(i) / float64(points) * float64(maxSize)
		sizeRatio := fileSize / float64(maxSize)

		// Calculate retention for no key
		noKeyRetention := minAge
		if sizeRatio <= 1 {
			noKeyRetention += (maxNoKey - minAge) * math.Pow(1-sizeRatio, 3)
		}
		noKeyRetention = math.Max(minAge, math.Min(maxNoKey, noKeyRetention))

		// Calculate retention for with key (more generous curve)
		withKeyRetention := minAge
		if sizeRatio <= 1 {
			withKeyRetention += (maxKey - minAge) * math.Pow(1-sizeRatio, 3)
		}
		withKeyRetention = math.Max(minAge, math.Min(maxKey, withKeyRetention))

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
		NoKeyRange:   fmt.Sprintf("%d-%d days", int(minAge), int(maxNoKey)),
		WithKeyRange: fmt.Sprintf("%d-%d days", int(minAge), int(maxKey)),
		Data:         data,
	}, nil
}
