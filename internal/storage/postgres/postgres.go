package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

func RunMigrations(cfg *config.Config, log *slog.Logger) error {
	const op = "storage.postgres.RunMigrations"

	migrationDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.DBName, cfg.Database.SSLMode,
	)

	m, err := migrate.New("file://migrations", migrationDSN)
	if err != nil {
		return fmt.Errorf("%s: failed to create migrate instance: %w", op, err)
	}
	defer m.Close()

	log.Info("checking and applying migrations...")
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("no new migrations to apply")
			return nil
		}
		return fmt.Errorf("%s: failed to run up migrations: %w", op, err)
	}

	log.Info("migrations applied successfully")
	return nil
}
