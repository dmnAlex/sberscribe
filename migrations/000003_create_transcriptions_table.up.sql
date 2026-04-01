CREATE TYPE transcription_status AS ENUM (
    'UNDEFINED',
    'NEW',
    'RUNNING',
    'CANCELED',
    'DONE',
    'ERROR'
);

CREATE TABLE IF NOT EXISTS transcriptions (
    meeting_id BIGINT PRIMARY KEY REFERENCES meetings(id) ON DELETE CASCADE,
    request_file_id TEXT NOT NULL,
    response_file_id TEXT,
    content TEXT,
    raw JSONB,
    status transcription_status NOT NULL DEFAULT 'UNDEFINED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transcriptions_meeting_id ON transcriptions(meeting_id);
CREATE INDEX idx_transcriptions_content_fts ON transcriptions USING GIN (to_tsvector('russian', content)); 