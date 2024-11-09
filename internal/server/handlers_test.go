package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/handlebars/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/watzon/paste69/internal/config"
	"github.com/watzon/paste69/internal/database"
	"github.com/watzon/paste69/internal/middleware"
	"github.com/watzon/paste69/internal/models"
	"go.uber.org/zap"
)

// MockStore implements the storage.Store interface for testing
type MockStore struct {
	mock.Mock
	contents map[string][]byte // In-memory storage for test data
}

func NewMockStore() *MockStore {
	return &MockStore{
		contents: make(map[string][]byte),
	}
}

func (m *MockStore) Save(content io.Reader, filename string) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("test/%s", filename)
	m.contents[path] = data
	return path, nil
}

func (m *MockStore) Get(path string) (io.ReadCloser, error) {
	if data, ok := m.contents[path]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return nil, fmt.Errorf("not found: %s", path)
}

func (m *MockStore) Delete(path string) error {
	delete(m.contents, path)
	return nil
}

func (m *MockStore) GetURL(path string) string {
	return fmt.Sprintf("http://test.local/%s", path)
}

func (m *MockStore) GetSize(path string) (int64, error) {
	if data, ok := m.contents[path]; ok {
		return int64(len(data)), nil
	}
	return 0, fmt.Errorf("not found")
}

func (m *MockStore) SetExpiry(path string, expiry time.Time) error {
	return nil
}

func (m *MockStore) Type() string {
	return "mock"
}

