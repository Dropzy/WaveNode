-- Add missing columns to scan_status table
ALTER TABLE scan_status ADD COLUMN IF NOT EXISTS songs_added INTEGER DEFAULT 0;
ALTER TABLE scan_status ADD COLUMN IF NOT EXISTS duplicates INTEGER DEFAULT 0;
