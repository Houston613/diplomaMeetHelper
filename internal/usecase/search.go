package usecase

import (
	"context"

	"diplomaMeetHelper/internal/domain"

	"go.uber.org/zap"
)

type SearchRepository interface {
	Search(ctx context.Context, userID string, query string) ([]domain.SearchResult, error)
}

type SearchUsecase struct {
	searchRepo SearchRepository
	logger     *zap.Logger
}

func NewSearchUsecase(searchRepo SearchRepository, logger *zap.Logger) *SearchUsecase {
	return &SearchUsecase{
		searchRepo: searchRepo,
		logger:     logger,
	}
}

func (u *SearchUsecase) Search(ctx context.Context, userID string, query string) ([]domain.SearchResult, error) {
	return u.searchRepo.Search(ctx, userID, query)
}
