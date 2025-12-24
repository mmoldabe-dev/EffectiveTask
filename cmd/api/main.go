package main

import (
	"fmt"

	"github.com/mmoldabe-dev/EffectiveTask/internal/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error to load config: %s", err)
	}
	fmt.Println(cfg)
}
