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

	user1 := "fts-user-1-" + uuid.NewString()
	user2 := "fts-user-2-" + uuid.NewString()

	_, _, _ = userRepo.EnsureUser(ctx, user1)
	_, _, _ = userRepo.EnsureUser(ctx, user2)

	meetingID := uuid.New()
	jobID := uuid.New()
	now := time.Now()

	meeting := &domain.Meeting{
		ID:        meetingID,
		UserID:    user1,
		Filename:  "architecture.mp3",
		FilePath:  "/tmp/architecture.mp3",
		Status:    domain.StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}

	job := &domain.ProcessingJob{
		ID:        jobID,
		MeetingID: meetingID,
		UserID:    user1,
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

	// 1. Test Multi-word Search with special characters (safe via plainto_tsquery)
	results, err := searchRepo.Search(ctx, user1, "архитектура и миграции?")
	if err != nil {
		t.Fatalf("Search failed on multi-word query: %v", err)
	}

	if len(results) == 0 {
		t.Errorf("expected search results for 'архитектура и миграции?'")
	}

	// 2. Test User Isolation (user2 must not see user1's results)
	user2Results, err := searchRepo.Search(ctx, user2, "архитектура")
	if err != nil {
		t.Fatalf("user2 search failed: %v", err)
	}
	if len(user2Results) != 0 {
		t.Errorf("expected 0 results for user2, got %d", len(user2Results))
	}

	// 3. Test Save & Get QA History
	qaItem := &domain.QAItem{
		ID:        uuid.New(),
		MeetingID: meetingID,
		UserID:    user1,
		Question:  "Какая архитектура выбрана?",
		Answer:    "Выбрана чистая архитектура.",
		CreatedAt: time.Now(),
	}

	if err := qaRepo.SaveQA(ctx, qaItem); err != nil {
		t.Fatalf("SaveQA failed: %v", err)
	}

	history, err := qaRepo.GetQAHistory(ctx, meetingID, user1)
	if err != nil {
		t.Fatalf("GetQAHistory failed: %v", err)
	}

	if len(history) != 1 || history[0].Question != "Какая архитектура выбрана?" {
		t.Errorf("unexpected QA history: %+v", history)
	}
}
