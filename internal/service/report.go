package service

import (
	"context"
	"fmt"
	"time"

	"report_srv/internal/models"
	"report_srv/internal/storage"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ReportService handles report generation and management
type ReportService struct {
	db      *gorm.DB
	storage storage.Storage
}

// NewReportService creates a new report service
func NewReportService(db *gorm.DB, storage storage.Storage) *ReportService {
	return &ReportService{
		db:      db,
		storage: storage,
	}
}

// CreateReport creates a new report
func (s *ReportService) CreateReport(ctx context.Context, report *models.Report) error {
	report.Status = "pending"
	report.GeneratedAt = time.Now()

	if err := s.db.Create(report).Error; err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	// Start report generation in background
	go s.generateReport(ctx, report)

	return nil
}

// GetReport retrieves a report by ID
func (s *ReportService) GetReport(ctx context.Context, id uint) (*models.Report, error) {
	var report models.Report
	if err := s.db.First(&report, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}
	return &report, nil
}

// ListReports retrieves all reports
func (s *ReportService) ListReports(ctx context.Context) ([]models.Report, error) {
	var reports []models.Report
	if err := s.db.Find(&reports).Error; err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}
	return reports, nil
}

// DeleteReport deletes a report
func (s *ReportService) DeleteReport(ctx context.Context, id uint) error {
	var report models.Report
	if err := s.db.First(&report, id).Error; err != nil {
		return fmt.Errorf("failed to find report: %w", err)
	}

	// Delete the file from storage if it exists
	if report.FileKey != "" {
		if err := s.storage.Delete(ctx, report.FileKey); err != nil {
			return fmt.Errorf("failed to delete report file: %w", err)
		}
	}

	if err := s.db.Delete(&report).Error; err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	return nil
}

// generateReport generates the report file
func (s *ReportService) generateReport(ctx context.Context, report *models.Report) {
	// Create a new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create a new sheet
	sheet := "Report"
	f.SetSheetName("Sheet1", sheet)

	// Add some sample data
	f.SetCellValue(sheet, "A1", "Report Title")
	f.SetCellValue(sheet, "B1", report.Title)
	f.SetCellValue(sheet, "A2", "Generated At")
	f.SetCellValue(sheet, "B2", report.GeneratedAt.Format(time.RFC3339))

	// Save the file to a buffer
	buffer, err := f.WriteToBuffer()
	if err != nil {
		s.updateReportStatus(ctx, report.ID, "failed", fmt.Sprintf("Failed to generate report: %v", err))
		return
	}

	// Generate a unique file key
	fileKey := fmt.Sprintf("reports/%d_%s.xlsx", report.ID, time.Now().Format("20060102150405"))

	// Save the file to storage
	if err := s.storage.Save(ctx, fileKey, buffer); err != nil {
		s.updateReportStatus(ctx, report.ID, "failed", fmt.Sprintf("Failed to save report: %v", err))
		return
	}

	// Update report status
	s.updateReportStatus(ctx, report.ID, "completed", fileKey)
}

// updateReportStatus updates the report status
func (s *ReportService) updateReportStatus(ctx context.Context, id uint, status string, fileKey string) {
	updates := map[string]interface{}{
		"status": status,
	}
	if fileKey != "" {
		updates["file_key"] = fileKey
	}

	if err := s.db.Model(&models.Report{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		// Log the error but don't return it since this is a background operation
		fmt.Printf("Failed to update report status: %v\n", err)
	}
}
