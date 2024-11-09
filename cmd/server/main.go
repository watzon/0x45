package main

import (
	"log"

	"github.com/watzon/paste69/internal/config"
	"github.com/watzon/paste69/internal/database"
	"github.com/watzon/paste69/internal/server"
	"github.com/watzon/paste69/internal/storage"
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

	// Initialize storage
	store, err := storage.NewStore(cfg)
	if err != nil {
		log.Fatalf("Error initializing storage: %v", err)
	}

	// Create and setup server
	srv := server.New(db, store, cfg)
	srv.SetupRoutes()

	// Start server
	log.Printf("Starting server on %s", cfg.Server.Address)
	log.Fatal(srv.Start(cfg.Server.Address))
}
