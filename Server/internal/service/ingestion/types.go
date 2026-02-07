package ingestion

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	DefaultMaxAttempts = 3

	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

type Job struct {
	ID           uuid.UUID  `json:"id"`
	JournalID    uuid.UUID  `json:"journal_id"`
	UserID       uuid.UUID  `json:"user_id"`
	Title        *string    `json:"title,omitempty"`
	Content      string     `json:"content"`
	Status       JobStatus  `json:"status"`
	Attempts     int        `json:"attempts"`
	MaxAttempts  int        `json:"max_attempts"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}
