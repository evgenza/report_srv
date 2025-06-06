package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"report_srv/internal/config"
)

// Storage interface for file storage operations
type Storage interface {
	Save(ctx context.Context, key string, data io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// LocalStorage implements Storage interface for local file system
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{
		basePath: basePath,
	}
}

// Save saves a file to local storage
func (l *LocalStorage) Save(ctx context.Context, key string, data io.Reader) error {
	fullPath := filepath.Join(l.basePath, key)

	// Создаем директорию если она не существует
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Создаем файл
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Копируем данные
	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Get retrieves a file from local storage
func (l *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(l.basePath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes a file from local storage
func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(l.basePath, key)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists checks if a file exists in local storage
func (l *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := filepath.Join(l.basePath, key)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// NewStorageFromConfig creates a storage instance based on configuration
func NewStorageFromConfig(cfg config.Config) (Storage, error) {
	switch cfg.Storage.Type {
	case "s3":
		s3cfg := S3Config{
			Region:    cfg.Storage.S3.Region,
			Bucket:    cfg.Storage.S3.Bucket,
			Endpoint:  cfg.Storage.S3.Endpoint,
			AccessKey: cfg.Storage.S3.AccessKey,
			SecretKey: cfg.Storage.S3.SecretKey,
		}
		return NewS3Storage(s3cfg)
	case "local":
		return NewLocalStorage(cfg.Storage.BasePath), nil
	default:
		return NewLocalStorage(cfg.Storage.BasePath), nil
	}
}
