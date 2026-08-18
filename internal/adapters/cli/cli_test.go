package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"diplomaMeetHelper/internal/adapters/cli"
	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
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

type mockMeetingService struct {
	loadMeetingFn       func(ctx context.Context, userID string, filePath string) (uuid.UUID, error)
	getMeetingStatusFn  func(ctx context.Context, userID string, meetingID uuid.UUID) (*domain.JobStatusInfo, error)
	listMeetingsFn      func(ctx context.Context, userID string) ([]domain.MeetingListItem, error)
	getMeetingDetailsFn func(ctx context.Context, userID string, meetingID uuid.UUID) (*domain.MeetingDetails, error)
}

func (m *mockMeetingService) LoadMeeting(ctx context.Context, userID string, filePath string) (uuid.UUID, error) {
	if m.loadMeetingFn != nil {
		return m.loadMeetingFn(ctx, userID, filePath)
	}
	return uuid.New(), nil
}

func (m *mockMeetingService) GetMeetingStatus(ctx context.Context, userID string, meetingID uuid.UUID) (*domain.JobStatusInfo, error) {
	if m.getMeetingStatusFn != nil {
		return m.getMeetingStatusFn(ctx, userID, meetingID)
	}
	return nil, nil
}

func (m *mockMeetingService) ListMeetings(ctx context.Context, userID string) ([]domain.MeetingListItem, error) {
	if m.listMeetingsFn != nil {
		return m.listMeetingsFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockMeetingService) GetMeetingDetails(ctx context.Context, userID string, meetingID uuid.UUID) (*domain.MeetingDetails, error) {
	if m.getMeetingDetailsFn != nil {
		return m.getMeetingDetailsFn(ctx, userID, meetingID)
	}
	return nil, nil
}

func TestCLI_MissingUserID(t *testing.T) {
	mockUser := &mockUserService{}
	mockMeeting := &mockMeetingService{}
	handler := cli.NewHandler(mockUser, mockMeeting, zap.NewNop())

	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"start"})
	if err == nil {
		t.Fatalf("expected error when --user-id is missing, got nil")
	}

	if !strings.Contains(err.Error(), "--user-id обязателен") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCLI_StartNewUser(t *testing.T) {
	mockUser := &mockUserService{
		ensureUserFn: func(ctx context.Context, userID string) (*domain.User, bool, error) {
			return &domain.User{
				ID:        userID,
				CreatedAt: time.Now(),
			}, true, nil
		},
	}
	mockMeeting := &mockMeetingService{}

	handler := cli.NewHandler(mockUser, mockMeeting, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"start", "--user-id", "alex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Пользователь alex успешно зарегистрирован\n"
	if outBuf.String() != expected {
		t.Errorf("expected output %q, got %q", expected, outBuf.String())
	}
}

func TestCLI_LoadCommand(t *testing.T) {
	expectedID := uuid.New()
	mockMeeting := &mockMeetingService{
		loadMeetingFn: func(ctx context.Context, userID string, filePath string) (uuid.UUID, error) {
			return expectedID, nil
		},
	}
	handler := cli.NewHandler(&mockUserService{}, mockMeeting, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"load", "audio.mp3", "-u", "alex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(outBuf.String(), expectedID.String()) {
		t.Errorf("expected output to contain meeting ID %s, got %s", expectedID, outBuf.String())
	}
}

func TestCLI_ListCommand(t *testing.T) {
	mockMeeting := &mockMeetingService{
		listMeetingsFn: func(ctx context.Context, userID string) ([]domain.MeetingListItem, error) {
			return []domain.MeetingListItem{
				{
					ID:        uuid.New(),
					Filename:  "standup.mp3",
					Status:    "completed",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}, nil
		},
	}
	handler := cli.NewHandler(&mockUserService{}, mockMeeting, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"list", "-u", "alex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(outBuf.String(), "standup.mp3") {
		t.Errorf("expected output to contain filename, got %s", outBuf.String())
	}
}
