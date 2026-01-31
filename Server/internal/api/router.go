package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/mindgarden/server/internal/api/handlers"
)

func MountRoutes(r chi.Router) {
	r.Get("/healthz", handlers.HealthCheck)
	
	// Initialize services (Simple global init for MVP)
	// handlers.InitServices()

	// r.Post("/ingest", handlers.IngestJournal)
	// r.Post("/ask", handlers.AskAI)
}
