package database

import (
	"fmt"
	"strings"

	"github.com/glebarez/sqlite"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/database/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	*gorm.DB
}

func New(config *config.Config) (*Database, error) {
	var dialector gorm.Dialector

	switch config.Database.Driver {
	case "postgres":
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Database.Host,
			config.Database.Port,
			config.Database.User,
			config.Database.Password,
			config.Database.Name,
			config.Database.SSLMode,
		)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(config.Database.Name)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", config.Database.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Database{db}, nil
}

func (d *Database) Migrate(config *config.Config) error {
	_, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	dsn := ""
	switch config.Database.Driver {
	case "postgres":
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Database.Host,
			config.Database.Port,
			config.Database.User,
			config.Database.Password,
			config.Database.Name,
			config.Database.SSLMode,
		)
	case "sqlite":
		dsn = config.Database.Name
	}

	url, err := dsnToUrl(config.Database.Driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to convert DSN to URL: %w", err)
	}

	return migrations.RunMigrations(url)
}

// DsnToUrl converts a database DSN to a URL format suitable for migrations
func dsnToUrl(driver string, dsn string) (string, error) {
	switch driver {
	case "postgres":
		// Parse the DSN into components
		params := make(map[string]string)
		for _, pair := range strings.Split(dsn, " ") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				params[parts[0]] = parts[1]
			}
		}

		// Build auth part of URL (optional)
		auth := ""
		if user, ok := params["user"]; ok {
			auth = user
			if pass, ok := params["password"]; ok {
				auth += ":" + pass
			}
			auth += "@"
		}

		// Construct the URL
		host := params["host"]
		if host == "" {
			host = "localhost"
		}
		port := params["port"]
		if port == "" {
			port = "5432"
		}
		dbname := params["dbname"]
		sslmode := params["sslmode"]
		if sslmode == "" {
			sslmode = "disable"
		}

		url := fmt.Sprintf("postgres://%s%s:%s/%s?sslmode=%s",
			auth,
			host,
			port,
			dbname,
			sslmode,
		)
		return url, nil

	case "sqlite":
		return "sqlite://" + dsn, nil

	default:
		return "", fmt.Errorf("unsupported database driver: %s", driver)
	}
}
