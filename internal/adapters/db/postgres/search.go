package postgres

import (
	"context"
	"fmt"

	"diplomaMeetHelper/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SearchRepository struct {
	pool *pgxpool.Pool
}

func NewSearchRepository(pool *pgxpool.Pool) *SearchRepository {
	return &SearchRepository{
		pool: pool,
	}
}

func (r *SearchRepository) Search(ctx context.Context, userID string, query string) ([]domain.SearchResult, error) {
	sqlQuery := `
		SELECT
    m.id,
    m.filename,
    m.status,
    m.created_at,
    'transcript' AS match_source,
    ts_headline('russian', t.content, to_tsquery('russian', $2), 'StartSel=, StopSel=, MaxWords=25, MinWords=10') AS snippet,
    ts_rank(t.tsv, to_tsquery('russian', $2)) AS rank
FROM
    meethelper.transcriptions t
    JOIN meethelper.meetings m ON m.id = t.meeting_id
WHERE
    m.user_id = $1
    AND t.tsv @@ to_tsquery('russian', $2)
UNION ALL
SELECT
    m.id,
    m.filename,
    m.status,
    m.created_at,
    'summary' AS match_source,
    ts_headline('russian', s.content, to_tsquery('russian', $2), 'StartSel=, StopSel=, MaxWords=25, MinWords=10') AS snippet,
    ts_rank(s.tsv, to_tsquery('russian', $2)) AS rank
FROM
    meethelper.summaries s
    JOIN meethelper.meetings m ON m.id = s.meeting_id
WHERE
    s.tsv @@ to_tsquery('russian', $2)
ORDER BY
    rank DESC;
	`

	rows, err := r.pool.Query(ctx, sqlQuery, userID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute full-text search: %w", err)
	}
	defer rows.Close()

	var results []domain.SearchResult
	for rows.Next() {
		var res domain.SearchResult
		if err := rows.Scan(
			&res.MeetingID,
			&res.Filename,
			&res.Status,
			&res.CreatedAt,
			&res.MatchSource,
			&res.Snippet,
			&res.Rank,
		); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, res)
	}

	return results, nil
}
