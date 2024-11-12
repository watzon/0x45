package main

import (
	"log"

	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/database"
	"github.com/watzon/0x45/internal/server"
	"github.com/watzon/0x45/internal/storage"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(cfg); err != nil {
		log.Fatalf("Error running migrations: %v", err)
	}

	// Initialize storage manager
	storageManager, err := storage.NewStorageManager(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize server with storage manager
	srv := server.New(db, storageManager, cfg)

	// Create and setup server
	srv.SetupRoutes()

	// Start server
	log.Printf("Starting server on %s", cfg.Server.Address)
	log.Fatal(srv.Start(cfg.Server.Address))
}
