-- Fix music table constraints to ensure proper duplicate handling
-- This migration addresses the issue where file_path has UNIQUE constraint
-- in setup_database.go but not in setup_database.sql

-- First, check if the unique constraint exists and remove it if necessary
DO $$
BEGIN
    -- Check if the unique constraint on file_path exists and drop it
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'music_file_path_key' 
        AND table_name = 'music'
    ) THEN
        ALTER TABLE music DROP CONSTRAINT music_file_path_key;
        RAISE NOTICE 'Dropped unique constraint on music.file_path';
    END IF;
    
    -- Check for any other unique constraint on file_path
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu 
             ON tc.constraint_name = kcu.constraint_name
        WHERE tc.table_name = 'music' 
        AND kcu.column_name = 'file_path'
        AND tc.constraint_type = 'UNIQUE'
    ) THEN
        -- Get the constraint name
        DECLARE 
            constraint_name text;
        BEGIN
            SELECT tc.constraint_name INTO constraint_name
            FROM information_schema.table_constraints tc
            JOIN information_schema.key_column_usage kcu 
                 ON tc.constraint_name = kcu.constraint_name
            WHERE tc.table_name = 'music' 
            AND kcu.column_name = 'file_path'
            AND tc.constraint_type = 'UNIQUE';
            
            EXECUTE 'ALTER TABLE music DROP CONSTRAINT ' || constraint_name;
            RAISE NOTICE 'Dropped unique constraint % on music.file_path', constraint_name;
        END;
    END IF;
END $$;

-- Add a composite unique constraint to prevent exact duplicates
-- This allows same file paths if they have different file names (for different versions)
-- but prevents exact duplicates
ALTER TABLE music 
ADD CONSTRAINT music_unique_file 
UNIQUE (file_path, file_name);

-- Add index for better duplicate detection performance
CREATE INDEX IF NOT EXISTS idx_music_file_path_file_name 
ON music(file_path, file_name);

-- Add index for title+artist duplicate detection
CREATE INDEX IF NOT EXISTS idx_music_title_artist 
ON music(title, artist);

-- Add comment to document the constraint change
COMMENT ON CONSTRAINT music_unique_file ON music IS 
'Prevents exact duplicate files but allows same path with different filenames for version management';

COMMENT ON TABLE music IS 
'Music tracks with improved duplicate handling - file_path is no longer uniquely constrained to allow updates';
