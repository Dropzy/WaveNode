-- Add cover art columns to the music table
ALTER TABLE music ADD COLUMN IF NOT EXISTS cover_art_url TEXT;
ALTER TABLE music ADD COLUMN IF NOT EXISTS cover_art_small_url TEXT;
ALTER TABLE music ADD COLUMN IF NOT EXISTS cover_art_medium_url TEXT;
ALTER TABLE music ADD COLUMN IF NOT EXISTS cover_art_large_url TEXT;
ALTER TABLE music ADD COLUMN IF NOT EXISTS cover_art_source VARCHAR(50) DEFAULT 'embedded';
ALTER TABLE music ADD COLUMN IF NOT EXISTS last_cover_art_enriched_at TIMESTAMP;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_music_cover_art_source ON music(cover_art_source);
CREATE INDEX IF NOT EXISTS idx_music_last_cover_art_enriched ON music(last_cover_art_enriched_at);
