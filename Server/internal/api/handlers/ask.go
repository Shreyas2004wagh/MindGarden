package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/mindgarden/server/internal/service/vector"
)

type AskRequest struct {
	Question string `json:"question"`
	UserID   string `json:"user_id"`
}

type AskResponse struct {
	Answer  string                   `json:"answer"`
	Sources []map[string]interface{} `json:"sources,omitempty"`
}

func AskAI(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		http.Error(w, "Invalid or expired token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

	if req.UserID != "" && req.UserID != userID {
		http.Error(w, "user_id does not match authenticated user", http.StatusForbidden)
		return
	}
	req.UserID = userID

	if llmService == nil || vectorStore == nil {
		http.Error(w, "AI services are not initialized", http.StatusServiceUnavailable)
		return
	}

	// 1. Embed Question
	qEmbedding, err := llmService.GetEmbedding(r.Context(), req.Question)
	if err != nil {
		http.Error(w, "Failed to embed question", http.StatusInternalServerError)
		return
	}

	// 2. Search with user isolation and similarity threshold
	// Use SearchByUser from user_search.go for privacy and quality
	pgStore, ok := vectorStore.(*vector.PostgresStore)
	if !ok {
		// Fallback to regular search if not PostgresStore
		http.Error(w, "Vector store not properly initialized", http.StatusInternalServerError)
		return
	}

	// Minimum similarity of 0.35 (35% cosine similarity)
	// This balances between quality and recall for journal queries
	// Lower threshold allows more relevant chunks while tiered confidence handles quality
	docs, err := pgStore.SearchByUser(qEmbedding, 5, req.UserID, 0.35)
	if err != nil {
		http.Error(w, "Vector search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Check if we have enough relevant context
	if len(docs) == 0 {
		json.NewEncoder(w).Encode(AskResponse{
			Answer: "I don't have enough information in your journals to answer that question.",
		})
		return
	}

	// Sort results chronologically (most recent first) for better temporal context
	sort.Slice(docs, func(i, j int) bool {
		ti, okI := docs[i].Metadata["timestamp"].(time.Time)
		tj, okJ := docs[j].Metadata["timestamp"].(time.Time)
		if okI && okJ {
			return ti.After(tj)
		}
		return false
	})

	// Calculate average similarity for confidence assessment
	var totalSimilarity float32
	for _, doc := range docs {
		if similarity, ok := doc.Metadata["similarity"].(float32); ok {
			totalSimilarity += similarity
		}
	}
	avgSimilarity := totalSimilarity / float32(len(docs))

	// Tiered confidence approach
	// Low confidence: reject if average similarity is too low
	if avgSimilarity < 0.35 {
		json.NewEncoder(w).Encode(AskResponse{
			Answer: "I don't have enough information in your journals to answer that question confidently.",
		})
		return
	}

	// For single results, require slightly higher confidence
	if len(docs) == 1 {
		similarity, ok := docs[0].Metadata["similarity"].(float32)
		if !ok || similarity < 0.50 {
			json.NewEncoder(w).Encode(AskResponse{
				Answer: "I'm not confident enough to answer based on your journals. The information might not be directly related to your question.",
			})
			return
		}
	}

	// 4. Construct context with source attribution and dates
	var context strings.Builder
	var sources []map[string]interface{}

	for i, doc := range docs {
		// Extract and format date
		dateStr := "Unknown date"
		if timestamp, ok := doc.Metadata["timestamp"].(time.Time); ok {
			dateStr = timestamp.Format("January 2, 2006")
		}

		// Add chunk content to context with date
		context.WriteString(fmt.Sprintf("Journal Entry %d (Written on %s):\n%s\n\n---\n\n",
			i+1, dateStr, doc.Content))

		// Collect source information
		source := map[string]interface{}{
			"journal_id": doc.Metadata["journal_id"],
			"title":      doc.Metadata["title"],
			"similarity": doc.Metadata["similarity"],
		}
		if timestamp, ok := doc.Metadata["timestamp"].(time.Time); ok {
			source["date"] = timestamp.Format("2006-01-02")
		}
		sources = append(sources, source)
	}

	// 5. Enhanced prompt with strict anti-hallucination rules and temporal awareness
	confidenceInstruction := ""
	if avgSimilarity >= 0.65 {
		// High confidence: answer directly
		confidenceInstruction = ""
	} else if avgSimilarity >= 0.35 {
		// Medium confidence: add caveat
		confidenceInstruction = "\n9. Start your answer with 'Based on what I found in your journals...' to indicate moderate confidence"
	}

	prompt := fmt.Sprintf(`You are a helpful assistant for the user's personal journal system called Mind Garden.

STRICT RULES - YOU MUST FOLLOW THESE:
1. Answer ONLY using the journal entries provided below
2. Each entry includes the date it was written - use this for temporal questions
3. When answering "when" questions, reference the specific dates from the entries
4. If the answer is not in the journals, say "I don't find that in your journals"
5. Never make assumptions or use external knowledge
6. Never invent facts or details not present in the journals
7. If uncertain, say "I'm not sure based on your journals"
8. Quote or reference specific journal entries when possible%s

Journal Entries (sorted by date, most recent first):
%s

Question: %s

Answer (remember: ONLY use information from the journal entries above):`, confidenceInstruction, context.String(), req.Question)

	// 6. Generate Answer
	answer, err := llmService.GenerateAnswer(r.Context(), prompt)
	if err != nil {
		http.Error(w, "Failed to generate answer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 7. Return answer with source attribution
	json.NewEncoder(w).Encode(AskResponse{
		Answer:  answer,
		Sources: sources,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
