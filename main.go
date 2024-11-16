package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/database"
	"github.com/watzon/0x45/internal/server"
	"github.com/watzon/0x45/internal/storage"
)

// @title 0x45 API
// @version 1.0
// @description API for 0x45
// @license.name MIT
// @license.url https://github.com/watzon/0x45/blob/main/LICENSE
// @host localhost:3000
// @BasePath /
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Initialize server with storage manager
	srv := server.New(db, storageManager, cfg)

	// Create and setup server
	srv.SetupRoutes()

	go func() {
		// Start server
		log.Printf("Starting server on %s", cfg.Server.Address)
		log.Fatal(srv.Start(cfg.Server.Address))
	}()

	// Wait for shutdown signal and initiate graceful shutdown once received.
	<-ctx.Done()
	defer func() {
		if err := srv.Cleanup(); err != nil {
			log.Printf("failed cleaning up server: %v", err)
		}
	}()
	log.Print("shutdown signal received, initiate shutdown")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("failed shutting down gracefully: %v", err)
	}
}
