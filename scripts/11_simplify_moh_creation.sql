-- ALTER Script: Simplify MOH Account Creation with Temporary Password
--
-- This script replaces the OTP-based MOH account creation with a direct
-- temporary password approach. Instead of:
--   1. Request OTP
--   2. Wait for OTP
--   3. Complete account with OTP
--
-- We now:
--   1. Create account immediately with temporary password
--   2. Send temporary password via WhatsApp
--   3. User logs in with temp password, must change it on first login

BEGIN;

-- Step 1: Create temporary password tracking table
CREATE TABLE IF NOT EXISTS moh_account_temp_passwords (
    id              TEXT PRIMARY KEY,
    employee_id     TEXT NOT NULL,
    email           TEXT NOT NULL,
    nic             TEXT NOT NULL,
    name            TEXT NOT NULL,
    phone_number    TEXT NOT NULL,
    assigned_area   TEXT NOT NULL,
    admin_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    temp_password   TEXT NOT NULL,
    used_at         TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_moh_temp_passwords_email ON moh_account_temp_passwords(email);
CREATE INDEX IF NOT EXISTS idx_moh_temp_passwords_admin_id ON moh_account_temp_passwords(admin_id);
CREATE INDEX IF NOT EXISTS idx_moh_temp_passwords_expires_at ON moh_account_temp_passwords(expires_at);

-- Step 2: Archive old OTP table (don't delete, keep for reference)
-- This keeps the old structure intact but marks it as deprecated
ALTER TABLE moh_account_otps RENAME TO moh_account_otps_deprecated;

-- Add a comment indicating it's deprecated
COMMENT ON TABLE moh_account_otps_deprecated IS 'DEPRECATED: Old OTP-based MOH account creation. Use moh_account_temp_passwords instead.';

-- Step 3: Create indexes on new table
CREATE INDEX IF NOT EXISTS idx_moh_temp_passwords_created_at ON moh_account_temp_passwords(created_at);

COMMIT;

-- IMPORTANT NOTES:
-- ================================================================
-- 1. The old moh_account_otps table is renamed to moh_account_otps_deprecated
--    and kept for historical reference only
--
-- 2. The new workflow is:
--    - Admin calls: POST /api/v1/admin/moh-accounts/create
--    - System generates temporary password immediately
--    - System creates MOH user with password hash
--    - System sends temp password via WhatsApp
--    - System returns user ID and masked phone number
--
-- 3. MOH user logs in with email/NIC + temp password
--
-- 4. MOH user MUST change password on first login
--    - POST /api/v1/auth/change-password
--    - firstLogin flag is set to false
--    - Account fully activated
--
-- 4. If you want to completely remove old table later:
--    DROP TABLE moh_account_otps_deprecated;
-- ================================================================

