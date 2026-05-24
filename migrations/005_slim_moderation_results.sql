-- Slim storage: transcript on final_verdicts; drop per-frame/per-audio aggregate tables.

ALTER TABLE final_verdicts
    ADD COLUMN IF NOT EXISTS transcript TEXT NOT NULL DEFAULT '';

DROP TABLE IF EXISTS frame_results;
DROP TABLE IF EXISTS audio_results;
