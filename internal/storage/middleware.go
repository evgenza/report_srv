package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"
)

// LoggingMiddleware добавляет логирование к операциям хранилища
type LoggingMiddleware struct {
	storage Storage
	logger  *logrus.Logger
}

// NewLoggingMiddleware создает новый logging middleware
func NewLoggingMiddleware(storage Storage, logger *logrus.Logger) Storage {
	return &LoggingMiddleware{
		storage: storage,
		logger:  logger,
	}
}

// Save логирует операцию сохранения
func (m *LoggingMiddleware) Save(ctx context.Context, key string, reader io.Reader) error {
	start := time.Now()
	logger := m.logger.WithFields(logrus.Fields{
		"operation": "save",
		"key":       key,
	})

	logger.Debug("Начало сохранения файла")

	err := m.storage.Save(ctx, key, reader)

	duration := time.Since(start)
	if err != nil {
		logger.WithError(err).WithField("duration", duration).Error("Ошибка сохранения файла")
	} else {
		logger.WithField("duration", duration).Info("Файл сохранен успешно")
	}

	return err
}

// Get логирует операцию получения
func (m *LoggingMiddleware) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	start := time.Now()
	logger := m.logger.WithFields(logrus.Fields{
		"operation": "get",
		"key":       key,
	})

	logger.Debug("Начало получения файла")

	reader, err := m.storage.Get(ctx, key)

	duration := time.Since(start)
	if err != nil {
		logger.WithError(err).WithField("duration", duration).Error("Ошибка получения файла")
	} else {
		logger.WithField("duration", duration).Info("Файл получен успешно")
	}

	return reader, err
}

// Delete логирует операцию удаления
func (m *LoggingMiddleware) Delete(ctx context.Context, key string) error {
	start := time.Now()
	logger := m.logger.WithFields(logrus.Fields{
		"operation": "delete",
		"key":       key,
	})

	logger.Debug("Начало удаления файла")

	err := m.storage.Delete(ctx, key)

	duration := time.Since(start)
	if err != nil {
		logger.WithError(err).WithField("duration", duration).Error("Ошибка удаления файла")
	} else {
		logger.WithField("duration", duration).Info("Файл удален успешно")
	}

	return err
}

// Остальные методы просто делегируют вызовы
func (m *LoggingMiddleware) Exists(ctx context.Context, key string) (bool, error) {
	return m.storage.Exists(ctx, key)
}

func (m *LoggingMiddleware) GetMetadata(ctx context.Context, key string) (*FileMetadata, error) {
	return m.storage.GetMetadata(ctx, key)
}

func (m *LoggingMiddleware) GetSize(ctx context.Context, key string) (int64, error) {
	return m.storage.GetSize(ctx, key)
}

func (m *LoggingMiddleware) GetURL(ctx context.Context, key string) (string, error) {
	return m.storage.GetURL(ctx, key)
}

func (m *LoggingMiddleware) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	return m.storage.GetPresignedURL(ctx, key, expiration)
}

func (m *LoggingMiddleware) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	return m.storage.List(ctx, prefix)
}

func (m *LoggingMiddleware) Copy(ctx context.Context, srcKey, dstKey string) error {
	return m.storage.Copy(ctx, srcKey, dstKey)
}

func (m *LoggingMiddleware) Move(ctx context.Context, srcKey, dstKey string) error {
	return m.storage.Move(ctx, srcKey, dstKey)
}

func (m *LoggingMiddleware) JoinPath(elem ...string) string {
	return m.storage.JoinPath(elem...)
}

func (m *LoggingMiddleware) ValidateKey(key string) error {
	return m.storage.ValidateKey(key)
}

// RetryMiddleware добавляет retry логику к операциям хранилища
type RetryMiddleware struct {
	storage    Storage
	maxRetries int
	retryDelay time.Duration
	logger     *logrus.Logger
}

// NewRetryMiddleware создает новый retry middleware
func NewRetryMiddleware(storage Storage, maxRetries int, retryDelay time.Duration, logger *logrus.Logger) Storage {
	return &RetryMiddleware{
		storage:    storage,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
		logger:     logger,
	}
}

// Save выполняет операцию сохранения с retry
func (m *RetryMiddleware) Save(ctx context.Context, key string, reader io.Reader) error {
	return m.retryOperation(ctx, "save", func() error {
		return m.storage.Save(ctx, key, reader)
	})
}

// Get выполняет операцию получения с retry
func (m *RetryMiddleware) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	var result io.ReadCloser
	err := m.retryOperation(ctx, "get", func() error {
		var err error
		result, err = m.storage.Get(ctx, key)
		return err
	})
	return result, err
}

// Delete выполняет операцию удаления с retry
func (m *RetryMiddleware) Delete(ctx context.Context, key string) error {
	return m.retryOperation(ctx, "delete", func() error {
		return m.storage.Delete(ctx, key)
	})
}

