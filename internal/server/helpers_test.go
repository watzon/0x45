package server

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/watzon/paste69/internal/database"
	"github.com/watzon/paste69/internal/models"
	"github.com/watzon/paste69/internal/testutil"
	"go.uber.org/zap"
)

func setupTestServer(t *testing.T) (*Server, *fiber.App) {
	cfg := testutil.TestConfig(t)
	store := testutil.NewTestStorage(t)
	logger, _ := zap.NewDevelopment()
	db := newTestDB(t)

	srv := &Server{
		config: cfg,
		db:     db,
		store:  store,
		logger: logger,
	}

	app := fiber.New()
	return srv, app
}

func TestCreatePasteFromRaw(t *testing.T) {
	srv, app := setupTestServer(t)

	t.Run("Basic Text Content", func(t *testing.T) {
		content := []byte("Hello, World!")

		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetBody(content)
		ctx.Request.Header.SetContentType("text/plain")

		fctx := app.AcquireCtx(ctx)
		defer app.ReleaseCtx(fctx)

		opts := &PasteOptions{
			Filename: "test.txt",
		}

		paste, err := srv.createPasteFromRaw(fctx, content, opts)
		assert.NoError(t, err)
		assert.NotNil(t, paste)
		assert.Equal(t, int64(len(content)), paste.Size)
		// Check if MIME type starts with text/plain (ignore charset)
		assert.True(t, strings.HasPrefix(paste.MimeType, "text/plain"))
	})

	t.Run("With Expiry", func(t *testing.T) {
		content := []byte("Temporary content")

		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetBody(content)

		fctx := app.AcquireCtx(ctx)
		defer app.ReleaseCtx(fctx)

		opts := &PasteOptions{
			Filename:  "temp.txt",
			ExpiresIn: "1h",
		}

		paste, err := srv.createPasteFromRaw(fctx, content, opts)
		assert.NoError(t, err)
		assert.NotNil(t, paste)
		assert.NotNil(t, paste.ExpiresAt)
		assert.True(t, paste.ExpiresAt.After(time.Now()))
	})
}

func TestCreatePasteFromMultipart(t *testing.T) {
	srv, app := setupTestServer(t)

	t.Run("Valid File Upload", func(t *testing.T) {
		content := []byte("Hello, Multipart!")
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Create the form file
		part, err := writer.CreateFormFile("file", "test.txt")
		assert.NoError(t, err)

		_, err = part.Write(content)
		assert.NoError(t, err)

		err = writer.Close()
		assert.NoError(t, err)

		// Create request context
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetBody(body.Bytes())
		ctx.Request.Header.SetContentType(writer.FormDataContentType())

		fctx := app.AcquireCtx(ctx)
		defer app.ReleaseCtx(fctx)

		// Parse the form
		form, err := multipart.NewReader(bytes.NewReader(body.Bytes()), writer.Boundary()).ReadForm(32 << 20)
		assert.NoError(t, err)

		file := form.File["file"][0]
		opts := &PasteOptions{
			Filename: file.Filename,
		}

		paste, err := srv.createPasteFromMultipart(fctx, file, opts)
		assert.NoError(t, err)
		assert.NotNil(t, paste)
		assert.Equal(t, "test.txt", paste.Filename)
		assert.Equal(t, int64(len(content)), paste.Size)
		assert.True(t, strings.HasPrefix(paste.MimeType, "text/plain"))
	})

	t.Run("Empty File", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Create empty file part
		_, err := writer.CreateFormFile("file", "empty.txt")
		assert.NoError(t, err)

		err = writer.Close()
		assert.NoError(t, err)

		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetBody(body.Bytes())
		ctx.Request.Header.SetContentType(writer.FormDataContentType())

		fctx := app.AcquireCtx(ctx)
		defer app.ReleaseCtx(fctx)

		form, err := multipart.NewReader(bytes.NewReader(body.Bytes()), writer.Boundary()).ReadForm(32 << 20)
		assert.NoError(t, err)

		file := form.File["file"][0]
		opts := &PasteOptions{
			Filename: file.Filename,
		}

		paste, err := srv.createPasteFromMultipart(fctx, file, opts)
		assert.NoError(t, err)
		assert.NotNil(t, paste)
		assert.Equal(t, "empty.txt", paste.Filename)
		assert.Equal(t, int64(0), paste.Size)
	})

	t.Run("Large File", func(t *testing.T) {
		// Create a large file that's just under the default limit
		content := make([]byte, 10<<20) // 10MB
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", "large.bin")
		assert.NoError(t, err)

		_, err = part.Write(content)
		assert.NoError(t, err)

		err = writer.Close()
		assert.NoError(t, err)

		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetBody(body.Bytes())
		ctx.Request.Header.SetContentType(writer.FormDataContentType())

		fctx := app.AcquireCtx(ctx)
		defer app.ReleaseCtx(fctx)

		form, err := multipart.NewReader(bytes.NewReader(body.Bytes()), writer.Boundary()).ReadForm(32 << 20)
		assert.NoError(t, err)

		file := form.File["file"][0]
		opts := &PasteOptions{
			Filename: file.Filename,
		}

		paste, err := srv.createPasteFromMultipart(fctx, file, opts)
		assert.NoError(t, err)
		assert.NotNil(t, paste)
		assert.Equal(t, "large.bin", paste.Filename)
		assert.Equal(t, int64(len(content)), paste.Size)
		assert.True(t, isBinaryContent(paste.MimeType))
	})
}

