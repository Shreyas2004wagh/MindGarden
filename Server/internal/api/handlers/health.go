package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mindgarden/server/internal/service/vector"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	checks := map[string]bool{
		"database":     false,
		"vector_store": false,
		"gemini_api":   false,
	}

	status := "healthy"

	// Database check
	db := GetDB()
	if db == nil {
		status = "unhealthy"
	} else {
		dbCtx, dbCancel := context.WithTimeout(r.Context(), 2*time.Second)
		if err := db.Ping(dbCtx); err != nil {
			status = "unhealthy"
		} else {
			checks["database"] = true
		}
		dbCancel()
	}

	// Vector store check
	if pgStore, ok := vectorStore.(*vector.PostgresStore); ok && pgStore.Pool == nil {
		status = "unhealthy"
	} else if vectorStore != nil {
		probe := make(vector.Vector, embeddingDimensions())
		if _, err := vectorStore.Search(probe, 1); err == nil {
			checks["vector_store"] = true
		} else {
			status = "unhealthy"
		}
	} else {
		status = "unhealthy"
	}

	// Gemini API check
	if llmService != nil {
		llmCtx, llmCancel := context.WithTimeout(r.Context(), 10*time.Second)
		geminiProbe := "health-check-" + time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := llmService.GetEmbedding(llmCtx, geminiProbe); err == nil {
			checks["gemini_api"] = true
		} else if status != "unhealthy" {
			status = "degraded"
		}
		llmCancel()
	} else if status != "unhealthy" {
		status = "degraded"
	}

	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"checks": checks,
	})
}

func embeddingDimensions() int {
	raw := strings.TrimSpace(os.Getenv("EMBEDDING_DIMENSIONS"))
	if raw == "" {
		return 768
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 768
	}
	return value
}
