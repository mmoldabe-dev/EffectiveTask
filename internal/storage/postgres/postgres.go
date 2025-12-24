package postgres

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"
	_ "github.com/lib/pq"
	"github.com/mmoldabe-dev/EffectiveTask/internal/config"
)

func NewPostgres(cfg *config.Config, log *slog.Logger) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password,
		cfg.Database.DBName, cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Error("failed to connect database", slog.String("error", err.Error()))
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		log.Error("database ping failed", slog.String("error", err.Error()))
		return nil, err
	}

	log.Info("database connection established",
		slog.String("host", cfg.Database.Host),
		slog.Int("port", cfg.Database.Port),
		slog.String("database", cfg.Database.DBName),
	)

	return db, nil
}
