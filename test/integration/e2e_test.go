package integration_test

import (
	"context"
	"strings"
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

func TestFullE2E_UserJourney(t *testing.T) {
	pool, ctx := setupTestDB(t)
	defer pool.Close()

	logger := zap.NewNop()

	// Repositories
	userRepo := postgres.NewUserRepository(pool)
	meetingRepo := postgres.NewMeetingRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)
	searchRepo := postgres.NewSearchRepository(pool)
	qaRepo := postgres.NewQARepository(pool)

	// External Mock Clients
	speechClient := mockSpeech.NewSpeechClient(10*time.Millisecond, 0)
	llmClient := mockLLM.NewLLMClient(10*time.Millisecond, 0)

	// Worker Pool
	jobHandler := func(jobCtx context.Context, job domain.ProcessingJob) error {
		return usecase.ProcessJobHandler(jobCtx, job, meetingRepo, jobRepo, speechClient, llmClient, logger)
	}
	workerPool := workerpool.New(2, 10, 5*time.Second, jobHandler, logger)
	workerPool.Start()
	defer workerPool.Stop(5 * time.Second)

	// Usecases
	userUC := usecase.NewUserUsecase(userRepo)
	meetingUC := usecase.NewMeetingUsecase(meetingRepo, jobRepo, workerPool, logger)
	searchUC := usecase.NewSearchUsecase(searchRepo, logger)
	chatUC := usecase.NewChatUsecase(meetingRepo, qaRepo, llmClient, logger)

	userID := "e2e-user-" + uuid.NewString()

	// 1. User Registration (start)
	user, created, err := userUC.EnsureUser(ctx, userID)
	if err != nil {
		t.Fatalf("EnsureUser failed: %v", err)
	}
	if !created || user.ID != userID {
		t.Fatalf("expected new user %s to be created", userID)
	}

	// 2. Load Meeting (load)
	fixturePath := "../../test/fixtures/meeting1_architecture.txt"
	meetingID, err := meetingUC.LoadMeeting(ctx, userID, fixturePath)
	if err != nil {
		t.Fatalf("LoadMeeting failed: %v", err)
	}

	// 3. Wait for WorkerPool processing
	var jobStatus *domain.JobStatusInfo
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		jobStatus, err = meetingUC.GetMeetingStatus(ctx, userID, meetingID)
		if err == nil && jobStatus.Status == domain.StatusCompleted {
			break
		}
	}
	if jobStatus == nil || jobStatus.Status != domain.StatusCompleted {
		t.Fatalf("expected job to be completed, got status: %v", jobStatus)
	}

	// 4. List Meetings (list)
	list, err := meetingUC.ListMeetings(ctx, userID)
	if err != nil {
		t.Fatalf("ListMeetings failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != meetingID {
		t.Fatalf("expected 1 meeting in list, got: %d", len(list))
	}
	if !strings.Contains(list[0].Summary, "Краткое содержание") {
		t.Errorf("expected list item to contain summary, got: %q", list[0].Summary)
	}

	// 5. Get Meeting Details (get)
	details, err := meetingUC.GetMeetingDetails(ctx, userID, meetingID)
	if err != nil {
		t.Fatalf("GetMeetingDetails failed: %v", err)
	}
	if details.Transcript == nil || !strings.Contains(details.Transcript.Content, "чистой архитектуры") {
		t.Errorf("expected transcript to contain 'чистой архитектуры', got: %+v", details.Transcript)
	}
	if details.Summary == nil || !strings.Contains(details.Summary.Content, "Краткое содержание") {
		t.Errorf("expected summary to contain 'Краткое содержание', got: %+v", details.Summary)
	}

	// 6. Full-Text Search (find)
	searchResults, err := searchUC.Search(ctx, userID, "архитектура")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(searchResults) == 0 {
		t.Errorf("expected search results for 'архитектура'")
	}

	// 7. Chat with LLM (chat)
	answer, err := chatUC.Ask(ctx, userID, meetingID, "какие решения приняты?")
	if err != nil {
		t.Fatalf("Chat Ask failed: %v", err)
	}
	if !strings.Contains(answer, "Ответ на вопрос") {
		t.Errorf("unexpected chat answer: %s", answer)
	}

	// 8. Cascade Delete (delete)
	err = meetingUC.DeleteMeeting(ctx, userID, meetingID)
	if err != nil {
		t.Fatalf("DeleteMeeting failed: %v", err)
	}

	listAfterDelete, err := meetingUC.ListMeetings(ctx, userID)
	if err != nil {
		t.Fatalf("ListMeetings after delete failed: %v", err)
	}
	if len(listAfterDelete) != 0 {
		t.Errorf("expected 0 meetings after delete, got: %d", len(listAfterDelete))
	}
}
