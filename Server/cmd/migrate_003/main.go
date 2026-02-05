package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
    "strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
    // Load .env file
    if err := godotenv.Load(".env"); err != nil {
        log.Println("Warning: Error loading .env file")
    }

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set in .env")
	}

    // Supabase requires SSL
    if !strings.Contains(dbURL, "sslmode=") {
        if strings.Contains(dbURL, "?") {
            dbURL += "&sslmode=require"
        } else {
            dbURL += "?sslmode=require"
        }
    }

    fmt.Println("Connecting to DB...")
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
    fmt.Println("Connected!")

	statements := []string{
        // 1. Add content_tsv column
        `ALTER TABLE embeddings ADD COLUMN IF NOT EXISTS content_tsv tsvector;`,
        
        // 2. Create GIN index
        `CREATE INDEX IF NOT EXISTS idx_content_fts ON embeddings USING gin(content_tsv);`,

        // 3. Create function
        `CREATE OR REPLACE FUNCTION tsvector_update_trigger() RETURNS trigger AS $$
        BEGIN
          NEW.content_tsv := to_tsvector('pg_catalog.english', COALESCE(NEW.content, ''));
          RETURN NEW;
        END
        $$ LANGUAGE plpgsql;`,

        // 4. Create trigger
        `DROP TRIGGER IF EXISTS embeddings_content_tsv_update ON embeddings;`,
        `CREATE TRIGGER embeddings_content_tsv_update
        BEFORE INSERT OR UPDATE ON embeddings
        FOR EACH ROW EXECUTE FUNCTION
        tsvector_update_trigger();`,

        // 5. Create search_feedback table
        `CREATE TABLE IF NOT EXISTS search_feedback (
            id UUID PRIMARY KEY,
            user_id UUID NOT NULL,
            query TEXT NOT NULL,
            result_journal_id UUID NOT NULL,
            similarity_score FLOAT,
            was_helpful BOOLEAN,
            created_at TIMESTAMP DEFAULT NOW()
        );`,

        // 6. Backfill
        `UPDATE embeddings SET content_tsv = to_tsvector('pg_catalog.english', COALESCE(content, '')) WHERE content_tsv IS NULL;`,
    }

    for i, stmt := range statements {
        fmt.Printf("Executing step %d...\n", i+1)
        _, err = db.Exec(stmt)
        if err != nil {
            log.Fatalf("Migration failed at step %d: %v\nStatement: %s", i+1, err, stmt)
        }
    }

	fmt.Println("Migration completed successfully!")
}
