package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
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

	// Minimum similarity of 0.50 (50% cosine similarity)
	// This balances between quality and recall for journal queries
	docs, err := pgStore.SearchByUser(qEmbedding, 5, req.UserID, 0.50)
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

	// Require at least 2 chunks for confidence (unless only 1 journal exists)
	if len(docs) < 2 {
		// Check similarity score of the single result
		similarity, ok := docs[0].Metadata["similarity"].(float32)
		if !ok || similarity < 0.75 {
			json.NewEncoder(w).Encode(AskResponse{
				Answer: "I'm not confident enough to answer based on your journals. The information might not be directly related to your question.",
			})
			return
		}
	}

	// 4. Construct context with source attribution
	var context strings.Builder
	var sources []map[string]interface{}

	for i, doc := range docs {
		// Add chunk content to context
		context.WriteString(fmt.Sprintf("Journal Entry %d:\n%s\n\n---\n\n", i+1, doc.Content))

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

	// 5. Enhanced prompt with strict anti-hallucination rules
	prompt := fmt.Sprintf(`You are a helpful assistant for the user's personal journal system called Mind Garden.

STRICT RULES - YOU MUST FOLLOW THESE:
1. Answer ONLY using the journal entries provided below
2. If the answer is not in the journals, say "I don't find that in your journals"
3. Never make assumptions or use external knowledge
4. Never invent facts or details not present in the journals
5. If uncertain, say "I'm not sure based on your journals"
6. Quote or reference specific journal entries when possible
7. Keep answers concise and grounded in the provided context

Journal Entries:
%s

Question: %s

Answer (remember: ONLY use information from the journal entries above):`, context.String(), req.Question)

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
