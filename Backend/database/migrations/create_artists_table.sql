-- Create artists table with comprehensive fields for Spotify integration
CREATE TABLE IF NOT EXISTS artists (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    spotify_id VARCHAR(255) UNIQUE,
    spotify_url TEXT,
    image_url TEXT,
    image_small_url TEXT,
    image_medium_url TEXT,
    image_large_url TEXT,
    followers INTEGER DEFAULT 0,
    popularity INTEGER DEFAULT 0,
    genres JSONB,
    biography TEXT,
    country VARCHAR(2),
    external_urls JSONB,
    uri TEXT,
    href TEXT,
    type VARCHAR(50) DEFAULT 'artist',
    api_data JSONB, -- Store raw Spotify API response
    last_enriched_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_artists_name ON artists(name);
CREATE INDEX IF NOT EXISTS idx_artists_spotify_id ON artists(spotify_id);
CREATE INDEX IF NOT EXISTS idx_artists_popularity ON artists(popularity DESC);
CREATE INDEX IF NOT EXISTS idx_artists_last_enriched ON artists(last_enriched_at);

-- Add artist_id foreign key to music table
ALTER TABLE music ADD COLUMN IF NOT EXISTS artist_id VARCHAR(255);

-- Create index for the new foreign key
CREATE INDEX IF NOT EXISTS idx_music_artist_id ON music(artist_id);

-- Migration script to populate artists table from existing music data
-- This will be run only if there are existing music records
INSERT INTO artists (id, name, created_at, updated_at)
SELECT 
    DISTINCT 'artist_' || MD5(artist) as id,
    artist as name,
    CURRENT_TIMESTAMP as created_at,
    CURRENT_TIMESTAMP as updated_at
FROM music 
WHERE artist IS NOT NULL AND artist != ''
AND NOT EXISTS (
    SELECT 1 FROM artists WHERE artists.name = music.artist
);

-- Update music table to reference artists
UPDATE music 
SET artist_id = (SELECT id FROM artists WHERE artists.name = music.artist)
WHERE artist_id IS NULL AND artist IS NOT NULL AND artist != '';

-- Remove the artist_image_url column from music table since it's now in artists table
ALTER TABLE music DROP COLUMN IF EXISTS artist_image_url;
