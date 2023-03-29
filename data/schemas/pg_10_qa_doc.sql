
CREATE OR REPLACE FUNCTION qa_match_documents (
  query_embedding vector(1536),
  similarity_threshold float,
  match_count int
)
RETURNS table (
  id bigint,
  title text,
  heading text,
  content text,
  similarity float
)
language plpgsql
AS $$
BEGIN
  RETURN query
  SELECT
    qcd.id,
    qcd.title,
    qcd.heading,
    qcd.content,
    (qcd.embedding <=> query_embedding) as similarity
  FROM qa_corpus_document qcd
  WHERE (qcd.embedding <=> query_embedding) < similarity_threshold
  ORDER BY qcd.embedding <=> query_embedding
  LIMIT match_count;
end;
$$;
