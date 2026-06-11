package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mindgarden/server/internal/service/ingestion"
)

type CreateJournalRequest struct {
	Title   *string `json:"title"`
	Content string  `json:"content"`
}

type JournalResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Title     *string    `json:"title"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

func CreateJournal(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid or expired token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req CreateJournalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate content
	trimmedContent := strings.TrimSpace(req.Content)
	if trimmedContent == "" {
		http.Error(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	// Trim title if provided
	var title *string
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed != "" {
			title = &trimmed
		}
	}

	// Create journal entry in database
	journalID := uuid.New().String()
	createdAt := time.Now()

	journal, err := insertJournal(r.Context(), journalID, userID, title, trimmedContent, createdAt)
	if err != nil {
		http.Error(w, "Failed to create journal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse UUIDs safely
	journalUID, err := uuid.Parse(journalID)
	if err != nil {
		http.Error(w, "Invalid internal journal ID", http.StatusInternalServerError)
		return
	}
	userUID, err := uuid.Parse(userID)
	if err != nil {
		// user_id comes from JWT, so this might mean JWT sub is not UUID
		http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
		return
	}

	// Queue ingestion job
	if ingestionRepo == nil {
		log.Printf("Ingestion service unavailable; journal %s was saved without an ingestion job", journalID)
	} else {
		jobID := uuid.New()
		job := &ingestion.Job{
			ID:        jobID,
			JournalID: journalUID,
			UserID:    userUID,
			Title:     title,
			Content:   trimmedContent,
			Status:    ingestion.StatusPending,
			CreatedAt: createdAt,
			UpdatedAt: time.Now(),
		}

		if err := ingestionRepo.CreateJob(r.Context(), job); err != nil {
			log.Printf("Failed to queue ingestion job for journal %s: %v", journalID, err)
		}
	}

	// Return created journal immediately (don't wait for ingestion)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(journal)
}
