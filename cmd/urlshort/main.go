package main

import (
	"URLShort/internal/config"
	"URLShort/internal/http-server/handlers/redirect"
	"URLShort/internal/http-server/handlers/url/save"
	mwLogger "URLShort/internal/http-server/middleware/logger"
	"URLShort/internal/lib/handlers/slogpretty"
	"URLShort/internal/lib/logger/sl"
	"URLShort/internal/storage/sqlite"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	envlocal = "local"
	envdev   = "dev"
	envprod  = "prod"
)

func main() {
	//init config cleanenv
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("start", slog.String("env", cfg.Env))
	log.Debug("debug")

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("Failed to init DB", sl.Err(err))
		os.Exit(1)
	}
	_ = storage
	router := chi.NewRouter()
	//middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.URLFormat)
	router.Use(mwLogger.New(log))

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("urlShot", map[string]string{
			cfg.HTTPServer.User: cfg.HTTPServer.Password,
		}))
		r.Post("/", save.New(log, storage))
	})

	router.Get("/{alias}", redirect.New(log, storage))

	log.Info("starting server")

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}
	log.Error("server stopped")
	// go func() {
	// 	if err := srv.ListenAndServe(); err != nil {
	// 		log.Error("failed to start server")
	// 	}
	// }()

	// log.Info("server started")

	// <-done
	// log.Info("stopping server")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envlocal:
		log = setupPrettySlog()
	case envdev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envprod:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	}
	return log
}
func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
