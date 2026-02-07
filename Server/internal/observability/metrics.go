package observability

import (
	"os"
	"strconv"
	"strings"
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
)

func init() {
	prometheus.MustRegister(
		ingestionsTotal,
		ingestionDuration,
		searchDuration,
		embeddingCost,
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
	rate := embeddingCostRatePer1KTokens()
	if rate <= 0 {
		return
	}

	tokens := estimateTokenCount(text)
	cost := (float64(tokens) / 1000.0) * rate
	if cost > 0 {
		embeddingCost.Add(cost)
	}
}

func embeddingCostRatePer1KTokens() float64 {
	raw := strings.TrimSpace(os.Getenv("EMBEDDING_COST_PER_1K_TOKENS_USD"))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func estimateTokenCount(text string) int {
	if text == "" {
		return 0
	}
	// Rough approximation for English-like text.
	charCount := utf8.RuneCountInString(text)
	tokens := charCount / 4
	if tokens <= 0 {
		return 1
	}
	return tokens
}
