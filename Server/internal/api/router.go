package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/mindgarden/server/internal/api/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func MountRoutes(r chi.Router) {
	r.Get("/healthz", handlers.HealthCheck)
	r.Handle("/metrics", promhttp.Handler())

	// Initialize services (Simple global init for MVP)
	handlers.InitServices()

	r.Group(func(r chi.Router) {
		r.Use(handlers.RateLimitMiddleware)

		// Journal endpoints
		r.Post("/journals", handlers.CreateJournal)

		// AI/RAG endpoints
		r.Post("/ingest", handlers.IngestJournal)
		r.Post("/ingest/batch", handlers.IngestJournalBatch)
		r.Post("/ask", handlers.AskAI)
	})
}
