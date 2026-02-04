package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mindgarden/server/internal/service/chunker"
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
	llmService  *llm.Service
	vectorStore vector.VectorStore
)

func InitServices() {
	llmService = llm.NewService()

	// Initialize Postgres Vector Store
	pgStore := vector.NewPostgresStore(getDB())
	if err := pgStore.InitSchema(); err != nil {
		// Log but don't panic? Or panic since RAG depends on it.
		// For MVP, printing is enough, it will fail on requests.
		// Use standard logger
		println("Failed to init vector schema:", err.Error())
	}
	vectorStore = pgStore
}

func IngestJournal(w http.ResponseWriter, r *http.Request) {
	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.JournalID == "" || req.UserID == "" || strings.TrimSpace(req.Content) == "" {
		http.Error(w, "journal_id, user_id, and content are required", http.StatusBadRequest)
		return
	}

	// Check for idempotency: if journal already ingested, skip
	// This prevents duplicate embeddings if the endpoint is called multiple times
	// We can check by querying for existing embeddings with this journal_id
	// For MVP, we'll skip this check and rely on application logic

	// 1. Chunk the content using semantic chunking
	config := chunker.DefaultConfig()
	chunks := chunker.ChunkText(req.Content, config)

	// 2. Generate embeddings and store each chunk
	var ingestedIDs []string
	for _, chunk := range chunks {
		// Generate embedding for this chunk
		embedding, err := llmService.GetEmbedding(r.Context(), chunk.Content)
		if err != nil {
			http.Error(w, "Failed to generate embedding: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Prepare metadata
		titleStr := ""
		if req.Title != nil {
			titleStr = *req.Title
		}

		// Use provided timestamp or default to now
		timestamp := req.CreatedAt
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		// Create document with enhanced metadata
		doc := vector.Document{
			ID:        uuid.New().String(),
			Embedding: embedding,
			Content:   chunk.Content,
			Metadata: map[string]interface{}{
				"user_id":      req.UserID,
				"journal_id":   req.JournalID,
				"chunk_index":  chunk.Index,
				"total_chunks": chunk.TotalCount,
				"title":        titleStr,
				"timestamp":    timestamp, // Use journal creation time
			},
		}

		// Save to vector store
		if err := vectorStore.Add(doc); err != nil {
			http.Error(w, "Failed to save to vector store: "+err.Error(), http.StatusInternalServerError)
			return
		}

		ingestedIDs = append(ingestedIDs, doc.ID)
	}

	// Call Save() for interface compliance (no-op for Postgres)
	vectorStore.Save()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ingested",
		"journal_id":     req.JournalID,
		"chunks_created": len(chunks),
		"embedding_ids":  ingestedIDs,
	})
}
