-- Add play_count column to music table
ALTER TABLE music ADD COLUMN play_count INTEGER DEFAULT 0;

-- Add index for better performance on play_count queries
CREATE INDEX IF NOT EXISTS idx_music_play_count ON music(play_count DESC);
