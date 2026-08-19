package integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"diplomaMeetHelper/internal/adapters/db/postgres"
	mockLLM "diplomaMeetHelper/internal/adapters/llm/mock"
	mockSpeech "diplomaMeetHelper/internal/adapters/speech/mock"
	"diplomaMeetHelper/internal/adapters/workerpool"
	"diplomaMeetHelper/internal/domain"
	"diplomaMeetHelper/internal/usecase"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestStartupRecovery_Integration(t *testing.T) {
	pool, ctx := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	userRepo := postgres.NewUserRepository(pool)
	meetingRepo := postgres.NewMeetingRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)
	speechClient := mockSpeech.NewSpeechClient(5*time.Millisecond, 0)
	llmClient := mockLLM.NewLLMClient(5*time.Millisecond, 0)

	userID := "recovery-user-" + uuid.NewString()
	_, _, _ = userRepo.EnsureUser(ctx, userID)

	// 1. Simulate 3 unfinished jobs in database (created prior to app startup)
	var meetingIDs []uuid.UUID
	for i := 0; i < 3; i++ {
		mID := uuid.New()
		jID := uuid.New()
		now := time.Now()

		meeting := &domain.Meeting{
			ID:        mID,
			UserID:    userID,
			Filename:  fmt.Sprintf("meeting_%d.txt", i),
			FilePath:  "../../test/fixtures/meeting1_architecture.txt",
			Status:    domain.StatusCreated,
			CreatedAt: now,
			UpdatedAt: now,
		}

		job := &domain.ProcessingJob{
			ID:        jID,
			MeetingID: mID,
			UserID:    userID,
			Status:    domain.StatusCreated,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := meetingRepo.CreateMeetingWithJob(ctx, meeting, job); err != nil {
			t.Fatalf("CreateMeetingWithJob failed: %v", err)
		}
		meetingIDs = append(meetingIDs, mID)
	}

	// 2. Start WorkerPool and perform Startup Recovery
	jobHandler := func(jobCtx context.Context, job domain.ProcessingJob) error {
		return usecase.ProcessJobHandler(jobCtx, job, meetingRepo, jobRepo, speechClient, llmClient, logger)
	}
	workerPool := workerpool.New(3, 10, 5*time.Second, jobHandler, logger)
	workerPool.Start()
	defer workerPool.Stop(5 * time.Second)

	pendingJobs, err := jobRepo.GetPendingJobs(ctx)
	if err != nil {
		t.Fatalf("GetPendingJobs failed: %v", err)
	}

	if len(pendingJobs) < 3 {
		t.Fatalf("expected at least 3 pending jobs, got %d", len(pendingJobs))
	}

	for _, j := range pendingJobs {
		_ = workerPool.Submit(j)
	}

	// 3. Wait for all recovered jobs to complete
	for _, mID := range meetingIDs {
		var status *domain.JobStatusInfo
		for attempt := 0; attempt < 30; attempt++ {
			time.Sleep(50 * time.Millisecond)
			status, err = jobRepo.GetJobByMeetingID(ctx, mID, userID)
			if err == nil && status.Status == domain.StatusCompleted {
				break
			}
		}
		if status == nil || status.Status != domain.StatusCompleted {
			t.Fatalf("job for meeting %s was not recovered successfully: %+v", mID, status)
		}
	}
}

func TestConcurrentLoad_StressTest(t *testing.T) {
	pool, ctx := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()
	userRepo := postgres.NewUserRepository(pool)
	meetingRepo := postgres.NewMeetingRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)
	speechClient := mockSpeech.NewSpeechClient(2*time.Millisecond, 0)
	llmClient := mockLLM.NewLLMClient(2*time.Millisecond, 0)

	jobHandler := func(jobCtx context.Context, job domain.ProcessingJob) error {
		return usecase.ProcessJobHandler(jobCtx, job, meetingRepo, jobRepo, speechClient, llmClient, logger)
	}
	workerPool := workerpool.New(5, 50, 10*time.Second, jobHandler, logger)
	workerPool.Start()
	defer workerPool.Stop(10 * time.Second)

	meetingUC := usecase.NewMeetingUsecase(meetingRepo, jobRepo, workerPool, logger)

	const numUsers = 4
	const tasksPerUser = 4
	var wg sync.WaitGroup

	for u := 0; u < numUsers; u++ {
		userID := fmt.Sprintf("stress-user-%d-%s", u, uuid.NewString()[:8])
		_, _, _ = userRepo.EnsureUser(ctx, userID)

		for tIdx := 0; tIdx < tasksPerUser; tIdx++ {
			wg.Add(1)
			go func(uid string) {
				defer wg.Done()
				_, err := meetingUC.LoadMeeting(ctx, uid, "../../test/fixtures/meeting1_architecture.txt")
				if err != nil {
					t.Errorf("LoadMeeting failed under stress: %v", err)
				}
			}(userID)
		}
	}

	wg.Wait()
}
