

ALTER TABLE IF EXISTS qa_corpus_document DROP IF EXISTS tokens;
ALTER TABLE IF EXISTS qa_corpus_document DROP IF EXISTS embedding;

CREATE INDEX ON qa_corpus_prompt
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
