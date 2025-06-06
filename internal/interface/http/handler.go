package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"report_srv/internal/usecase"
)

// ReportHandler обрабатывает HTTP-запросы к сервису отчётов.
type ReportHandler struct {
	Service *usecase.ReportService
	Logger  *logrus.Logger
}

func NewHandler(svc *usecase.ReportService, log *logrus.Logger) *ReportHandler {
	return &ReportHandler{Service: svc, Logger: log}
}

// Routes возвращает настроенный роутер.
func (h *ReportHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/reports/{id}", h.Generate)
	return r
}

// Generate запускает генерацию отчёта и отдаёт файл в ответе.
func (h *ReportHandler) Generate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	data, err := h.Service.Generate(r.Context(), id)
	if err != nil {
		h.Logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
