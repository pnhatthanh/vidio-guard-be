-- Risk-scoring verdict: replace peak/flagged columns with modal scores.

ALTER TABLE final_verdicts
    DROP COLUMN IF EXISTS peak_violence_score,
    DROP COLUMN IF EXISTS peak_nsfw_score,
    DROP COLUMN IF EXISTS flagged_frames_count,
    DROP COLUMN IF EXISTS flagged_timestamps;

ALTER TABLE final_verdicts
    ADD COLUMN IF NOT EXISTS frame_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS audio_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS final_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_frames INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS video_duration_sec DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS hard_rule_triggered BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS hard_rule_reason TEXT NOT NULL DEFAULT '';

-- Backfill final_score from legacy risk_score when present.
UPDATE final_verdicts
SET final_score = risk_score
WHERE final_score = 0 AND risk_score > 0;

-- Map legacy categorical verdicts to safe | warning | violation.
UPDATE final_verdicts
SET verdict = CASE
    WHEN verdict IN ('nsfw', 'violence') THEN 'violation'
    WHEN verdict = 'safe' THEN 'safe'
    ELSE verdict
END
WHERE verdict IN ('nsfw', 'violence');
