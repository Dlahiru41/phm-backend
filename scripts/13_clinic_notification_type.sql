-- Add clinic_reminder notification type
ALTER TABLE notifications DROP CONSTRAINT notifications_type_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_type_check
    CHECK (type IN ('reminder', 'missed', 'upcoming', 'info', 'vaccination_due', 'growth_record', 'clinic_reminder'));

