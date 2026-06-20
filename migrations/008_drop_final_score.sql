-- risk_score is the single fused score; final_score was redundant.

UPDATE final_verdicts
SET risk_score = final_score
WHERE risk_score = 0 AND final_score > 0;

ALTER TABLE final_verdicts
    DROP COLUMN IF EXISTS final_score;
