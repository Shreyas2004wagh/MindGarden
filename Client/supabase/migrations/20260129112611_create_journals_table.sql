/*
  # Create journals table for Mind Garden
  
  1. New Tables
    - `journals`
      - `id` (uuid, primary key) - unique identifier for each journal entry
      - `user_id` (uuid, foreign key) - references auth.users, owner of the journal
      - `title` (text, nullable) - optional title for the journal entry
      - `content` (text, not null) - main journal content
      - `created_at` (timestamptz) - when the entry was created
      - `updated_at` (timestamptz) - when the entry was last modified
  
  2. Security
    - Enable RLS on `journals` table
    - Add policy for users to read their own journal entries
    - Add policy for users to create their own journal entries
    - Add policy for users to update their own journal entries
    - Add policy for users to delete their own journal entries
  
  3. Indexes
    - Index on user_id for efficient queries
    - Index on created_at for chronological ordering
*/

-- Create journals table
CREATE TABLE IF NOT EXISTS journals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  title text,
  content text NOT NULL,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS journals_user_id_idx ON journals(user_id);
CREATE INDEX IF NOT EXISTS journals_created_at_idx ON journals(created_at DESC);

-- Enable Row Level Security
ALTER TABLE journals ENABLE ROW LEVEL SECURITY;

-- Policy: Users can read their own journal entries
CREATE POLICY "Users can view own journals"
  ON journals
  FOR SELECT
  TO authenticated
  USING (auth.uid() = user_id);

-- Policy: Users can create their own journal entries
CREATE POLICY "Users can create own journals"
  ON journals
  FOR INSERT
  TO authenticated
  WITH CHECK (auth.uid() = user_id);

-- Policy: Users can update their own journal entries
CREATE POLICY "Users can update own journals"
  ON journals
  FOR UPDATE
  TO authenticated
  USING (auth.uid() = user_id)
  WITH CHECK (auth.uid() = user_id);

-- Policy: Users can delete their own journal entries
CREATE POLICY "Users can delete own journals"
  ON journals
  FOR DELETE
  TO authenticated
  USING (auth.uid() = user_id);

-- Create function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update updated_at
DROP TRIGGER IF EXISTS update_journals_updated_at ON journals;
CREATE TRIGGER update_journals_updated_at
  BEFORE UPDATE ON journals
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();