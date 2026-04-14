-- ClinicSchedule table for clinic scheduling and due child identification
CREATE TABLE clinic_schedules (
    id              TEXT PRIMARY KEY,
    phm_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    clinic_date     DATE NOT NULL,
    gn_division     TEXT NOT NULL,
    location        TEXT NOT NULL,
    description     TEXT,
    status          TEXT NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'completed', 'cancelled')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Junction table for clinic-child mapping (for attendance tracking and reporting)
CREATE TABLE clinic_children (
    id              TEXT PRIMARY KEY,
    clinic_id       TEXT NOT NULL REFERENCES clinic_schedules(id) ON DELETE CASCADE,
    child_id        TEXT NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    attended        BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(clinic_id, child_id)
);

-- Index for clinic queries
CREATE INDEX idx_clinic_schedules_phm_id ON clinic_schedules(phm_id);
CREATE INDEX idx_clinic_schedules_date ON clinic_schedules(clinic_date);
CREATE INDEX idx_clinic_schedules_gn_division ON clinic_schedules(gn_division);
CREATE INDEX idx_clinic_children_clinic_id ON clinic_children(clinic_id);
CREATE INDEX idx_clinic_children_child_id ON clinic_children(child_id);

-- Trigger for clinic_schedules updated_at
CREATE TRIGGER clinic_schedules_updated_at
    BEFORE UPDATE ON clinic_schedules FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

-- Trigger for clinic_children updated_at
CREATE TRIGGER clinic_children_updated_at
    BEFORE UPDATE ON clinic_children FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

