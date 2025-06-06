package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"report_srv/internal/config"
	"report_srv/internal/models"
	"report_srv/internal/service"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server
type Server struct {
	echo    *echo.Echo
	service *service.ReportService
	logger  *logrus.Logger
}

// NewServer creates a new HTTP server
func NewServer(cfg config.Config, reportService *service.ReportService, logger *logrus.Logger) *Server {
	e := echo.New()
	e.Debug = cfg.Server.Debug
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RequestID())

	if cfg.Server.Debug {
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format: "${time_rfc3339} ${method} ${uri} ${status} ${latency_human} ${error}\n",
		}))
	}

	server := &Server{
		echo:    e,
		service: reportService,
		logger:  logger,
	}

	server.setupRoutes()
	return server
}

// Start starts the HTTP server
func (s *Server) Start(address string) error {
	s.logger.WithField("address", address).Info("Starting HTTP server")
	return s.echo.Start(address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.echo.Shutdown(ctx)
}

// setupRoutes configures the server routes
func (s *Server) setupRoutes() {
	// Health check
	s.echo.GET("/health", s.healthCheck)

	// API routes
	api := s.echo.Group("/api/v1")
	{
		reports := api.Group("/reports")
		{
			reports.POST("", s.createReport)
			reports.GET("", s.listReports)
			reports.GET("/:id", s.getReport)
			reports.DELETE("/:id", s.deleteReport)
			reports.GET("/:id/download", s.downloadReport)
		}
	}
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "report-service",
	})
}

// createReport handles report creation
func (s *Server) createReport(c echo.Context) error {
	var req struct {
		Title       string                 `json:"title" validate:"required"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters"`
		CreatedBy   string                 `json:"created_by" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		s.logger.WithError(err).Error("Failed to bind request")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Title is required",
		})
	}

	if req.CreatedBy == "" {
		req.CreatedBy = "anonymous"
	}

	report := &models.Report{
		Title:       req.Title,
		Description: req.Description,
		Parameters:  req.Parameters,
		CreatedBy:   req.CreatedBy,
		UpdatedBy:   req.CreatedBy,
	}

	if err := s.service.CreateReport(c.Request().Context(), report); err != nil {
		s.logger.WithError(err).Error("Failed to create report")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create report",
		})
	}

	return c.JSON(http.StatusCreated, report)
}

// listReports handles listing reports
func (s *Server) listReports(c echo.Context) error {
	reports, err := s.service.ListReports(c.Request().Context())
	if err != nil {
		s.logger.WithError(err).Error("Failed to list reports")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list reports",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"reports": reports,
		"count":   len(reports),
	})
}

// getReport handles getting a single report
func (s *Server) getReport(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid report ID",
		})
	}

	report, err := s.service.GetReport(c.Request().Context(), uint(id))
	if err != nil {
		s.logger.WithError(err).Error("Failed to get report")
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Report not found",
		})
	}

	return c.JSON(http.StatusOK, report)
}

// deleteReport handles report deletion
func (s *Server) deleteReport(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid report ID",
		})
	}

	if err := s.service.DeleteReport(c.Request().Context(), uint(id)); err != nil {
		s.logger.WithError(err).Error("Failed to delete report")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete report",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Report deleted successfully",
	})
}

// downloadReport handles report download
func (s *Server) downloadReport(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid report ID",
		})
	}

	report, err := s.service.GetReport(c.Request().Context(), uint(id))
	if err != nil {
		s.logger.WithError(err).Error("Failed to get report")
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Report not found",
		})
	}

	if report.Status != "completed" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Report is not ready for download",
		})
	}

	if report.FileKey == "" {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Report file not found",
		})
	}

	// For now, return the file key. In a real implementation,
	// you would stream the file from storage
	return c.JSON(http.StatusOK, map[string]interface{}{
		"download_url": "/files/" + report.FileKey,
		"filename":     report.Title + ".xlsx",
		"status":       "ready",
	})
}
