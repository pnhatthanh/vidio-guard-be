-- Migrate audio_results from 3-class (offensive/hate) to 2-class (toxic).

ALTER TABLE audio_results ADD COLUMN IF NOT EXISTS toxic_score double precision NOT NULL DEFAULT 0;

UPDATE audio_results
SET toxic_score = GREATEST(COALESCE(offensive_score, 0), COALESCE(hate_score, 0))
WHERE toxic_score = 0 AND (offensive_score > 0 OR hate_score > 0);

ALTER TABLE audio_results DROP COLUMN IF EXISTS offensive_score;
ALTER TABLE audio_results DROP COLUMN IF EXISTS hate_score;
