package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ReportStatus типизированный статус отчета
type ReportStatus string

const (
	// StatusPending отчет ожидает генерации
	StatusPending ReportStatus = "pending"
	// StatusProcessing отчет генерируется
	StatusProcessing ReportStatus = "processing"
	// StatusCompleted отчет успешно сгенерирован
	StatusCompleted ReportStatus = "completed"
	// StatusFailed ошибка при генерации отчета
	StatusFailed ReportStatus = "failed"
	// StatusCanceled отчет отменен
	StatusCanceled ReportStatus = "canceled"
)

// String возвращает строковое представление статуса
func (s ReportStatus) String() string {
	return string(s)
}

// IsValid проверяет валидность статуса
func (s ReportStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusProcessing, StatusCompleted, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

// IsFinal возвращает true для финальных статусов
func (s ReportStatus) IsFinal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCanceled
}

// CanTransitionTo проверяет возможность перехода к новому статусу
func (s ReportStatus) CanTransitionTo(newStatus ReportStatus) bool {
	transitions := map[ReportStatus][]ReportStatus{
		StatusPending:    {StatusProcessing, StatusCanceled},
		StatusProcessing: {StatusCompleted, StatusFailed, StatusCanceled},
		StatusCompleted:  {},              // финальный статус
		StatusFailed:     {StatusPending}, // можно попробовать снова
		StatusCanceled:   {StatusPending}, // можно возобновить
	}

	allowedTransitions, exists := transitions[s]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == newStatus {
			return true
		}
	}
	return false
}

// ReportEntity интерфейс для работы с отчетами
type ReportEntity interface {
	GetID() uint
	GetTitle() string
	GetStatus() ReportStatus
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetCreatedBy() string
	GetUpdatedBy() string
	IsCompleted() bool
	IsPending() bool
	IsFailed() bool
	IsProcessing() bool
	SetStatus(status ReportStatus, updatedBy string) error
	Validate() error
}

// Auditable интерфейс для аудита изменений
type Auditable interface {
	SetCreatedBy(user string)
	SetUpdatedBy(user string)
	GetAuditInfo() (createdBy, updatedBy string, createdAt, updatedAt time.Time)
}

// Report представляет сгенерированный отчет
type Report struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	Title       string         `json:"title" gorm:"size:255;not null" validate:"required,min=1,max=255"`
	Description string         `json:"description" gorm:"size:1000" validate:"max=1000"`
	Status      ReportStatus   `json:"status" gorm:"size:50;not null;default:'pending'" validate:"required"`
	FileKey     string         `json:"file_key,omitempty" gorm:"size:255" validate:"max=255"`
	GeneratedAt *time.Time     `json:"generated_at,omitempty"`
	Parameters  JSON           `json:"parameters,omitempty" gorm:"type:jsonb"`
	CreatedBy   string         `json:"created_by" gorm:"size:255;not null" validate:"required,min=1,max=255"`
	UpdatedBy   string         `json:"updated_by" gorm:"size:255;not null" validate:"required,min=1,max=255"`
}

// JSON кастомный тип для работы с JSONB данными
type JSON map[string]interface{}

// NewJSON создает новый JSON объект
func NewJSON() JSON {
	return make(JSON)
}

// Set устанавливает значение по ключу
func (j JSON) Set(key string, value interface{}) {
	j[key] = value
}

// Get получает значение по ключу
func (j JSON) Get(key string) (interface{}, bool) {
	value, exists := j[key]
	return value, exists
}

// GetString получает строковое значение по ключу
func (j JSON) GetString(key string) (string, bool) {
	if value, exists := j[key]; exists {
		if str, ok := value.(string); ok {
			return str, true
		}
	}
	return "", false
}

// GetInt получает целочисленное значение по ключу
func (j JSON) GetInt(key string) (int, bool) {
	if value, exists := j[key]; exists {
		switch v := value.(type) {
		case int:
			return v, true
		case float64:
			return int(v), true
		}
	}
	return 0, false
}

// Has проверяет наличие ключа
func (j JSON) Has(key string) bool {
	_, exists := j[key]
	return exists
}

// Delete удаляет ключ
func (j JSON) Delete(key string) {
	delete(j, key)
}

