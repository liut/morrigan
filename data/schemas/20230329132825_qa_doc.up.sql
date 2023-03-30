

CREATE INDEX ON qa_corpus_document
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
