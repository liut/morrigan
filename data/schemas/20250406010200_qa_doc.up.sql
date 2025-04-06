

ALTER TABLE IF EXISTS qa_corpus_document DROP COLUMN IF EXISTS qas;

CREATE INDEX ON qa_corpus_vector_400
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
