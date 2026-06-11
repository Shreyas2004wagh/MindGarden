package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mindgarden/server/internal/service/ingestion"
	"github.com/mindgarden/server/internal/service/llm"
	"github.com/mindgarden/server/internal/service/vector"
)

type IngestRequest struct {
	JournalID string    `json:"journal_id"` // Required: journal ID
	Title     *string   `json:"title"`      // Optional: journal title
	Content   string    `json:"content"`    // Required: journal content
	UserID    string    `json:"user_id"`    // Required: user ID
	CreatedAt time.Time `json:"created_at"` // Optional: journal creation time
}

// Global instances for MVP simplicity. In production use dependency injection.
var (
	llmService      *llm.Service
	vectorStore     vector.VectorStore
	ingestionRepo   ingestion.Repository
	ingestionWorker *ingestion.Worker
)

func InitServices() {
	log.Println("InitServices: Starting...")

	llmService = llm.NewService()
	log.Println("InitServices: LLM Service initialized")

	database := getDB()
	if database == nil {
		log.Println("InitServices: Database is not initialized; vector search and ingestion are disabled")
		return
	}

	// Initialize Postgres Vector Store
	pgStore := vector.NewPostgresStore(database)
	if err := pgStore.InitSchema(); err != nil {
		log.Println("InitServices: Failed to init vector schema:", err.Error())
	} else {
		log.Println("InitServices: Vector schema initialized")
	}
	vectorStore = pgStore

	// Initialize Ingestion System
	ingestionRepo = ingestion.NewPostgresRepository(database)
	if err := ingestionRepo.InitSchema(context.Background()); err != nil {
		log.Println("InitServices: Failed to init ingestion schema:", err.Error())
	} else {
		log.Println("InitServices: Ingestion schema initialized")
	}

	ingestionWorker = ingestion.NewWorker(ingestionRepo, llmService, vectorStore)
	ingestionWorker.Start()
	log.Println("InitServices: Worker started. Initialization complete.")
}

func IngestJournal(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid or expired token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.JournalID == "" || strings.TrimSpace(req.Content) == "" {
		http.Error(w, "journal_id and content are required", http.StatusBadRequest)
		return
	}
	if req.UserID != "" && req.UserID != userID {
		http.Error(w, "user_id does not match authenticated user", http.StatusForbidden)
		return
	}
	req.UserID = userID

	// Parse UUIDs
	journalUID, err := uuid.Parse(req.JournalID)
	if err != nil {
		http.Error(w, "Invalid journal_id format", http.StatusBadRequest)
		return
	}
	userUID, err := uuid.Parse(req.UserID)
	if err != nil {
		http.Error(w, "Invalid user_id format", http.StatusBadRequest)
		return
	}

	// Prepare Job
	jobID := uuid.New()
	// Use provided timestamp or default to now
	timestamp := req.CreatedAt
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	job := &ingestion.Job{
		ID:        jobID,
		JournalID: journalUID,
		UserID:    userUID,
		Title:     req.Title,
		Content:   req.Content,
		Status:    ingestion.StatusPending,
		CreatedAt: timestamp,
		UpdatedAt: time.Now(),
	}

	// Create Job in DB
	if ingestionRepo == nil {
		http.Error(w, "Ingestion service not initialized", http.StatusServiceUnavailable)
		return
	}
	if err := ingestionRepo.CreateJob(r.Context(), job); err != nil {
		http.Error(w, "Failed to queue ingestion job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "pending",
		"job_id":  jobID.String(),
		"message": "Ingestion job queued",
	})
}
