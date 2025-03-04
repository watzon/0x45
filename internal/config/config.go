package config

import (
	"fmt"
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

type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

type CleanupConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Interval int    `mapstructure:"interval"` // in seconds
	MaxAge   string `mapstructure:"max_age"`  // duration string (e.g., "168h")
}

type GlobalRateLimitConfig struct {
	Enabled bool    `mapstructure:"enabled"` // Enable global rate limiting
	Rate    float64 `mapstructure:"rate"`    // Requests per second
	Burst   int     `mapstructure:"burst"`   // Maximum burst size
}

type PerIPRateLimitConfig struct {
	Enabled bool    `mapstructure:"enabled"` // Enable per-IP rate limiting
	Rate    float64 `mapstructure:"rate"`    // Requests per second per IP
	Burst   int     `mapstructure:"burst"`   // Maximum burst size
}

type RateLimitConfig struct {
	Global            GlobalRateLimitConfig `mapstructure:"global"`
	PerIP             PerIPRateLimitConfig  `mapstructure:"per_ip"`
	UseRedis          bool                  `mapstructure:"use_redis"`           // Use Redis for rate limiting if it's available (required for prefork)
	IPCleanupInterval time.Duration         `mapstructure:"ip_cleanup_interval"` // Duration string (e.g., "1h")
}

type ServerConfig struct {
	Address           string          `mapstructure:"address"`
	BaseURL           string          `mapstructure:"base_url"`
	MaxUploadSize     int             `mapstructure:"max_upload_size"`
	DefaultUploadSize int             `mapstructure:"default_upload_size"`
	APIUploadSize     int             `mapstructure:"api_upload_size"`
	Prefork           bool            `mapstructure:"prefork"`
	ServerHeader      string          `mapstructure:"server_header"`
	AppName           string          `mapstructure:"app_name"`
	Cleanup           CleanupConfig   `mapstructure:"cleanup"`
	RateLimit         RateLimitConfig `mapstructure:"rate_limit"`
	CORSOrigins       []string        `mapstructure:"cors_origins"`
	ViewsDirectory    string          `mapstructure:"views_directory"`
	PublicDirectory   string          `mapstructure:"public_directory"`
}

type SMTPConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
	StartTLS bool   `mapstructure:"starttls"`
}

type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type RetentionLimitConfig struct {
	MinAge float64 `mapstructure:"min_age"`
	MaxAge float64 `mapstructure:"max_age"`
}

type RetentionConfig struct {
	NoKey   RetentionLimitConfig `mapstructure:"no_key"`
	WithKey RetentionLimitConfig `mapstructure:"with_key"`
	Points  int                  `mapstructure:"points"` // Number of points to generate for the curve
}

