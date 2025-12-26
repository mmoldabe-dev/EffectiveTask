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

//@title Effective Task
//@version 1.0
//@description The test task for "effective mobile"

// host@ localhost:8080
// basePath /
func main() {
	// грузим конфиг
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("cant load config: %s", err)
		os.Exit(1)
	}

	log := logger.SetupLogger(cfg.Logger.Level, "effective_task")

	// запускаем миграции перед стартом
	if err := postgres.RunMigrations(cfg, log); err != nil {
		log.Error("migration faild", slog.String("err", err.Error()))
		os.Exit(1)
	}

	db, err := postgres.NewPostgres(cfg, log)
	if err != nil {
		log.Error("db init error")
		os.Exit(1)
	}
	defer db.Close()

	// собираем слои
	repo := repository.NewSubscriptionRepository(db, log)
	svc := service.NewSubscriptionService(repo, log)
	h := handler.NewHandlerSubscription(svc, log)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      h.SetupRouter(), // прокидываем роутер
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info("server starting...", slog.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen error", slog.String("err", err.Error()))
		}
	}()

	// ждем сигнал на выход
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("stopping server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", slog.String("error", err.Error()))
	}

	log.Info("server stopped")
}
