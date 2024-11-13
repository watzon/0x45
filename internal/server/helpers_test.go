package server

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/watzon/0x45/internal/config"
)

func TestCalculateExpiry(t *testing.T) {
	// Create a server instance with test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxUploadSize: 10 * 1024 * 1024, // 10MB
		},
		Retention: config.RetentionConfig{
			NoKey: config.RetentionLimitConfig{
				MinAge: 7,  // 7 days
				MaxAge: 30, // 30 days
			},
			WithKey: config.RetentionLimitConfig{
				MinAge: 30, // 30 days
				MaxAge: 90, // 90 days
			},
		},
	}

	s := &Server{config: cfg}

	tests := []struct {
		name        string
		opts        ExpiryOptions
		wantErr     bool
		errContains string
		wantDays    float64 // Single expected value instead of min/max range
		epsilon     float64 // Tolerance for floating point comparison
	}{
		{
			name: "explicit expiry within bounds (no key)",
			opts: ExpiryOptions{
				Size:            1024,
				HasAPIKey:       false,
				RequestedExpiry: "48h",
			},
			wantDays: 2.0,
			epsilon:  1e-3,
		},
		{
			name: "explicit expiry exceeds max (no key)",
			opts: ExpiryOptions{
				Size:            1024,
				HasAPIKey:       false,
				RequestedExpiry: "720h", // 30 days
			},
			wantErr:     true,
			errContains: "Maximum allowed expiry",
		},
		{
			name: "permanent paste without API key",
			opts: ExpiryOptions{
				Size:            1024,
				HasAPIKey:       false,
				RequestedExpiry: "never",
			},
			wantErr:     true,
			errContains: "Permanent pastes require an API key",
		},
		{
			name: "permanent paste with API key",
			opts: ExpiryOptions{
				Size:            1024,
				HasAPIKey:       true,
				RequestedExpiry: "never",
			},
			wantDays: 0.0,
			epsilon:  1e-3,
		},
		{
			name: "invalid duration format",
			opts: ExpiryOptions{
				Size:            1024,
				HasAPIKey:       false,
				RequestedExpiry: "invalid",
			},
			wantErr:     true,
			errContains: "Invalid expiration format",
		},
		{
			name: "small file no key (auto expiry)",
			opts: ExpiryOptions{
				Size:      1024, // 1KB
				HasAPIKey: false,
			},
			wantDays: 29.9,
			epsilon:  1e-3,
		},
		{
			name: "large file no key (auto expiry)",
			opts: ExpiryOptions{
				Size:      9 * 1024 * 1024, // 9MB
				HasAPIKey: false,
			},
			wantDays: 7.2,
			epsilon:  1e-3,
		},
		{
			name: "small file with key (auto expiry)",
			opts: ExpiryOptions{
				Size:      1024, // 1KB
				HasAPIKey: true,
			},
			wantDays: 89.9,
			epsilon:  1e-3,
		},
		{
			name: "large file with key (auto expiry)",
			opts: ExpiryOptions{
				Size:      9 * 1024 * 1024, // 9MB
				HasAPIKey: true,
			},
			wantDays: 31.9,
			epsilon:  1e-3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expiryTime, err := s.calculateExpiry(tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			assert.NoError(t, err)

			if tt.opts.RequestedExpiry == "never" {
				assert.Nil(t, expiryTime)
				return
			}

			assert.NotNil(t, expiryTime)

			// Calculate days and compare with nearlyEqual
			days := expiryTime.Sub(time.Now()).Hours() / 24
			if !nearlyEqual(days, tt.wantDays, tt.epsilon) {
				t.Errorf("Expected %.10f days, got %.10f days", tt.wantDays, days)
			}
		})
	}
}

func TestCalculateMaxRetention(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			MaxUploadSize: 10 * 1024 * 1024, // 10MB
		},
		Retention: config.RetentionConfig{
			NoKey: config.RetentionLimitConfig{
				MinAge: 7,
				MaxAge: 30,
			},
			WithKey: config.RetentionLimitConfig{
				MinAge: 30,
				MaxAge: 90,
			},
		},
	}

	s := &Server{config: cfg}

	tests := []struct {
		name      string
		size      int64
		hasAPIKey bool
		want      float64
		epsilon   float64
	}{
		{
			name:      "minimum size no key",
			size:      0,
			hasAPIKey: false,
			want:      30,
			epsilon:   1e-3,
		},
		{
			name:      "minimum size with key",
			size:      0,
			hasAPIKey: true,
			want:      90,
			epsilon:   1e-3,
		},
		{
			name:      "maximum size no key",
			size:      10 * 1024 * 1024,
			hasAPIKey: false,
			want:      7,
			epsilon:   1e-3,
		},
		{
			name:      "maximum size with key",
			size:      10 * 1024 * 1024,
			hasAPIKey: true,
			want:      30,
			epsilon:   1e-3,
		},
		{
			name:      "half max size no key",
			size:      5 * 1024 * 1024,
			hasAPIKey: false,
			want:      12.75,
			epsilon:   1e-3,
		},
		{
			name:      "half max size with key",
			size:      5 * 1024 * 1024,
			hasAPIKey: true,
			want:      51.2,
			epsilon:   1e-3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.calculateMaxRetention(tt.size, tt.hasAPIKey)
			if !nearlyEqual(got, tt.want, tt.epsilon) {
				t.Errorf("calculateMaxRetention() = %.10f, want %.10f", got, tt.want)
			}
		})
	}
}

func nearlyEqual(a, b, epsilon float64) bool {

	// already equal?
	if a == b {
		return true
	}

	diff := math.Abs(a - b)
	if a == 0.0 || b == 0.0 || diff < math.SmallestNonzeroFloat64 {
		return diff < epsilon*math.SmallestNonzeroFloat64
	}

	return diff/(math.Abs(a)+math.Abs(b)) < epsilon
}
