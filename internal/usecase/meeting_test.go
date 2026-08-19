package usecase_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"diplomaMeetHelper/internal/domain"
	"diplomaMeetHelper/internal/usecase"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockMeetingRepo struct {
	createMeetingWithJobFn func(ctx context.Context, meeting *domain.Meeting, job *domain.ProcessingJob) error
	getMeetingFn           func(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.Meeting, error)
	listMeetingsFn         func(ctx context.Context, userID string) ([]domain.MeetingListItem, error)
	getMeetingDetailsFn    func(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.MeetingDetails, error)
	deleteMeetingFn        func(ctx context.Context, meetingID uuid.UUID, userID string) error
}

func (m *mockMeetingRepo) CreateMeetingWithJob(ctx context.Context, meeting *domain.Meeting, job *domain.ProcessingJob) error {
	if m.createMeetingWithJobFn != nil {
		return m.createMeetingWithJobFn(ctx, meeting, job)
	}
	return nil
}

func (m *mockMeetingRepo) GetMeeting(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.Meeting, error) {
	if m.getMeetingFn != nil {
		return m.getMeetingFn(ctx, meetingID, userID)
	}
	return nil, nil
}

func (m *mockMeetingRepo) ListMeetings(ctx context.Context, userID string) ([]domain.MeetingListItem, error) {
	if m.listMeetingsFn != nil {
		return m.listMeetingsFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockMeetingRepo) GetMeetingDetails(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.MeetingDetails, error) {
	if m.getMeetingDetailsFn != nil {
		return m.getMeetingDetailsFn(ctx, meetingID, userID)
	}
	return nil, nil
}

func (m *mockMeetingRepo) DeleteMeeting(ctx context.Context, meetingID uuid.UUID, userID string) error {
	if m.deleteMeetingFn != nil {
		return m.deleteMeetingFn(ctx, meetingID, userID)
	}
	return nil
}

type mockJobRepo struct {
	getJobByMeetingIDFn      func(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.JobStatusInfo, error)
	updateJobStatusFn        func(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, status string) error
	completeJobWithResultsFn func(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, transcript string, summary string) error
	failJobFn                func(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, errorMessage string) error
	retryJobFn               func(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.ProcessingJob, error)
}

func (m *mockJobRepo) GetJobByMeetingID(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.JobStatusInfo, error) {
	if m.getJobByMeetingIDFn != nil {
		return m.getJobByMeetingIDFn(ctx, meetingID, userID)
	}
	return nil, nil
}

func (m *mockJobRepo) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, status string) error {
	if m.updateJobStatusFn != nil {
		return m.updateJobStatusFn(ctx, jobID, meetingID, status)
	}
	return nil
}

func (m *mockJobRepo) CompleteJobWithResults(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, transcript string, summary string) error {
	if m.completeJobWithResultsFn != nil {
		return m.completeJobWithResultsFn(ctx, jobID, meetingID, transcript, summary)
	}
	return nil
}

func (m *mockJobRepo) FailJob(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, errorMessage string) error {
	if m.failJobFn != nil {
		return m.failJobFn(ctx, jobID, meetingID, errorMessage)
	}
	return nil
}

func (m *mockJobRepo) RetryJob(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.ProcessingJob, error) {
	if m.retryJobFn != nil {
		return m.retryJobFn(ctx, meetingID, userID)
	}
	return nil, nil
}

type mockSubmitter struct {
	submitFn func(job domain.ProcessingJob) error
}

func (m *mockSubmitter) Submit(job domain.ProcessingJob) error {
	if m.submitFn != nil {
		return m.submitFn(job)
	}
	return nil
}

type mockSpeech struct {
	transcribeFn func(ctx context.Context, filePath string) (string, error)
}

func (m *mockSpeech) Transcribe(ctx context.Context, filePath string) (string, error) {
	if m.transcribeFn != nil {
		return m.transcribeFn(ctx, filePath)
	}
	return "mock transcript", nil
}

