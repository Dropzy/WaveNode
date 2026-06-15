-- Add image_url column to music table for track-specific artwork
ALTER TABLE music ADD COLUMN IF NOT EXISTS image_url TEXT;

-- Add index for better performance if needed
CREATE INDEX IF NOT EXISTS idx_music_image_url ON music(image_url) WHERE image_url IS NOT NULL;