// Keys возвращает все ключи
func (j JSON) Keys() []string {
	keys := make([]string, 0, len(j))
	for key := range j {
		keys = append(keys, key)
	}
	return keys
}

// IsEmpty проверяет, пуст ли JSON объект
func (j JSON) IsEmpty() bool {
	return len(j) == 0
}

// Value реализует интерфейс driver.Valuer для JSON
func (j JSON) Value() (driver.Value, error) {
	if j == nil || j.IsEmpty() {
		return nil, nil
	}

	data, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	return data, nil
}

// Scan реализует интерфейс sql.Scanner для JSON
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*j = NewJSON()
			return nil
		}
		bytes = v
	case string:
		if v == "" {
			*j = NewJSON()
			return nil
		}
		bytes = []byte(v)
	default:
		return fmt.Errorf("невозможно сканировать %T в JSON", value)
	}

	result := make(JSON)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return fmt.Errorf("ошибка десериализации JSON: %w", err)
	}

	*j = result
	return nil
}

// ReportBuilder строитель для создания отчетов
type ReportBuilder struct {
	report *Report
}

// NewReportBuilder создает новый строитель отчетов
func NewReportBuilder() *ReportBuilder {
	return &ReportBuilder{
		report: &Report{
			Status:     StatusPending,
			Parameters: NewJSON(),
		},
	}
}

// WithTitle устанавливает заголовок отчета
func (b *ReportBuilder) WithTitle(title string) *ReportBuilder {
	b.report.Title = strings.TrimSpace(title)
	return b
}

// WithDescription устанавливает описание отчета
func (b *ReportBuilder) WithDescription(description string) *ReportBuilder {
	b.report.Description = strings.TrimSpace(description)
	return b
}

// WithCreatedBy устанавливает создателя отчета
func (b *ReportBuilder) WithCreatedBy(user string) *ReportBuilder {
	b.report.CreatedBy = strings.TrimSpace(user)
	b.report.UpdatedBy = strings.TrimSpace(user)
	return b
}

// WithParameters устанавливает параметры отчета
func (b *ReportBuilder) WithParameters(params JSON) *ReportBuilder {
	if params != nil {
		b.report.Parameters = params
	}
	return b
}

// AddParameter добавляет параметр к отчету
func (b *ReportBuilder) AddParameter(key string, value interface{}) *ReportBuilder {
	if b.report.Parameters == nil {
		b.report.Parameters = NewJSON()
	}
	b.report.Parameters.Set(key, value)
	return b
}

// Build создает отчет
func (b *ReportBuilder) Build() (*Report, error) {
	if err := b.report.Validate(); err != nil {
		return nil, fmt.Errorf("ошибка валидации отчета: %w", err)
	}
	return b.report, nil
}

// TableName указывает имя таблицы для модели Report
func (Report) TableName() string {
	return "reports"
}

// GetID возвращает ID отчета
func (r *Report) GetID() uint {
	return r.ID
}

// GetTitle возвращает заголовок отчета
func (r *Report) GetTitle() string {
	return r.Title
}

// GetStatus возвращает статус отчета
func (r *Report) GetStatus() ReportStatus {
	return r.Status
}

// GetCreatedAt возвращает время создания
func (r *Report) GetCreatedAt() time.Time {
	return r.CreatedAt
}

// GetUpdatedAt возвращает время последнего обновления
func (r *Report) GetUpdatedAt() time.Time {
	return r.UpdatedAt
}

// GetCreatedBy возвращает создателя отчета
func (r *Report) GetCreatedBy() string {
	return r.CreatedBy
}

// GetUpdatedBy возвращает последнего редактора отчета
func (r *Report) GetUpdatedBy() string {
	return r.UpdatedBy
}

// IsCompleted возвращает true, если генерация отчета завершена
func (r *Report) IsCompleted() bool {
	return r.Status == StatusCompleted
}

// IsPending возвращает true, если отчет ожидает генерации
func (r *Report) IsPending() bool {
	return r.Status == StatusPending
}

// IsFailed возвращает true, если генерация отчета завершилась ошибкой
func (r *Report) IsFailed() bool {
	return r.Status == StatusFailed
}

