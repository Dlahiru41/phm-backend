-- Migration: Parent-child linking via WhatsApp OTP (schema)
-- Safe to run multiple times.

ALTER TABLE children
    ADD COLUMN IF NOT EXISTS parent_whatsapp_number TEXT;

CREATE TABLE IF NOT EXISTS child_link_otps (
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

