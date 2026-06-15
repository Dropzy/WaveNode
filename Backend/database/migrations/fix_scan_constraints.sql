-- Fix scan constraints - run this with: psql -d music_server -f database/migrations/fix_scan_constraints.sql

-- Step 1: Drop existing file_path unique constraint if it exists
DO $$
BEGIN
    -- Check if music_file_path_key constraint exists and drop it
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'music_file_path_key' 
        AND table_name = 'music'
    ) THEN
        ALTER TABLE music DROP CONSTRAINT music_file_path_key;
        RAISE NOTICE 'Dropped music_file_path_key constraint';
    END IF;
    
    -- Check for any other unique constraints on file_path
    FOR constraint_name IN (
        SELECT tc.constraint_name
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu 
             ON tc.constraint_name = kcu.constraint_name
        WHERE tc.table_name = 'music' 
        AND kcu.column_name = 'file_path'
        AND tc.constraint_type = 'UNIQUE'
    ) LOOP
        EXECUTE 'ALTER TABLE music DROP CONSTRAINT ' || constraint_name;
        RAISE NOTICE 'Dropped constraint % on music.file_path', constraint_name;
    END LOOP;
END $$;

-- Step 2: Add composite unique constraint to prevent exact duplicates
-- This allows same file paths if they have different file names (for different versions)
-- but prevents exact duplicates
ALTER TABLE music 
ADD CONSTRAINT IF NOT EXISTS music_unique_file 
UNIQUE (file_path, file_name);

-- Step 3: Add performance indexes
CREATE INDEX IF NOT EXISTS idx_music_file_path_file_name 
ON music(file_path, file_name);

CREATE INDEX IF NOT EXISTS idx_music_title_artist 
ON music(title, artist);

-- Step 4: Add comments to document the changes
COMMENT ON CONSTRAINT music_unique_file ON music IS 
'Prevents exact duplicate files but allows same path with different filenames for version management';

COMMENT ON TABLE music IS 
'Music tracks with improved duplicate handling - file_path is no longer uniquely constrained to allow updates';

-- Step 5: Show results
DO $$
BEGIN
    RAISE NOTICE '✓ Migration completed successfully';
    RAISE NOTICE 'The following changes have been applied:';
    RAISE NOTICE '1. Removed problematic UNIQUE constraint on file_path';
    RAISE NOTICE '2. Added composite UNIQUE constraint on (file_path, file_name)';
    RAISE NOTICE '3. Added performance indexes for duplicate detection';
    RAISE NOTICE '4. Added documentation comments';
    RAISE NOTICE '';
    RAISE NOTICE 'Your library scan should now work correctly and handle duplicates properly.';
END $$;
