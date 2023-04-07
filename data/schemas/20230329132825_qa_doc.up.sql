

CREATE INDEX ON qa_corpus_document
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

ALTER TABLE IF EXISTS qa_corpus_document ADD IF NOT EXISTS qas JSONB NOT NULL DEFAULT '[]';
