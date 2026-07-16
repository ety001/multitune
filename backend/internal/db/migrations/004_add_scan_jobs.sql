CREATE TABLE IF NOT EXISTS scan_jobs (
    id TEXT PRIMARY KEY,
    playlist_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    total INTEGER NOT NULL DEFAULT 0,
    current INTEGER NOT NULL DEFAULT 0,
    added INTEGER NOT NULL DEFAULT 0,
    updated INTEGER NOT NULL DEFAULT 0,
    message TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_scan_jobs_status ON scan_jobs(status);
