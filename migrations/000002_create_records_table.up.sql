CREATE TABLE IF NOT EXISTS records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    bot_file_id TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    upload_file_id TEXT,
    task_id TEXT,
    download_file_id TEXT,
    content TEXT,
    content_tsv TSVECTOR GENERATED ALWAYS AS (to_tsvector('russian', content)) STORED,
    title TEXT,
    summary TEXT,
    raw JSONB,
    status SMALLINT NOT NULL DEFAULT 0,
    attempts SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_records_user_id ON records(user_id);
CREATE INDEX idx_records_status ON records(status);
CREATE INDEX idx_records_attempts_updated_at ON records(attempts, updated_at);
CREATE INDEX idx_records_content_tsv ON records USING GIN (content_tsv);
