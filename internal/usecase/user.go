package usecase

import (
	"context"
	"strings"

	"diplomaMeetHelper/internal/domain"
)

type UserRepository interface {
	EnsureUser(ctx context.Context, userID string) (*domain.User, bool, error)
	GetUser(ctx context.Context, userID string) (*domain.User, error)
}

type UserUsecase struct {
	repo UserRepository
}

func NewUserUsecase(repo UserRepository) *UserUsecase {
	return &UserUsecase{
		repo: repo,
	}
}

func (u *UserUsecase) EnsureUser(ctx context.Context, userID string) (*domain.User, bool, error) {
	trimmedID := strings.TrimSpace(userID)
	if trimmedID == "" {
		return nil, false, domain.ErrInvalidUserID
	}

	return u.repo.EnsureUser(ctx, trimmedID)
}

func (u *UserUsecase) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	trimmedID := strings.TrimSpace(userID)
	if trimmedID == "" {
		return nil, domain.ErrInvalidUserID
	}

	return u.repo.GetUser(ctx, trimmedID)
}
