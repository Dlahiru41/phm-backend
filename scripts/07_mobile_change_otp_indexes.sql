-- Migration: Mobile number change OTP flow (indexes)
-- Safe to run multiple times.

CREATE INDEX IF NOT EXISTS idx_user_mobile_change_otps_user
    ON user_mobile_change_otps(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_mobile_change_otps_active
    ON user_mobile_change_otps(user_id, new_phone_number, expires_at)
    WHERE consumed_at IS NULL;

