-- Migration script to update embeddings table schema
-- Run this in your Supabase SQL Editor

-- Step 1: Drop the old table (this will delete existing embeddings)
DROP TABLE IF EXISTS embeddings CASCADE;

-- Step 2: Create the new table with enhanced schema
CREATE TABLE embeddings (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    journal_id UUID NOT NULL,
    chunk_index INT DEFAULT 0,
    title TEXT,
    content TEXT,
    metadata JSONB,
    embedding vector(768),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Step 3: Create indexes for fast filtering
CREATE INDEX idx_embeddings_user_id ON embeddings(user_id);
CREATE INDEX idx_embeddings_journal_id ON embeddings(journal_id);
CREATE INDEX idx_embeddings_user_journal ON embeddings(user_id, journal_id);

-- Step 4: Create HNSW index for fast vector search
CREATE INDEX embeddings_embedding_idx ON embeddings USING hnsw (embedding vector_cosine_ops);

-- Verify the table was created
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'embeddings'
ORDER BY ordinal_position;
