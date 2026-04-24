-- Capability vector stored procedure for semantic matching
CREATE OR REPLACE FUNCTION vector_match_capability_4 (
  query_embedding vector(1024),
  similarity_threshold float,
  match_count int
)
RETURNS TABLE (
  doc_id bigint,
  subject text,
  similarity float
)
AS $$
BEGIN
  RETURN QUERY
  SELECT
    acv.cap_id as doc_id,
    acv.subject,
    (acv.embedding <=> query_embedding) as similarity
  FROM api_capability_vector acv
  WHERE (acv.embedding <=> query_embedding) < similarity_threshold
  ORDER BY acv.embedding <=> query_embedding
  LIMIT match_count;
END;
$$ LANGUAGE plpgsql;

-- IVFFlat index for vector search performance
CREATE INDEX IF NOT EXISTS idx_capability_vector_embedding
ON api_capability_vector
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);
