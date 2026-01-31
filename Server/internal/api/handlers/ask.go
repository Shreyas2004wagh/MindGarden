package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type AskRequest struct {
	Question string `json:"question"`
	UserID   string `json:"user_id"`
}

func AskAI(w http.ResponseWriter, r *http.Request) {
	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Embed Question
	qEmbedding, err := llmService.GetEmbedding(r.Context(), req.Question)
	if err != nil {
		http.Error(w, "Failed to embed question", http.StatusInternalServerError)
		return
	}

	// 2. Search
	docs := vectorStore.Search(qEmbedding, 3)
	if len(docs) == 0 {
		json.NewEncoder(w).Encode(map[string]string{"answer": " I don't have enough information in your journals to answer that."})
		return
	}

	// 3. Construct Prompt
	var context strings.Builder
	for _, doc := range docs {
		context.WriteString(doc.Content + "\n---\n")
	}

	prompt := fmt.Sprintf(`You are a helpful assistant for the user's personal journal. 
Answer the question based ONLY on the context below.
If you don't know, say "I don't find that in your journals."

Context:
%s

Question: %s`, context.String(), req.Question)

	// 4. Generate Answer
	answer, err := llmService.GenerateAnswer(r.Context(), prompt)
	if err != nil {
		http.Error(w, "Failed to generate answer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"answer": answer})
}
