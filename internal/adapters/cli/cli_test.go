package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"diplomaMeetHelper/internal/adapters/cli"
	"diplomaMeetHelper/internal/domain"

	"go.uber.org/zap"
)

type mockUserService struct {
	ensureUserFn func(ctx context.Context, userID string) (*domain.User, bool, error)
	getUserFn    func(ctx context.Context, userID string) (*domain.User, error)
}

func (m *mockUserService) EnsureUser(ctx context.Context, userID string) (*domain.User, bool, error) {
	if m.ensureUserFn != nil {
		return m.ensureUserFn(ctx, userID)
	}
	return nil, false, nil
}

func (m *mockUserService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	if m.getUserFn != nil {
		return m.getUserFn(ctx, userID)
	}
	return nil, nil
}

func TestCLI_MissingUserID(t *testing.T) {
	mockService := &mockUserService{}
	app := cli.NewApp(mockService, zap.NewNop())

	var outBuf, errBuf bytes.Buffer
	app.SetOutput(&outBuf, &errBuf)

	err := app.Execute(context.Background(), []string{"start"})
	if err == nil {
		t.Fatalf("expected error when --user-id is missing, got nil")
	}

	if !strings.Contains(err.Error(), "--user-id обязателен") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCLI_StartNewUser(t *testing.T) {
	mockService := &mockUserService{
		ensureUserFn: func(ctx context.Context, userID string) (*domain.User, bool, error) {
			return &domain.User{
				ID:        userID,
				CreatedAt: time.Now(),
			}, true, nil
		},
	}

	app := cli.NewApp(mockService, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	app.SetOutput(&outBuf, &errBuf)

	err := app.Execute(context.Background(), []string{"start", "--user-id", "alex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Пользователь alex успешно зарегистрирован\n"
	if outBuf.String() != expected {
		t.Errorf("expected output %q, got %q", expected, outBuf.String())
	}
}

func TestCLI_StartExistingUser(t *testing.T) {
	mockService := &mockUserService{
		ensureUserFn: func(ctx context.Context, userID string) (*domain.User, bool, error) {
			return &domain.User{
				ID:        userID,
				CreatedAt: time.Now(),
			}, false, nil
		},
	}

	app := cli.NewApp(mockService, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	app.SetOutput(&outBuf, &errBuf)

	err := app.Execute(context.Background(), []string{"start", "-u", "bob"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Пользователь bob уже зарегистрирован\n"
	if outBuf.String() != expected {
		t.Errorf("expected output %q, got %q", expected, outBuf.String())
	}
}
