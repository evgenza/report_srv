package models

import (
	"time"

	"gorm.io/gorm"
)

// Report represents a generated report
type Report struct {
	gorm.Model
	Title       string `gorm:"size:255;not null"`
	Description string `gorm:"size:1000"`
	Status      string `gorm:"size:50;not null;default:'pending'"`
	FileKey     string `gorm:"size:255"`
	GeneratedAt time.Time
	Parameters  JSON   `gorm:"type:jsonb"`
	CreatedBy   string `gorm:"size:255;not null"`
	UpdatedBy   string `gorm:"size:255;not null"`
}

// JSON is a custom type for handling JSONB data
type JSON map[string]interface{}

// TableName specifies the table name for the Report model
func (Report) TableName() string {
	return "reports"
}
