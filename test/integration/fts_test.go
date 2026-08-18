package integration_test

import (
	"testing"
	"time"

	"diplomaMeetHelper/internal/adapters/db/postgres"
	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
)

func TestFTSAndQA_Integration(t *testing.T) {
	pool, ctx := setupTestDB(t)
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	meetingRepo := postgres.NewMeetingRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)
	searchRepo := postgres.NewSearchRepository(pool)
	qaRepo := postgres.NewQARepository(pool)

	userID := "fts-user-" + uuid.NewString()
	_, _, err := userRepo.EnsureUser(ctx, userID)
	if err != nil {
		t.Fatalf("EnsureUser failed: %v", err)
	}

	meetingID := uuid.New()
	jobID := uuid.New()
	now := time.Now()

	meeting := &domain.Meeting{
		ID:        meetingID,
		UserID:    userID,
		Filename:  "architecture.mp3",
		FilePath:  "/tmp/architecture.mp3",
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

	transcript := "Сегодня на встрече мы подробно обсудили архитектуру микросервисов и стратегию миграции базы данных."
	summary := "Кратко: выбрана чистая архитектура и миграции PostgreSQL."
	if err := jobRepo.CompleteJobWithResults(ctx, jobID, meetingID, transcript, summary); err != nil {
		t.Fatalf("CompleteJobWithResults failed: %v", err)
	}

	// 1. Test FTS Search by single word
	results, err := searchRepo.Search(ctx, userID, "микросервисов")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Errorf("expected at least 1 search result for 'микросервисов'")
	}

	// 2. Test Save & Get QA History
	qaItem := &domain.QAItem{
		ID:        uuid.New(),
		MeetingID: meetingID,
		UserID:    userID,
		Question:  "Какая архитектура выбрана?",
		Answer:    "Выбрана чистая архитектура.",
		CreatedAt: time.Now(),
	}

	if err := qaRepo.SaveQA(ctx, qaItem); err != nil {
		t.Fatalf("SaveQA failed: %v", err)
	}

	history, err := qaRepo.GetQAHistory(ctx, meetingID, userID)
	if err != nil {
		t.Fatalf("GetQAHistory failed: %v", err)
	}

	if len(history) != 1 || history[0].Question != "Какая архитектура выбрана?" {
		t.Errorf("unexpected QA history: %+v", history)
	}
}
