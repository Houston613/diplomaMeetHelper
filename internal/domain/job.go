package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusCreated     = "created"
	StatusProcessing  = "processing"
	StatusTranscribed = "transcribed"
	StatusSummarized  = "summarized"
	StatusCompleted   = "completed"
	StatusFailed      = "failed"
)

type ProcessingJob struct {
	ID           uuid.UUID `json:"id"`
	MeetingID    uuid.UUID `json:"meeting_id"`
	UserID       string    `json:"user_id"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message"`
	RetryCount   int       `json:"retry_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type JobStatusInfo struct {
	MeetingID    uuid.UUID `json:"meeting_id"`
	JobID        uuid.UUID `json:"job_id"`
	Status       string    `json:"status"`
	RetryCount   int       `json:"retry_count"`
	ErrorMessage string    `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
