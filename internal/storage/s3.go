package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"report_srv/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"
)

const (
	// Типы хранилищ
	StorageTypeLocal = "local"
	StorageTypeS3    = "s3"

	// Таймауты по умолчанию
	DefaultUploadTimeout    = 30 * time.Minute
	DefaultDownloadTimeout  = 10 * time.Minute
	DefaultOperationTimeout = 30 * time.Second

	// Настройки retry
	DefaultMaxRetries = 3
	DefaultRetryDelay = time.Second
)

// Storage интерфейс для работы с файловыми хранилищами
type Storage interface {
	// Основные операции
	Save(ctx context.Context, key string, reader io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Метаданные
	GetMetadata(ctx context.Context, key string) (*FileMetadata, error)
	GetSize(ctx context.Context, key string) (int64, error)

	// Работа с URL
	GetURL(ctx context.Context, key string) (string, error)
	GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)

	// Утилиты
	JoinPath(elem ...string) string
	ValidateKey(key string) error

	// Операции с множественными файлами
	List(ctx context.Context, prefix string) ([]FileInfo, error)
	Copy(ctx context.Context, srcKey, dstKey string) error
	Move(ctx context.Context, srcKey, dstKey string) error
}

// FileMetadata метаданные файла
type FileMetadata struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ContentType  string            `json:"content_type"`
	ETag         string            `json:"etag,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// FileInfo информация о файле
type FileInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	IsDir        bool      `json:"is_dir"`
}

// StorageConfig общая конфигурация хранилища
type StorageConfig struct {
	Type            string        `json:"type"`
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	UploadTimeout   time.Duration `json:"upload_timeout"`
	DownloadTimeout time.Duration `json:"download_timeout"`
	EnableMetrics   bool          `json:"enable_metrics"`
	EnableLogging   bool          `json:"enable_logging"`
}

// S3Config конфигурация S3 хранилища
type S3Config struct {
	StorageConfig
	Region            string        `json:"region"`
	Bucket            string        `json:"bucket"`
	Endpoint          string        `json:"endpoint,omitempty"`
	AccessKey         string        `json:"access_key"`
	SecretKey         string        `json:"secret_key"`
	ForcePathStyle    bool          `json:"force_path_style"`
	DisableSSL        bool          `json:"disable_ssl"`
	PresignExpiration time.Duration `json:"presign_expiration"`
}

// LocalConfig конфигурация локального хранилища
type LocalConfig struct {
	StorageConfig
	BasePath    string      `json:"base_path"`
	Permissions os.FileMode `json:"permissions"`
	CreateDirs  bool        `json:"create_dirs"`
}

// StorageFactory фабрика для создания хранилищ
type StorageFactory interface {
	CreateStorage(cfg interface{}) (Storage, error)
	SupportedTypes() []string
}

// StorageBuilder строитель для конфигурации хранилища
type StorageBuilder struct {
	config config.Config
	logger *logrus.Logger
}

// NewStorageBuilder создает новый строитель хранилища
func NewStorageBuilder(cfg config.Config, logger *logrus.Logger) *StorageBuilder {
	return &StorageBuilder{
		config: cfg,
		logger: logger,
	}
}

// Build создает хранилище на основе конфигурации
func (b *StorageBuilder) Build() (Storage, error) {
	factory := NewDefaultStorageFactory(b.logger)

	switch b.config.Storage.Type {
	case StorageTypeS3:
		s3Config := b.buildS3Config()
		storage, err := factory.CreateStorage(s3Config)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания S3 хранилища: %w", err)
		}
		return b.wrapWithMiddleware(storage), nil

	case StorageTypeLocal:
		localConfig := b.buildLocalConfig()
		storage, err := factory.CreateStorage(localConfig)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания локального хранилища: %w", err)
		}
		return b.wrapWithMiddleware(storage), nil

	default:
		return nil, fmt.Errorf("неподдерживаемый тип хранилища: %s", b.config.Storage.Type)
	}
}

// buildS3Config создает конфигурацию S3
func (b *StorageBuilder) buildS3Config() S3Config {
	return S3Config{
		StorageConfig: StorageConfig{
			Type:            StorageTypeS3,
			MaxRetries:      DefaultMaxRetries,
			RetryDelay:      DefaultRetryDelay,
			UploadTimeout:   DefaultUploadTimeout,
			DownloadTimeout: DefaultDownloadTimeout,
			EnableMetrics:   true,
			EnableLogging:   true,
		},
		Region:            b.config.Storage.S3.Region,
		Bucket:            b.config.Storage.S3.Bucket,
		Endpoint:          b.config.Storage.S3.Endpoint,
		AccessKey:         b.config.Storage.S3.AccessKey,
		SecretKey:         b.config.Storage.S3.SecretKey,
		ForcePathStyle:    true,
		PresignExpiration: 1 * time.Hour,
	}
}

// buildLocalConfig создает конфигурацию локального хранилища
func (b *StorageBuilder) buildLocalConfig() LocalConfig {
	return LocalConfig{
		StorageConfig: StorageConfig{
			Type:            StorageTypeLocal,
			MaxRetries:      DefaultMaxRetries,
			RetryDelay:      DefaultRetryDelay,
			UploadTimeout:   DefaultUploadTimeout,
			DownloadTimeout: DefaultDownloadTimeout,
			EnableMetrics:   true,
			EnableLogging:   true,
		},
		BasePath:    b.config.Storage.BasePath,
		Permissions: 0755,
		CreateDirs:  true,
	}
}

// wrapWithMiddleware оборачивает хранилище в middleware
func (b *StorageBuilder) wrapWithMiddleware(storage Storage) Storage {
	// Добавляем логирование
	if b.logger != nil {
		storage = NewLoggingMiddleware(storage, b.logger)
	}

	// Добавляем retry логику
	storage = NewRetryMiddleware(storage, DefaultMaxRetries, DefaultRetryDelay, b.logger)

	// Добавляем валидацию
	storage = NewValidationMiddleware(storage, b.logger)

	return storage
}

// DefaultStorageFactory реализация фабрики хранилищ
type DefaultStorageFactory struct {
	logger *logrus.Logger
}

// NewDefaultStorageFactory создает новую фабрику хранилищ
func NewDefaultStorageFactory(logger *logrus.Logger) StorageFactory {
	return &DefaultStorageFactory{logger: logger}
}

// CreateStorage создает хранилище по конфигурации
func (f *DefaultStorageFactory) CreateStorage(cfg interface{}) (Storage, error) {
	switch config := cfg.(type) {
	case S3Config:
		return NewS3Storage(config, f.logger)
	case LocalConfig:
		return NewLocalStorage(config, f.logger)
	default:
		return nil, fmt.Errorf("неподдерживаемый тип конфигурации: %T", cfg)
	}
}

// SupportedTypes возвращает поддерживаемые типы хранилищ
func (f *DefaultStorageFactory) SupportedTypes() []string {
	return []string{StorageTypeS3, StorageTypeLocal}
}

// S3Storage реализация хранилища для AWS S3
type S3Storage struct {
	client            *s3.Client
	bucket            string
	presignExpiration time.Duration
	logger            *logrus.Logger
}

// NewS3Storage создает новое S3 хранилище
func NewS3Storage(cfg S3Config, logger *logrus.Logger) (*S3Storage, error) {
	if err := validateS3Config(cfg); err != nil {
		return nil, fmt.Errorf("неверная конфигурация S3: %w", err)
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(cfg.Region),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки AWS конфигурации: %w", err)
	}

	// Настройка custom endpoint если указан
	if cfg.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				HostnameImmutable: true,
			}, nil
		})
		awsCfg.EndpointResolverWithOptions = customResolver
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForcePathStyle
	})

	return &S3Storage{
		client:            client,
		bucket:            cfg.Bucket,
		presignExpiration: cfg.PresignExpiration,
		logger:            logger,
	}, nil
}

// Save сохраняет файл в S3
func (s *S3Storage) Save(ctx context.Context, key string, reader io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		return fmt.Errorf("ошибка сохранения файла в S3: %w", err)
	}
	return nil
}

// Get получает файл из S3
func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения файла из S3: %w", err)
	}
	return result.Body, nil
}

// Delete удаляет файл из S3
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("ошибка удаления файла из S3: %w", err)
	}
	return nil
}

// Exists проверяет существование файла в S3
func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NoSuchKey
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, fmt.Errorf("ошибка проверки существования файла: %w", err)
	}
	return true, nil
}

// GetMetadata получает метаданные файла
func (s *S3Storage) GetMetadata(ctx context.Context, key string) (*FileMetadata, error) {
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения метаданных: %w", err)
	}

	size := int64(0)
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	return &FileMetadata{
		Key:          key,
		Size:         size,
		LastModified: *result.LastModified,
		ContentType:  aws.ToString(result.ContentType),
		ETag:         aws.ToString(result.ETag),
		Metadata:     result.Metadata,
	}, nil
}

// GetSize возвращает размер файла
func (s *S3Storage) GetSize(ctx context.Context, key string) (int64, error) {
	metadata, err := s.GetMetadata(ctx, key)
	if err != nil {
		return 0, err
	}
	return metadata.Size, nil
}

// GetURL возвращает публичный URL файла
func (s *S3Storage) GetURL(ctx context.Context, key string) (string, error) {
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key), nil
}

// GetPresignedURL возвращает pre-signed URL
func (s *S3Storage) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)
	presignedURL, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("ошибка генерации pre-signed URL: %w", err)
	}
	return presignedURL.URL, nil
}

// List возвращает список файлов по префиксу
func (s *S3Storage) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка файлов: %w", err)
	}

	files := make([]FileInfo, len(result.Contents))
	for i, obj := range result.Contents {
		size := int64(0)
		if obj.Size != nil {
			size = *obj.Size
		}
		files[i] = FileInfo{
			Key:          aws.ToString(obj.Key),
			Size:         size,
			LastModified: *obj.LastModified,
			IsDir:        false,
		}
	}

	return files, nil
}

// Copy копирует файл
func (s *S3Storage) Copy(ctx context.Context, srcKey, dstKey string) error {
	copySource := fmt.Sprintf("%s/%s", s.bucket, srcKey)
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return fmt.Errorf("ошибка копирования файла: %w", err)
	}
	return nil
}

// Move перемещает файл
func (s *S3Storage) Move(ctx context.Context, srcKey, dstKey string) error {
	if err := s.Copy(ctx, srcKey, dstKey); err != nil {
		return err
	}
	return s.Delete(ctx, srcKey)
}

// JoinPath объединяет элементы пути
func (s *S3Storage) JoinPath(elem ...string) string {
	return path.Join(elem...)
}

// ValidateKey валидирует ключ файла
func (s *S3Storage) ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("ключ файла не может быть пустым")
	}
	if len(key) > 1024 {
		return fmt.Errorf("ключ файла слишком длинный: %d символов (максимум 1024)", len(key))
	}
	return nil
}

// LocalStorage реализация локального файлового хранилища
type LocalStorage struct {
	basePath    string
	permissions os.FileMode
	createDirs  bool
	logger      *logrus.Logger
}

// NewLocalStorage создает новое локальное хранилище
func NewLocalStorage(cfg LocalConfig, logger *logrus.Logger) (*LocalStorage, error) {
	if err := validateLocalConfig(cfg); err != nil {
		return nil, fmt.Errorf("неверная конфигурация локального хранилища: %w", err)
	}

	// Создаем базовую директорию если нужно
	if cfg.CreateDirs {
		if err := os.MkdirAll(cfg.BasePath, cfg.Permissions); err != nil {
			return nil, fmt.Errorf("ошибка создания базовой директории: %w", err)
		}
	}

	return &LocalStorage{
		basePath:    cfg.BasePath,
		permissions: cfg.Permissions,
		createDirs:  cfg.CreateDirs,
		logger:      logger,
	}, nil
}

// Save сохраняет файл локально
func (l *LocalStorage) Save(ctx context.Context, key string, reader io.Reader) error {
	fullPath := l.getFullPath(key)

	// Создаем директорию если нужно
	if l.createDirs {
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, l.permissions); err != nil {
			return fmt.Errorf("ошибка создания директории: %w", err)
		}
	}

	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, l.permissions)
	if err != nil {
		return fmt.Errorf("ошибка создания файла: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	return nil
}

// Get получает файл локально
func (l *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := l.getFullPath(key)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("файл не найден: %s", key)
		}
		return nil, fmt.Errorf("ошибка открытия файла: %w", err)
	}
	return file, nil
}

// Delete удаляет файл локально
func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := l.getFullPath(key)
	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ошибка удаления файла: %w", err)
	}
	return nil
}

// Exists проверяет существование файла
func (l *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := l.getFullPath(key)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("ошибка проверки существования файла: %w", err)
	}
	return true, nil
}

// GetMetadata получает метаданные файла
func (l *LocalStorage) GetMetadata(ctx context.Context, key string) (*FileMetadata, error) {
	fullPath := l.getFullPath(key)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о файле: %w", err)
	}

	return &FileMetadata{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
		ContentType:  "application/octet-stream", // базовый тип для локальных файлов
	}, nil
}

// GetSize возвращает размер файла
func (l *LocalStorage) GetSize(ctx context.Context, key string) (int64, error) {
	metadata, err := l.GetMetadata(ctx, key)
	if err != nil {
		return 0, err
	}
	return metadata.Size, nil
}

// GetURL возвращает файловый URL
func (l *LocalStorage) GetURL(ctx context.Context, key string) (string, error) {
	fullPath := l.getFullPath(key)
	return "file://" + fullPath, nil
}

// GetPresignedURL для локального хранилища возвращает обычный URL
func (l *LocalStorage) GetPresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	return l.GetURL(ctx, key)
}

// List возвращает список файлов
func (l *LocalStorage) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	prefixPath := l.getFullPath(prefix)
	baseDir := filepath.Dir(prefixPath)

	var files []FileInfo
	err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Получаем относительный путь
		relPath, err := filepath.Rel(l.basePath, path)
		if err != nil {
			return err
		}

		// Проверяем префикс
		if !strings.HasPrefix(relPath, prefix) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		files = append(files, FileInfo{
			Key:          relPath,
			Size:         info.Size(),
			LastModified: info.ModTime(),
			IsDir:        d.IsDir(),
		})

		return nil
	})

	return files, err
}

// Copy копирует файл
func (l *LocalStorage) Copy(ctx context.Context, srcKey, dstKey string) error {
	srcPath := l.getFullPath(srcKey)
	dstPath := l.getFullPath(dstKey)

	// Создаем директорию назначения если нужно
	if l.createDirs {
		dir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dir, l.permissions); err != nil {
			return fmt.Errorf("ошибка создания директории: %w", err)
		}
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("ошибка открытия исходного файла: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, l.permissions)
	if err != nil {
		return fmt.Errorf("ошибка создания файла назначения: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("ошибка копирования файла: %w", err)
	}

	return nil
}

// Move перемещает файл
func (l *LocalStorage) Move(ctx context.Context, srcKey, dstKey string) error {
	srcPath := l.getFullPath(srcKey)
	dstPath := l.getFullPath(dstKey)

	// Создаем директорию назначения если нужно
	if l.createDirs {
		dir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dir, l.permissions); err != nil {
			return fmt.Errorf("ошибка создания директории: %w", err)
		}
	}

	err := os.Rename(srcPath, dstPath)
	if err != nil {
		return fmt.Errorf("ошибка перемещения файла: %w", err)
	}

	return nil
}

// JoinPath объединяет элементы пути
func (l *LocalStorage) JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// ValidateKey валидирует ключ файла
func (l *LocalStorage) ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("ключ файла не может быть пустым")
	}
	if strings.Contains(key, "..") {
		return fmt.Errorf("ключ файла не может содержать '..'")
	}
	return nil
}

// getFullPath возвращает полный путь к файлу
func (l *LocalStorage) getFullPath(key string) string {
	return filepath.Join(l.basePath, key)
}

// Функции валидации

// validateS3Config валидирует конфигурацию S3
func validateS3Config(cfg S3Config) error {
	if cfg.Region == "" {
		return fmt.Errorf("регион S3 не может быть пустым")
	}
	if cfg.Bucket == "" {
		return fmt.Errorf("bucket S3 не может быть пустым")
	}
	if cfg.AccessKey == "" {
		return fmt.Errorf("access key не может быть пустым")
	}
	if cfg.SecretKey == "" {
		return fmt.Errorf("secret key не может быть пустым")
	}
	if cfg.PresignExpiration <= 0 {
		return fmt.Errorf("время истечения presigned URL должно быть положительным")
	}
	return nil
}

// validateLocalConfig валидирует конфигурацию локального хранилища
func validateLocalConfig(cfg LocalConfig) error {
	if cfg.BasePath == "" {
		return fmt.Errorf("базовый путь не может быть пустым")
	}
	if !filepath.IsAbs(cfg.BasePath) {
		return fmt.Errorf("базовый путь должен быть абсолютным")
	}
	return nil
}

// NewStorageFromConfig создает хранилище из конфигурации (обратная совместимость)
func NewStorageFromConfig(cfg config.Config, logger *logrus.Logger) (Storage, error) {
	builder := NewStorageBuilder(cfg, logger)
	return builder.Build()
}
