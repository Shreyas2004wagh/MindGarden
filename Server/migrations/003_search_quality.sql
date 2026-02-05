-- Migration for Search Quality Improvements
-- Adds Full Text Search capabilities and Relevance Feedback tracking

-- 1. Add content_tsv column for Full Text Search
ALTER TABLE embeddings ADD COLUMN IF NOT EXISTS content_tsv tsvector;

-- 2. Create GIN index for fast text search
CREATE INDEX IF NOT EXISTS idx_content_fts ON embeddings USING gin(content_tsv);

-- 3. Create function to update tsvector automatically
CREATE OR REPLACE FUNCTION tsvector_update_trigger() RETURNS trigger AS $$
BEGIN
  NEW.content_tsv := to_tsvector('pg_catalog.english', NEW.content);
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- 4. Create trigger to update content_tsv on insert or update
DROP TRIGGER IF EXISTS embeddings_content_tsv_update ON embeddings;
CREATE TRIGGER embeddings_content_tsv_update
BEFORE INSERT OR UPDATE ON embeddings
FOR EACH ROW EXECUTE FUNCTION
tsvector_update_trigger();

-- 5. Create search_feedback table
CREATE TABLE IF NOT EXISTS search_feedback (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    query TEXT NOT NULL,
    result_journal_id UUID NOT NULL,
    similarity_score FLOAT,
    was_helpful BOOLEAN,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 6. Backfill existing data
UPDATE embeddings SET content_tsv = to_tsvector('pg_catalog.english', content) WHERE content_tsv IS NULL;
