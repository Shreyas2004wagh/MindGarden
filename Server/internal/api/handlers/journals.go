package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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
	// Extract JWT from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	// Extract Bearer token
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		http.Error(w, "Invalid authorization format. Expected: Bearer <token>", http.StatusUnauthorized)
		return
	}

	// Verify JWT and extract user_id (using simple HS256 verification)
	userID, err := verifySupabaseJWT(token)
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

	// Return created journal
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(journal)
}
