package http

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/fx"
)

// NewServer создаёт и запускает HTTP-сервер.
func NewServer(lc fx.Lifecycle, handler *ReportHandler, addr string) *http.Server {
	srv := &http.Server{
		Addr:    addr,
		Handler: handler.Routes(),
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go srv.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
