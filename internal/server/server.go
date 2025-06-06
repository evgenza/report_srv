package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Config holds the server configuration
type Config struct {
	Address string
	Debug   bool
}

// Server represents the HTTP server
type Server struct {
	echo *echo.Echo
	db   *gorm.DB
	log  *logrus.Logger
}

// NewServer creates a new HTTP server
func NewServer(cfg Config, db *gorm.DB, log *logrus.Logger) *Server {
	e := echo.New()
	e.Debug = cfg.Debug

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	if cfg.Debug {
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format: "${time_rfc3339} ${method} ${uri} ${status} ${latency_human}\n",
		}))
	}

	return &Server{
		echo: e,
		db:   db,
		log:  log,
	}
}

// Start starts the HTTP server
func (s *Server) Start(address string) error {
	s.setupRoutes()
	return s.echo.Start(address)
}

// setupRoutes configures the server routes
func (s *Server) setupRoutes() {
	// Health check
	s.echo.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

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

// createReport handles report creation
func (s *Server) createReport(c echo.Context) error {
	// TODO: Implement report creation
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "not implemented",
	})
}

// listReports handles listing reports
func (s *Server) listReports(c echo.Context) error {
	// TODO: Implement report listing
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "not implemented",
	})
}

// getReport handles getting a single report
func (s *Server) getReport(c echo.Context) error {
	// TODO: Implement getting a single report
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "not implemented",
	})
}

// deleteReport handles report deletion
func (s *Server) deleteReport(c echo.Context) error {
	// TODO: Implement report deletion
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "not implemented",
	})
}

// downloadReport handles report download
func (s *Server) downloadReport(c echo.Context) error {
	// TODO: Implement report download
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "not implemented",
	})
}
