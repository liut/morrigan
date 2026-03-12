
ALTER TABLE IF EXISTS qa_corpus_document RENAME TO corpus_document;
ALTER TABLE IF EXISTS qa_corpus_vector_400 RENAME TO corpus_vector_400;


CREATE INDEX ON corpus_vector_400
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
