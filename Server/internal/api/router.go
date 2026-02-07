package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/mindgarden/server/internal/api/handlers"
)

func MountRoutes(r chi.Router) {
	r.Get("/healthz", handlers.HealthCheck)

	// Initialize services (Simple global init for MVP)
	handlers.InitServices()

	// Journal endpoints
	r.Post("/journals", handlers.CreateJournal)

	// AI/RAG endpoints
	r.Post("/ingest", handlers.IngestJournal)
	r.Post("/ingest/batch", handlers.IngestJournalBatch)
	r.Post("/ask", handlers.AskAI)
}
