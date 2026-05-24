-- Violation time ranges for video moderation timeline UI.

CREATE TABLE IF NOT EXISTS violation_segments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id    UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    source      VARCHAR(10) NOT NULL,
    category    VARCHAR(20) NOT NULL,
    start_sec   DOUBLE PRECISION NOT NULL,
    end_sec     DOUBLE PRECISION NOT NULL,
    peak_score  DOUBLE PRECISION NOT NULL DEFAULT 0,
    evidence    TEXT,
    CONSTRAINT chk_violation_segments_source CHECK (source IN ('visual', 'audio')),
    CONSTRAINT chk_violation_segments_time CHECK (end_sec >= start_sec)
);

CREATE INDEX IF NOT EXISTS idx_violation_segments_video_start
    ON violation_segments (video_id, start_sec);