type Config struct {
	Database  DatabaseConfig  `mapstructure:"database"`
	Storage   []StorageConfig `mapstructure:"storage"`
	Server    ServerConfig    `mapstructure:"server"`
	SMTP      SMTPConfig      `mapstructure:"smtp"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Retention RetentionConfig `mapstructure:"retention"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// Database bindings
	_ = viper.BindEnv("database.driver", "0X_DATABASE_DRIVER")
	_ = viper.BindEnv("database.host", "0X_DATABASE_HOST")
	_ = viper.BindEnv("database.port", "0X_DATABASE_PORT")
	_ = viper.BindEnv("database.user", "0X_DATABASE_USER")
	_ = viper.BindEnv("database.password", "0X_DATABASE_PASSWORD")
	_ = viper.BindEnv("database.name", "0X_DATABASE_NAME")
	_ = viper.BindEnv("database.sslmode", "0X_DATABASE_SSLMODE")

	// Server bindings
	_ = viper.BindEnv("server.address", "0X_SERVER_ADDRESS")
	_ = viper.BindEnv("server.base_url", "0X_SERVER_BASE_URL")
	_ = viper.BindEnv("server.max_upload_size", "0X_SERVER_MAX_UPLOAD_SIZE")
	_ = viper.BindEnv("server.default_upload_size", "0X_SERVER_DEFAULT_UPLOAD_SIZE")
	_ = viper.BindEnv("server.api_upload_size", "0X_SERVER_API_UPLOAD_SIZE")
	_ = viper.BindEnv("server.prefork", "0X_SERVER_PREFORK")
	_ = viper.BindEnv("server.server_header", "0X_SERVER_SERVER_HEADER")
	_ = viper.BindEnv("server.app_name", "0X_SERVER_APP_NAME")
	_ = viper.BindEnv("server.cors_origins", "0X_SERVER_CORS_ORIGINS")
	_ = viper.BindEnv("server.views_directory", "0X_SERVER_VIEWS_DIRECTORY")
	_ = viper.BindEnv("server.public_directory", "0X_SERVER_PUBLIC_DIRECTORY")

	// Server cleanup bindings
	_ = viper.BindEnv("server.cleanup.enabled", "0X_SERVER_CLEANUP_ENABLED")
	_ = viper.BindEnv("server.cleanup.interval", "0X_SERVER_CLEANUP_INTERVAL")
	_ = viper.BindEnv("server.cleanup.max_age", "0X_SERVER_CLEANUP_MAX_AGE")

	// Rate limit bindings
	_ = viper.BindEnv("server.rate_limit.global.enabled", "0X_SERVER_RATE_LIMIT_GLOBAL_ENABLED")
	_ = viper.BindEnv("server.rate_limit.global.rate", "0X_SERVER_RATE_LIMIT_GLOBAL_RATE")
	_ = viper.BindEnv("server.rate_limit.global.burst", "0X_SERVER_RATE_LIMIT_GLOBAL_BURST")
	_ = viper.BindEnv("server.rate_limit.per_ip.enabled", "0X_SERVER_RATE_LIMIT_PER_IP_ENABLED")
	_ = viper.BindEnv("server.rate_limit.per_ip.rate", "0X_SERVER_RATE_LIMIT_PER_IP_RATE")
	_ = viper.BindEnv("server.rate_limit.per_ip.burst", "0X_SERVER_RATE_LIMIT_PER_IP_BURST")
	_ = viper.BindEnv("server.rate_limit.use_redis", "0X_SERVER_RATE_LIMIT_USE_REDIS")
	_ = viper.BindEnv("server.rate_limit.ip_cleanup_interval", "0X_SERVER_RATE_LIMIT_IP_CLEANUP_INTERVAL")

	// SMTP bindings
	_ = viper.BindEnv("smtp.enabled", "0X_SMTP_ENABLED")
	_ = viper.BindEnv("smtp.host", "0X_SMTP_HOST")
	_ = viper.BindEnv("smtp.port", "0X_SMTP_PORT")
	_ = viper.BindEnv("smtp.username", "0X_SMTP_USERNAME")
	_ = viper.BindEnv("smtp.password", "0X_SMTP_PASSWORD")
	_ = viper.BindEnv("smtp.from", "0X_SMTP_FROM")
	_ = viper.BindEnv("smtp.from_name", "0X_SMTP_FROM_NAME")
	_ = viper.BindEnv("smtp.starttls", "0X_SMTP_STARTTLS")

	// Redis bindings
	_ = viper.BindEnv("redis.enabled", "0X_REDIS_ENABLED")
	_ = viper.BindEnv("redis.address", "0X_REDIS_ADDRESS")
	_ = viper.BindEnv("redis.password", "0X_REDIS_PASSWORD")
	_ = viper.BindEnv("redis.db", "0X_REDIS_DB")

	// Retention bindings
	_ = viper.BindEnv("retention.no_key.min_age", "0X_RETENTION_NO_KEY_MIN_AGE")
	_ = viper.BindEnv("retention.no_key.max_age", "0X_RETENTION_NO_KEY_MAX_AGE")
	_ = viper.BindEnv("retention.with_key.min_age", "0X_RETENTION_WITH_KEY_MIN_AGE")
	_ = viper.BindEnv("retention.with_key.max_age", "0X_RETENTION_WITH_KEY_MAX_AGE")
	_ = viper.BindEnv("retention.points", "0X_RETENTION_POINTS")

	// Now set defaults
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.name", "paste69.db")
	viper.SetDefault("database.sslmode", "disable")

	viper.SetDefault("smtp.enabled", false)
	viper.SetDefault("smtp.port", 587)
	viper.SetDefault("smtp.starttls", true)
	viper.SetDefault("smtp.tls_verify", true)
	viper.SetDefault("smtp.from_name", "Paste69")

	viper.SetDefault("storage", []map[string]any{
		{
			"name":    "local",
			"type":    "local",
			"path":    "./uploads",
			"default": true,
		},
	})

	viper.SetDefault("server.address", ":3000")
	viper.SetDefault("server.max_upload_size", 5242880)      // 5MB default
	viper.SetDefault("server.default_upload_size", 10485760) // 10MB default
	viper.SetDefault("server.api_upload_size", 52428800)     // 50MB default
	viper.SetDefault("server.prefork", false)
	viper.SetDefault("server.server_header", "Paste69")
	viper.SetDefault("server.app_name", "Paste69")
	viper.SetDefault("server.cleanup.enabled", true)
	viper.SetDefault("server.cleanup.interval", 3600)
	viper.SetDefault("server.cleanup.max_age", "168h")
	viper.SetDefault("server.cors_origins", []string{"*"})
	viper.SetDefault("server.views_directory", "./views")
	viper.SetDefault("server.public_directory", "./public")

	viper.SetDefault("server.rate_limit.global.enabled", true) // Enable global rate limiting by default
	viper.SetDefault("server.rate_limit.global.rate", 6969.0)  // 6969 requests per second globally
	viper.SetDefault("server.rate_limit.global.burst", 250)    // Allow bursts of up to 250 requests
	viper.SetDefault("server.rate_limit.per_ip.enabled", true) // Enable per-IP rate limiting by default
	viper.SetDefault("server.rate_limit.per_ip.rate", 2.0)     // 2 request per second per IP
	viper.SetDefault("server.rate_limit.per_ip.burst", 5)      // Allow bursts of up to 5 requests
	viper.SetDefault("server.rate_limit.use_redis", false)     // Use Redis for rate limiting if it's available (required for prefork)
	viper.SetDefault("server.rate_limit.ip_cleanup_interval", "1h")

	viper.SetDefault("redis.enabled", false)
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	viper.SetDefault("retention.no_key.min_age", 7.0)     // 7 days minimum
	viper.SetDefault("retention.no_key.max_age", 128.0)   // 128 days without key
	viper.SetDefault("retention.with_key.min_age", 30.0)  // 30 days minimum
	viper.SetDefault("retention.with_key.max_age", 730.0) // 2 years with key
	viper.SetDefault("retention.points", 50)              // Number of points to generate

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Add support for multiple storage backends via environment variables
	const maxStorageBackends = 10 // reasonable limit
	storageConfigs := []StorageConfig{}

	for i := 0; i < maxStorageBackends; i++ {
		prefix := fmt.Sprintf("STORAGE_%d_", i)

		// Bind storage config environment variables
		_ = viper.BindEnv(fmt.Sprintf("storage.%d.name", i), "0X_"+prefix+"NAME")
		_ = viper.BindEnv(fmt.Sprintf("storage.%d.type", i), "0X_"+prefix+"TYPE")
		_ = viper.BindEnv(fmt.Sprintf("storage.%d.default", i), "0X_"+prefix+"DEFAULT")
		_ = viper.BindEnv(fmt.Sprintf("storage.%d.path", i), "0X_"+prefix+"PATH")
		_ = viper.BindEnv(fmt.Sprintf("storage.%d.s3_bucket", i), "0X_"+prefix+"S3_BUCKET")
		_ = viper.BindEnv(fmt.Sprintf("storage.%d.s3_region", i), "0X_"+prefix+"S3_REGION")
		_ = viper.BindEnv(fmt.Sprintf("storage.%d.s3_key", i), "0X_"+prefix+"S3_KEY")
		_ = viper.BindEnv(fmt.Sprintf("storage.%d.s3_secret", i), "0X_"+prefix+"S3_SECRET")
		_ = viper.BindEnv(fmt.Sprintf("storage.%d.s3_endpoint", i), "0X_"+prefix+"S3_ENDPOINT")

		// Check if this storage backend is configured
		if name := viper.GetString(fmt.Sprintf("storage.%d.name", i)); name != "" {
			storage := StorageConfig{
				Name:       name,
				Type:       viper.GetString(fmt.Sprintf("storage.%d.type", i)),
				IsDefault:  viper.GetBool(fmt.Sprintf("storage.%d.default", i)),
				Path:       viper.GetString(fmt.Sprintf("storage.%d.path", i)),
				S3Bucket:   viper.GetString(fmt.Sprintf("storage.%d.s3_bucket", i)),
				S3Region:   viper.GetString(fmt.Sprintf("storage.%d.s3_region", i)),
				S3Key:      viper.GetString(fmt.Sprintf("storage.%d.s3_key", i)),
				S3Secret:   viper.GetString(fmt.Sprintf("storage.%d.s3_secret", i)),
				S3Endpoint: viper.GetString(fmt.Sprintf("storage.%d.s3_endpoint", i)),
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