type mockLLM struct {
	summarizeFn func(ctx context.Context, transcript string) (string, error)
}

func (m *mockLLM) Summarize(ctx context.Context, transcript string) (string, error) {
	if m.summarizeFn != nil {
		return m.summarizeFn(ctx, transcript)
	}
	return "mock summary", nil
}

func TestMeetingUsecase_LoadMeeting(t *testing.T) {
	ctx := context.Background()

	// 1. File not found
	uc := usecase.NewMeetingUsecase(&mockMeetingRepo{}, &mockJobRepo{}, &mockSubmitter{}, zap.NewNop())
	_, err := uc.LoadMeeting(ctx, "user-1", "non_existing_audio.mp3")
	if err == nil {
		t.Fatalf("expected error for non-existing file, got nil")
	}

	// 2. Unsupported format (.pdf)
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "document.pdf")
	_ = os.WriteFile(invalidFile, []byte("fake pdf"), 0644)
	_, err = uc.LoadMeeting(ctx, "user-1", invalidFile)
	if !errors.Is(err, domain.ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got: %v", err)
	}

	// 3. Valid file (.txt)
	tmpFile := filepath.Join(tmpDir, "meeting.txt")
	_ = os.WriteFile(tmpFile, []byte("test meeting notes"), 0644)

	var savedMeeting *domain.Meeting
	var submittedJob *domain.ProcessingJob

	mRepo := &mockMeetingRepo{
		createMeetingWithJobFn: func(ctx context.Context, meeting *domain.Meeting, job *domain.ProcessingJob) error {
			savedMeeting = meeting
			return nil
		},
	}
	sub := &mockSubmitter{
		submitFn: func(job domain.ProcessingJob) error {
			submittedJob = &job
			return nil
		},
	}

	uc = usecase.NewMeetingUsecase(mRepo, &mockJobRepo{}, sub, zap.NewNop())
	meetingID, err := uc.LoadMeeting(ctx, "user-1", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meetingID == uuid.Nil {
		t.Errorf("expected non-nil meeting ID")
	}
	if savedMeeting == nil || savedMeeting.ID != meetingID {
		t.Errorf("saved meeting ID does not match returned ID")
	}
	if submittedJob == nil || submittedJob.MeetingID != meetingID {
		t.Errorf("submitted job meeting ID does not match")
	}
}

