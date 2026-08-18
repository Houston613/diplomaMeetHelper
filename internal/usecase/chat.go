package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type QARepository interface {
	SaveQA(ctx context.Context, item *domain.QAItem) error
	GetQAHistory(ctx context.Context, meetingID uuid.UUID, userID string) ([]domain.QAItem, error)
}

type ChatLLMClient interface {
	AskQuestion(ctx context.Context, contextText string, question string) (string, error)
}

type ChatUsecase struct {
	meetingRepo MeetingRepository
	qaRepo      QARepository
	llmClient   ChatLLMClient
	logger      *zap.Logger
}

func NewChatUsecase(
	meetingRepo MeetingRepository,
	qaRepo QARepository,
	llmClient ChatLLMClient,
	logger *zap.Logger,
) *ChatUsecase {
	return &ChatUsecase{
		meetingRepo: meetingRepo,
		qaRepo:      qaRepo,
		llmClient:   llmClient,
		logger:      logger,
	}
}

func (u *ChatUsecase) Ask(ctx context.Context, userID string, meetingID uuid.UUID, question string) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", domain.ErrEmptyQuestion
	}

	details, err := u.meetingRepo.GetMeetingDetails(ctx, meetingID, userID)
	if err != nil {
		return "", err
	}

	if details.Meeting.Status != domain.StatusCompleted {
		return "", domain.ErrJobNotCompleted
	}

	contextText := ""
	if details.Summary != nil {
		contextText += "Выжимка: " + details.Summary.Content + "\n"
	}
	if details.Transcript != nil {
		contextText += "Транскрипция: " + details.Transcript.Content
	}

	answer, err := u.llmClient.AskQuestion(ctx, contextText, question)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer from LLM: %w", err)
	}

	qaItem := &domain.QAItem{
		ID:        uuid.New(),
		MeetingID: meetingID,
		UserID:    userID,
		Question:  question,
		Answer:    answer,
		CreatedAt: time.Now(),
	}

	if err := u.qaRepo.SaveQA(ctx, qaItem); err != nil {
		u.logger.Warn("Failed to save QA history",
			zap.String("meeting_id", meetingID.String()),
			zap.Error(err),
		)
	}

	return answer, nil
}
