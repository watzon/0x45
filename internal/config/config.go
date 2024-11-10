package config

import (
	"fmt"

	"github.com/spf13/viper"
)

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

	Storage struct {
		Type       string `mapstructure:"type"` // "local" or "s3"
		Path       string `mapstructure:"path"` // for local storage
		S3Bucket   string `mapstructure:"s3_bucket"`
		S3Region   string `mapstructure:"s3_region"`
		S3Key      string `mapstructure:"s3_key"`
		S3Secret   string `mapstructure:"s3_secret"`
		S3Endpoint string `mapstructure:"s3_endpoint"` // for custom S3-compatible services
	} `mapstructure:"storage"`

	Server struct {
		Address       string `mapstructure:"address"`
		BaseURL       string `mapstructure:"base_url"`
		MaxUploadSize int    `mapstructure:"max_upload_size"`
		Cleanup       struct {
			Enabled  bool   `mapstructure:"enabled"`
			Interval int    `mapstructure:"interval"` // in seconds
			MaxAge   string `mapstructure:"max_age"`  // duration string (e.g., "168h")
		} `mapstructure:"cleanup"`
	} `mapstructure:"server"`

	// Add SMTP configuration
	SMTP struct {
		Enabled   bool   `mapstructure:"enabled"`
		Host      string `mapstructure:"host"`
		Port      int    `mapstructure:"port"`
		Username  string `mapstructure:"username"`
		Password  string `mapstructure:"password"`
		From      string `mapstructure:"from"`
		FromName  string `mapstructure:"from_name"`
		StartTLS  bool   `mapstructure:"starttls"`
		TLSVerify bool   `mapstructure:"tls_verify"`
	} `mapstructure:"smtp"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.name", "paste69.db")
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.path", "./uploads")
	viper.SetDefault("server.address", ":3000")
	viper.SetDefault("server.max_upload_size", 50*1024*1024) // 50MB default
	viper.SetDefault("server.cleanup.enabled", true)
	viper.SetDefault("server.cleanup.interval", 3600)
	viper.SetDefault("server.cleanup.max_age", "168h")

	// SMTP defaults
	viper.SetDefault("smtp.enabled", false)
	viper.SetDefault("smtp.port", 587)
	viper.SetDefault("smtp.starttls", true)
	viper.SetDefault("smtp.tls_verify", true)
	viper.SetDefault("smtp.from_name", "Paste69")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}
