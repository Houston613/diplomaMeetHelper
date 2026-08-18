package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MeetingRepository interface {
	CreateMeetingWithJob(ctx context.Context, meeting *domain.Meeting, job *domain.ProcessingJob) error
	GetMeeting(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.Meeting, error)
	ListMeetings(ctx context.Context, userID string) ([]domain.MeetingListItem, error)
	GetMeetingDetails(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.MeetingDetails, error)
}

type JobRepository interface {
	GetJobByMeetingID(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.JobStatusInfo, error)
	CompleteJobWithResults(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, transcript string, summary string) error
	FailJob(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, errorMessage string) error
}

type SpeechClient interface {
	Transcribe(ctx context.Context, filePath string) (string, error)
}

type LLMClient interface {
	Summarize(ctx context.Context, transcript string) (string, error)
}

type WorkerSubmitter interface {
	Submit(job domain.ProcessingJob) error
}

type MeetingUsecase struct {
	meetingRepo MeetingRepository
	jobRepo     JobRepository
	submitter   WorkerSubmitter
	logger      *zap.Logger
}

func NewMeetingUsecase(
	meetingRepo MeetingRepository,
	jobRepo JobRepository,
	submitter WorkerSubmitter,
	logger *zap.Logger,
) *MeetingUsecase {
	return &MeetingUsecase{
		meetingRepo: meetingRepo,
		jobRepo:     jobRepo,
		submitter:   submitter,
		logger:      logger,
	}
}

func (u *MeetingUsecase) LoadMeeting(ctx context.Context, userID string, filePath string) (uuid.UUID, error) {
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return uuid.Nil, domain.ErrFileNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to access file %s: %w", filePath, err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3", ".wav", ".txt", ".json":
	default:
		return uuid.Nil, domain.ErrUnsupportedFormat
	}

	filename := filepath.Base(filePath)
	now := time.Now()
	meetingID := uuid.New()
	jobID := uuid.New()

	meeting := &domain.Meeting{
		ID:        meetingID,
		UserID:    userID,
		Filename:  filename,
		FilePath:  filePath,
		Status:    domain.StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}

	job := &domain.ProcessingJob{
		ID:           jobID,
		MeetingID:    meetingID,
		UserID:       userID,
		Status:       domain.StatusCreated,
		ErrorMessage: "",
		RetryCount:   0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := u.meetingRepo.CreateMeetingWithJob(ctx, meeting, job); err != nil {
		return uuid.Nil, fmt.Errorf("failed to save meeting and job: %w", err)
	}

	if err := u.submitter.Submit(*job); err != nil {
		u.logger.Warn("Failed to submit job to worker queue",
			zap.String("job_id", jobID.String()),
			zap.Error(err),
		)
	}

	return meetingID, nil
}

func (u *MeetingUsecase) GetMeetingStatus(ctx context.Context, userID string, meetingID uuid.UUID) (*domain.JobStatusInfo, error) {
	return u.jobRepo.GetJobByMeetingID(ctx, meetingID, userID)
}

func (u *MeetingUsecase) ListMeetings(ctx context.Context, userID string) ([]domain.MeetingListItem, error) {
	return u.meetingRepo.ListMeetings(ctx, userID)
}

func (u *MeetingUsecase) GetMeetingDetails(ctx context.Context, userID string, meetingID uuid.UUID) (*domain.MeetingDetails, error) {
	details, err := u.meetingRepo.GetMeetingDetails(ctx, meetingID, userID)
	if err != nil {
		return nil, err
	}

	if details.Meeting.Status != domain.StatusCompleted {
		return nil, domain.ErrJobNotCompleted
	}

	return details, nil
}

func ProcessJobHandler(
	ctx context.Context,
	job domain.ProcessingJob,
	meetingRepo MeetingRepository,
	jobRepo JobRepository,
	speech SpeechClient,
	llm LLMClient,
	logger *zap.Logger,
) error {
	logger.Info("Starting processing job",
		zap.String("job_id", job.ID.String()),
		zap.String("meeting_id", job.MeetingID.String()),
	)

	meeting, err := meetingRepo.GetMeeting(ctx, job.MeetingID, job.UserID)
	if err != nil {
		_ = jobRepo.FailJob(ctx, job.ID, job.MeetingID, err.Error())
		return fmt.Errorf("failed to fetch meeting for processing: %w", err)
	}

	// 1. Transcription step
	transcript, err := speech.Transcribe(ctx, meeting.FilePath)
	if err != nil {
		_ = jobRepo.FailJob(ctx, job.ID, job.MeetingID, "speech recognition failed: "+err.Error())
		return fmt.Errorf("transcription failed: %w", err)
	}

	// 2. Summarization step
	summary, err := llm.Summarize(ctx, transcript)
	if err != nil {
		_ = jobRepo.FailJob(ctx, job.ID, job.MeetingID, "summarization failed: "+err.Error())
		return fmt.Errorf("summarization failed: %w", err)
	}

	// 3. Complete job with results atomically
	if err := jobRepo.CompleteJobWithResults(ctx, job.ID, job.MeetingID, transcript, summary); err != nil {
		_ = jobRepo.FailJob(ctx, job.ID, job.MeetingID, "failed to persist results: "+err.Error())
		return fmt.Errorf("persisting results failed: %w", err)
	}

	logger.Info("Job processed successfully",
		zap.String("job_id", job.ID.String()),
		zap.String("meeting_id", job.MeetingID.String()),
	)
	return nil
}
