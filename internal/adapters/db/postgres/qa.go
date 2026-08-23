package postgres

import (
	"context"
	"fmt"

	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QARepository struct {
	pool *pgxpool.Pool
}

func NewQARepository(pool *pgxpool.Pool) *QARepository {
	return &QARepository{
		pool: pool,
	}
}

func (r *QARepository) SaveQA(ctx context.Context, item *domain.QAItem) error {
	query := `
		INSERT INTO meethelper.qa_history (id, meeting_id, user_id, question, answer, created_at)
		VALUES ($1, $2, $3, $4, $5, $6);
	`

	_, err := r.pool.Exec(ctx, query,
		item.ID,
		item.MeetingID,
		item.UserID,
		item.Question,
		item.Answer,
		item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save QA item: %w", err)
	}

	return nil
}

func (r *QARepository) GetQAHistory(ctx context.Context, meetingID uuid.UUID, userID string) ([]domain.QAItem, error) {
	query := `
		SELECT id, meeting_id, user_id, question, answer, created_at
		FROM meethelper.qa_history
		WHERE meeting_id = $1 AND user_id = $2
		ORDER BY created_at ASC;
	`

	rows, err := r.pool.Query(ctx, query, meetingID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get QA history: %w", err)
	}
	defer rows.Close()

	var items []domain.QAItem
	for rows.Next() {
		var it domain.QAItem
		if err := rows.Scan(
			&it.ID,
			&it.MeetingID,
			&it.UserID,
			&it.Question,
			&it.Answer,
			&it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan QA item: %w", err)
		}
		items = append(items, it)
	}

	return items, nil
}
