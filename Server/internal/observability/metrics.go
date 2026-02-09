package observability

import (
	"time"
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ingestionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingestions_total",
			Help: "Total number of ingestion attempts",
		},
		[]string{"status"},
	)

	ingestionDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ingestion_duration_seconds",
			Help:    "Time taken to ingest a journal",
			Buckets: prometheus.DefBuckets,
		},
	)

	searchDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "search_duration_seconds",
			Help:    "Time taken to search",
			Buckets: prometheus.DefBuckets,
		},
	)

	embeddingCost = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "embedding_api_cost_usd",
			Help: "Estimated cost of embedding API calls",
		},
	)

	llmCost = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "llm_api_cost_usd",
			Help: "Estimated cost of LLM API calls",
		},
	)
)

func init() {
	prometheus.MustRegister(
		ingestionsTotal,
		ingestionDuration,
		searchDuration,
		embeddingCost,
		llmCost,
	)
}

func RecordIngestion(status string, duration time.Duration) {
	if status != "success" && status != "failure" {
		status = "failure"
	}
	ingestionsTotal.WithLabelValues(status).Inc()
	ingestionDuration.Observe(duration.Seconds())
}

func RecordSearchDuration(duration time.Duration) {
	searchDuration.Observe(duration.Seconds())
}

func RecordEmbeddingCostEstimate(text string) {
	TrackEmbeddingCost(utf8.RuneCountInString(text))
}

func TrackEmbeddingCost(textLength int) {
	if textLength <= 0 {
		return
	}
	// Gemini pricing approximation: $0.00001 per 1000 chars.
	cost := float64(textLength) / 1000.0 * 0.00001
	if cost > 0 {
		embeddingCost.Add(cost)
	}
}

func TrackLLMCost(promptTokens, completionTokens int) {
	if promptTokens < 0 {
		promptTokens = 0
	}
	if completionTokens < 0 {
		completionTokens = 0
	}
	// Groq pricing approximation by token type.
	cost := (float64(promptTokens) * 0.00001) + (float64(completionTokens) * 0.00002)
	if cost > 0 {
		llmCost.Add(cost)
	}
}
