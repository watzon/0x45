package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type StorageConfig struct {
	Name       string `mapstructure:"name"`    // Unique name for this storage config
	Type       string `mapstructure:"type"`    // "local" or "s3"
	IsDefault  bool   `mapstructure:"default"` // Whether this is the default storage
	Path       string `mapstructure:"path"`    // for local storage
	S3Bucket   string `mapstructure:"s3_bucket"`
	S3Region   string `mapstructure:"s3_region"`
	S3Key      string `mapstructure:"s3_key"`
	S3Secret   string `mapstructure:"s3_secret"`
	S3Endpoint string `mapstructure:"s3_endpoint"`
}

type Config struct {
	Database struct {
		Driver   string `mapstructure:"driver"`
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
		SSLMode  string `mapstructure:"sslmode"`
	} `mapstructure:"database"`

	Storage []StorageConfig `mapstructure:"storage"`

	Server struct {
		Address       string `mapstructure:"address"`
		BaseURL       string `mapstructure:"base_url"`
		MaxUploadSize int    `mapstructure:"max_upload_size"`
		Prefork       bool   `mapstructure:"prefork"`
		ServerHeader  string `mapstructure:"server_header"`
		AppName       string `mapstructure:"app_name"`
		Cleanup       struct {
			Enabled  bool   `mapstructure:"enabled"`
			Interval int    `mapstructure:"interval"` // in seconds
			MaxAge   string `mapstructure:"max_age"`  // duration string (e.g., "168h")
		} `mapstructure:"cleanup"`
		RateLimit struct {
			Global struct {
				Enabled bool    `mapstructure:"enabled"` // Enable global rate limiting
				Rate    float64 `mapstructure:"rate"`    // Requests per second
				Burst   int     `mapstructure:"burst"`   // Maximum burst size
			} `mapstructure:"global"`
			PerIP struct {
				Enabled bool    `mapstructure:"enabled"` // Enable per-IP rate limiting
				Rate    float64 `mapstructure:"rate"`    // Requests per second per IP
				Burst   int     `mapstructure:"burst"`   // Maximum burst size
			} `mapstructure:"per_ip"`
			UseRedis          bool          `mapstructure:"use_redis"`           // Use Redis for rate limiting if it's available (required for prefork)
			IPCleanupInterval time.Duration `mapstructure:"ip_cleanup_interval"` // Duration string (e.g., "1h")
		} `mapstructure:"rate_limit"`
	} `mapstructure:"server"`

	// Add SMTP configuration
	SMTP struct {
		Enabled  bool   `mapstructure:"enabled"`
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		From     string `mapstructure:"from"`
		FromName string `mapstructure:"from_name"`
		StartTLS bool   `mapstructure:"starttls"`
	} `mapstructure:"smtp"`

	Redis struct {
		Enabled  bool   `mapstructure:"enabled"`
		Address  string `mapstructure:"address"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`

	Retention struct {
		NoKey struct {
			MinAge float64 `mapstructure:"min_age"` // Minimum retention in days
			MaxAge float64 `mapstructure:"max_age"` // Maximum retention in days
		} `mapstructure:"no_key"`
		WithKey struct {
			MinAge float64 `mapstructure:"min_age"`
			MaxAge float64 `mapstructure:"max_age"`
		} `mapstructure:"with_key"`
		Points int `mapstructure:"points"` // Number of points to generate for the curve
	} `mapstructure:"retention"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults AFTER binding environment variables
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.host", "localhost") // Add default host
	viper.SetDefault("database.port", 5432)        // Add default port
	viper.SetDefault("database.user", "")          // Add default user
	viper.SetDefault("database.password", "")      // Add default password
	viper.SetDefault("database.name", "paste69.db")
	viper.SetDefault("database.sslmode", "disable") // Add default sslmode

	// SMTP defaults
	viper.SetDefault("smtp.enabled", false)
	viper.SetDefault("smtp.port", 587)
	viper.SetDefault("smtp.starttls", true)
	viper.SetDefault("smtp.tls_verify", true)
	viper.SetDefault("smtp.from_name", "Paste69")

	viper.SetDefault("storage", []map[string]interface{}{
		{
			"name":    "local",
			"type":    "local",
			"path":    "./uploads",
			"default": true,
		},
	})

	viper.SetDefault("server.address", ":3000")
	viper.SetDefault("server.max_upload_size", 5*1024*1024) // 5MB default
	viper.SetDefault("server.prefork", false)
	viper.SetDefault("server.server_header", "Paste69")
	viper.SetDefault("server.app_name", "Paste69")
	viper.SetDefault("server.cleanup.enabled", true)
	viper.SetDefault("server.cleanup.interval", 3600)
	viper.SetDefault("server.cleanup.max_age", "168h")

	// Rate limiting defaults
	viper.SetDefault("server.rate_limit.global.enabled", true) // Enable global rate limiting by default
	viper.SetDefault("server.rate_limit.global.rate", 100.0)   // 100 requests per second globally
	viper.SetDefault("server.rate_limit.global.burst", 50)     // Allow bursts of up to 50 requests
	viper.SetDefault("server.rate_limit.per_ip.enabled", true) // Enable per-IP rate limiting by default
	viper.SetDefault("server.rate_limit.per_ip.rate", 1.0)     // 1 request per second per IP
	viper.SetDefault("server.rate_limit.per_ip.burst", 5)      // Allow bursts of up to 5 requests
	viper.SetDefault("server.rate_limit.use_redis", false)     // Use Redis for rate limiting if it's available (required for prefork)
	viper.SetDefault("server.rate_limit.ip_cleanup_interval", "1h")

	// Redis defaults
	viper.SetDefault("redis.enabled", false)
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Set retention defaults
	viper.SetDefault("retention.no_key.min_age", 7.0)     // 7 days minimum
	viper.SetDefault("retention.no_key.max_age", 128.0)   // 128 days without key
	viper.SetDefault("retention.with_key.min_age", 30.0)  // 30 days minimum
	viper.SetDefault("retention.with_key.max_age", 730.0) // 2 years with key
	viper.SetDefault("retention.points", 50)              // Number of points to generate

	viper.AutomaticEnv()
	viper.SetEnvPrefix("0X_")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Debug output
	fmt.Printf("Database Config: driver=%s host=%s port=%d user=%s dbname=%s sslmode=%s\n",
		config.Database.Driver,
		config.Database.Host,
		config.Database.Port,
		config.Database.User,
		config.Database.Name,
		config.Database.SSLMode,
	)

	// Add support for multiple storage backends via environment variables
	const maxStorageBackends = 10 // reasonable limit
	storageConfigs := []StorageConfig{}

	for i := 0; i < maxStorageBackends; i++ {
		prefix := fmt.Sprintf("STORAGE_%d_", i)

		// Check if this storage backend is configured
		if name := viper.GetString(prefix + "NAME"); name != "" {
			storage := StorageConfig{
				Name:       name,
				Type:       viper.GetString(prefix + "TYPE"),
				IsDefault:  viper.GetBool(prefix + "DEFAULT"),
				Path:       viper.GetString(prefix + "PATH"),
				S3Bucket:   viper.GetString(prefix + "S3_BUCKET"),
				S3Region:   viper.GetString(prefix + "S3_REGION"),
				S3Key:      viper.GetString(prefix + "S3_KEY"),
				S3Secret:   viper.GetString(prefix + "S3_SECRET"),
				S3Endpoint: viper.GetString(prefix + "S3_ENDPOINT"),
			}
			storageConfigs = append(storageConfigs, storage)
		}
	}

	// If no storage configs were found in env vars, use the default from viper
	if len(storageConfigs) > 0 {
		config.Storage = storageConfigs
	}

	return &config, nil
}
