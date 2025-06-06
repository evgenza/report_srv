package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"report_srv/internal/models"
	"report_srv/internal/storage"

	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ReportService handles report generation and management
type ReportService struct {
	db      *gorm.DB
	storage storage.Storage
	logger  *logrus.Logger
}

// NewReportService creates a new report service
func NewReportService(db *gorm.DB, storage storage.Storage, logger *logrus.Logger) *ReportService {
	return &ReportService{
		db:      db,
		storage: storage,
		logger:  logger,
	}
}

// CreateReport creates a new report
func (s *ReportService) CreateReport(ctx context.Context, report *models.Report) error {
	report.Status = "pending"
	report.GeneratedAt = time.Now()

	if err := s.db.WithContext(ctx).Create(report).Error; err != nil {
		s.logger.WithError(err).Error("Failed to create report in database")
		return fmt.Errorf("failed to create report: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"report_id": report.ID,
		"title":     report.Title,
		"status":    report.Status,
	}).Info("Report created successfully")

	// Start report generation in background
	go s.generateReport(context.Background(), report)

	return nil
}

// GetReport retrieves a report by ID
func (s *ReportService) GetReport(ctx context.Context, id uint) (*models.Report, error) {
	var report models.Report
	if err := s.db.WithContext(ctx).First(&report, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.WithField("report_id", id).Warn("Report not found")
			return nil, fmt.Errorf("report not found")
		}
		s.logger.WithError(err).WithField("report_id", id).Error("Failed to get report from database")
		return nil, fmt.Errorf("failed to get report: %w", err)
	}
	return &report, nil
}

// ListReports retrieves all reports
func (s *ReportService) ListReports(ctx context.Context) ([]models.Report, error) {
	var reports []models.Report
	if err := s.db.WithContext(ctx).Order("created_at DESC").Find(&reports).Error; err != nil {
		s.logger.WithError(err).Error("Failed to list reports from database")
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}

	s.logger.WithField("count", len(reports)).Info("Retrieved reports list")
	return reports, nil
}

// DeleteReport deletes a report
func (s *ReportService) DeleteReport(ctx context.Context, id uint) error {
	var report models.Report
	if err := s.db.WithContext(ctx).First(&report, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.WithField("report_id", id).Warn("Report not found for deletion")
			return fmt.Errorf("report not found")
		}
		return fmt.Errorf("failed to find report: %w", err)
	}

	// Delete the file from storage if it exists
	if report.FileKey != "" {
		if err := s.storage.Delete(ctx, report.FileKey); err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"report_id": id,
				"file_key":  report.FileKey,
			}).Error("Failed to delete report file from storage")
			return fmt.Errorf("failed to delete report file: %w", err)
		}
	}

	if err := s.db.WithContext(ctx).Delete(&report).Error; err != nil {
		s.logger.WithError(err).WithField("report_id", id).Error("Failed to delete report from database")
		return fmt.Errorf("failed to delete report: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"report_id": id,
		"title":     report.Title,
	}).Info("Report deleted successfully")

	return nil
}

// generateReport generates the report file
func (s *ReportService) generateReport(ctx context.Context, report *models.Report) {
	logger := s.logger.WithFields(logrus.Fields{
		"report_id": report.ID,
		"title":     report.Title,
	})

	logger.Info("Starting report generation")

	// Create a new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create a new sheet
	sheet := "Report"
	f.SetSheetName("Sheet1", sheet)

	// Add header styling
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E6E6FA"},
			Pattern: 1,
		},
	})
	if err != nil {
		logger.WithError(err).Error("Failed to create header style")
	}

	// Add some sample data with better formatting
	headers := []string{"Field", "Value"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
		if style != 0 {
			f.SetCellStyle(sheet, cell, cell, style)
		}
	}

	// Add report data
	data := [][]interface{}{
		{"Report Title", report.Title},
		{"Description", report.Description},
		{"Status", report.Status},
		{"Generated At", report.GeneratedAt.Format(time.RFC3339)},
		{"Created By", report.CreatedBy},
		{"Report ID", report.ID},
	}

	for rowIndex, row := range data {
		for colIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			f.SetCellValue(sheet, cell, value)
		}
	}

	// Auto-adjust column width
	f.SetColWidth(sheet, "A", "B", 25)

	// Save the file to a buffer
	var buffer bytes.Buffer
	if err := f.Write(&buffer); err != nil {
		s.updateReportStatus(ctx, report.ID, "failed", "", fmt.Sprintf("Failed to generate report: %v", err))
		return
	}

	// Generate a unique file key
	fileKey := fmt.Sprintf("reports/%d_%s.xlsx", report.ID, time.Now().Format("20060102150405"))

	// Save the file to storage
	if err := s.storage.Save(ctx, fileKey, &buffer); err != nil {
		s.updateReportStatus(ctx, report.ID, "failed", "", fmt.Sprintf("Failed to save report: %v", err))
		return
	}

	// Update report status
	s.updateReportStatus(ctx, report.ID, "completed", fileKey, "")
	logger.WithField("file_key", fileKey).Info("Report generation completed successfully")
}

// updateReportStatus updates the report status
func (s *ReportService) updateReportStatus(ctx context.Context, id uint, status string, fileKey string, errorMsg string) {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if fileKey != "" {
		updates["file_key"] = fileKey
	}

	if errorMsg != "" {
		// In a real application, you might want to store error messages in a separate field
		s.logger.WithFields(logrus.Fields{
			"report_id": id,
			"status":    status,
			"error":     errorMsg,
		}).Error("Report generation failed")
	}

	if err := s.db.WithContext(ctx).Model(&models.Report{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		s.logger.WithError(err).WithField("report_id", id).Error("Failed to update report status")
	} else {
		s.logger.WithFields(logrus.Fields{
			"report_id": id,
			"status":    status,
		}).Info("Report status updated")
	}
}