func setupTestServerWithMockStore(t *testing.T) (*Server, *fiber.App, *MockStore) {
	store := NewMockStore()

	// Initialize test database
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("failed to setup test database: %v", err)
	}

	config := &config.Config{}
	config.Server.BaseURL = "http://localhost:3000"
	config.Server.MaxUploadSize = 1024 * 1024 * 10 // 10MB
	config.Server.Cleanup.MaxAge = "24h"

	// Create a temporary test template
	testTemplate := `
<div class="paste-header">
    <div class="paste-info">
        <h2>{{filename}}</h2>
        <div class="metadata">
            <span title="{{created}}">Created: {{created}}</span>
            {{#if expires}}
            <span title="{{expires}}">Expires: {{expires}}</span>
            {{/if}}
            <span>Language: {{language}}</span>
        </div>
    </div>
    <div class="actions">
        <button class="action-btn" data-clipboard="#paste-content">Copy</button>
        <a href="/raw/{{id}}" class="action-btn">Raw</a>
        <a href="/download/{{id}}" class="action-btn">Download</a>
    </div>
</div>

<div id="paste-content" class="paste-content">
    {{{content}}}
</div>`

	// Write test templates to temporary directory
	tmpDir := t.TempDir()
	err = os.MkdirAll(filepath.Join(tmpDir, "views", "layouts"), 0755)
	if err != nil {
		t.Fatalf("failed to create template directories: %v", err)
	}

	// Write paste template
	err = os.WriteFile(filepath.Join(tmpDir, "views", "paste.hbs"), []byte(testTemplate), 0644)
	if err != nil {
		t.Fatalf("failed to write paste template: %v", err)
	}

	// Write main layout template
	mainLayout := `
<!DOCTYPE html>
<html>
<head>
    <title>Test Layout</title>
</head>
<body>
    {{{embed}}}
</body>
</html>`

	err = os.WriteFile(filepath.Join(tmpDir, "views", "layouts", "main.hbs"), []byte(mainLayout), 0644)
	if err != nil {
		t.Fatalf("failed to write main layout template: %v", err)
	}

	// Initialize Fiber with handlebars engine
	app := fiber.New(fiber.Config{
		Views: handlebars.New(filepath.Join(tmpDir, "views"), ".hbs"),
	})

	srv := &Server{
		app:    app,
		store:  store,
		config: config,
		logger: zap.NewNop(),
		db:     db,
		auth:   middleware.NewAuthMiddleware(db.DB),
	}

	srv.SetupRoutes()

	return srv, app, store
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB() (*database.Database, error) {
	config := &config.Config{}
	config.Database.Driver = "sqlite"
	config.Database.Name = ":memory:"

	db, err := database.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Auto migrate the schemas
	err = db.AutoMigrate(&models.Paste{}, &models.APIKey{}, &models.Shortlink{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func TestHandleUpload(t *testing.T) {
	srv, app, store := setupTestServerWithMockStore(t)

	t.Run("Multipart Upload", func(t *testing.T) {
		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.txt")
		assert.NoError(t, err)

		content := []byte("Hello, World!")
		_, err = part.Write(content)
		assert.NoError(t, err)
		err = writer.Close()
		assert.NoError(t, err)

		// Create request
		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Test
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verify response
		var result struct {
			Success bool         `json:"success"`
			Data    models.Paste `json:"data"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "test.txt", result.Data.Filename)
		assert.Equal(t, int64(13), result.Data.Size)

		// Verify the paste was stored in the database
		var storedPaste models.Paste
		err = srv.db.First(&storedPaste, "id = ?", result.Data.ID).Error
		assert.NoError(t, err)

		// Verify content was stored
		storedContent, err := store.Get(storedPaste.StoragePath)
		assert.NoError(t, err)
		data, err := io.ReadAll(storedContent)
		assert.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("JSON Upload", func(t *testing.T) {
		payload := map[string]interface{}{
			"content":  "Hello, JSON!",
			"filename": "test.json",
		}
		jsonData, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result struct {
			Success bool `json:"success"`
			Data    struct {
				ID       string `json:"id"`
				Filename string `json:"filename"`
				Size     int64  `json:"size"`
			} `json:"data"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "test.json", result.Data.Filename)
	})

	t.Run("Raw Upload", func(t *testing.T) {
		content := []byte("Raw content test")
		req := httptest.NewRequest("POST", "/?filename=test.txt", bytes.NewReader(content))
		req.Header.Set("Content-Type", "text/plain")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result struct {
			Success bool `json:"success"`
			Data    struct {
				ID       string `json:"id"`
				Filename string `json:"filename"`
				Size     int64  `json:"size"`
			} `json:"data"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "test.txt", result.Data.Filename)
		assert.Equal(t, int64(len(content)), result.Data.Size)
	})
}

func TestHandleView(t *testing.T) {
	srv, app, store := setupTestServerWithMockStore(t)

	t.Run("View Text Paste", func(t *testing.T) {
		// Create test content in storage
		content := []byte("Hello, World!")
		storagePath := "test/view.txt"
		store.contents[storagePath] = content

		paste := &models.Paste{
			ID:          "test123",
			MimeType:    "text/plain",
			Filename:    "test.txt",
			Size:        int64(len(content)),
			StoragePath: storagePath,
			CreatedAt:   time.Now(),
		}
		err := srv.db.Create(paste).Error
		assert.NoError(t, err)

		// Clean up after test
		defer func() {
			srv.db.Unscoped().Delete(paste)
			delete(store.contents, storagePath)
		}()

		req := httptest.NewRequest("GET", "/"+paste.ID, nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		bodyStr := string(body)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
		assert.Contains(t, bodyStr, "Hello, World!")
		assert.Contains(t, bodyStr, "test.txt")
		assert.Contains(t, bodyStr, "plaintext")
	})

	t.Run("View Non-Text Paste", func(t *testing.T) {
		content := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic numbers
		storagePath := "test/image.png"
		store.contents[storagePath] = content

		paste := &models.Paste{
			ID:          "test456",
			MimeType:    "image/png",
			Filename:    "test.png",
			Size:        int64(len(content)),
			StoragePath: storagePath,
			CreatedAt:   time.Now(),
		}
		err := srv.db.Create(paste).Error
		assert.NoError(t, err)

		defer func() {
			srv.db.Unscoped().Delete(paste)
			delete(store.contents, storagePath)
		}()

		req := httptest.NewRequest("GET", "/"+paste.ID, nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusTemporaryRedirect, resp.StatusCode)
		assert.Equal(t, "/download/"+paste.ID, resp.Header.Get("Location"))
	})

	t.Run("View Non-Existent Paste", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("View Expired Paste", func(t *testing.T) {
		expiredTime := time.Now().Add(-24 * time.Hour)
		paste := &models.Paste{
			ID:        "expired123",
			MimeType:  "text/plain",
			Filename:  "expired.txt",
			ExpiresAt: &expiredTime,
			CreatedAt: time.Now(),
		}
		err := srv.db.Create(paste).Error
		assert.NoError(t, err)

		defer srv.db.Unscoped().Delete(paste)

		req := httptest.NewRequest("GET", "/"+paste.ID, nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
