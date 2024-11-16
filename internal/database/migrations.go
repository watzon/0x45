package database

import (
	"fmt"

	"github.com/watzon/0x45/internal/models"
	"gorm.io/gorm"
)

// Models is a list of all models that need to be migrated
var Models = []interface{}{
	&models.Paste{},
	&models.APIKey{},
	&models.Shortlink{},
	&models.AnalyticsEvent{},
}

// RunMigrations runs all necessary database migrations
func RunMigrations(db *gorm.DB) error {
	if err := runGormMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := createConstraints(db); err != nil {
		return fmt.Errorf("failed to create constraints: %w", err)
	}

	return nil
}

func runGormMigrations(db *gorm.DB) error {
	// Enable foreign key constraints for SQLite
	// This is a no-op for PostgreSQL as it's enabled by default
	db = db.Set("gorm:auto_foreign_key", true)

	// Run migrations for each model
	return db.AutoMigrate(Models...)
}

func GetMigrator(db *gorm.DB) gorm.Migrator {
	return db.Migrator()
}

func DropAllTables(db *gorm.DB) error {
	return db.Migrator().DropTable(Models...)
}

func HasTable(db *gorm.DB, model interface{}) bool {
	return db.Migrator().HasTable(model)
}

func createConstraints(db *gorm.DB) error {
	// Add any custom constraints here that GORM's automigrate doesn't handle
	// For example, if you need to add a custom index or foreign key that
	// isn't defined in the model tags

	return nil
}
