package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"diplomaMeetHelper/internal/adapters/cli"
	"diplomaMeetHelper/internal/adapters/db/postgres"
	mockLLM "diplomaMeetHelper/internal/adapters/llm/mock"
	mockSpeech "diplomaMeetHelper/internal/adapters/speech/mock"
	"diplomaMeetHelper/internal/adapters/workerpool"
	"diplomaMeetHelper/internal/config"
	"diplomaMeetHelper/internal/domain"
	"diplomaMeetHelper/internal/usecase"
	"diplomaMeetHelper/internal/version"
	"diplomaMeetHelper/migrations"
	"diplomaMeetHelper/pkg/logger"
	"diplomaMeetHelper/pkg/migrator"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func Run(parentCtx context.Context) error {
	version.PrintBuildInfo()

	ctx, stop := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
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

	// Repositories
	userRepo := postgres.NewUserRepository(dbPool)
	meetingRepo := postgres.NewMeetingRepository(dbPool)
	jobRepo := postgres.NewJobRepository(dbPool)
	searchRepo := postgres.NewSearchRepository(dbPool)
	qaRepo := postgres.NewQARepository(dbPool)

	// External Speech Client
	var speechClient usecase.SpeechClient
	switch strings.ToLower(cfg.Speech.Provider) {
	case "mock":
		speechClient = mockSpeech.NewSpeechClient(cfg.Speech.Mock.Delay, cfg.Speech.Mock.ErrorRate)
	default:
		return fmt.Errorf("unsupported speech provider: %q (supported: mock)", cfg.Speech.Provider)
	}

	// External LLM Client
	type fullLLMClient interface {
		usecase.LLMClient
		usecase.ChatLLMClient
	}

	var llmClient fullLLMClient
	switch strings.ToLower(cfg.LLM.Provider) {
	case "mock":
		llmClient = mockLLM.NewLLMClient(cfg.LLM.Mock.Delay, cfg.LLM.Mock.ErrorRate)
	default:
		return fmt.Errorf("unsupported LLM provider: %q (supported: mock)", cfg.LLM.Provider)
	}

	// Worker Pool
	jobHandler := func(jobCtx context.Context, job domain.ProcessingJob) error {
		return usecase.ProcessJobHandler(jobCtx, job, meetingRepo, jobRepo, speechClient, llmClient, log)
	}

	pool := workerpool.New(cfg.App.Workers, 100, cfg.App.JobTimeout, jobHandler, log)
	pool.Start()
	defer pool.Stop(cfg.App.JobTimeout)

	// Startup Reconciliation: автовосстановление незавершенных задач при рестарте
	pendingJobs, err := jobRepo.GetPendingJobs(ctx)
	if err != nil {
		log.Warn("Failed to fetch pending jobs for recovery", zap.Error(err))
	} else if len(pendingJobs) > 0 {
		log.Info("Recovering pending jobs on startup", zap.Int("count", len(pendingJobs)))
		for _, job := range pendingJobs {
			if err := pool.Submit(job); err != nil {
				log.Warn("Failed to submit recovered job to queue",
					zap.String("job_id", job.ID.String()),
					zap.Error(err),
				)
			}
		}
	}

	// Usecases
	userUsecase := usecase.NewUserUsecase(userRepo)
	meetingUsecase := usecase.NewMeetingUsecase(meetingRepo, jobRepo, pool, log)
	searchUsecase := usecase.NewSearchUsecase(searchRepo, log)
	chatUsecase := usecase.NewChatUsecase(meetingRepo, qaRepo, llmClient, log)

	// CLI Presentation Layer
	cliHandler := cli.NewHandler(userUsecase, meetingUsecase, searchUsecase, chatUsecase, log)
	cliHandler.RootCmd().Version = version.Version

	return cliHandler.Execute(ctx, os.Args[1:])
}
