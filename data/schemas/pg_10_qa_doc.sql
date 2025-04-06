
-- Note: Different providers have varying dimensions for Vector settings.
-- 4=400=1024 (bge-m3), 6=600=1536 (openai embedding)

CREATE OR REPLACE FUNCTION qa_match_docs_4 (
  query_embedding vector(1024),
  similarity_threshold float,
  match_count int
)
RETURNS table (
  doc_id bigint,
  subject text,
  similarity float
)
AS $$

BEGIN
  RETURN query
  SELECT
    qcv.doc_id,
    qcv.subject,
    (qcv.embedding <=> query_embedding) as similarity
  FROM qa_corpus_vector_400 qcv
  WHERE (qcv.embedding <=> query_embedding) < similarity_threshold
  ORDER BY qcv.embedding <=> query_embedding
  LIMIT match_count;
END;

$$ LANGUAGE plpgsql;
