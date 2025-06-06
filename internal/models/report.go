package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Report represents a generated report
type Report struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	Title       string         `json:"title" gorm:"size:255;not null"`
	Description string         `json:"description" gorm:"size:1000"`
	Status      string         `json:"status" gorm:"size:50;not null;default:'pending'"`
	FileKey     string         `json:"file_key,omitempty" gorm:"size:255"`
	GeneratedAt time.Time      `json:"generated_at"`
	Parameters  JSON           `json:"parameters,omitempty" gorm:"type:jsonb"`
	CreatedBy   string         `json:"created_by" gorm:"size:255;not null"`
	UpdatedBy   string         `json:"updated_by" gorm:"size:255;not null"`
}

// JSON is a custom type for handling JSONB data
type JSON map[string]interface{}

// Value implements the driver.Valuer interface for JSON
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSON
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSON", value)
	}

	return json.Unmarshal(bytes, j)
}

// TableName specifies the table name for the Report model
func (Report) TableName() string {
	return "reports"
}

// IsCompleted returns true if the report generation is completed
func (r *Report) IsCompleted() bool {
	return r.Status == "completed"
}

// IsPending returns true if the report is pending generation
func (r *Report) IsPending() bool {
	return r.Status == "pending"
}

// IsFailed returns true if the report generation failed
func (r *Report) IsFailed() bool {
	return r.Status == "failed"
}

// SetStatus updates the report status
func (r *Report) SetStatus(status string) {
	r.Status = status
	r.UpdatedAt = time.Now()
}
