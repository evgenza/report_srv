package storage

import (
	"io/ioutil"
	"path/filepath"
)

// S3Storage is a placeholder implementation that reads templates from a local directory
// representing an S3 bucket. In a real implementation this would use the AWS SDK.
type S3Storage struct {
	BasePath string
}

// Download returns the contents of the object identified by key.
func (s S3Storage) Download(key string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(s.BasePath, key))
}
