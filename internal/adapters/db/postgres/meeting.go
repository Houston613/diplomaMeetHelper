package postgres

import (
	"context"
	"errors"
	"fmt"

	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MeetingRepository struct {
	pool *pgxpool.Pool
}

func NewMeetingRepository(pool *pgxpool.Pool) *MeetingRepository {
	return &MeetingRepository{
		pool: pool,
	}
}

func (r *MeetingRepository) CreateMeetingWithJob(ctx context.Context, meeting *domain.Meeting, job *domain.ProcessingJob) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queryMeeting := `
INSERT INTO meethelper.meetings (id, user_id, filename, file_path, status, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7);
	`
	_, err = tx.Exec(ctx, queryMeeting,
		meeting.ID,
		meeting.UserID,
		meeting.Filename,
		meeting.FilePath,
		meeting.Status,
		meeting.CreatedAt,
		meeting.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert meeting: %w", err)
	}

	queryJob := `
INSERT INTO meethelper.processing_jobs (id, meeting_id, user_id, status, error_message, retry_count, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
	`
	_, err = tx.Exec(ctx, queryJob,
		job.ID,
		job.MeetingID,
		job.UserID,
		job.Status,
		job.ErrorMessage,
		job.RetryCount,
		job.CreatedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert processing job: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *MeetingRepository) GetMeeting(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.Meeting, error) {
	query := `
SELECT
    id,
    user_id,
    filename,
    file_path,
    status,
    created_at,
    updated_at
FROM
    meethelper.meetings
WHERE
    id = $1;
	`

	var m domain.Meeting
	err := r.pool.QueryRow(ctx, query, meetingID).Scan(
		&m.ID,
		&m.UserID,
		&m.Filename,
		&m.FilePath,
		&m.Status,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMeetingNotFound
		}
		return nil, fmt.Errorf("failed to get meeting: %w", err)
	}

	if m.UserID != userID {
		return nil, domain.ErrAccessDenied
	}

	return &m, nil
}

func (r *MeetingRepository) ListMeetings(ctx context.Context, userID string) ([]domain.MeetingListItem, error) {
	query := `
SELECT
    m.id,
    m.filename,
    m.status,
    COALESCE(s.content, '') AS summary,
    m.created_at,
    m.updated_at
FROM
    meethelper.meetings m
    LEFT JOIN meethelper.summaries s ON m.id = s.meeting_id
WHERE
    m.user_id = $1
ORDER BY
    m.created_at DESC;
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list meetings: %w", err)
	}
	defer rows.Close()

	var items []domain.MeetingListItem
	for rows.Next() {
		var item domain.MeetingListItem
		if err := rows.Scan(&item.ID, &item.Filename, &item.Status, &item.Summary, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan meeting item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *MeetingRepository) GetMeetingDetails(ctx context.Context, meetingID uuid.UUID, userID string) (*domain.MeetingDetails, error) {
	meeting, err := r.GetMeeting(ctx, meetingID, userID)
	if err != nil {
		return nil, err
	}

	details := &domain.MeetingDetails{
		Meeting: *meeting,
	}

	// Fetch transcript if available
	var tr domain.Transcript
	err = r.pool.QueryRow(ctx, `
SELECT
    id,
    meeting_id,
    content,
    created_at
FROM
    meethelper.transcriptions
WHERE
    meeting_id = $1;
	`, meetingID).Scan(&tr.ID, &tr.MeetingID, &tr.Content, &tr.CreatedAt)

	if err == nil {
		details.Transcript = &tr
	}

	// Fetch summary if available
	var sm domain.Summary
	err = r.pool.QueryRow(ctx, `
SELECT
    id,
    meeting_id,
    content,
    created_at
FROM
    meethelper.summaries
WHERE
    meeting_id = $1;
	`, meetingID).Scan(&sm.ID, &sm.MeetingID, &sm.Content, &sm.CreatedAt)

	if err == nil {
		details.Summary = &sm
	}

	return details, nil
}

func (r *MeetingRepository) DeleteMeeting(ctx context.Context, meetingID uuid.UUID, userID string) error {
	// 1. Verify existence & ownership
	_, err := r.GetMeeting(ctx, meetingID, userID)
	if err != nil {
		return err
	}

	// 2. Cascade delete
	query := `
		DELETE FROM meethelper.meetings
		WHERE id = $1 AND user_id = $2;
	`
	_, err = r.pool.Exec(ctx, query, meetingID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete meeting: %w", err)
	}

	return nil
}
