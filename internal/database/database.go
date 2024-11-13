package database

import (
	"fmt"

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

	switch config.Database.Driver {
	case "postgres":
		pgURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			config.Database.User,
			config.Database.Password,
			config.Database.Host,
			config.Database.Port,
			config.Database.Name,
			config.Database.SSLMode,
		)
		return migrations.RunMigrations(pgURL)
	case "sqlite":
		return migrations.RunMigrations("sqlite://" + config.Database.Name)
	}

	return fmt.Errorf("unsupported database driver")
}
