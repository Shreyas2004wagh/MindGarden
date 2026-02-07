package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SearchByUser searches for similar documents filtered by user_id with similarity threshold
func (s *PostgresStore) SearchByUser(query Vector, k int, userID string, minSimilarity float32) ([]Document, error) {
	embeddingBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query embedding: %w", err)
	}
	embeddingStr := string(embeddingBytes)

	// Use <=> for cosine distance, filter by user_id
	// cosine_distance = 1 - cosine_similarity
	// So if we want similarity > 0.7, we need distance < 0.3
	maxDistance := 1.0 - minSimilarity

	sqlQuery := `
		SELECT id, content, metadata, embedding, 
		       (embedding <=> $1) as distance
		FROM embeddings
		WHERE user_id = $2::uuid
		  AND (embedding <=> $1) < $3
		ORDER BY distance
		LIMIT $4
	`

	rows, err := s.Pool.Query(context.Background(), sqlQuery, embeddingStr, userID, maxDistance, k)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		var metadataBytes []byte
		var embeddingStr string
		var distance float64 // pgx returns float64 for float8

		if err := rows.Scan(&doc.ID, &doc.Content, &metadataBytes, &embeddingStr, &distance); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataBytes, &doc.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		// Parse embedding
		if err := json.Unmarshal([]byte(embeddingStr), &doc.Embedding); err != nil {
			return nil, fmt.Errorf("failed to parse embedding from db: %w", err)
		}

		// Fix timestamp: JSON unmarshaling converts time.Time to string
		// We need to parse it back to time.Time
		if timestampStr, ok := doc.Metadata["timestamp"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, timestampStr); err == nil {
				doc.Metadata["timestamp"] = parsedTime
			}
		}

		// Store similarity score in metadata for reference
		doc.Metadata["similarity"] = float32(1.0 - distance)

		docs = append(docs, doc)
	}
	return docs, nil
}

// DeleteByJournalID deletes all embeddings for a specific journal
func (s *PostgresStore) DeleteByJournalID(journalID string, userID string) error {
	query := `DELETE FROM embeddings WHERE journal_id = $1 AND user_id = $2`
	_, err := s.Pool.Exec(context.Background(), query, journalID, userID)
	return err
}
