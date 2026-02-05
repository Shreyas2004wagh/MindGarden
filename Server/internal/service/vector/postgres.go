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

	// Extract user_id, journal_id, chunk_index, and title from metadata
	userID, _ := doc.Metadata["user_id"].(string)
	journalID, _ := doc.Metadata["journal_id"].(string)
	chunkIndex, _ := doc.Metadata["chunk_index"].(int)
	title, _ := doc.Metadata["title"].(string)

	query := `
		INSERT INTO embeddings (id, user_id, journal_id, chunk_index, title, content, metadata, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = s.DB.Exec(query, doc.ID, userID, journalID, chunkIndex, title, doc.Content, metadataBytes, embeddingStr)
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

// HybridSearch performs combined keyword (BM25) and semantic (Vector) search
// This implementation uses Reciprocal Rank Fusion (RRF) implicitly by simple weighting or just retrieving both sets and deduping, 
// OR more advanced SQL that combines scores.
// Here we'll do a simple weighted score in SQL for efficiency.
//
// Combined Score = (Semantic Score * alpha) + (Keyword Score * beta)
// Semantic Score in pgvector with <=> is distance. Similarity is 1/(1+distance) or similar?
// Let's use 1 - (embedding <=> query) as similarity (approx).
// Keyword Score is ts_rank_cd using cover density.
func (s *PostgresStore) HybridSearch(query string, embedding Vector, k int, userID string) ([]Document, error) {
	embeddingBytes, err := json.Marshal(embedding)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query embedding: %w", err)
	}
	embeddingStr := string(embeddingBytes)

	// Expand query with synonyms
	expandedQuery := s.ExpandQuery(query)
	// Convert to tsquery format: "word1 | word2 | ..."
	tsQueryStr := strings.Join(strings.Fields(expandedQuery), " | ")

	sqlQuery := `
		WITH semantic_search AS (
			SELECT id, content, metadata, embedding,
				   1 - (embedding <=> $1) as semantic_score
			FROM embeddings
			WHERE user_id = $2
			ORDER BY embedding <=> $1
			LIMIT $3
		),
		keyword_search AS (
			SELECT id, content, metadata, embedding,
				   ts_rank_cd(content_tsv, to_tsquery('english', $4)) as keyword_score
			FROM embeddings
			WHERE user_id = $2
			  AND content_tsv @@ to_tsquery('english', $4)
			ORDER BY keyword_score DESC
			LIMIT $3
		)
		SELECT 
			COALESCE(s.id, k.id) as id,
			COALESCE(s.content, k.content) as content,
			COALESCE(s.metadata, k.metadata) as metadata,
			COALESCE(s.embedding, k.embedding) as embedding,
			(COALESCE(s.semantic_score, 0) * 0.7 + COALESCE(k.keyword_score, 0) * 0.3) as final_score
		FROM semantic_search s
		FULL OUTER JOIN keyword_search k ON s.id = k.id
		ORDER BY final_score DESC
		LIMIT $3
	`

	rows, err := s.DB.Query(sqlQuery, embeddingStr, userID, k, tsQueryStr)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		var metadataBytes []byte
		var embeddingStr string
		var score float64

		if err := rows.Scan(&doc.ID, &doc.Content, &metadataBytes, &embeddingStr, &score); err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}

		if err := json.Unmarshal(metadataBytes, &doc.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		
		// Add score to metadata for debugging/UI
		doc.Metadata["score"] = score

		if err := json.Unmarshal([]byte(embeddingStr), &doc.Embedding); err != nil {
			return nil, fmt.Errorf("failed to parse embedding: %w", err)
		}

		docs = append(docs, doc)
	}
	return docs, nil
}

// ExpandQuery expands the user query with synonyms
func (s *PostgresStore) ExpandQuery(query string) string {
	expansions := []string{query}
	lowerQuery := strings.ToLower(query)

	// Hardcoded synonyms for now (as per requirements)
	// In production this could be DB-backed or use an external service
	synonyms := map[string][]string{
		"travel": {"visit", "trip", "journey", "vacation"},
		"work":   {"job", "office", "career", "project"},
		"feel":   {"emotion", "mood", "sentiment"},
	}

	for word, syns := range synonyms {
		if strings.Contains(lowerQuery, word) {
			expansions = append(expansions, syns...)
		}
	}
	
	// Join all unique terms
	return strings.Join(expansions, " ")
}
func (s *PostgresStore) InitSchema() error {
	// Enable pgvector
	_, err := s.DB.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create table with enhanced schema
	// Using 768 dimensions for Gemini text-embedding-004
	query := `
		CREATE TABLE IF NOT EXISTS embeddings (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			journal_id UUID NOT NULL,
			chunk_index INT DEFAULT 0,
			title TEXT,
			content TEXT,
			metadata JSONB,
			embedding vector(768),
			created_at TIMESTAMP DEFAULT NOW()
		)
	`
	_, err = s.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create embeddings table: %w", err)
	}

	// Create indexes for fast filtering
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_embeddings_user_id ON embeddings(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_embeddings_journal_id ON embeddings(journal_id)",
		"CREATE INDEX IF NOT EXISTS idx_embeddings_user_journal ON embeddings(user_id, journal_id)",
	}

	for _, indexQuery := range indexes {
		_, err = s.DB.Exec(indexQuery)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Create HNSW index for performance
	// "vector_cosine_ops" is for <=>
	indexQuery := `
		CREATE INDEX IF NOT EXISTS embeddings_embedding_idx 
		ON embeddings 
		USING hnsw (embedding vector_cosine_ops)
	`
	_, err = s.DB.Exec(indexQuery)
	if err != nil {
		// HNSW index creation might fail if extension not fully set up
		// Log but don't fail - the table is still usable
		fmt.Printf("Warning: failed to create HNSW index: %v\n", err)
	}

	return nil
}
