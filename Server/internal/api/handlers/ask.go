package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/mindgarden/server/internal/observability"
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
	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate user_id
	if req.UserID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	// 1. Embed Question
	qEmbedding, err := llmService.GetEmbedding(r.Context(), req.Question)
	if err != nil {
		log.Printf("AskAI: Embedding failed: %v", err)
		http.Error(w, "Failed to embed question", http.StatusInternalServerError)
		return
	}

	// 2. Search with user isolation and similarity threshold
	// Use HybridSearch for better recall (keywords + semantic)
	pgStore, ok := vectorStore.(*vector.PostgresStore)
	if !ok {
		// Fallback to regular search if not PostgresStore
		log.Printf("AskAI: Vector Store Type Error")
		http.Error(w, "Vector store not properly initialized", http.StatusInternalServerError)
		return
	}

	// Hybrid Search with 5 results
	// We don't filter by min similarity in DB anymore, we filter by confidence later
	searchStart := time.Now()
	docs, err := pgStore.HybridSearch(req.Question, qEmbedding, 5, req.UserID)
	observability.RecordSearchDuration(time.Since(searchStart))
	if err != nil {
		log.Printf("AskAI: HybridSearch failed: %v", err)
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

	// Calculate average score for confidence assessment
	var totalScore float64
	for _, doc := range docs {
		if score, ok := doc.Metadata["score"].(float64); ok {
			totalScore += score
		} else if similarity, ok := doc.Metadata["similarity"].(float32); ok {
			totalScore += float64(similarity)
		}
	}
	avgScore := float32(totalScore) / float32(len(docs))

	// Tiered confidence approach
	if avgScore < 0.30 {
		json.NewEncoder(w).Encode(AskResponse{
			Answer: "I don't have enough information in your journals to answer that question confidently.",
		})
		return
	}

	// For single results, require slightly higher confidence
	if len(docs) == 1 {
		score, ok := docs[0].Metadata["score"].(float64)
		if !ok {
			if sim, ok := docs[0].Metadata["similarity"].(float32); ok {
				score = float64(sim)
			}
		}
		if score < 0.45 {
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
		dateStr := "Unknown date"
		if timestamp, ok := doc.Metadata["timestamp"].(time.Time); ok {
			dateStr = timestamp.Format("January 2, 2006")
		}

		context.WriteString(fmt.Sprintf("Journal Entry %d (Written on %s):\n%s\n\n---\n\n",
			i+1, dateStr, doc.Content))

		source := map[string]interface{}{
			"journal_id": doc.Metadata["journal_id"],
			"title":      doc.Metadata["title"],
			"score":      doc.Metadata["score"],
		}
		if timestamp, ok := doc.Metadata["timestamp"].(time.Time); ok {
			source["date"] = timestamp.Format("2006-01-02")
		}
		sources = append(sources, source)
	}

	// 5. Enhanced prompt
	confidenceInstruction := ""
	if avgScore >= 0.60 {
		confidenceInstruction = ""
	} else if avgScore >= 0.30 {
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
		log.Printf("AskAI: GenerateAnswer failed: %v", err)
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
