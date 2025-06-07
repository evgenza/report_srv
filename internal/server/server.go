package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"report_srv/internal/config"
	"report_srv/internal/models"
	"report_srv/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

const (
	// API версии
	APIVersion = "v1"
	APIPrefix  = "/api/" + APIVersion

	// HTTP заголовки
	HeaderContentType   = "Content-Type"
	HeaderAuthorization = "Authorization"
	HeaderRequestID     = "X-Request-ID"

	// Лимиты
	DefaultPageSize = 20
	MaxPageSize     = 100

	// Таймауты
	DefaultRequestTimeout  = 30 * time.Second
	DefaultShutdownTimeout = 10 * time.Second
)

// HTTPServer интерфейс для HTTP сервера
type HTTPServer interface {
	Start(address string) error
	Shutdown(ctx context.Context) error
	GetEcho() *echo.Echo
}

// Handler интерфейс для обработчиков
type Handler interface {
	Register(group *echo.Group)
}

// Middleware интерфейс для middleware
type Middleware interface {
	Apply(e *echo.Echo)
}

// ResponseWriter интерфейс для формирования ответов
type ResponseWriter interface {
	Success(c echo.Context, data interface{}) error
	Error(c echo.Context, err error) error
	ValidationError(c echo.Context, err error) error
	NotFound(c echo.Context, message string) error
}

