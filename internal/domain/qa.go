package domain

import (
	"time"

	"github.com/google/uuid"
)

type QAItem struct {
	ID        uuid.UUID `json:"id"`
	MeetingID uuid.UUID `json:"meeting_id"`
	UserID    string    `json:"user_id"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	CreatedAt time.Time `json:"created_at"`
}
