package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"diplomaMeetHelper/internal/domain"
	"diplomaMeetHelper/internal/usecase"
)

type mockUserRepo struct {
	ensureUserFn func(ctx context.Context, userID string) (*domain.User, bool, error)
	getUserFn    func(ctx context.Context, userID string) (*domain.User, error)
}

func (m *mockUserRepo) EnsureUser(ctx context.Context, userID string) (*domain.User, bool, error) {
	if m.ensureUserFn != nil {
		return m.ensureUserFn(ctx, userID)
	}
	return nil, false, errors.New("not implemented")
}

func (m *mockUserRepo) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	if m.getUserFn != nil {
		return m.getUserFn(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func TestUserUsecase_EnsureUser(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		mockRepo    *mockUserRepo
		wantCreated bool
		wantErr     error
	}{
		{
			name:   "empty user id",
			userID: "   ",
			mockRepo: &mockUserRepo{
				ensureUserFn: func(ctx context.Context, userID string) (*domain.User, bool, error) {
					t.Fatalf("repo should not be called for empty userID")
					return nil, false, nil
				},
			},
			wantCreated: false,
			wantErr:     domain.ErrInvalidUserID,
		},
		{
			name:   "successfully creates new user",
			userID: "user-123",
			mockRepo: &mockUserRepo{
				ensureUserFn: func(ctx context.Context, userID string) (*domain.User, bool, error) {
					return &domain.User{ID: userID, CreatedAt: time.Now()}, true, nil
				},
			},
			wantCreated: true,
			wantErr:     nil,
		},
		{
			name:   "existing user returned",
			userID: "user-existing",
			mockRepo: &mockUserRepo{
				ensureUserFn: func(ctx context.Context, userID string) (*domain.User, bool, error) {
					return &domain.User{ID: userID, CreatedAt: time.Now()}, false, nil
				},
			},
			wantCreated: false,
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewUserUsecase(tt.mockRepo)
			user, created, err := uc.EnsureUser(ctx, tt.userID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr == nil {
				if created != tt.wantCreated {
					t.Errorf("expected created = %v, got %v", tt.wantCreated, created)
				}
				if user == nil || user.ID != tt.userID {
					t.Errorf("expected user with ID %q, got %+v", tt.userID, user)
				}
			}
		})
	}
}

func TestUserUsecase_GetUser(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		userID   string
		mockRepo *mockUserRepo
		wantErr  error
	}{
		{
			name:     "empty user id",
			userID:   "",
			mockRepo: &mockUserRepo{},
			wantErr:  domain.ErrInvalidUserID,
		},
		{
			name:   "user found",
			userID: "user-456",
			mockRepo: &mockUserRepo{
				getUserFn: func(ctx context.Context, userID string) (*domain.User, error) {
					return &domain.User{ID: userID, CreatedAt: time.Now()}, nil
				},
			},
			wantErr: nil,
		},
		{
			name:   "user not found",
			userID: "user-unknown",
			mockRepo: &mockUserRepo{
				getUserFn: func(ctx context.Context, userID string) (*domain.User, error) {
					return nil, domain.ErrUserNotFound
				},
			},
			wantErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewUserUsecase(tt.mockRepo)
			user, err := uc.GetUser(ctx, tt.userID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr == nil {
				if user == nil || user.ID != tt.userID {
					t.Errorf("expected user with ID %q, got %+v", tt.userID, user)
				}
			}
		})
	}
}
