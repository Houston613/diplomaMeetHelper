-- 000002_add_fts.down.sql

DROP INDEX IF EXISTS meethelper.idx_summaries_tsv;
DROP INDEX IF EXISTS meethelper.idx_transcriptions_tsv;

ALTER TABLE meethelper.summaries DROP COLUMN IF EXISTS tsv;
ALTER TABLE meethelper.transcriptions DROP COLUMN IF EXISTS tsv;

DROP TABLE IF EXISTS meethelper.qa_history;
