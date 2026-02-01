package vector

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type PostgresStore struct {
	DB *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{DB: db}
}

func (s *PostgresStore) Add(doc Document) error {
	// pgvector input format: "[1.1,2.2,3.3]"
	// JSON marshaling a float array gives exactly that format.
	embeddingBytes, err := json.Marshal(doc.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}
	embeddingStr := string(embeddingBytes)

	// Marshal metadata to JSONB
	metadataBytes, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO embeddings (id, content, metadata, embedding)
		VALUES ($1, $2, $3, $4)
	`
	_, err = s.DB.Exec(query, doc.ID, doc.Content, metadataBytes, embeddingStr)
	return err
}

func (s *PostgresStore) Save() error {
	// No-op for Postgres
	return nil
}

func (s *PostgresStore) Load() error {
	// No-op for Postgres
	return nil
}

func (s *PostgresStore) Search(query Vector, k int) ([]Document, error) {
	embeddingBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query embedding: %w", err)
	}
	embeddingStr := string(embeddingBytes)

	// Use <=> operator for cosine distance (ASC order usually means closest distance = most similar if normalized? 
	// Wait, cosine distance: 0 is identical, 2 is opposite.
	// So Order By distance ASC.
	// Note: 1 - cosine_similarity = cosine_distance for normalized vectors.
	sqlQuery := `
		SELECT id, content, metadata, embedding
		FROM embeddings
		ORDER BY embedding <=> $1
		LIMIT $2
	`

	rows, err := s.DB.Query(sqlQuery, embeddingStr, k)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		var metadataBytes []byte
		var embeddingStr string

		if err := rows.Scan(&doc.ID, &doc.Content, &metadataBytes, &embeddingStr); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataBytes, &doc.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		// Parse embedding string back to Vector if needed, 
		// or typically we don't strictly need the embedding returned in search results 
		// if we just want content/metadata. 
		// But let's parse it to be compliant with Document struct.
		// pgvector returns string "[1,2,3]"
		if err := json.Unmarshal([]byte(embeddingStr), &doc.Embedding); err != nil {
			// Try to handle postgres specific formatting if json fails, 
			// but usually json unmarshal works on "[...]" string.
			// If it fails, we might need manual parsing.
			// Let's assume standard format for now.
			return nil, fmt.Errorf("failed to parse embedding from db: %w", err)
		}

		docs = append(docs, doc)
	}
	return docs, nil
}

// Helper to ensure the table exists (for simple migration)
func (s *PostgresStore) InitSchema() error {
	// Enable pgvector
	_, err := s.DB.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create table
	// Using 768 dimensions for Gemini text-embedding-004
	query := `
		CREATE TABLE IF NOT EXISTS embeddings (
			id UUID PRIMARY KEY,
			content TEXT,
			metadata JSONB,
			embedding vector(768)
		)
	`
	_, err = s.DB.Exec(query)
	if err != nil {
		// Fallback for different dimension or if table exists with diff schema?
		// For now simple error
		if strings.Contains(err.Error(), "dimension") {
             // Try creating without dimension check if it fails? No, better to face it.
		}
		return fmt.Errorf("failed to create embeddings table: %w", err)
	}
	
	// Create HNSW index for performance
	// "vector_cosine_ops" is for <=>
	indexQuery := `
		CREATE INDEX IF NOT EXISTS embeddings_embedding_idx 
		ON embeddings 
		USING hnsw (embedding vector_cosine_ops)
	`
	_, err = s.DB.Exec(indexQuery)
    // Index creation might fail if not enough data or other reasons, but usually fine.
    // Allow it to fail silently? No, log it.
    if err != nil {
        return fmt.Errorf("failed to create index: %w", err)
    }

	return nil
}
