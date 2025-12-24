package main

import (
	"fmt"

	"github.com/mmoldabe-dev/EffectiveTask/internal/config"
	"github.com/mmoldabe-dev/EffectiveTask/pkg/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error to load config: %s", err)
	}
	log := logger.SetupLogger(cfg.Logger.Level, "main")

	log.Info("Config:", cfg)
}
