
CREATE OR REPLACE FUNCTION qa_match_prompts (
  query_embedding vector(1536),
  similarity_threshold float,
  match_count int
)
RETURNS table (
  doc_id bigint,
  prompt text,
  similarity float
)
AS $$

BEGIN
  RETURN query
  SELECT
    qcp.doc_id,
    qcp.prompt,
    (qcp.embedding <=> query_embedding) as similarity
  FROM qa_corpus_prompt qcp
  WHERE (qcp.embedding <=> query_embedding) < similarity_threshold
  ORDER BY qcp.embedding <=> query_embedding
  LIMIT match_count;
END;

$$ LANGUAGE plpgsql;
