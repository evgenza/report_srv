package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"report_srv/internal/models"
	"report_srv/internal/storage"

	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

const (
	// Таймауты
	defaultGenerationTimeout = 30 * time.Minute
	defaultContextTimeout    = 5 * time.Second

	// Лимиты
	maxConcurrentGeneration = 5
	maxRetryAttempts        = 3
)

// ReportService интерфейс для работы с отчетами
type ReportService interface {
	CreateReport(ctx context.Context, report *models.Report) error
	GetReport(ctx context.Context, id uint) (*models.Report, error)
	ListReports(ctx context.Context, params ListReportParams) (*ReportList, error)
	UpdateReport(ctx context.Context, id uint, updates ReportUpdateParams) error
	DeleteReport(ctx context.Context, id uint) error
	CancelReportGeneration(ctx context.Context, id uint) error
	GetReportFile(ctx context.Context, id uint) (io.ReadCloser, string, error)
}

// ReportRepository интерфейс для работы с базой данных отчетов
type ReportRepository interface {
	Create(ctx context.Context, report *models.Report) error
	GetByID(ctx context.Context, id uint) (*models.Report, error)
	List(ctx context.Context, params ListReportParams) ([]models.Report, int64, error)
	Update(ctx context.Context, id uint, updates map[string]interface{}) error
	Delete(ctx context.Context, id uint) error
	UpdateStatus(ctx context.Context, id uint, status models.ReportStatus, fileKey string) error
}

// ReportGenerator интерфейс для генерации отчетов
type ReportGenerator interface {
	Generate(ctx context.Context, report *models.Report) (io.Reader, string, error)
	GetMimeType() string
	GetFileExtension() string
}

// ReportFileStorage интерфейс для работы с файлами отчетов
type ReportFileStorage interface {
	Save(ctx context.Context, key string, data io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GenerateKey(report *models.Report) string
}

// BackgroundProcessor интерфейс для фоновой обработки
type BackgroundProcessor interface {
	SubmitTask(ctx context.Context, task Task) error
	CancelTask(taskID string) error
	GetTaskStatus(taskID string) TaskStatus
}

// Task представляет фоновую задачу
type Task struct {
	ID       string
	Type     TaskType
	Data     interface{}
	Priority Priority
	Timeout  time.Duration
}

// TaskType тип задачи
type TaskType string

const (
	TaskTypeReportGeneration TaskType = "report_generation"
)

// TaskStatus статус задачи
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCanceled  TaskStatus = "canceled"
)

// Priority приоритет задачи
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// ListReportParams параметры для получения списка отчетов
type ListReportParams struct {
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	Status   *models.ReportStatus `json:"status,omitempty"`
	Search   string               `json:"search,omitempty"`
	SortBy   string               `json:"sort_by,omitempty"`
	SortDesc bool                 `json:"sort_desc,omitempty"`
}

// ReportUpdateParams параметры для обновления отчета
type ReportUpdateParams struct {
	Title       *string              `json:"title,omitempty"`
	Description *string              `json:"description,omitempty"`
	Status      *models.ReportStatus `json:"status,omitempty"`
	Parameters  *models.JSON         `json:"parameters,omitempty"`
	UpdatedBy   string               `json:"updated_by"`
}

