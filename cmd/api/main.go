package main

import (
	"fmt"
	"os"

	"github.com/mmoldabe-dev/EffectiveTask/internal/config"
	"github.com/mmoldabe-dev/EffectiveTask/internal/storage/postgres"
	"github.com/mmoldabe-dev/EffectiveTask/pkg/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error to load config: %s", err)
	}
	log := logger.SetupLogger(cfg.Logger.Level, "effective_task")

	db, err := postgres.NewPostgres(cfg, log)
	if err != nil {
		log.Error("could not initialize database storage")
		os.Exit(1)
	}
	defer db.Close()

	log.Info("application is ready to work")
}
