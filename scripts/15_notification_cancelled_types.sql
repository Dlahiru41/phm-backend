-- Add cancelled clinic/vaccination notification categories.
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
        'missed_clinic',
        'cancelled_clinic',
        'cancelled_vaccination'
    ));

