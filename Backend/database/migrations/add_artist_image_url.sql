-- Add artist_image_url column to music table for Spotify enrichment
-- This migration adds support for storing high-quality artist images fetched from Spotify

-- Add the artist_image_url column to store Spotify artist image URLs
ALTER TABLE music ADD COLUMN IF NOT EXISTS artist_image_url TEXT;

-- Add index for better performance on artist queries
CREATE INDEX IF NOT EXISTS idx_music_artist_image_url ON music(artist_image_url);

-- Update the updated_at timestamp for existing records
UPDATE music SET updated_at = CURRENT_TIMESTAMP WHERE artist_image_url IS NOT NULL;

-- Add comment to document the new column
COMMENT ON COLUMN music.artist_image_url IS 'URL to high-quality artist image fetched from Spotify API';
