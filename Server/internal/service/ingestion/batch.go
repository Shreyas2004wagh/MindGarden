package ingestion

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// IngestionJournal represents the input for batch ingestion
type IngestionJournal struct {
	JournalID string
	UserID    string
	Title     *string
	Content   string
	CreatedAt time.Time
}

// BatchIngestResult summarizes a batch ingestion run.
type BatchIngestResult struct {
	Total   int `json:"total"`
	Queued  int `json:"queued"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

// BatchIngest ingests a list of journals in batches
func (w *Worker) BatchIngest(ctx context.Context, journals []IngestionJournal) (*BatchIngestResult, error) {
	if w == nil || w.repo == nil {
		return nil, fmt.Errorf("ingestion worker not initialized")
	}

	result := &BatchIngestResult{
		Total: len(journals),
	}

	// Process in batches of 10
	batchSize := 10
	for i := 0; i < len(journals); i += batchSize {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		end := i + batchSize
		if end > len(journals) {
			end = len(journals)
		}
		batch := journals[i:end]

		// Process batch in parallel
		var wg sync.WaitGroup
		var mu sync.Mutex
		for _, journal := range batch {
			wg.Add(1)
			go func(j IngestionJournal) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					mu.Lock()
					result.Failed++
					mu.Unlock()
					return
				default:
				}

				// Validate
				if j.JournalID == "" || j.UserID == "" || strings.TrimSpace(j.Content) == "" {
					log.Printf("BatchIngest: Skipping invalid journal %v", j.JournalID)
					mu.Lock()
					result.Skipped++
					mu.Unlock()
					return
				}

				// Create Job
				jobID := uuid.New()
				// Parse UUIDs (assuming valid for now, or log error)
				journalUID, err := uuid.Parse(j.JournalID)
				if err != nil {
					log.Printf("BatchIngest: Invalid journal ID %s: %v", j.JournalID, err)
					return
				}
				userUID, err := uuid.Parse(j.UserID)
				if err != nil {
					log.Printf("BatchIngest: Invalid user ID %s: %v", j.UserID, err)
					return
				}

				timestamp := j.CreatedAt
				if timestamp.IsZero() {
					timestamp = time.Now()
				}

				job := &Job{
					ID:          jobID,
					JournalID:   journalUID,
					UserID:      userUID,
					Title:       j.Title,
					Content:     j.Content,
					Status:      StatusPending,
					Attempts:    0,
					MaxAttempts: DefaultMaxAttempts,
					CreatedAt:   timestamp,
					UpdatedAt:   time.Now(),
				}

				if err := w.repo.CreateJob(ctx, job); err != nil {
					log.Printf("BatchIngest: Failed to create job for journal %s: %v", j.JournalID, err)
					mu.Lock()
					result.Failed++
					mu.Unlock()
					return
				}
				mu.Lock()
				result.Queued++
				mu.Unlock()
			}(journal)
		}
		wg.Wait()

		// Rate limit to avoid overwhelming DB or if we move to immediate processing
		if end < len(journals) {
			time.Sleep(1 * time.Second)
		}
	}
	return result, nil
}
