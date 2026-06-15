-- Migration to fix artist_id relationships between music and artists tables

-- Create a temporary mapping of artist names to their correct IDs
CREATE TEMPORARY TABLE artist_name_mapping AS
SELECT 
    name,
    MIN(id) as correct_id
FROM artists 
WHERE name IS NOT NULL AND name != ''
GROUP BY name;

-- Update music table to link to correct artist IDs based on artist name
UPDATE music m
SET artist_id = am.correct_id
FROM artist_name_mapping am
WHERE m.artist = am.name 
AND (m.artist_id IS NULL OR m.artist_id = '');

-- For any remaining tracks where artist_id is still null but artist exists,
-- create new artist records with generated hashes
INSERT INTO artists (id, name, created_at, updated_at)
SELECT 
    DISTINCT
    CASE 
        WHEN LENGTH(md5(LOWER(TRIM(m.artist)))) >= 8 THEN SUBSTRING(md5(LOWER(TRIM(m.artist))), 1, 8)
        ELSE 'artist_' || SUBSTRING(md5(LOWER(TRIM(m.artist))), 1, 8)
    END,
    TRIM(m.artist),
    NOW(),
    NOW()
FROM music m
WHERE m.artist IS NOT NULL 
AND m.artist != ''
AND m.artist_id IS NULL
AND NOT EXISTS (
    SELECT 1 FROM artists a WHERE a.name = TRIM(m.artist)
);

-- Update the newly created artist IDs
UPDATE music m
SET artist_id = a.id
FROM artists a
WHERE m.artist = a.name 
AND m.artist_id IS NULL;

-- Clean up temporary table
DROP TABLE artist_name_mapping;

-- Create an index on artist_id for better performance
CREATE INDEX IF NOT EXISTS idx_music_artist_id ON music(artist_id);

-- Create an index on artists.name for faster lookups
CREATE INDEX IF NOT EXISTS idx_artists_name ON artists(name);
