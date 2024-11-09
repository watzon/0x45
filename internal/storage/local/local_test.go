package local

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalStorage(t *testing.T) {
	tempDir := t.TempDir()
	baseURL := "http://localhost:3000"
	store, err := New(tempDir, baseURL)
	assert.NoError(t, err)

	t.Run("Save and Get", func(t *testing.T) {
		content := "test content"
		reader := strings.NewReader(content)

		// Test Save
		path, err := store.Save(reader, "test.txt")
		assert.NoError(t, err)
		assert.NotEmpty(t, path)

		// Test Get
		file, err := store.Get(path)
		assert.NoError(t, err)
		defer file.Close()

		data, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("Delete", func(t *testing.T) {
		content := "delete test"
		reader := strings.NewReader(content)

		path, err := store.Save(reader, "delete.txt")
		assert.NoError(t, err)

		err = store.Delete(path)
		assert.NoError(t, err)

		_, err = os.Stat(filepath.Join(tempDir, path))
		assert.True(t, os.IsNotExist(err))
	})
}
