
ALTER TABLE qa_corpus_document ALTER title TYPE text;
ALTER TABLE qa_corpus_document ALTER heading TYPE text;
ALTER TABLE qa_corpus_document ALTER content TYPE text;

CREATE INDEX ON qa_corpus_document
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
