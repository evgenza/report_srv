package service

import (
	"context"
	"io"
	"testing"

	"report_srv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockStorage is a mock implementation of the Storage interface
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Save(ctx context.Context, key string, reader io.Reader) error {
	args := m.Called(ctx, key, reader)
	return args.Error(0)
}

func (m *MockStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorage) GetURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) JoinPath(elem ...string) string {
	args := m.Called(elem)
	return args.String(0)
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.Report{})
	assert.NoError(t, err)

	return db
}

func TestCreateReport(t *testing.T) {
	db := setupTestDB(t)
	mockStorage := new(MockStorage)
	service := NewReportService(db, mockStorage)

	report := &models.Report{
		Title:       "Test Report",
		Description: "Test Description",
		CreatedBy:   "test-user",
		UpdatedBy:   "test-user",
	}

	err := service.CreateReport(context.Background(), report)
	assert.NoError(t, err)
	assert.NotZero(t, report.ID)
	assert.Equal(t, "pending", report.Status)
}

func TestGetReport(t *testing.T) {
	db := setupTestDB(t)
	mockStorage := new(MockStorage)
	service := NewReportService(db, mockStorage)

	// Create a test report
	report := &models.Report{
		Title:       "Test Report",
		Description: "Test Description",
		Status:      "completed",
		CreatedBy:   "test-user",
		UpdatedBy:   "test-user",
	}
	err := db.Create(report).Error
	assert.NoError(t, err)

	// Test getting the report
	retrieved, err := service.GetReport(context.Background(), report.ID)
	assert.NoError(t, err)
	assert.Equal(t, report.Title, retrieved.Title)
	assert.Equal(t, report.Description, retrieved.Description)
}

func TestListReports(t *testing.T) {
	db := setupTestDB(t)
	mockStorage := new(MockStorage)
	service := NewReportService(db, mockStorage)

	// Create test reports
	reports := []models.Report{
		{
			Title:       "Report 1",
			Description: "Description 1",
			Status:      "completed",
			CreatedBy:   "test-user",
			UpdatedBy:   "test-user",
		},
		{
			Title:       "Report 2",
			Description: "Description 2",
			Status:      "pending",
			CreatedBy:   "test-user",
			UpdatedBy:   "test-user",
		},
	}

	for i := range reports {
		err := db.Create(&reports[i]).Error
		assert.NoError(t, err)
	}

	// Test listing reports
	retrieved, err := service.ListReports(context.Background())
	assert.NoError(t, err)
	assert.Len(t, retrieved, 2)
}

func TestDeleteReport(t *testing.T) {
	db := setupTestDB(t)
	mockStorage := new(MockStorage)
	service := NewReportService(db, mockStorage)

	// Create a test report
	report := &models.Report{
		Title:       "Test Report",
		Description: "Test Description",
		Status:      "completed",
		FileKey:     "test-file.xlsx",
		CreatedBy:   "test-user",
		UpdatedBy:   "test-user",
	}
	err := db.Create(report).Error
	assert.NoError(t, err)

	// Mock storage delete
	mockStorage.On("Delete", mock.Anything, report.FileKey).Return(nil)

	// Test deleting the report
	err = service.DeleteReport(context.Background(), report.ID)
	assert.NoError(t, err)

	// Verify report is deleted
	var count int64
	db.Model(&models.Report{}).Where("id = ?", report.ID).Count(&count)
	assert.Equal(t, int64(0), count)

	mockStorage.AssertExpectations(t)
}
