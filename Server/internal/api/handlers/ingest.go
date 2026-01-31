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
	vectorStore *vector.Store
)

func InitServices() {
	llmService = llm.NewService()
	// Per-user store in real app. Here single store for MVP demo.
	os.MkdirAll("storage", 0755)
	vectorStore = vector.NewStore("storage/index.json")
	vectorStore.Load()
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
	vectorStore.Add(doc)
	vectorStore.Save()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ingested", "id": doc.ID})
}
