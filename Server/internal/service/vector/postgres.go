package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	Pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{Pool: pool}
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

	// Handle chunk_index safely
	var chunkIndex int
	if ci, ok := doc.Metadata["chunk_index"].(int); ok {
		chunkIndex = ci
	} else if ci, ok := doc.Metadata["chunk_index"].(float64); ok {
		chunkIndex = int(ci)
	}

	title, _ := doc.Metadata["title"].(string)

	query := `
		INSERT INTO embeddings (id, user_id, journal_id, chunk_index, title, content, metadata, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = s.Pool.Exec(context.Background(), query, doc.ID, userID, journalID, chunkIndex, title, doc.Content, metadataBytes, embeddingStr)
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

	sqlQuery := `
		SELECT id, content, metadata, embedding
		FROM embeddings
		ORDER BY embedding <=> $1
		LIMIT $2
	`

	rows, err := s.Pool.Query(context.Background(), sqlQuery, embeddingStr, k)
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

		if err := json.Unmarshal([]byte(embeddingStr), &doc.Embedding); err != nil {
			return nil, fmt.Errorf("failed to parse embedding from db: %w", err)
		}

		docs = append(docs, doc)
	}
	return docs, nil
}

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

	rows, err := s.Pool.Query(context.Background(), sqlQuery, embeddingStr, userID, k, tsQueryStr)
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

func (s *PostgresStore) ExpandQuery(query string) string {
	expansions := []string{query}
	lowerQuery := strings.ToLower(query)

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
	ctx := context.Background()
	// Enable pgvector
	_, err := s.Pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create table with enhanced schema
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
			created_at TIMESTAMP DEFAULT NOW(),
			content_tsv TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', content)) STORED
		)
	`
	_, err = s.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create embeddings table: %w", err)
	}

	// Clean up legacy trigger/function introduced by earlier migrations.
	// We use generated column `content_tsv`, so trigger maintenance is unnecessary.
	cleanupQueries := []string{
		"DROP TRIGGER IF EXISTS embeddings_content_tsv_update ON embeddings",
	}
	for _, cleanupQuery := range cleanupQueries {
		if _, err = s.Pool.Exec(ctx, cleanupQuery); err != nil {
			return fmt.Errorf("failed schema cleanup query: %w", err)
		}
	}

	// Normalize `content_tsv` so existing environments with a plain TSVECTOR column
	// (or incompatible trigger setup) are migrated to the generated-column design.
	_, err = s.Pool.Exec(ctx, `
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'embeddings'
				  AND column_name = 'content_tsv'
				  AND is_generated <> 'ALWAYS'
			) THEN
				ALTER TABLE embeddings DROP COLUMN content_tsv;
			END IF;

			IF NOT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'embeddings'
				  AND column_name = 'content_tsv'
			) THEN
				ALTER TABLE embeddings
				ADD COLUMN content_tsv TSVECTOR
				GENERATED ALWAYS AS (to_tsvector('english', COALESCE(content, ''))) STORED;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("failed to normalize content_tsv column: %w", err)
	}

	// Create indexes for fast filtering
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_embeddings_user_id ON embeddings(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_embeddings_journal_id ON embeddings(journal_id)",
		"CREATE INDEX IF NOT EXISTS idx_embeddings_user_journal ON embeddings(user_id, journal_id)",
		"CREATE INDEX IF NOT EXISTS idx_embeddings_content_tsv ON embeddings USING GIN(content_tsv)",
	}

	for _, indexQuery := range indexes {
		_, err = s.Pool.Exec(ctx, indexQuery)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Create HNSW index for performance
	indexQuery := `
		CREATE INDEX IF NOT EXISTS embeddings_embedding_idx 
		ON embeddings 
		USING hnsw (embedding vector_cosine_ops)
	`
	_, err = s.Pool.Exec(ctx, indexQuery)
	if err != nil {
		fmt.Printf("Warning: failed to create HNSW index: %v\n", err)
	}

	return nil
}
