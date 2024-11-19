package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/database"
	"github.com/watzon/0x45/internal/models"
	"github.com/watzon/0x45/internal/server"
	"github.com/watzon/0x45/internal/storage"
	"github.com/watzon/0x45/internal/utils/bytesize"
	"go.uber.org/zap"
)

type TestEnv struct {
	App       *fiber.App
	Server    *server.Server
	DB        *database.Database
	Config    *config.Config
	Storage   *storage.StorageManager
	Logger    *zap.Logger
	TempDir   string
	CleanupFn func()
}

func SetupTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	// Create temp directory for uploads and views
	tempDir, err := os.MkdirTemp("", "0x45-test-*")
	if err != nil {
		t.Fatal(err)
	}

	// Create views directory and copy templates if needed
	viewsDir := filepath.Join(tempDir, "views")
	if err := os.MkdirAll(viewsDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatal(err)
	}

	pubDir := filepath.Join(tempDir, "public")
	if err := os.MkdirAll(pubDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatal(err)
	}

	// Create test config
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Name:   tempDir + "/paste69.db",
		},
		Storage: []config.StorageConfig{
			{
				Name:      "local",
				Type:      "local",
				Path:      tempDir,
				IsDefault: true,
			},
		},
		Server: config.ServerConfig{
			MaxUploadSize:     bytesize.ByteSize(10 * 1024 * 1024), // 10MB
			DefaultUploadSize: bytesize.ByteSize(5 * 1024 * 1024),  // 5MB
			APIUploadSize:     bytesize.ByteSize(10 * 1024 * 1024), // 10MB
			AppName:           "0x45-test",
			ServerHeader:      "0x45-test",
			ViewsDirectory:    viewsDir,
			PublicDirectory:   pubDir,
		},
		Retention: config.RetentionConfig{
			NoKey: config.RetentionLimitConfig{
				MinAge: 1,  // 1 day minimum
				MaxAge: 30, // 30 days maximum
			},
			WithKey: config.RetentionLimitConfig{
				MinAge: 1,   // 1 day minimum
				MaxAge: 365, // 365 days maximum
			},
		},
	}

	// Initialize test logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatal(err)
	}
	defer func() { _ = logger.Sync() }()

	// Create server instance with modified config
	origCfg := *cfg                                                // Make a copy of the original config
	cfg.Server.MaxUploadSize = bytesize.ByteSize(10 * 1024 * 1024) // 10MB
	cfg.Server.AppName = "0x45-test"
	cfg.Server.ServerHeader = "0x45-test"

	// Create server instance
	srv := server.New(cfg, logger)
	srv.SetupRoutes()

	// Add test API key
	err = srv.GetDB().Create(&models.APIKey{Email: "test@example.com", Key: "test-api-key", Verified: true, AllowShortlinks: true}).Error
	if err != nil {
		logger.Error("Error creating test API key", zap.Error(err))
		os.RemoveAll(tempDir)
		t.Fatal(err)
	}

	cleanup := func() {
		if err := srv.Cleanup(); err != nil {
			logger.Error("failed cleaning up server", zap.Error(err))
		}
		os.RemoveAll(tempDir)
	}

	return &TestEnv{
		App:       srv.GetApp(),
		Server:    srv,
		DB:        srv.GetDB(),
		Config:    &origCfg,
		Storage:   srv.GetStorage(),
		Logger:    logger,
		TempDir:   tempDir,
		CleanupFn: cleanup,
	}
}
