package di

import (
	"net/http"

	"report_srv/internal/config"
	sqlinfra "report_srv/internal/infrastructure/sql"
	"report_srv/internal/infrastructure/storage"
	"report_srv/internal/infrastructure/template"
	httpapi "report_srv/internal/interface/http"
	"report_srv/internal/usecase"

	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

func InitializeApp() *fx.App {
	return fx.New(
		fx.Provide(
			config.Load,
			newLogger,
			newDB,
			storage.NewS3,
			template.NewXLSX,
			newReportRepo,
			usecase.NewReportService,
			httpapi.NewHandler,
			newServer,
		),
	)
}

func newLogger() *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	return l
}

func newDB(cfg config.Config) (*sqlinfra.DB, error) {
	return sqlinfra.Open(cfg.DB.Driver, cfg.DB.DSN)
}
func newReportRepo(db *sqlinfra.DB) sqlinfra.ReportRepository {
	return sqlinfra.ReportRepository{DB: db.DB}
}

func newServer(lc fx.Lifecycle, h *httpapi.ReportHandler, cfg config.Config) *http.Server {
	return httpapi.NewServer(lc, h, cfg.Server.Address)
}
