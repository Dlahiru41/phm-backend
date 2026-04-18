-- Enhancements for clinic/vaccination attendance tracking and missed notifications

-- Vaccination schedules: track whether missed alerts were sent.
ALTER TABLE vaccination_schedules
    ADD COLUMN IF NOT EXISTS missed_notified BOOLEAN NOT NULL DEFAULT false;

-- Clinic children: explicit attendance state and missed alert tracking.
ALTER TABLE clinic_children
    ADD COLUMN IF NOT EXISTS attendance_status TEXT NOT NULL DEFAULT 'pending';

ALTER TABLE clinic_children
    ADD COLUMN IF NOT EXISTS missed_notified BOOLEAN NOT NULL DEFAULT false;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'clinic_children_attendance_status_check'
    ) THEN
        ALTER TABLE clinic_children
            ADD CONSTRAINT clinic_children_attendance_status_check
            CHECK (attendance_status IN ('pending', 'attended', 'not_attended', 'missed'));
    END IF;
END $$;

-- Bring old rows into an explicit status.
UPDATE clinic_children
SET attendance_status = CASE
    WHEN attended THEN 'attended'
    ELSE 'pending'
END
WHERE attendance_status IS NULL OR attendance_status = '';

-- Notifications: support missed event types.
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_type_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_type_check
    CHECK (type IN (
        'reminder',
        'missed',
        'upcoming',
        'info',
        'vaccination_due',
        'growth_record',
        'clinic_reminder',
        'missed_vaccination',
        'missed_clinic'
    ));

