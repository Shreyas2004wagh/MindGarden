package ingestion

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mindgarden/server/internal/service/chunker"
	"github.com/mindgarden/server/internal/service/llm"
	"github.com/mindgarden/server/internal/service/vector"
)

type Worker struct {
	repo        Repository
	llmService  *llm.Service
	vectorStore vector.VectorStore
	stop        chan struct{}
}

func NewWorker(repo Repository, llmService *llm.Service, vectorStore vector.VectorStore) *Worker {
	return &Worker{
		repo:        repo,
		llmService:  llmService,
		vectorStore: vectorStore,
		stop:        make(chan struct{}),
	}
}

func (w *Worker) Start() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Ingestion worker panicked: %v", r)
			}
		}()
		log.Println("Ingestion worker background routine started")
		ticker := time.NewTicker(1 * time.Second) // Poll every second
		defer ticker.Stop()

		for {
			select {
			case <-w.stop:
				return
			case <-ticker.C:
				w.processNextJob()
			}
		}
	}()
	log.Println("Ingestion worker started")
}

func (w *Worker) Stop() {
	close(w.stop)
	log.Println("Ingestion worker stopped")
}

func (w *Worker) processNextJob() {
	ctx := context.Background()
	// Fetch pending job
	job, err := w.repo.GetPendingJob(ctx)
	if err != nil {
		log.Printf("Error fetching pending job: %v", err)
		return
	}
	if job == nil {
		return // No jobs
	}

	log.Printf("Processing job %s", job.ID)

	// Update status to processing
	if err := w.repo.UpdateJobStatus(ctx, job.ID, StatusProcessing, nil); err != nil {
		log.Printf("Error updating job status to processing: %v", err)
		return
	}

	// Perform ingestion
	if err := w.ingest(ctx, job); err != nil {
		log.Printf("Job %s failed: %v", job.ID, err)
		errMsg := err.Error()
		w.repo.IncrementAttempts(ctx, job.ID)

		// Check max attempts
		// Need to refresh job to get current attempts if they were incremented concurrently,
		// but since we are the only worker processing this ID, we can assume local copy + 1 is accurate enough or just use the IncrementAttempts behavior.
		// However, IncrementAttempts happens in DB.
		// Let's assume we want to retry if attempts < MaxAttempts.
		// We should probably read back the job or pass the incremented attempt count.
		// For MVP, simplistic check:
		if job.Attempts < job.MaxAttempts {
			log.Printf("Job %s failed (attempt %d/%d). Retrying...", job.ID, job.Attempts+1, job.MaxAttempts)
			// Reset to pending so it gets picked up again
			// Ideally we would add a backoff (e.g. valid_after column) but for MVP immediate retry or simple delay is fine.
			w.repo.UpdateJobStatus(ctx, job.ID, StatusPending, &errMsg)
		} else {
			log.Printf("Job %s failed permanently after %d attempts.", job.ID, job.MaxAttempts)
			w.repo.UpdateJobStatus(ctx, job.ID, StatusFailed, &errMsg)
		}
	} else {
		log.Printf("Job %s completed successfully", job.ID)
		w.repo.UpdateJobStatus(ctx, job.ID, StatusCompleted, nil)
	}
}

func (w *Worker) ingest(ctx context.Context, job *Job) error {
	// 1. Chunk content with adaptive configuration
	config := chunker.GetOptimalConfig(len(job.Content))
	chunks := chunker.ChunkTextEnriched(job.Content, config)

	// 2. Generate embeddings and store
	for _, chunk := range chunks {
		embedding, err := w.llmService.GetEmbedding(ctx, chunk.Content)
		if err != nil {
			return fmt.Errorf("embedding generation failed: %w", err)
		}

		titleStr := ""
		if job.Title != nil {
			titleStr = *job.Title
		}

		doc := vector.Document{
			ID:        uuid.New().String(),
			Embedding: embedding,
			Content:   chunk.Content,
			Metadata: map[string]interface{}{
				"user_id":      job.UserID.String(),
				"journal_id":   job.JournalID.String(),
				"chunk_index":  chunk.Index,
				"total_chunks": chunk.TotalCount,
				"title":        titleStr,
				"timestamp":    job.CreatedAt,
				"job_id":       job.ID.String(),
				// Enriched metadata
				"word_count":     chunk.WordCount,
				"sentence_count": chunk.SentenceCount,
				"has_questions":  chunk.HasQuestions,
				"has_dates":      chunk.HasDates,
				"sentiment":      chunk.Sentiment,
				"topics":         chunk.Topics,
			},
		}

		if err := w.vectorStore.Add(doc); err != nil {
			return fmt.Errorf("failed to save to vector store: %w", err)
		}
	}

	// Ensure implementation saves (commit/flush if needed)
	w.vectorStore.Save()

	return nil
}
