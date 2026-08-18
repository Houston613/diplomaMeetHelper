-- 000002_add_fts.up.sql

CREATE TABLE IF NOT EXISTS meethelper.qa_history (
    id UUID PRIMARY KEY,
    meeting_id UUID NOT NULL REFERENCES meethelper.meetings(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL REFERENCES meethelper.users(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_qa_meeting_id ON meethelper.qa_history(meeting_id);
CREATE INDEX IF NOT EXISTS idx_qa_user_id ON meethelper.qa_history(user_id);

ALTER TABLE meethelper.transcriptions 
ADD COLUMN IF NOT EXISTS tsv tsvector GENERATED ALWAYS AS (to_tsvector('russian', content)) STORED;

ALTER TABLE meethelper.summaries 
ADD COLUMN IF NOT EXISTS tsv tsvector GENERATED ALWAYS AS (to_tsvector('russian', content)) STORED;

CREATE INDEX IF NOT EXISTS idx_transcriptions_tsv ON meethelper.transcriptions USING GIN(tsv);
CREATE INDEX IF NOT EXISTS idx_summaries_tsv ON meethelper.summaries USING GIN(tsv);