// ReportList результат получения списка отчетов с пагинацией
type ReportList struct {
	Reports    []models.Report `json:"reports"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// ReportServiceImpl реализация сервиса отчетов
type ReportServiceImpl struct {
	repository  ReportRepository
	generator   ReportGenerator
	fileStorage ReportFileStorage
	processor   BackgroundProcessor
	logger      *logrus.Logger

	// Канал для отмены генерации
	cancellations sync.Map // map[uint]context.CancelFunc
}

// NewReportService создает новый сервис отчетов
func NewReportService(
	repository ReportRepository,
	generator ReportGenerator,
	fileStorage ReportFileStorage,
	processor BackgroundProcessor,
	logger *logrus.Logger,
) ReportService {
	return &ReportServiceImpl{
		repository:  repository,
		generator:   generator,
		fileStorage: fileStorage,
		processor:   processor,
		logger:      logger,
	}
}

// CreateReport создает новый отчет
func (s *ReportServiceImpl) CreateReport(ctx context.Context, report *models.Report) error {
	logger := s.logger.WithFields(logrus.Fields{
		"title":      report.Title,
		"created_by": report.CreatedBy,
	})

	logger.Info("Создание нового отчета")

	// Валидация отчета
	if err := report.Validate(); err != nil {
		logger.WithError(err).Error("Ошибка валидации отчета")
		return fmt.Errorf("ошибка валидации отчета: %w", err)
	}

	// Сохранение в БД
	if err := s.repository.Create(ctx, report); err != nil {
		logger.WithError(err).Error("Ошибка сохранения отчета в БД")
		return fmt.Errorf("ошибка создания отчета: %w", err)
	}

	logger.WithField("report_id", report.ID).Info("Отчет создан, запуск генерации")

	// Запуск фоновой генерации
	task := Task{
		ID:       fmt.Sprintf("report_%d", report.ID),
		Type:     TaskTypeReportGeneration,
		Data:     report.ID,
		Priority: PriorityNormal,
		Timeout:  defaultGenerationTimeout,
	}

	if err := s.processor.SubmitTask(ctx, task); err != nil {
		logger.WithError(err).Error("Ошибка запуска фоновой генерации")
		// Обновляем статус на failed
		s.updateReportStatus(ctx, report.ID, models.StatusFailed, "")
		return fmt.Errorf("ошибка запуска генерации отчета: %w", err)
	}

	return nil
}

// GetReport получает отчет по ID
func (s *ReportServiceImpl) GetReport(ctx context.Context, id uint) (*models.Report, error) {
	report, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("отчет с ID %d не найден", id)
		}
		s.logger.WithError(err).WithField("report_id", id).Error("Ошибка получения отчета")
		return nil, fmt.Errorf("ошибка получения отчета: %w", err)
	}

	return report, nil
}

// ListReports получает список отчетов с пагинацией
func (s *ReportServiceImpl) ListReports(ctx context.Context, params ListReportParams) (*ReportList, error) {
	// Валидация параметров пагинации
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	reports, total, err := s.repository.List(ctx, params)
	if err != nil {
		s.logger.WithError(err).Error("Ошибка получения списка отчетов")
		return nil, fmt.Errorf("ошибка получения списка отчетов: %w", err)
	}

	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &ReportList{
		Reports:    reports,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateReport обновляет отчет
func (s *ReportServiceImpl) UpdateReport(ctx context.Context, id uint, params ReportUpdateParams) error {
	logger := s.logger.WithFields(logrus.Fields{
		"report_id":  id,
		"updated_by": params.UpdatedBy,
	})

	// Получаем текущий отчет для валидации
	report, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("отчет с ID %d не найден", id)
		}
		return fmt.Errorf("ошибка получения отчета: %w", err)
	}

	// Подготавливаем обновления
	updates := make(map[string]interface{})
	updates["updated_by"] = params.UpdatedBy
	updates["updated_at"] = time.Now().UTC()

	if params.Title != nil {
		updates["title"] = *params.Title
	}
	if params.Description != nil {
		updates["description"] = *params.Description
	}
	if params.Parameters != nil {
		updates["parameters"] = *params.Parameters
	}

	// Обработка изменения статуса
	if params.Status != nil {
		if !report.Status.CanTransitionTo(*params.Status) {
			return fmt.Errorf("невозможен переход со статуса %s на %s", report.Status, *params.Status)
		}
		updates["status"] = *params.Status

		// Если отменяем генерацию
		if *params.Status == models.StatusCanceled {
			s.cancelGeneration(id)
		}
	}

	if err := s.repository.Update(ctx, id, updates); err != nil {
		logger.WithError(err).Error("Ошибка обновления отчета")
		return fmt.Errorf("ошибка обновления отчета: %w", err)
	}

	logger.Info("Отчет обновлен успешно")
	return nil
}

// DeleteReport удаляет отчет
func (s *ReportServiceImpl) DeleteReport(ctx context.Context, id uint) error {
	logger := s.logger.WithField("report_id", id)

	// Получаем отчет для проверки существования и получения file_key
	report, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("отчет с ID %d не найден", id)
		}
		return fmt.Errorf("ошибка получения отчета: %w", err)
	}

	// Отменяем генерацию, если она идет
	s.cancelGeneration(id)

	// Удаляем файл из хранилища, если он существует
	if report.HasFile() {
		if err := s.fileStorage.Delete(ctx, report.FileKey); err != nil {
			logger.WithError(err).WithField("file_key", report.FileKey).
				Error("Ошибка удаления файла отчета")
			// Не прерываем удаление отчета из-за ошибки удаления файла
		}
	}

	// Удаляем отчет из БД
	if err := s.repository.Delete(ctx, id); err != nil {
		logger.WithError(err).Error("Ошибка удаления отчета из БД")
		return fmt.Errorf("ошибка удаления отчета: %w", err)
	}

	logger.WithField("title", report.Title).Info("Отчет удален успешно")
	return nil
}

// CancelReportGeneration отменяет генерацию отчета
func (s *ReportServiceImpl) CancelReportGeneration(ctx context.Context, id uint) error {
	logger := s.logger.WithField("report_id", id)

	// Проверяем существование отчета
	report, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("отчет с ID %d не найден", id)
		}
		return fmt.Errorf("ошибка получения отчета: %w", err)
	}

	// Проверяем, что отчет можно отменить
	if !report.Status.CanTransitionTo(models.StatusCanceled) {
		return fmt.Errorf("отчет в статусе %s нельзя отменить", report.Status)
	}

	// Отменяем задачу в процессоре
	taskID := fmt.Sprintf("report_%d", id)
	if err := s.processor.CancelTask(taskID); err != nil {
		logger.WithError(err).Error("Ошибка отмены задачи в процессоре")
	}

	// Отменяем генерацию
	s.cancelGeneration(id)

	// Обновляем статус
	if err := s.updateReportStatus(ctx, id, models.StatusCanceled, ""); err != nil {
		return fmt.Errorf("ошибка обновления статуса отчета: %w", err)
	}

	logger.Info("Генерация отчета отменена")
	return nil
}

// GetReportFile возвращает файл отчета
func (s *ReportServiceImpl) GetReportFile(ctx context.Context, id uint) (io.ReadCloser, string, error) {
	report, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, "", fmt.Errorf("отчет с ID %d не найден", id)
		}
		return nil, "", fmt.Errorf("ошибка получения отчета: %w", err)
	}

	if !report.IsCompleted() {
		return nil, "", fmt.Errorf("отчет еще не готов")
	}

	if !report.HasFile() {
		return nil, "", fmt.Errorf("файл отчета не найден")
	}

	reader, err := s.fileStorage.Get(ctx, report.FileKey)
	if err != nil {
		s.logger.WithError(err).WithField("file_key", report.FileKey).
			Error("Ошибка получения файла из хранилища")
		return nil, "", fmt.Errorf("ошибка получения файла: %w", err)
	}

	filename := fmt.Sprintf("%s.%s", report.Title, s.generator.GetFileExtension())
	return reader, filename, nil
}

// cancelGeneration отменяет генерацию отчета
func (s *ReportServiceImpl) cancelGeneration(reportID uint) {
	if cancel, exists := s.cancellations.LoadAndDelete(reportID); exists {
		if cancelFunc, ok := cancel.(context.CancelFunc); ok {
			cancelFunc()
		}
	}
}

// updateReportStatus обновляет статус отчета
func (s *ReportServiceImpl) updateReportStatus(ctx context.Context, id uint, status models.ReportStatus, fileKey string) error {
	return s.repository.UpdateStatus(ctx, id, status, fileKey)
}

// ExcelReportGenerator генератор Excel отчетов
type ExcelReportGenerator struct {
	logger *logrus.Logger
}

// NewExcelReportGenerator создает новый генератор Excel отчетов
func NewExcelReportGenerator(logger *logrus.Logger) ReportGenerator {
	return &ExcelReportGenerator{logger: logger}
}

// Generate генерирует Excel отчет
func (g *ExcelReportGenerator) Generate(ctx context.Context, report *models.Report) (io.Reader, string, error) {
	logger := g.logger.WithFields(logrus.Fields{
		"report_id": report.ID,
		"title":     report.Title,
	})

	logger.Info("Генерация Excel отчета")

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Report"
	f.SetSheetName("Sheet1", sheet)

	// Стиль для заголовков
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E6E6FA"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	if err != nil {
		logger.WithError(err).Warn("Ошибка создания стиля заголовка")
	}

	// Заголовки
	headers := []string{"Параметр", "Значение"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
		if headerStyle != 0 {
			f.SetCellStyle(sheet, cell, cell, headerStyle)
		}
	}

	// Данные отчета
	data := [][]interface{}{
		{"ID отчета", report.ID},
		{"Название", report.Title},
		{"Описание", report.Description},
		{"Статус", string(report.Status)},
		{"Создал", report.CreatedBy},
		{"Дата создания", report.CreatedAt.Format("2006-01-02 15:04:05")},
	}

	// Добавляем параметры
	if report.Parameters != nil && !report.Parameters.IsEmpty() {
		data = append(data, []interface{}{"--- Параметры ---", ""})
		for key, value := range report.Parameters {
			data = append(data, []interface{}{key, fmt.Sprintf("%v", value)})
		}
	}

	// Заполняем данные
	for rowIndex, row := range data {
		for colIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			f.SetCellValue(sheet, cell, value)
		}
	}

	// Автоширина колонок
	f.SetColWidth(sheet, "A", "B", 30)

	// Генерируем буфер
	var buffer bytes.Buffer
	if err := f.Write(&buffer); err != nil {
		logger.WithError(err).Error("Ошибка записи Excel файла")
		return nil, "", fmt.Errorf("ошибка генерации Excel файла: %w", err)
	}

	filename := fmt.Sprintf("report_%d_%s.xlsx", report.ID, time.Now().Format("20060102_150405"))

	logger.WithField("filename", filename).Info("Excel отчет сгенерирован успешно")
	return &buffer, filename, nil
}

// GetMimeType возвращает MIME тип для Excel файлов
func (g *ExcelReportGenerator) GetMimeType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// GetFileExtension возвращает расширение файла для Excel
func (g *ExcelReportGenerator) GetFileExtension() string {
	return "xlsx"
}

// ReportFileStorageImpl реализация хранилища файлов отчетов
type ReportFileStorageImpl struct {
	storage storage.Storage
	logger  *logrus.Logger
}

// NewReportFileStorage создает новое хранилище файлов отчетов
func NewReportFileStorage(storage storage.Storage, logger *logrus.Logger) ReportFileStorage {
	return &ReportFileStorageImpl{
		storage: storage,
		logger:  logger,
	}
}

// Save сохраняет файл в хранилище
func (s *ReportFileStorageImpl) Save(ctx context.Context, key string, data io.Reader) error {
	return s.storage.Save(ctx, key, data)
}

// Get получает файл из хранилища
func (s *ReportFileStorageImpl) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.storage.Get(ctx, key)
}

// Delete удаляет файл из хранилища
func (s *ReportFileStorageImpl) Delete(ctx context.Context, key string) error {
	return s.storage.Delete(ctx, key)
}

// GenerateKey генерирует ключ для файла отчета
func (s *ReportFileStorageImpl) GenerateKey(report *models.Report) string {
	return fmt.Sprintf("reports/%d/%s_%s.xlsx",
		report.ID,
		report.Title,
		time.Now().Format("20060102150405"))
}

// GormReportRepository реализация репозитория отчетов для GORM
type GormReportRepository struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// NewGormReportRepository создает новый GORM репозиторий отчетов
func NewGormReportRepository(db *gorm.DB, logger *logrus.Logger) ReportRepository {
	return &GormReportRepository{
		db:     db,
		logger: logger,
	}
}

// Create создает новый отчет в БД
func (r *GormReportRepository) Create(ctx context.Context, report *models.Report) error {
	return r.db.WithContext(ctx).Create(report).Error
}

// GetByID получает отчет по ID
func (r *GormReportRepository) GetByID(ctx context.Context, id uint) (*models.Report, error) {
	var report models.Report
	err := r.db.WithContext(ctx).First(&report, id).Error
	return &report, err
}

// List получает список отчетов с фильтрацией и пагинацией
func (r *GormReportRepository) List(ctx context.Context, params ListReportParams) ([]models.Report, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Report{})

	// Фильтрация по статусу
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// Поиск
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	// Подсчет общего количества
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Сортировка
	if params.SortBy != "" {
		order := params.SortBy
		if params.SortDesc {
			order += " DESC"
		}
		query = query.Order(order)
	} else {
		query = query.Order("created_at DESC")
	}

	// Пагинация
	offset := (params.Page - 1) * params.PageSize
	query = query.Offset(offset).Limit(params.PageSize)

	var reports []models.Report
	err := query.Find(&reports).Error

	return reports, total, err
}

// Update обновляет отчет
func (r *GormReportRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Report{}).Where("id = ?", id).Updates(updates).Error
}

// Delete удаляет отчет
func (r *GormReportRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Report{}, id).Error
}

// UpdateStatus обновляет статус отчета
func (r *GormReportRepository) UpdateStatus(ctx context.Context, id uint, status models.ReportStatus, fileKey string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}

	if fileKey != "" {
		updates["file_key"] = fileKey
	}

	if status == models.StatusCompleted {
		now := time.Now().UTC()
		updates["generated_at"] = &now
	}

	return r.db.WithContext(ctx).Model(&models.Report{}).Where("id = ?", id).Updates(updates).Error
}

// NewReportServiceFromDB создает полностью настроенный сервис отчетов (обратная совместимость)
func NewReportServiceFromDB(db *gorm.DB, storage storage.Storage, logger *logrus.Logger) ReportService {
	repository := NewGormReportRepository(db, logger)
	generator := NewExcelReportGenerator(logger)
	fileStorage := NewReportFileStorage(storage, logger)

	// Создаем простой синхронный процессор для совместимости
	processor := NewSyncBackgroundProcessor(repository, generator, fileStorage, logger)

	service := NewReportService(repository, generator, fileStorage, processor, logger)

	// Запускаем обработку фоновых задач для синхронного процессора
	if syncProcessor, ok := processor.(*SyncBackgroundProcessor); ok {
		go syncProcessor.Start()
	}

	return service
}

// SyncBackgroundProcessor простая синхронная реализация фонового процессора
type SyncBackgroundProcessor struct {
	repository    ReportRepository
	generator     ReportGenerator
	fileStorage   ReportFileStorage
	logger        *logrus.Logger
	tasks         chan Task
	cancellations sync.Map
}

// NewSyncBackgroundProcessor создает новый синхронный фоновый процессор
func NewSyncBackgroundProcessor(
	repository ReportRepository,
	generator ReportGenerator,
	fileStorage ReportFileStorage,
	logger *logrus.Logger,
) BackgroundProcessor {
	return &SyncBackgroundProcessor{
		repository:  repository,
		generator:   generator,
		fileStorage: fileStorage,
		logger:      logger,
		tasks:       make(chan Task, 100),
	}
}

// SubmitTask отправляет задачу на выполнение
func (p *SyncBackgroundProcessor) SubmitTask(ctx context.Context, task Task) error {
	select {
	case p.tasks <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("очередь задач переполнена")
	}
}

// CancelTask отменяет задачу
func (p *SyncBackgroundProcessor) CancelTask(taskID string) error {
	if cancel, exists := p.cancellations.Load(taskID); exists {
		if cancelFunc, ok := cancel.(context.CancelFunc); ok {
			cancelFunc()
			return nil
		}
	}
	return fmt.Errorf("задача %s не найдена", taskID)
}

// GetTaskStatus возвращает статус задачи
func (p *SyncBackgroundProcessor) GetTaskStatus(taskID string) TaskStatus {
	// Простая реализация - в реальном приложении нужно отслеживать статусы
	return TaskStatusRunning
}

// Start запускает обработку фоновых задач
func (p *SyncBackgroundProcessor) Start() {
	for task := range p.tasks {
		go p.processTask(task)
	}
}

// processTask обрабатывает задачу
func (p *SyncBackgroundProcessor) processTask(task Task) {
	ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
	defer cancel()

	// Сохраняем функцию отмены
	p.cancellations.Store(task.ID, cancel)
	defer p.cancellations.Delete(task.ID)

	switch task.Type {
	case TaskTypeReportGeneration:
		p.processReportGeneration(ctx, task)
	default:
		p.logger.WithField("task_type", task.Type).Warn("Неизвестный тип задачи")
	}
}

// processReportGeneration обрабатывает генерацию отчета
func (p *SyncBackgroundProcessor) processReportGeneration(ctx context.Context, task Task) {
	reportID, ok := task.Data.(uint)
	if !ok {
		p.logger.Error("Неверный тип данных для задачи генерации отчета")
		return
	}

	logger := p.logger.WithField("report_id", reportID)

	// Обновляем статус на "processing"
	if err := p.repository.UpdateStatus(ctx, reportID, models.StatusProcessing, ""); err != nil {
		logger.WithError(err).Error("Ошибка обновления статуса на processing")
		return
	}

	// Получаем отчет
	report, err := p.repository.GetByID(ctx, reportID)
	if err != nil {
		logger.WithError(err).Error("Ошибка получения отчета для генерации")
		p.repository.UpdateStatus(ctx, reportID, models.StatusFailed, "")
		return
	}

	// Генерируем файл
	fileReader, filename, err := p.generator.Generate(ctx, report)
	if err != nil {
		logger.WithError(err).Error("Ошибка генерации файла отчета")
		p.repository.UpdateStatus(ctx, reportID, models.StatusFailed, "")
		return
	}

	// Генерируем ключ файла
	fileKey := p.fileStorage.GenerateKey(report)

	// Сохраняем файл
	if err := p.fileStorage.Save(ctx, fileKey, fileReader); err != nil {
		logger.WithError(err).Error("Ошибка сохранения файла отчета")
		p.repository.UpdateStatus(ctx, reportID, models.StatusFailed, "")
		return
	}

	// Обновляем статус на "completed"
	if err := p.repository.UpdateStatus(ctx, reportID, models.StatusCompleted, fileKey); err != nil {
		logger.WithError(err).Error("Ошибка обновления статуса на completed")
		return
	}

	logger.WithFields(logrus.Fields{
		"filename": filename,
		"file_key": fileKey,
	}).Info("Отчет сгенерирован успешно")
}
