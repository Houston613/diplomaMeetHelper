package usecase_test

import (
	"context"
	"testing"
	"time"

	"diplomaMeetHelper/internal/domain"
	"diplomaMeetHelper/internal/usecase"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockQARepo struct {
	saveQAFn        func(ctx context.Context, item *domain.QAItem) error
	getQAHistoryFn func(ctx context.Context, meetingID uuid.UUID, userID string) ([]domain.QAItem, error)
}

func (m *mockQARepo) SaveQA(ctx context.Context, item *domain.QAItem) error {
	if m.saveQAFn != nil {
		return m.saveQAFn(ctx, item)
	}
	return nil
}

func (m *mockQARepo) GetQAHistory(ctx context.Context, meetingID uuid.UUID, userID string) ([]domain.QAItem, error) {
	if m.getQAHistoryFn != nil {
		return m.getQAHistoryFn(ctx, meetingID, userID)
	}
	return nil, nil
}

type mockChatLLM struct {
	askQuestionFn func(ctx context.Context, contextText string, question string) (string, error)
}

func (m *mockChatLLM) AskQuestion(ctx context.Context, contextText string, question string) (string, error) {
	if m.askQuestionFn != nil {
		return m.askQuestionFn(ctx, contextText, question)
	}
	return "mock answer", nil
}

func TestChatUsecase_Ask(t *testing.T) {
	ctx := context.Background()
	meetingID := uuid.New()

	mRepo := &mockMeetingRepo{
		getMeetingDetailsFn: func(ctx context.Context, mID uuid.UUID, uID string) (*domain.MeetingDetails, error) {
			return &domain.MeetingDetails{
				Meeting: domain.Meeting{
					ID:        mID,
					UserID:    uID,
					Status:    domain.StatusCompleted,
					CreatedAt: time.Now(),
				},
				Transcript: &domain.Transcript{Content: "transcript text"},
				Summary:    &domain.Summary{Content: "summary text"},
			}, nil
		},
	}

	qaRepo := &mockQARepo{}
	llm := &mockChatLLM{
		askQuestionFn: func(ctx context.Context, contextText string, question string) (string, error) {
			return "Ответ: согласовано 10 сторипоинтов", nil
		},
	}

	uc := usecase.NewChatUsecase(mRepo, qaRepo, llm, zap.NewNop())
	answer, err := uc.Ask(ctx, "user-1", meetingID, "сколько сторипоинтов?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if answer != "Ответ: согласовано 10 сторипоинтов" {
		t.Errorf("unexpected answer: %s", answer)
	}
}