func TestFindPaste(t *testing.T) {
	srv, _ := setupTestServer(t)

	t.Run("Find Existing Paste", func(t *testing.T) {
		paste := &models.Paste{
			Filename: "test.txt",
			MimeType: "text/plain",
			Size:     100,
		}
		err := srv.db.Create(paste).Error
		assert.NoError(t, err)

		found, err := srv.findPaste(paste.ID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, paste.ID, found.ID)
	})

	t.Run("Find Expired Paste", func(t *testing.T) {
		paste := &models.Paste{
			Filename:  "expired.txt",
			ExpiresAt: &time.Time{}, // Set to zero time to ensure it's expired
		}
		err := srv.db.Create(paste).Error
		assert.NoError(t, err)

		found, err := srv.findPaste(paste.ID)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Equal(t, fiber.StatusNotFound, err.(*fiber.Error).Code)
	})
}

func TestMimeTypeHelpers(t *testing.T) {
	tests := []struct {
		mimeType string
		isText   bool
		isImage  bool
		isBinary bool
	}{
		{"text/plain", true, false, false},
		{"text/plain; charset=utf-8", true, false, false},
		{"text/html", true, false, false},
		{"application/json", true, false, false},
		{"application/javascript", true, false, false},
		{"image/jpeg", false, true, false},
		{"image/png", false, true, false},
		{"application/pdf", false, false, true},
		{"application/octet-stream", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			assert.Equal(t, tt.isText, isTextContent(tt.mimeType))
			assert.Equal(t, tt.isImage, isImageContent(tt.mimeType))
			assert.Equal(t, tt.isBinary, isBinaryContent(tt.mimeType))
		})
	}
}

func TestGetStatsHistory(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Create some test data
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	// Create pastes
	paste1 := &models.Paste{Size: 100, CreatedAt: now}
	paste2 := &models.Paste{Size: 200, CreatedAt: yesterday}
	srv.db.Create(paste1)
	srv.db.Create(paste2)

	// Create shortlinks
	link1 := &models.Shortlink{TargetURL: "https://example.com", CreatedAt: now, APIKey: "test"}
	link2 := &models.Shortlink{TargetURL: "https://example.org", CreatedAt: yesterday, APIKey: "test"}
	srv.db.Create(link1)
	srv.db.Create(link2)

	// Get stats for last 2 days
	stats, err := srv.getStatsHistory(2)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Len(t, stats.Pastes, 2)
	assert.Len(t, stats.URLs, 2)
	assert.Len(t, stats.Storage, 2)
}

// Helper function to create a test database
func newTestDB(t *testing.T) *database.Database {
	cfg := testutil.TestConfig(t)
	cfg.Database.Driver = "sqlite"
	cfg.Database.Name = ":memory:"

	db, err := database.New(cfg)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Auto-migrate all required models
	err = db.AutoMigrate(
		&models.Paste{},
		&models.Shortlink{},
		&models.APIKey{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}
