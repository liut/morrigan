ALTER TABLE convo_memory ADD COLUMN IF NOT EXISTS tier text NOT NULL DEFAULT 'working';
ALTER TABLE convo_memory ADD COLUMN IF NOT EXISTS importance_score float NOT NULL DEFAULT 0.5;
ALTER TABLE convo_memory ADD COLUMN IF NOT EXISTS last_accessed_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE convo_memory ADD COLUMN IF NOT EXISTS access_count int NOT NULL DEFAULT 0;
ALTER TABLE convo_memory ADD COLUMN IF NOT EXISTS decay_rate float NOT NULL DEFAULT 1.0;