// APIResponse стандартная структура ответа API
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Meta      *APIMeta    `json:"meta,omitempty"`
	Timestamp string      `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// APIError структура ошибки API
type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// APIMeta метаинформация для пагинации
type APIMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PaginationParams параметры пагинации
type PaginationParams struct {
	Page     int `query:"page" validate:"min=1"`
	PageSize int `query:"page_size" validate:"min=1,max=100"`
}

// CreateReportRequest запрос на создание отчета
type CreateReportRequest struct {
	Title       string                 `json:"title" validate:"required,min=1,max=255"`
	Description string                 `json:"description" validate:"max=1000"`
	Parameters  map[string]interface{} `json:"parameters"`
	CreatedBy   string                 `json:"created_by" validate:"required,min=1,max=255"`
}

// Server реализация HTTP сервера
type Server struct {
	echo           *echo.Echo
	config         config.Config
	logger         *logrus.Logger
	validator      *validator.Validate
	responseWriter ResponseWriter
	handlers       []Handler
	middlewares    []Middleware
}

// ServerBuilder строитель для сервера
type ServerBuilder struct {
	config          config.Config
	logger          *logrus.Logger
	reportService   service.ReportService
	handlers        []Handler
	middlewares     []Middleware
	customValidator *validator.Validate
}

// NewServerBuilder создает новый строитель сервера
func NewServerBuilder(cfg config.Config, logger *logrus.Logger) *ServerBuilder {
	return &ServerBuilder{
		config:      cfg,
		logger:      logger,
		handlers:    make([]Handler, 0),
		middlewares: make([]Middleware, 0),
	}
}

// WithReportService добавляет сервис отчетов
func (b *ServerBuilder) WithReportService(service service.ReportService) *ServerBuilder {
	// Автоматически добавляем handler для отчетов
	b.handlers = append(b.handlers, NewReportHandler(service, b.logger))
	return b
}

// WithHandler добавляет кастомный handler
func (b *ServerBuilder) WithHandler(handler Handler) *ServerBuilder {
	b.handlers = append(b.handlers, handler)
	return b
}

// WithMiddleware добавляет кастомный middleware
func (b *ServerBuilder) WithMiddleware(middleware Middleware) *ServerBuilder {
	b.middlewares = append(b.middlewares, middleware)
	return b
}

// WithValidator устанавливает кастомный валидатор
func (b *ServerBuilder) WithValidator(v *validator.Validate) *ServerBuilder {
	b.customValidator = v
	return b
}

// Build создает и настраивает сервер
func (b *ServerBuilder) Build() HTTPServer {
	e := echo.New()
	e.Debug = b.config.Server.Debug
	e.HideBanner = true

	// Создаем валидатор
	v := b.customValidator
	if v == nil {
		v = validator.New()
	}

	// Создаем response writer
	responseWriter := NewJSONResponseWriter(b.logger)

	server := &Server{
		echo:           e,
		config:         b.config,
		logger:         b.logger,
		validator:      v,
		responseWriter: responseWriter,
		handlers:       b.handlers,
		middlewares:    b.middlewares,
	}

	server.setupMiddleware()
	server.setupRoutes()
	server.setupErrorHandler()

	return server
}

// JSONResponseWriter реализация ResponseWriter для JSON ответов
type JSONResponseWriter struct {
	logger *logrus.Logger
}

// NewJSONResponseWriter создает новый JSONResponseWriter
func NewJSONResponseWriter(logger *logrus.Logger) ResponseWriter {
	return &JSONResponseWriter{logger: logger}
}

// Success отправляет успешный ответ
func (w *JSONResponseWriter) Success(c echo.Context, data interface{}) error {
	response := &APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: getRequestID(c),
	}
	return c.JSON(http.StatusOK, response)
}

// Error отправляет ответ с ошибкой
func (w *JSONResponseWriter) Error(c echo.Context, err error) error {
	w.logger.WithError(err).Error("API error occurred")

	response := &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "INTERNAL_ERROR",
			Message: "Внутренняя ошибка сервера",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: getRequestID(c),
	}

	return c.JSON(http.StatusInternalServerError, response)
}

// ValidationError отправляет ответ с ошибкой валидации
func (w *JSONResponseWriter) ValidationError(c echo.Context, err error) error {
	details := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			details[fieldError.Field()] = getValidationMessage(fieldError)
		}
	}

	response := &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Ошибка валидации данных",
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: getRequestID(c),
	}

	return c.JSON(http.StatusBadRequest, response)
}

// NotFound отправляет ответ о том, что ресурс не найден
func (w *JSONResponseWriter) NotFound(c echo.Context, message string) error {
	response := &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "NOT_FOUND",
			Message: message,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: getRequestID(c),
	}

	return c.JSON(http.StatusNotFound, response)
}

// ReportHandler обработчик для отчетов
type ReportHandler struct {
	service        service.ReportService
	logger         *logrus.Logger
	responseWriter ResponseWriter
	validator      *validator.Validate
}

// NewReportHandler создает новый обработчик отчетов
func NewReportHandler(service service.ReportService, logger *logrus.Logger) Handler {
	return &ReportHandler{
		service:        service,
		logger:         logger,
		responseWriter: NewJSONResponseWriter(logger),
		validator:      validator.New(),
	}
}

// Register регистрирует маршруты для отчетов
func (h *ReportHandler) Register(group *echo.Group) {
	reports := group.Group("/reports")
	{
		reports.POST("", h.createReport)
		reports.GET("", h.listReports)
		reports.GET("/:id", h.getReport)
		reports.DELETE("/:id", h.deleteReport)
		reports.GET("/:id/download", h.downloadReport)
		reports.PUT("/:id/status", h.updateReportStatus)
	}
}

// HealthHandler обработчик для health check
type HealthHandler struct {
	responseWriter ResponseWriter
	startTime      time.Time
}

// NewHealthHandler создает новый health handler
func NewHealthHandler() Handler {
	return &HealthHandler{
		responseWriter: NewJSONResponseWriter(logrus.New()),
		startTime:      time.Now(),
	}
}

// Register регистрирует health маршруты
func (h *HealthHandler) Register(group *echo.Group) {
	group.GET("/health", h.healthCheck)
	group.GET("/health/ready", h.readinessCheck)
	group.GET("/health/live", h.livenessCheck)
}

// GetEcho возвращает экземпляр Echo
func (s *Server) GetEcho() *echo.Echo {
	return s.echo
}

// Start запускает HTTP сервер
func (s *Server) Start(address string) error {
	s.logger.WithField("address", address).Info("Запуск HTTP сервера")
	return s.echo.Start(address)
}

// Shutdown корректно останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Остановка HTTP сервера")
	return s.echo.Shutdown(ctx)
}

// setupMiddleware настраивает middleware
func (s *Server) setupMiddleware() {
	// Базовые middleware
	s.echo.Use(middleware.RequestID())
	s.echo.Use(middleware.Recover())
	s.echo.Use(middleware.CORS())

	// Логирование
	if s.config.Server.Debug {
		s.echo.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format: "${time_rfc3339} ${method} ${uri} ${status} ${latency_human} ${error}\n",
		}))
	}

	// Таймаут для запросов
	s.echo.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: DefaultRequestTimeout,
	}))

	// Rate limiting (базовый)
	s.echo.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(100)))

	// Кастомные middleware
	for _, mw := range s.middlewares {
		mw.Apply(s.echo)
	}
}

// setupRoutes настраивает маршруты
func (s *Server) setupRoutes() {
	// Группа API
	api := s.echo.Group(APIPrefix)

	// Health handler по умолчанию
	healthHandler := NewHealthHandler()
	healthHandler.Register(s.echo.Group(""))

	// Регистрируем все handlers
	for _, handler := range s.handlers {
		handler.Register(api)
	}
}

// setupErrorHandler настраивает обработчик ошибок
func (s *Server) setupErrorHandler() {
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		he, ok := err.(*echo.HTTPError)
		if !ok {
			he = &echo.HTTPError{
				Code:    http.StatusInternalServerError,
				Message: "Внутренняя ошибка сервера",
			}
		}

		response := &APIResponse{
			Success: false,
			Error: &APIError{
				Code:    fmt.Sprintf("HTTP_%d", he.Code),
				Message: fmt.Sprintf("%v", he.Message),
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: getRequestID(c),
		}

		if err := c.JSON(he.Code, response); err != nil {
			s.logger.WithError(err).Error("Ошибка отправки HTTP error response")
		}
	}
}

// createReport создает новый отчет
func (h *ReportHandler) createReport(c echo.Context) error {
	var req CreateReportRequest

	if err := c.Bind(&req); err != nil {
		return h.responseWriter.ValidationError(c, err)
	}

	if err := h.validator.Struct(&req); err != nil {
		return h.responseWriter.ValidationError(c, err)
	}

	// Создаем отчет через builder
	report, err := models.NewReportBuilder().
		WithTitle(req.Title).
		WithDescription(req.Description).
		WithCreatedBy(req.CreatedBy).
		WithParameters(req.Parameters).
		Build()

	if err != nil {
		return h.responseWriter.ValidationError(c, err)
	}

	if err := h.service.CreateReport(c.Request().Context(), report); err != nil {
		return h.responseWriter.Error(c, err)
	}

	return c.JSON(http.StatusCreated, &APIResponse{
		Success:   true,
		Data:      report,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: getRequestID(c),
	})
}

// listReports возвращает список отчетов с пагинацией
func (h *ReportHandler) listReports(c echo.Context) error {
	var pagination PaginationParams
	pagination.Page = 1
	pagination.PageSize = DefaultPageSize

	if err := c.Bind(&pagination); err != nil {
		return h.responseWriter.ValidationError(c, err)
	}

	if err := h.validator.Struct(&pagination); err != nil {
		return h.responseWriter.ValidationError(c, err)
	}

	// Создаем параметры для ListReports
	params := service.ListReportParams{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	reportList, err := h.service.ListReports(c.Request().Context(), params)
	if err != nil {
		return h.responseWriter.Error(c, err)
	}

	response := &APIResponse{
		Success: true,
		Data:    reportList.Reports,
		Meta: &APIMeta{
			Page:       reportList.Page,
			PageSize:   reportList.PageSize,
			Total:      int(reportList.Total),
			TotalPages: reportList.TotalPages,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: getRequestID(c),
	}

	return c.JSON(http.StatusOK, response)
}

// getReport возвращает отчет по ID
func (h *ReportHandler) getReport(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.responseWriter.ValidationError(c, fmt.Errorf("неверный ID отчета"))
	}

	report, err := h.service.GetReport(c.Request().Context(), id)
	if err != nil {
		return h.responseWriter.NotFound(c, "Отчет не найден")
	}

	return h.responseWriter.Success(c, report)
}

// deleteReport удаляет отчет
func (h *ReportHandler) deleteReport(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.responseWriter.ValidationError(c, fmt.Errorf("неверный ID отчета"))
	}

	if err := h.service.DeleteReport(c.Request().Context(), id); err != nil {
		return h.responseWriter.Error(c, err)
	}

	return h.responseWriter.Success(c, map[string]string{
		"message": "Отчет успешно удален",
	})
}

// downloadReport возвращает ссылку на скачивание отчета
func (h *ReportHandler) downloadReport(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.responseWriter.ValidationError(c, fmt.Errorf("неверный ID отчета"))
	}

	report, err := h.service.GetReport(c.Request().Context(), id)
	if err != nil {
		return h.responseWriter.NotFound(c, "Отчет не найден")
	}

	if !report.IsCompleted() {
		return c.JSON(http.StatusBadRequest, &APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "REPORT_NOT_READY",
				Message: "Отчет еще не готов для скачивания",
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: getRequestID(c),
		})
	}

	if !report.HasFile() {
		return h.responseWriter.NotFound(c, "Файл отчета не найден")
	}

	downloadInfo := map[string]interface{}{
		"download_url": "/files/" + report.FileKey,
		"filename":     report.Title + ".xlsx",
		"status":       "ready",
		"file_size":    "unknown", // В реальном приложении получили бы размер файла
	}

	return h.responseWriter.Success(c, downloadInfo)
}

// updateReportStatus обновляет статус отчета
func (h *ReportHandler) updateReportStatus(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return h.responseWriter.ValidationError(c, fmt.Errorf("неверный ID отчета"))
	}

	var req struct {
		Status    string `json:"status" validate:"required"`
		UpdatedBy string `json:"updated_by" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return h.responseWriter.ValidationError(c, err)
	}

	if err := h.validator.Struct(&req); err != nil {
		return h.responseWriter.ValidationError(c, err)
	}

	report, err := h.service.GetReport(c.Request().Context(), id)
	if err != nil {
		return h.responseWriter.NotFound(c, "Отчет не найден")
	}

	status := models.ReportStatus(req.Status)
	if err := report.SetStatus(status, req.UpdatedBy); err != nil {
		return h.responseWriter.ValidationError(c, err)
	}

	// В реальном приложении здесь был бы вызов service.UpdateReport

	return h.responseWriter.Success(c, report)
}

