package local

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type LocalStore struct {
	basePath string
	baseURL  string
}

func New(basePath, baseURL string) (*LocalStore, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStore{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

func (s *LocalStore) Save(content io.Reader, filename string) (string, error) {
	// Generate unique path
	storagePath := filepath.Join(time.Now().Format("2006/01/02"), filename)
	fullPath := filepath.Join(s.basePath, storagePath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Copy content
	if _, err := io.Copy(f, content); err != nil {
		return "", fmt.Errorf("failed to write content: %w", err)
	}

	return storagePath, nil
}

func (s *LocalStore) Get(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)
	return os.Open(fullPath)
}

func (s *LocalStore) Delete(path string) error {
	fullPath := filepath.Join(s.basePath, path)
	return os.Remove(fullPath)
}

func (s *LocalStore) GetURL(path string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, path)
}

func (s *LocalStore) GetSize(path string) (int64, error) {
	fullPath := filepath.Join(s.basePath, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (s *LocalStore) SetExpiry(path string, expiry time.Time) error {
	// Local filesystem doesn't support expiry directly
	// This would be handled by a cleanup routine
	return nil
}

func (s *LocalStore) Type() string {
	return "local"
}
