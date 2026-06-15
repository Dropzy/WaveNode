-- Create scan_status table for tracking enrichment and scan operations
CREATE TABLE IF NOT EXISTS scan_status (
    id VARCHAR(255) PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    progress INTEGER DEFAULT 0,
    total_files INTEGER DEFAULT 0,
    processed INTEGER DEFAULT 0,
    current_file TEXT,
    errors JSONB DEFAULT '[]',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_scan_status_started_at ON scan_status(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_scan_status_type ON scan_status(type);
CREATE INDEX IF NOT EXISTS idx_scan_status_status ON scan_status(status);