// healthCheck обработчик health check
func (h *HealthHandler) healthCheck(c echo.Context) error {
	data := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "report-service",
		"version":   APIVersion,
		"uptime":    time.Since(h.startTime).String(),
	}

	return h.responseWriter.Success(c, data)
}

// readinessCheck проверка готовности сервиса
func (h *HealthHandler) readinessCheck(c echo.Context) error {
	// Здесь можно добавить проверки готовности (БД, внешние сервисы)
	return h.responseWriter.Success(c, map[string]string{
		"status": "ready",
	})
}

// livenessCheck проверка жизни сервиса
func (h *HealthHandler) livenessCheck(c echo.Context) error {
	return h.responseWriter.Success(c, map[string]string{
		"status": "alive",
	})
}

// Вспомогательные функции

// getRequestID извлекает Request ID из контекста
func getRequestID(c echo.Context) string {
	return c.Response().Header().Get(echo.HeaderXRequestID)
}

// parseUintParam парсит uint параметр из URL
func parseUintParam(c echo.Context, paramName string) (uint, error) {
	idStr := c.Param(paramName)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// getValidationMessage возвращает человекочитаемое сообщение об ошибке валидации
func getValidationMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "Поле обязательно для заполнения"
	case "min":
		return fmt.Sprintf("Минимальная длина: %s", fieldError.Param())
	case "max":
		return fmt.Sprintf("Максимальная длина: %s", fieldError.Param())
	case "email":
		return "Неверный формат email"
	default:
		return "Неверное значение поля"
	}
}

// NewServer создает новый HTTP сервер (обратная совместимость)
func NewServer(cfg config.Config, reportService service.ReportService, logger *logrus.Logger) HTTPServer {
	return NewServerBuilder(cfg, logger).
		WithReportService(reportService).
		Build()
}
