-- SuwaCare LK / NCVMS - PostgreSQL Schema
-- Run as postgres superuser or a dedicated app user with CREATE privileges.
-- Database: create with: CREATE DATABASE ncvms;

SET client_encoding = 'UTF8';

-- Extensions (optional, for UUIDs and crypto)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id              TEXT PRIMARY KEY,
    email           TEXT NOT NULL UNIQUE,
    nic             TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL CHECK (role IN ('parent', 'phm', 'moh')),
    name            TEXT NOT NULL,
    phone_number    TEXT,
    address         TEXT,
    language_preference TEXT DEFAULT 'en',
    notification_settings JSONB DEFAULT '{"email": true, "sms": false, "push": true}',
    area_code       TEXT,
    employee_id     TEXT UNIQUE,
    assigned_area   TEXT,
    first_login     BOOLEAN DEFAULT false,
    created_by_moh  TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE password_reset_tokens (
    token       TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Children
-- ---------------------------------------------------------------------------
CREATE TABLE children (
    id                  TEXT PRIMARY KEY,
    registration_number TEXT NOT NULL UNIQUE,
    first_name          TEXT NOT NULL,
    last_name           TEXT NOT NULL,
    date_of_birth       DATE NOT NULL,
    gender              TEXT NOT NULL CHECK (gender IN ('male', 'female', 'other')),
    blood_group         TEXT,
    birth_weight        NUMERIC(5,2),
    birth_height        NUMERIC(5,2),
    head_circumference  NUMERIC(4,2),
    mother_name         TEXT,
    mother_nic          TEXT,
    father_name         TEXT,
    father_nic          TEXT,
    parent_id           TEXT REFERENCES users(id) ON DELETE SET NULL,
    registered_by       TEXT REFERENCES users(id) ON DELETE SET NULL,
    district            TEXT,
    ds_division         TEXT,
    gn_division         TEXT,
    address             TEXT,
    area_code           TEXT,
    parent_whatsapp_number TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE child_link_otps (
    id              TEXT PRIMARY KEY,
    child_id        TEXT NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    parent_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    otp_hash        TEXT NOT NULL,
    attempt_count   INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 5,
    expires_at      TIMESTAMPTZ NOT NULL,
    consumed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Vaccines (master data)
-- ---------------------------------------------------------------------------
CREATE TABLE vaccines (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    manufacturer      TEXT,
    dosage_info       TEXT,
    recommended_age   INT NOT NULL DEFAULT 0,
    interval_days     INT NOT NULL DEFAULT 0,
    description       TEXT,
    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Vaccination records
-- ---------------------------------------------------------------------------
CREATE TABLE vaccination_records (
    id                TEXT PRIMARY KEY,
    child_id           TEXT NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    vaccine_id         TEXT NOT NULL REFERENCES vaccines(id) ON DELETE RESTRICT,
    administered_date  DATE NOT NULL,
    batch_number       TEXT,
    administered_by    TEXT,
    location           TEXT,
    site               TEXT,
    dose_number        INT,
    next_due_date      DATE,
    status             TEXT NOT NULL DEFAULT 'administered' CHECK (status IN ('pending', 'administered', 'missed', 'cancelled')),
    notes              TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Vaccination schedule
-- ---------------------------------------------------------------------------
CREATE TABLE vaccination_schedules (
    id             TEXT PRIMARY KEY,
    child_id       TEXT NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    vaccine_id      TEXT NOT NULL REFERENCES vaccines(id) ON DELETE RESTRICT,
    scheduled_date  DATE NOT NULL,
    due_date       DATE NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'scheduled', 'completed', 'missed', 'cancelled')),
    reminder_sent   BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Growth records
-- ---------------------------------------------------------------------------
CREATE TABLE growth_records (
    id                 TEXT PRIMARY KEY,
    child_id            TEXT NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    recorded_date       DATE NOT NULL,
    height              NUMERIC(5,2),
    weight              NUMERIC(5,2),
    head_circumference  NUMERIC(4,2),
    recorded_by         TEXT,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Notifications
-- ---------------------------------------------------------------------------
CREATE TABLE notifications (
    id                TEXT PRIMARY KEY,
    recipient_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type               TEXT NOT NULL CHECK (type IN ('reminder', 'missed', 'upcoming', 'info', 'vaccination_due', 'growth_record')),
    message            TEXT NOT NULL,
    related_child_id   TEXT REFERENCES children(id) ON DELETE SET NULL,
    sent_date          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_read            BOOLEAN NOT NULL DEFAULT false,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Reports (metadata; files stored on disk or object storage)
-- ---------------------------------------------------------------------------
CREATE TABLE reports (
    id              TEXT PRIMARY KEY,
    report_type     TEXT NOT NULL CHECK (report_type IN ('vaccination_coverage', 'area_performance', 'audit_report', 'growth_analysis', 'monthly_summary')),
    generated_by    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    generated_date  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    start_date      DATE,
    end_date        DATE,
    format          TEXT NOT NULL CHECK (format IN ('pdf', 'excel', 'csv')),
    file_path       TEXT,
    filter_params   JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Audit logs
-- ---------------------------------------------------------------------------
CREATE TABLE audit_logs (
    id          TEXT PRIMARY KEY,
    user_id      TEXT REFERENCES users(id) ON DELETE SET NULL,
    user_role    TEXT,
    user_name    TEXT,
    action       TEXT NOT NULL CHECK (action IN ('CREATE', 'UPDATE', 'DELETE', 'VIEW', 'LOGIN', 'LOGOUT')),
    entity_type  TEXT CHECK (entity_type IN ('Child', 'VaccinationRecord', 'GrowthRecord', 'Report', 'User')),
    entity_id    TEXT,
    details      TEXT,
    timestamp    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Triggers: updated_at
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE set_updated_at();
CREATE TRIGGER children_updated_at
    BEFORE UPDATE ON children FOR EACH ROW EXECUTE PROCEDURE set_updated_at();
CREATE TRIGGER vaccination_records_updated_at
    BEFORE UPDATE ON vaccination_records FOR EACH ROW EXECUTE PROCEDURE set_updated_at();
CREATE TRIGGER vaccination_schedules_updated_at
    BEFORE UPDATE ON vaccination_schedules FOR EACH ROW EXECUTE PROCEDURE set_updated_at();
