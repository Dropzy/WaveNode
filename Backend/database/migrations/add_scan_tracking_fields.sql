-- Add additional tracking fields to scan_status table
ALTER TABLE scan_status 
ADD COLUMN IF NOT EXISTS songs_added INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS songs_updated INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS tracks_skipped INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS duplicates INTEGER DEFAULT 0;

-- Create indexes for the new fields for better performance
CREATE INDEX IF NOT EXISTS idx_scan_status_songs_added ON scan_status(songs_added);
CREATE INDEX IF NOT EXISTS idx_scan_status_songs_updated ON scan_status(songs_updated);
CREATE INDEX IF NOT EXISTS idx_scan_status_tracks_skipped ON scan_status(tracks_skipped);
CREATE INDEX IF NOT EXISTS idx_scan_status_duplicates ON scan_status(duplicates);
