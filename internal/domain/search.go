package domain

import (
	"time"

	"github.com/google/uuid"
)

type SearchResult struct {
	MeetingID   uuid.UUID `json:"meeting_id"`
	Filename    string    `json:"filename"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	MatchSource string    `json:"match_source"`
	Snippet     string    `json:"snippet"`
	Rank        float64   `json:"rank"`
}