// retryOperation выполняет операцию с retry логикой
func (m *RetryMiddleware) retryOperation(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= m.maxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Проверяем, стоит ли повторять операцию
		if !m.shouldRetry(lastErr) {
			break
		}

		if attempt < m.maxRetries {
			m.logger.WithFields(logrus.Fields{
				"operation":   operation,
				"attempt":     attempt + 1,
				"max_retries": m.maxRetries,
			}).WithError(lastErr).Warn("Повтор операции после ошибки")

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(m.retryDelay):
				// Продолжаем
			}
		}
	}

	return lastErr
}

// shouldRetry определяет, стоит ли повторять операцию
func (m *RetryMiddleware) shouldRetry(err error) bool {
	// Здесь можно добавить логику для определения, какие ошибки стоит повторять
	return true
}

func (m *RetryMiddleware) Exists(ctx context.Context, key string) (bool, error) {
	return m.storage.Exists(ctx, key)
}

func (m *RetryMiddleware) GetMetadata(ctx context.Context, key string) (*FileMetadata, error) {
	return m.storage.GetMetadata(ctx, key)
}

func (m *RetryMiddleware) GetSize(ctx context.Context, key string) (int64, error) {
	return m.storage.GetSize(ctx, key)
}

func (m *RetryMiddleware) GetURL(ctx context.Context, key string) (string, error) {
	return m.storage.GetURL(ctx, key)
}

func (m *RetryMiddleware) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	return m.storage.GetPresignedURL(ctx, key, expiration)
}

func (m *RetryMiddleware) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	return m.storage.List(ctx, prefix)
}

func (m *RetryMiddleware) Copy(ctx context.Context, srcKey, dstKey string) error {
	return m.storage.Copy(ctx, srcKey, dstKey)
}

func (m *RetryMiddleware) Move(ctx context.Context, srcKey, dstKey string) error {
	return m.storage.Move(ctx, srcKey, dstKey)
}

func (m *RetryMiddleware) JoinPath(elem ...string) string {
	return m.storage.JoinPath(elem...)
}

func (m *RetryMiddleware) ValidateKey(key string) error {
	return m.storage.ValidateKey(key)
}

// ValidationMiddleware добавляет валидацию к операциям хранилища
type ValidationMiddleware struct {
	storage Storage
	logger  *logrus.Logger
}

// NewValidationMiddleware создает новый validation middleware
func NewValidationMiddleware(storage Storage, logger *logrus.Logger) Storage {
	return &ValidationMiddleware{
		storage: storage,
		logger:  logger,
	}
}

// Save выполняет валидацию перед сохранением
func (m *ValidationMiddleware) Save(ctx context.Context, key string, reader io.Reader) error {
	if err := m.validateKey(key); err != nil {
		return err
	}
	return m.storage.Save(ctx, key, reader)
}

// Get выполняет валидацию перед получением
func (m *ValidationMiddleware) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	return m.storage.Get(ctx, key)
}

// Delete выполняет валидацию перед удалением
func (m *ValidationMiddleware) Delete(ctx context.Context, key string) error {
	if err := m.validateKey(key); err != nil {
		return err
	}
	return m.storage.Delete(ctx, key)
}

// validateKey проверяет корректность ключа
func (m *ValidationMiddleware) validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("ключ файла не может быть пустым")
	}
	return nil
}

func (m *ValidationMiddleware) Exists(ctx context.Context, key string) (bool, error) {
	if err := m.validateKey(key); err != nil {
		return false, err
	}
	return m.storage.Exists(ctx, key)
}

func (m *ValidationMiddleware) GetMetadata(ctx context.Context, key string) (*FileMetadata, error) {
	if err := m.validateKey(key); err != nil {
		return nil, err
	}
	return m.storage.GetMetadata(ctx, key)
}

func (m *ValidationMiddleware) GetSize(ctx context.Context, key string) (int64, error) {
	if err := m.validateKey(key); err != nil {
		return 0, err
	}
	return m.storage.GetSize(ctx, key)
}

func (m *ValidationMiddleware) GetURL(ctx context.Context, key string) (string, error) {
	if err := m.validateKey(key); err != nil {
		return "", err
	}
	return m.storage.GetURL(ctx, key)
}

func (m *ValidationMiddleware) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	if err := m.validateKey(key); err != nil {
		return "", err
	}
	return m.storage.GetPresignedURL(ctx, key, expiration)
}

func (m *ValidationMiddleware) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	return m.storage.List(ctx, prefix)
}

func (m *ValidationMiddleware) Copy(ctx context.Context, srcKey, dstKey string) error {
	if err := m.validateKey(srcKey); err != nil {
		return err
	}
	if err := m.validateKey(dstKey); err != nil {
		return err
	}
	return m.storage.Copy(ctx, srcKey, dstKey)
}

func (m *ValidationMiddleware) Move(ctx context.Context, srcKey, dstKey string) error {
	if err := m.validateKey(srcKey); err != nil {
		return err
	}
	if err := m.validateKey(dstKey); err != nil {
		return err
	}
	return m.storage.Move(ctx, srcKey, dstKey)
}

func (m *ValidationMiddleware) JoinPath(elem ...string) string {
	return m.storage.JoinPath(elem...)
}

func (m *ValidationMiddleware) ValidateKey(key string) error {
	return m.storage.ValidateKey(key)
}
