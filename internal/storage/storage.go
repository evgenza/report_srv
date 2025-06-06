package storage

import (
	"context"
	"io"
)

// Storage defines the interface for file storage operations
type Storage interface {
	// Save saves a file to storage
	Save(ctx context.Context, key string, reader io.Reader) error

	// Get retrieves a file from storage
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes a file from storage
	Delete(ctx context.Context, key string) error

	// GetURL returns a pre-signed URL for the file
	GetURL(ctx context.Context, key string) (string, error)

	// JoinPath joins path elements
	JoinPath(elem ...string) string
}
