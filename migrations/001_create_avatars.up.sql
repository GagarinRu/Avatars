CREATE TABLE IF NOT EXISTS avatars (
    user_id TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    staging_key TEXT NOT NULL DEFAULT '',
    original_key TEXT NOT NULL DEFAULT '',
    thumbnail_key TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    width INT NOT NULL DEFAULT 0,
    height INT NOT NULL DEFAULT 0,
    error_message TEXT,
    message_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_avatars_status ON avatars(status);
