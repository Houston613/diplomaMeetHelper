package integration_test

import (
	"errors"
	"testing"
	"time"

	"diplomaMeetHelper/internal/adapters/db/postgres"
	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
)

func TestCascadeDelete_Integration(t *testing.T) {
	pool, ctx := setupTestDB(t)
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	meetingRepo := postgres.NewMeetingRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)
	qaRepo := postgres.NewQARepository(pool)

	userID := "delete-user-" + uuid.NewString()
	_, _, _ = userRepo.EnsureUser(ctx, userID)

	meetingID := uuid.New()
	jobID := uuid.New()
	now := time.Now()

	meeting := &domain.Meeting{
		ID:        meetingID,
		UserID:    userID,
		Filename:  "cascade_test.mp3",
		FilePath:  "/tmp/cascade_test.mp3",
		Status:    domain.StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}

	job := &domain.ProcessingJob{
		ID:        jobID,
		MeetingID: meetingID,
		UserID:    userID,
		Status:    domain.StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := meetingRepo.CreateMeetingWithJob(ctx, meeting, job); err != nil {
		t.Fatalf("CreateMeetingWithJob failed: %v", err)
	}

	if err := jobRepo.CompleteJobWithResults(ctx, jobID, meetingID, "sample transcript", "sample summary"); err != nil {
		t.Fatalf("CompleteJobWithResults failed: %v", err)
	}

	qaItem := &domain.QAItem{
		ID:        uuid.New(),
		MeetingID: meetingID,
		UserID:    userID,
		Question:  "Вопрос по встрече?",
		Answer:    "Ответ.",
		CreatedAt: time.Now(),
	}
	if err := qaRepo.SaveQA(ctx, qaItem); err != nil {
		t.Fatalf("SaveQA failed: %v", err)
	}

	// 1. Delete Meeting
	if err := meetingRepo.DeleteMeeting(ctx, meetingID, userID); err != nil {
		t.Fatalf("DeleteMeeting failed: %v", err)
	}

	// 2. Check that meeting is deleted
	_, err := meetingRepo.GetMeeting(ctx, meetingID, userID)
	if !errors.Is(err, domain.ErrMeetingNotFound) {
		t.Errorf("expected ErrMeetingNotFound, got: %v", err)
	}

	// 3. Check that job is cascade deleted
	_, err = jobRepo.GetJobByMeetingID(ctx, meetingID, userID)
	if !errors.Is(err, domain.ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound for cascade deleted job, got: %v", err)
	}

	// 4. Check that QA history is cascade deleted
	history, err := qaRepo.GetQAHistory(ctx, meetingID, userID)
	if err != nil {
		t.Fatalf("GetQAHistory failed: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0 QA records after cascade delete, got %d", len(history))
	}
}

func TestRetryJob_Integration(t *testing.T) {
	pool, ctx := setupTestDB(t)
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	meetingRepo := postgres.NewMeetingRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)

	userID := "retry-user-" + uuid.NewString()
	_, _, _ = userRepo.EnsureUser(ctx, userID)

	meetingID := uuid.New()
	jobID := uuid.New()
	now := time.Now()

	meeting := &domain.Meeting{
		ID:        meetingID,
		UserID:    userID,
		Filename:  "retry_test.mp3",
		FilePath:  "/tmp/retry_test.mp3",
		Status:    domain.StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}

	job := &domain.ProcessingJob{
		ID:        jobID,
		MeetingID: meetingID,
		UserID:    userID,
		Status:    domain.StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := meetingRepo.CreateMeetingWithJob(ctx, meeting, job); err != nil {
		t.Fatalf("CreateMeetingWithJob failed: %v", err)
	}

	// 1. Retry on non-failed job should fail with ErrJobNotRetryable
	_, err := jobRepo.RetryJob(ctx, meetingID, userID)
	if !errors.Is(err, domain.ErrJobNotRetryable) {
		t.Fatalf("expected ErrJobNotRetryable on created job, got: %v", err)
	}

	// 2. Fail the job
	if err := jobRepo.FailJob(ctx, jobID, meetingID, "speech recognition failed: timeout"); err != nil {
		t.Fatalf("FailJob failed: %v", err)
	}

	// 3. Retry on failed job should succeed
	retriedJob, err := jobRepo.RetryJob(ctx, meetingID, userID)
	if err != nil {
		t.Fatalf("RetryJob failed: %v", err)
	}

	if retriedJob.Status != domain.StatusCreated {
		t.Errorf("expected status 'created', got %s", retriedJob.Status)
	}
	if retriedJob.RetryCount != 1 {
		t.Errorf("expected retry_count 1, got %d", retriedJob.RetryCount)
	}
	if retriedJob.ErrorMessage != "" {
		t.Errorf("expected empty error_message, got %s", retriedJob.ErrorMessage)
	}
}
