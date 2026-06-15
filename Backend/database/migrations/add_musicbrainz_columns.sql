-- Add MusicBrainz columns to artists table
ALTER TABLE artists ADD COLUMN IF NOT EXISTS musicbrainz_id VARCHAR(255) UNIQUE;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS musicbrainz_url TEXT;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS tags JSONB;

-- Update spotify_url column to be nullable (it was already in the original schema)
-- This column exists but our Go code doesn't use it anymore

-- Create index for MusicBrainz ID
CREATE INDEX IF NOT EXISTS idx_artists_musicbrainz_id ON artists(musicbrainz_id);
