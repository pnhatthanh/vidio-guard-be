-- Fix final_verdicts FK created by GORM without ON DELETE CASCADE (fk_videos_final_verdict).

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT c.conname
        FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        JOIN pg_class ref ON c.confrelid = ref.oid
        WHERE t.relname = 'final_verdicts'
          AND ref.relname = 'videos'
          AND c.contype = 'f'
    LOOP
        EXECUTE format('ALTER TABLE final_verdicts DROP CONSTRAINT %I', r.conname);
    END LOOP;
END $$;

ALTER TABLE final_verdicts
    ADD CONSTRAINT final_verdicts_video_id_fkey
    FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE ON UPDATE CASCADE;
