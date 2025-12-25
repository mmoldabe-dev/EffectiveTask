package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mmoldabe-dev/EffectiveTask/internal/config"
	"github.com/mmoldabe-dev/EffectiveTask/internal/handler"
	"github.com/mmoldabe-dev/EffectiveTask/internal/repository"
	"github.com/mmoldabe-dev/EffectiveTask/internal/service"
	"github.com/mmoldabe-dev/EffectiveTask/internal/storage/postgres"
	"github.com/mmoldabe-dev/EffectiveTask/pkg/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error to load config: %s", err)
		os.Exit(1)
	}

	log := logger.SetupLogger(cfg.Logger.Level, "effective_task")

	if err := postgres.RunMigrations(cfg, log); err != nil {
		log.Error("migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	db, err := postgres.NewPostgres(cfg, log)
	if err != nil {
		log.Error("could not initialize database storage")
		os.Exit(1)
	}
	defer db.Close()

	repo := repository.NewSubscriptionRepository(db, log)
	svc := service.NewSubscriptionService(repo, log)
	handler := handler.NewHandlerSubscription(svc, log)

	router := handler.SetupRouter()

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info("starting server", slog.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", slog.String("error", err.Error()))
	}

	log.Info("server stopped gracefully")
}
