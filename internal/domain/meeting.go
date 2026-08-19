package domain

import (
	"time"

	"github.com/google/uuid"
)

type Meeting struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	Filename  string    `json:"filename"`
	FilePath  string    `json:"file_path"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Transcript struct {
	ID        uuid.UUID `json:"id"`
	MeetingID uuid.UUID `json:"meeting_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Summary struct {
	ID        uuid.UUID `json:"id"`
	MeetingID uuid.UUID `json:"meeting_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type MeetingListItem struct {
	ID        uuid.UUID `json:"id"`
	Filename  string    `json:"filename"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MeetingDetails struct {
	Meeting    Meeting     `json:"meeting"`
	Transcript *Transcript `json:"transcript,omitempty"`
	Summary    *Summary    `json:"summary,omitempty"`
}
