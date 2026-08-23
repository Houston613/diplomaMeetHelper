package cli_test

import (
	"bytes"
	"context"
	"errors"
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
	retryMeetingFn      func(ctx context.Context, userID string, meetingID uuid.UUID) error
	deleteMeetingFn     func(ctx context.Context, userID string, meetingID uuid.UUID) error
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

func (m *mockMeetingService) RetryMeeting(ctx context.Context, userID string, meetingID uuid.UUID) error {
	if m.retryMeetingFn != nil {
		return m.retryMeetingFn(ctx, userID, meetingID)
	}
	return nil
}

func (m *mockMeetingService) DeleteMeeting(ctx context.Context, userID string, meetingID uuid.UUID) error {
	if m.deleteMeetingFn != nil {
		return m.deleteMeetingFn(ctx, userID, meetingID)
	}
	return nil
}

type mockSearchService struct {
	searchFn func(ctx context.Context, userID string, query string) ([]domain.SearchResult, error)
}

func (m *mockSearchService) Search(ctx context.Context, userID string, query string) ([]domain.SearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, userID, query)
	}
	return nil, nil
}

type mockChatService struct {
	askFn func(ctx context.Context, userID string, meetingID uuid.UUID, question string) (string, error)
}

func (m *mockChatService) Ask(ctx context.Context, userID string, meetingID uuid.UUID, question string) (string, error) {
	if m.askFn != nil {
		return m.askFn(ctx, userID, meetingID, question)
	}
	return "mock answer", nil
}

func TestCLI_MissingUserID(t *testing.T) {
	mockUser := &mockUserService{}
	mockMeeting := &mockMeetingService{}
	mockSearch := &mockSearchService{}
	mockChat := &mockChatService{}

	handler := cli.NewHandler(mockUser, mockMeeting, mockSearch, mockChat, zap.NewNop())

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
	mockSearch := &mockSearchService{}
	mockChat := &mockChatService{}

	handler := cli.NewHandler(mockUser, mockMeeting, mockSearch, mockChat, zap.NewNop())
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

func TestCLI_FindCommand(t *testing.T) {
	mockSearch := &mockSearchService{
		searchFn: func(ctx context.Context, userID string, query string) ([]domain.SearchResult, error) {
			return []domain.SearchResult{
				{
					MeetingID:   uuid.New(),
					Filename:    "standup.mp3",
					MatchSource: "transcript",
					CreatedAt:   time.Now(),
					Snippet:     "обсудили <b>архитектуру</b> диплома",
				},
			}, nil
		},
	}

	handler := cli.NewHandler(&mockUserService{}, &mockMeetingService{}, mockSearch, &mockChatService{}, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"find", "архитектура", "-u", "alex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(outBuf.String(), "архитектуру") {
		t.Errorf("expected output to contain snippet, got: %s", outBuf.String())
	}
}

func TestCLI_ChatCommand(t *testing.T) {
	meetingID := uuid.New()
	mockChat := &mockChatService{
		askFn: func(ctx context.Context, userID string, mID uuid.UUID, question string) (string, error) {
			return "Принято решение завершить срез 3.", nil
		},
	}

	handler := cli.NewHandler(&mockUserService{}, &mockMeetingService{}, &mockSearchService{}, mockChat, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"chat", meetingID.String(), "какое", "решение?", "-u", "alex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(outBuf.String(), "Принято решение") {
		t.Errorf("expected output to contain answer, got: %s", outBuf.String())
	}
}

func TestCLI_RetryCommand(t *testing.T) {
	meetingID := uuid.New()
	var retriedID uuid.UUID
	mockMeeting := &mockMeetingService{
		retryMeetingFn: func(ctx context.Context, userID string, mID uuid.UUID) error {
			retriedID = mID
			return nil
		},
	}

	handler := cli.NewHandler(&mockUserService{}, mockMeeting, &mockSearchService{}, &mockChatService{}, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"retry", meetingID.String(), "-u", "alex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retriedID != meetingID {
		t.Errorf("expected retried meeting ID %s, got %s", meetingID, retriedID)
	}
	if !strings.Contains(outBuf.String(), "Повторная обработка встречи") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}
}

func TestCLI_DeleteCommand(t *testing.T) {
	meetingID := uuid.New()
	var deletedID uuid.UUID
	mockMeeting := &mockMeetingService{
		deleteMeetingFn: func(ctx context.Context, userID string, mID uuid.UUID) error {
			deletedID = mID
			return nil
		},
	}

	handler := cli.NewHandler(&mockUserService{}, mockMeeting, &mockSearchService{}, &mockChatService{}, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"delete", meetingID.String(), "-u", "alex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deletedID != meetingID {
		t.Errorf("expected deleted meeting ID %s, got %s", meetingID, deletedID)
	}
	if !strings.Contains(outBuf.String(), "успешно удалены") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}
}

func TestCLI_ListCommand(t *testing.T) {
	meetingID := uuid.New()
	mockMeeting := &mockMeetingService{
		listMeetingsFn: func(ctx context.Context, userID string) ([]domain.MeetingListItem, error) {
			return []domain.MeetingListItem{
				{
					ID:        meetingID,
					Filename:  "weekly_sync.mp3",
					Status:    domain.StatusCompleted,
					Summary:   "Согласованы сроки релиза и архитектура сервиса.",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}, nil
		},
	}

	handler := cli.NewHandler(&mockUserService{}, mockMeeting, &mockSearchService{}, &mockChatService{}, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"list", "-u", "alex"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := outBuf.String()
	if !strings.Contains(output, "Выжимка") {
		t.Errorf("expected header to contain 'Выжимка', got: %s", output)
	}
	if !strings.Contains(output, "weekly_sync.mp3") {
		t.Errorf("expected output to contain filename, got: %s", output)
	}
	if !strings.Contains(output, "Согласованы") {
		t.Errorf("expected output to contain summary snippet, got: %s", output)
	}
}

func TestCLI_EnsureUser_Error(t *testing.T) {
	mockUser := &mockUserService{
		ensureUserFn: func(ctx context.Context, userID string) (*domain.User, bool, error) {
			return nil, false, errors.New("db error")
		},
	}

	handler := cli.NewHandler(mockUser, &mockMeetingService{}, &mockSearchService{}, &mockChatService{}, zap.NewNop())
	var outBuf, errBuf bytes.Buffer
	handler.SetOutput(&outBuf, &errBuf)

	err := handler.Execute(context.Background(), []string{"list", "-u", "alex"})
	if err == nil {
		t.Fatal("expected error from EnsureUser middleware, got nil")
	}

	if !strings.Contains(err.Error(), "ошибка проверки/регистрации пользователя") {
		t.Errorf("unexpected error message: %v", err)
	}
}
