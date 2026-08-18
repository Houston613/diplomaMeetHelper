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

type mockSearchRepo struct {
	searchFn func(ctx context.Context, userID string, query string) ([]domain.SearchResult, error)
}

func (m *mockSearchRepo) Search(ctx context.Context, userID string, query string) ([]domain.SearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, userID, query)
	}
	return nil, nil
}

func TestSearchUsecase_Success(t *testing.T) {
	ctx := context.Background()
	expectedResults := []domain.SearchResult{
		{
			MeetingID:   uuid.New(),
			Filename:    "standup.mp3",
			MatchSource: "transcript",
			Snippet:     "обсудили <b>миграции</b>",
			CreatedAt:   time.Now(),
			Rank:        0.8,
		},
	}

	repo := &mockSearchRepo{
		searchFn: func(ctx context.Context, userID string, query string) ([]domain.SearchResult, error) {
			return expectedResults, nil
		},
	}

	uc := usecase.NewSearchUsecase(repo, zap.NewNop())
	results, err := uc.Search(ctx, "user-1", "миграции")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 || results[0].Filename != "standup.mp3" {
		t.Errorf("unexpected results: %+v", results)
	}
}
