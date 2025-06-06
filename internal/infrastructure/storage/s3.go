package storage

import (
	"io/ioutil"
	"path/filepath"

	"report_srv/internal/config"
)

// S3Storage — упрощённая реализация, читающая шаблоны из локальной директории,
// которая имитирует бакет S3. В реальном проекте здесь использовался бы AWS SDK.
type S3Storage struct {
	BasePath string
}

// NewS3 создаёт хранилище с указанным базовым каталогом.
func NewS3(cfg config.Config) S3Storage {
	return S3Storage{BasePath: cfg.Storage.BasePath}
}

// Download возвращает содержимое объекта с указанным ключом.
func (s S3Storage) Download(key string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(s.BasePath, key))
}
