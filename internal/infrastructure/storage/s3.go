package storage

import (
	"io/ioutil"
	"path/filepath"
)

// S3Storage — упрощённая реализация, читающая шаблоны из локальной директории,
// которая имитирует бакет S3. В реальном проекте здесь использовался бы AWS SDK.
type S3Storage struct {
	BasePath string
}

// Download возвращает содержимое объекта с указанным ключом.
func (s S3Storage) Download(key string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(s.BasePath, key))
}
