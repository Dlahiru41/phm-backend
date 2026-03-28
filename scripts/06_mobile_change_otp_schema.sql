-- Migration: Mobile number change OTP flow (schema)
-- Safe to run multiple times.

CREATE TABLE IF NOT EXISTS user_mobile_change_otps (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    new_phone_number TEXT NOT NULL,
    otp_hash         TEXT NOT NULL,
    attempt_count    INT NOT NULL DEFAULT 0,
    max_attempts     INT NOT NULL DEFAULT 5,
    expires_at       TIMESTAMPTZ NOT NULL,
    consumed_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