func TestMeetingUsecase_RetryMeeting(t *testing.T) {
	ctx := context.Background()
	meetingID := uuid.New()

	var retriedJob *domain.ProcessingJob
	jRepo := &mockJobRepo{
		retryJobFn: func(ctx context.Context, mID uuid.UUID, userID string) (*domain.ProcessingJob, error) {
			job := &domain.ProcessingJob{
				ID:         uuid.New(),
				MeetingID:  mID,
				UserID:     userID,
				Status:     domain.StatusCreated,
				RetryCount: 1,
			}
			return job, nil
		},
	}
	sub := &mockSubmitter{
		submitFn: func(job domain.ProcessingJob) error {
			retriedJob = &job
			return nil
		},
	}

	uc := usecase.NewMeetingUsecase(&mockMeetingRepo{}, jRepo, sub, zap.NewNop())
	err := uc.RetryMeeting(ctx, "user-1", meetingID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retriedJob == nil || retriedJob.MeetingID != meetingID || retriedJob.RetryCount != 1 {
		t.Errorf("expected job to be resubmitted with retry count 1, got %+v", retriedJob)
	}
}

func TestMeetingUsecase_DeleteMeeting(t *testing.T) {
	ctx := context.Background()
	meetingID := uuid.New()

	var deletedID uuid.UUID
	mRepo := &mockMeetingRepo{
		deleteMeetingFn: func(ctx context.Context, mID uuid.UUID, userID string) error {
			deletedID = mID
			return nil
		},
	}

	uc := usecase.NewMeetingUsecase(mRepo, &mockJobRepo{}, &mockSubmitter{}, zap.NewNop())
	err := uc.DeleteMeeting(ctx, "user-1", meetingID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deletedID != meetingID {
		t.Errorf("expected deleted meeting ID %s, got %s", meetingID, deletedID)
	}
}

func TestMeetingUsecase_GetMeetingDetails_StatusCheck(t *testing.T) {
	ctx := context.Background()
	meetingID := uuid.New()

	// 1. Job not completed
	mRepo := &mockMeetingRepo{
		getMeetingDetailsFn: func(ctx context.Context, mID uuid.UUID, uID string) (*domain.MeetingDetails, error) {
			return &domain.MeetingDetails{
				Meeting: domain.Meeting{
					ID:     mID,
					UserID: uID,
					Status: domain.StatusProcessing,
				},
			}, nil
		},
	}
	uc := usecase.NewMeetingUsecase(mRepo, &mockJobRepo{}, &mockSubmitter{}, zap.NewNop())
	_, err := uc.GetMeetingDetails(ctx, "user-1", meetingID)
	if !errors.Is(err, domain.ErrJobNotCompleted) {
		t.Fatalf("expected ErrJobNotCompleted, got: %v", err)
	}

	// 2. Job completed
	mRepoCompleted := &mockMeetingRepo{
		getMeetingDetailsFn: func(ctx context.Context, mID uuid.UUID, uID string) (*domain.MeetingDetails, error) {
			return &domain.MeetingDetails{
				Meeting: domain.Meeting{
					ID:     mID,
					UserID: uID,
					Status: domain.StatusCompleted,
				},
				Transcript: &domain.Transcript{Content: "transcript"},
				Summary:    &domain.Summary{Content: "summary"},
			}, nil
		},
	}
	ucCompleted := usecase.NewMeetingUsecase(mRepoCompleted, &mockJobRepo{}, &mockSubmitter{}, zap.NewNop())
	details, err := ucCompleted.GetMeetingDetails(ctx, "user-1", meetingID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.Transcript == nil || details.Transcript.Content != "transcript" {
		t.Errorf("expected transcript content")
	}
}

func TestProcessJobHandler_Success(t *testing.T) {
	ctx := context.Background()
	meetingID := uuid.New()
	jobID := uuid.New()

	mRepo := &mockMeetingRepo{
		getMeetingFn: func(ctx context.Context, mID uuid.UUID, uID string) (*domain.Meeting, error) {
			return &domain.Meeting{
				ID:       mID,
				UserID:   uID,
				FilePath: "meeting.txt",
			}, nil
		},
	}

	var statusTransitions []string
	var completedTranscript, completedSummary string
	jRepo := &mockJobRepo{
		updateJobStatusFn: func(ctx context.Context, jID uuid.UUID, mID uuid.UUID, status string) error {
			statusTransitions = append(statusTransitions, status)
			return nil
		},
		completeJobWithResultsFn: func(ctx context.Context, jID uuid.UUID, mID uuid.UUID, transcript string, summary string) error {
			completedTranscript = transcript
			completedSummary = summary
			return nil
		},
	}

	speech := &mockSpeech{
		transcribeFn: func(ctx context.Context, filePath string) (string, error) {
			return "full transcription text", nil
		},
	}

	llm := &mockLLM{
		summarizeFn: func(ctx context.Context, transcript string) (string, error) {
			return "summary result", nil
		},
	}

	job := domain.ProcessingJob{
		ID:        jobID,
		MeetingID: meetingID,
		UserID:    "user-1",
		CreatedAt: time.Now(),
	}

	err := usecase.ProcessJobHandler(ctx, job, mRepo, jRepo, speech, llm, zap.NewNop())
	if err != nil {
		t.Fatalf("ProcessJobHandler failed: %v", err)
	}

	// Verify all intermediate statuses were reported in exact order
	expectedTransitions := []string{
		domain.StatusProcessing,
		domain.StatusTranscribed,
		domain.StatusSummarized,
	}
	if len(statusTransitions) != len(expectedTransitions) {
		t.Fatalf("expected %d transitions, got %d: %v", len(expectedTransitions), len(statusTransitions), statusTransitions)
	}
	for i, expected := range expectedTransitions {
		if statusTransitions[i] != expected {
			t.Errorf("transition [%d]: expected %s, got %s", i, expected, statusTransitions[i])
		}
	}

	if completedTranscript != "full transcription text" {
		t.Errorf("unexpected transcript: %q", completedTranscript)
	}
	if completedSummary != "summary result" {
		t.Errorf("unexpected summary: %q", completedSummary)
	}
}

func TestProcessJobHandler_SpeechFailure(t *testing.T) {
	ctx := context.Background()
	meetingID := uuid.New()
	jobID := uuid.New()

	mRepo := &mockMeetingRepo{
		getMeetingFn: func(ctx context.Context, mID uuid.UUID, uID string) (*domain.Meeting, error) {
			return &domain.Meeting{
				ID:       mID,
				UserID:   uID,
				FilePath: "error_meeting.txt",
			}, nil
		},
	}

	var failedErrorMessage string
	jRepo := &mockJobRepo{
		updateJobStatusFn: func(ctx context.Context, jID uuid.UUID, mID uuid.UUID, status string) error {
			return nil
		},
		failJobFn: func(ctx context.Context, jID uuid.UUID, mID uuid.UUID, errorMessage string) error {
			failedErrorMessage = errorMessage
			return nil
		},
	}

	speech := &mockSpeech{
		transcribeFn: func(ctx context.Context, filePath string) (string, error) {
			return "", errors.New("simulated speech recognition failure")
		},
	}

	llm := &mockLLM{}

	job := domain.ProcessingJob{
		ID:        jobID,
		MeetingID: meetingID,
		UserID:    "user-1",
		CreatedAt: time.Now(),
	}

	err := usecase.ProcessJobHandler(ctx, job, mRepo, jRepo, speech, llm, zap.NewNop())
	if err == nil {
		t.Fatalf("expected error from failed transcription, got nil")
	}

	if failedErrorMessage == "" {
		t.Errorf("expected FailJob to be called with error message")
	}
}

func TestProcessJobHandler_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	meetingID := uuid.New()
	jobID := uuid.New()

	mRepo := &mockMeetingRepo{
		getMeetingFn: func(ctx context.Context, mID uuid.UUID, uID string) (*domain.Meeting, error) {
			return &domain.Meeting{
				ID:       mID,
				UserID:   uID,
				FilePath: "meeting.txt",
			}, nil
		},
	}

	var failedErrorMessage string
	jRepo := &mockJobRepo{
		updateJobStatusFn: func(ctx context.Context, jID uuid.UUID, mID uuid.UUID, status string) error {
			return nil
		},
		failJobFn: func(ctx context.Context, jID uuid.UUID, mID uuid.UUID, errorMessage string) error {
			failedErrorMessage = errorMessage
			return nil
		},
	}

	speech := &mockSpeech{
		transcribeFn: func(ctx context.Context, filePath string) (string, error) {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "transcription", nil
		},
	}

	llm := &mockLLM{}

	job := domain.ProcessingJob{
		ID:        jobID,
		MeetingID: meetingID,
		UserID:    "user-1",
		CreatedAt: time.Now(),
	}

	err := usecase.ProcessJobHandler(ctx, job, mRepo, jRepo, speech, llm, zap.NewNop())
	if err == nil {
		t.Fatalf("expected error due to cancelled context, got nil")
	}
	if failedErrorMessage == "" {
		t.Errorf("expected FailJob to be called with cancellation error")
	}
}
