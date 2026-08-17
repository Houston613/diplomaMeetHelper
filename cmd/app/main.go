package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"diplomaMeetHelper/internal/adapters/cli"
	"diplomaMeetHelper/internal/adapters/db/postgres"
	"diplomaMeetHelper/internal/config"
	"diplomaMeetHelper/internal/usecase"
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	log, err := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	if err := migrator.Run(migrations.FS, ".", cfg.DB.DSN); err != nil {
		log.Error("Database migration failed", zap.Error(err))
		return fmt.Errorf("database migration failed: %w", err)
	}

	dbPool, err := pgxpool.New(ctx, cfg.DB.DSN)
	if err != nil {
		log.Error("Failed to initialize PostgreSQL pool", zap.Error(err))
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Warn("PostgreSQL ping failed. Please ensure database container is running.", zap.Error(err))
	}

	userRepo := postgres.NewUserRepository(dbPool)
	userUsecase := usecase.NewUserUsecase(userRepo)
	cliApp := cli.NewApp(userUsecase, log)
	cliApp.RootCmd().Version = version.Version

	return cliApp.Execute(ctx, os.Args[1:])
}
