package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{
		pool: pool,
	}
}

func (r *JobRepository) GetJobByMeetingID(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.JobStatusInfo, error) {
	query := `
		SELECT j.meeting_id, j.id, j.status, j.retry_count, j.error_message, j.created_at, j.updated_at, j.user_id
		FROM meethelper.processing_jobs j
		WHERE j.meeting_id = $1;
	`

	var info domain.JobStatusInfo
	var ownerID string
	err := r.pool.QueryRow(ctx, query, meetingID).Scan(
		&info.MeetingID,
		&info.JobID,
		&info.Status,
		&info.RetryCount,
		&info.ErrorMessage,
		&info.CreatedAt,
		&info.UpdatedAt,
		&ownerID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job status: %w", err)
	}

	if ownerID != userID {
		return nil, domain.ErrAccessDenied
	}

	return &info, nil
}

func (r *JobRepository) CompleteJobWithResults(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, transcript string, summary string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()

	// Insert Transcript
	_, err = tx.Exec(ctx, `
		INSERT INTO meethelper.transcriptions (id, meeting_id, content, created_at)
		VALUES ($1, $2, $3, $4);
	`, uuid.New(), meetingID, transcript, now)
	if err != nil {
		return fmt.Errorf("failed to save transcription: %w", err)
	}

	// Insert Summary
	_, err = tx.Exec(ctx, `
		INSERT INTO meethelper.summaries (id, meeting_id, content, created_at)
		VALUES ($1, $2, $3, $4);
	`, uuid.New(), meetingID, summary, now)
	if err != nil {
		return fmt.Errorf("failed to save summary: %w", err)
	}

	// Update Job Status
	_, err = tx.Exec(ctx, `
		UPDATE meethelper.processing_jobs
		SET status = $1, updated_at = $2, error_message = ''
		WHERE id = $3;
	`, domain.StatusCompleted, now, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Update Meeting Status
	_, err = tx.Exec(ctx, `
		UPDATE meethelper.meetings
		SET status = $1, updated_at = $2
		WHERE id = $3;
	`, domain.StatusCompleted, now, meetingID)
	if err != nil {
		return fmt.Errorf("failed to update meeting status: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *JobRepository) FailJob(ctx context.Context, jobID uuid.UUID, meetingID uuid.UUID, errorMessage string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()

	_, err = tx.Exec(ctx, `
		UPDATE meethelper.processing_jobs
		SET status = $1, error_message = $2, updated_at = $3
		WHERE id = $4;
	`, domain.StatusFailed, errorMessage, now, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job failure: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE meethelper.meetings
		SET status = $1, updated_at = $2
		WHERE id = $3;
	`, domain.StatusFailed, now, meetingID)
	if err != nil {
		return fmt.Errorf("failed to update meeting failure: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *JobRepository) RetryJob(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.ProcessingJob, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `
		SELECT id, meeting_id, user_id, status, error_message, retry_count, created_at, updated_at
		FROM meethelper.processing_jobs
		WHERE meeting_id = $1;
	`
	var job domain.ProcessingJob
	err = tx.QueryRow(ctx, query, meetingID).Scan(
		&job.ID,
		&job.MeetingID,
		&job.UserID,
		&job.Status,
		&job.ErrorMessage,
		&job.RetryCount,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if job.UserID != userID {
		return nil, domain.ErrAccessDenied
	}

	if job.Status != domain.StatusFailed {
		return nil, domain.ErrJobNotRetryable
	}

	now := time.Now()
	job.Status = domain.StatusCreated
	job.ErrorMessage = ""
	job.RetryCount++
	job.UpdatedAt = now

	_, err = tx.Exec(ctx, `
		UPDATE meethelper.processing_jobs
		SET status = $1, error_message = '', retry_count = $2, updated_at = $3
		WHERE id = $4;
	`, domain.StatusCreated, job.RetryCount, now, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update job for retry: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE meethelper.meetings
		SET status = $1, updated_at = $2
		WHERE id = $3;
	`, domain.StatusCreated, now, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to update meeting for retry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit retry transaction: %w", err)
	}

	return &job, nil
}

func (r *JobRepository) GetPendingJobs(ctx context.Context) ([]domain.ProcessingJob, error) {
	query := `
		SELECT id, meeting_id, user_id, status, error_message, retry_count, created_at, updated_at
		FROM meethelper.processing_jobs
		WHERE status IN ($1, $2)
		ORDER BY created_at ASC;
	`

	rows, err := r.pool.Query(ctx, query, domain.StatusCreated, domain.StatusProcessing)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []domain.ProcessingJob
	for rows.Next() {
		var j domain.ProcessingJob
		if err := rows.Scan(
			&j.ID,
			&j.MeetingID,
			&j.UserID,
			&j.Status,
			&j.ErrorMessage,
			&j.RetryCount,
			&j.CreatedAt,
			&j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pending job: %w", err)
		}
		jobs = append(jobs, j)
	}

	return jobs, nil
}
