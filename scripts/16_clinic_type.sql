-- Add clinic type classification: normal or vaccination.
ALTER TABLE clinic_schedules
    ADD COLUMN IF NOT EXISTS clinic_type TEXT NOT NULL DEFAULT 'normal';

UPDATE clinic_schedules
SET clinic_type = 'normal'
WHERE clinic_type IS NULL OR TRIM(clinic_type) = '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'clinic_schedules_clinic_type_check'
    ) THEN
        ALTER TABLE clinic_schedules
            ADD CONSTRAINT clinic_schedules_clinic_type_check
            CHECK (clinic_type IN ('normal', 'vaccination'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_clinic_schedules_clinic_type ON clinic_schedules(clinic_type);

