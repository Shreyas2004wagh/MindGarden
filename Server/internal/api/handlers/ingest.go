package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mindgarden/server/internal/service/llm"
	"github.com/mindgarden/server/internal/service/vector"
)

type IngestRequest struct {
	Content string `json:"content"`
	UserID  string `json:"user_id"` // In real app, get this from context/auth
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

	// 1. Chunking (Simple split by newlines for MVP)
	// Real app: robust token-based chunking
	
	// 2. Generate Embedding
	embedding, err := llmService.GetEmbedding(r.Context(), req.Content)
	if err != nil {
		http.Error(w, "Failed to generate embedding: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Save to Vector Store
	doc := vector.Document{
		ID:        uuid.New().String(),
		Embedding: embedding,
		Content:   req.Content,
		Metadata: map[string]interface{}{
			"user_id":   req.UserID,
			"timestamp": time.Now(),
		},
	}
	if err := vectorStore.Add(doc); err != nil {
		http.Error(w, "Failed to save to vector store: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// vectorStore.Save() // No-op for Postgres, but part of interface if we kept it. 
    // Actually the interface has Save(), so we can call it or ignore it.
    // Let's call it for correctness with interface, distinct from Add error.
    vectorStore.Save()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ingested", "id": doc.ID})
}