// IsProcessing возвращает true, если отчет в процессе генерации
func (r *Report) IsProcessing() bool {
	return r.Status == StatusProcessing
}

// IsCanceled возвращает true, если отчет отменен
func (r *Report) IsCanceled() bool {
	return r.Status == StatusCanceled
}

// SetStatus обновляет статус отчета с проверкой валидности перехода
func (r *Report) SetStatus(status ReportStatus, updatedBy string) error {
	if !status.IsValid() {
		return fmt.Errorf("неверный статус: %s", status)
	}

	if !r.Status.CanTransitionTo(status) {
		return fmt.Errorf("невозможен переход со статуса %s на %s", r.Status, status)
	}

	r.Status = status
	r.UpdatedBy = strings.TrimSpace(updatedBy)
	r.UpdatedAt = time.Now().UTC()

	// Устанавливаем время генерации для завершенных отчетов
	if status == StatusCompleted && r.GeneratedAt == nil {
		now := time.Now().UTC()
		r.GeneratedAt = &now
	}

	return nil
}

// SetCreatedBy устанавливает создателя отчета
func (r *Report) SetCreatedBy(user string) {
	r.CreatedBy = strings.TrimSpace(user)
}

// SetUpdatedBy устанавливает редактора отчета
func (r *Report) SetUpdatedBy(user string) {
	r.UpdatedBy = strings.TrimSpace(user)
	r.UpdatedAt = time.Now().UTC()
}

// GetAuditInfo возвращает информацию для аудита
func (r *Report) GetAuditInfo() (createdBy, updatedBy string, createdAt, updatedAt time.Time) {
	return r.CreatedBy, r.UpdatedBy, r.CreatedAt, r.UpdatedAt
}

// SetFileKey устанавливает ключ файла отчета
func (r *Report) SetFileKey(fileKey string) {
	r.FileKey = strings.TrimSpace(fileKey)
}

// HasFile возвращает true, если у отчета есть связанный файл
func (r *Report) HasFile() bool {
	return r.FileKey != ""
}

// Validate валидирует отчет
func (r *Report) Validate() error {
	var errors []string

	// Проверка заголовка
	if strings.TrimSpace(r.Title) == "" {
		errors = append(errors, "заголовок не может быть пустым")
	}
	if len(r.Title) > 255 {
		errors = append(errors, "заголовок не может быть длиннее 255 символов")
	}

	// Проверка описания
	if len(r.Description) > 1000 {
		errors = append(errors, "описание не может быть длиннее 1000 символов")
	}

	// Проверка статуса
	if !r.Status.IsValid() {
		errors = append(errors, fmt.Sprintf("неверный статус: %s", r.Status))
	}

	// Проверка создателя
	if strings.TrimSpace(r.CreatedBy) == "" {
		errors = append(errors, "поле created_by не может быть пустым")
	}
	if len(r.CreatedBy) > 255 {
		errors = append(errors, "поле created_by не может быть длиннее 255 символов")
	}

	// Проверка редактора
	if strings.TrimSpace(r.UpdatedBy) == "" {
		errors = append(errors, "поле updated_by не может быть пустым")
	}
	if len(r.UpdatedBy) > 255 {
		errors = append(errors, "поле updated_by не может быть длиннее 255 символов")
	}

	// Проверка ключа файла
	if len(r.FileKey) > 255 {
		errors = append(errors, "ключ файла не может быть длиннее 255 символов")
	}

	if len(errors) > 0 {
		return fmt.Errorf("ошибки валидации: %s", strings.Join(errors, "; "))
	}

	return nil
}

// BeforeCreate GORM hook, вызывается перед созданием записи
func (r *Report) BeforeCreate(tx *gorm.DB) error {
	r.CreatedAt = time.Now().UTC()
	r.UpdatedAt = time.Now().UTC()

	if r.Status == "" {
		r.Status = StatusPending
	}

	if r.Parameters == nil {
		r.Parameters = NewJSON()
	}

	return r.Validate()
}

// BeforeUpdate GORM hook, вызывается перед обновлением записи
func (r *Report) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = time.Now().UTC()
	return r.Validate()
}
