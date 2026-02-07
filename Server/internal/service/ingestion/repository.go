package ingestion

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	InitSchema(ctx context.Context) error
	CreateJob(ctx context.Context, job *Job) error
	GetPendingJob(ctx context.Context) (*Job, error)
	UpdateJobStatus(ctx context.Context, id uuid.UUID, status JobStatus, errorMessage *string) error
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) InitSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS ingestion_jobs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			journal_id UUID NOT NULL,
			user_id UUID NOT NULL,
			title TEXT,
			content TEXT NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			attempts INT DEFAULT 0,
			max_attempts INT DEFAULT 3,
			error_message TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON ingestion_jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_created ON ingestion_jobs(created_at)`,
		`UPDATE ingestion_jobs SET max_attempts = 3 WHERE max_attempts IS NULL OR max_attempts <= 0`,
		`UPDATE ingestion_jobs
		 SET status = 'pending', error_message = NULL
		 WHERE status = 'failed' AND attempts < max_attempts`,
	}

	for i, q := range queries {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("failed to execute query %d: %w", i, err)
		}
	}
	return nil
}

func (r *PostgresRepository) CreateJob(ctx context.Context, job *Job) error {
	if job.Attempts < 0 {
		job.Attempts = 0
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = DefaultMaxAttempts
	}

	query := `
		INSERT INTO ingestion_jobs (id, journal_id, user_id, title, content, status, attempts, max_attempts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		job.ID, job.JournalID, job.UserID, job.Title, job.Content,
		job.Status, job.Attempts, job.MaxAttempts, job.CreatedAt, job.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetPendingJob(ctx context.Context) (*Job, error) {
	// Simple polling: Get one pending job.
	// In a real system with multiple workers, we'd want 'FOR UPDATE SKIP LOCKED'
	// to prevent race conditions.
	query := `
		SELECT id, journal_id, user_id, title, content, status, attempts, max_attempts, created_at, updated_at
		FROM ingestion_jobs
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var job Job
	err := r.pool.QueryRow(ctx, query).Scan(
		&job.ID, &job.JournalID, &job.UserID, &job.Title, &job.Content,
		&job.Status, &job.Attempts, &job.MaxAttempts, &job.CreatedAt, &job.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // No jobs available
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending job: %w", err)
	}
	return &job, nil
}

func (r *PostgresRepository) UpdateJobStatus(ctx context.Context, id uuid.UUID, status JobStatus, errorMessage *string) error {
	query := `
		UPDATE ingestion_jobs
		SET status = $1::text, error_message = $2, updated_at = $3, 
		    completed_at = CASE WHEN $1::text = 'completed' THEN $3 ELSE completed_at END
		WHERE id = $4
	`
	_, err := r.pool.Exec(ctx, query, status, errorMessage, time.Now(), id)
	return err
}

// IncrementAttempts increments the attempts counter and potentially updates status
func (r *PostgresRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE ingestion_jobs
		SET attempts = attempts + 1, updated_at = $1
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, time.Now(), id)
	return err
}
