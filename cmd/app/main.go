package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"diplomaMeetHelper/internal/config"
	"diplomaMeetHelper/internal/version"
	"diplomaMeetHelper/migrations"
	"diplomaMeetHelper/pkg/logger"
	"diplomaMeetHelper/pkg/migrator"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	version.PrintBuildInfo()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Load & Validate Configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// 2. Initialize Structured Logger
	log, err := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	log.Info("Starting diplomaMeetHelper",
		zap.String("version", version.Version),
		zap.String("log_level", cfg.Log.Level),
		zap.String("log_format", cfg.Log.Format),
	)

	// 3. Run Automated Database Migrations
	log.Info("Applying database migrations")
	if err := migrator.Run(migrations.FS, ".", cfg.DB.DSN); err != nil {
		log.Error("Database migration failed", zap.Error(err))
		return fmt.Errorf("database migration failed: %w", err)
	}
	log.Info("Database migrations applied successfully")

	// 4. Verify PostgreSQL Pool Connection
	dbPool, err := pgxpool.New(ctx, cfg.DB.DSN)
	if err != nil {
		log.Error("Failed to initialize PostgreSQL pool", zap.Error(err))
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Warn("PostgreSQL ping failed. Please ensure database container is running.", zap.Error(err))
	} else {
		log.Info("PostgreSQL connection pool verified and ready")
	}

	log.Info("Just stop application")
	return nil
}
