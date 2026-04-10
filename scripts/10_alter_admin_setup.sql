-- ALTER Script: Update Existing Admin User Setup
--
-- IMPORTANT: This script is used when admin user already exists.
-- It updates the existing admin user instead of trying to insert a new one.
-- This avoids the "Only one admin user is allowed in the system" error.

-- Step 1: Check if admin user already exists
-- If it does, we'll update it. If not, we'll create it.

-- BEGIN TRANSACTION for safety
BEGIN;

-- Step 2: Try to update existing admin, or create if doesn't exist
-- Using PostgreSQL INSERT ... ON CONFLICT to handle both cases gracefully

DO $$
DECLARE
    admin_count INT;
    admin_id TEXT;
BEGIN
    -- Check if admin exists
    SELECT COUNT(*) INTO admin_count FROM users WHERE role = 'admin';

    IF admin_count > 0 THEN
        -- Admin exists, update it
        RAISE NOTICE 'Admin user already exists. Updating...';
        UPDATE users
        SET
            email = 'admin@moh.lk',
            password_hash = '$2a$10$Puf5Bp1O3xZ.41gmYdlGj.Qw5CDeU7fKIPZfX94WXSb3VTl69fbH.',
            name = 'System Administrator',
            phone_number = '+94711234567',
            address = 'Colombo, Sri Lanka',
            language_preference = 'en',
            first_login = false,
            updated_at = NOW()
        WHERE role = 'admin';

        RAISE NOTICE 'Admin user updated successfully';
    ELSE
        -- Admin doesn't exist, create it
        RAISE NOTICE 'Creating new admin user...';
        INSERT INTO users (
            id,
            email,
            nic,
            password_hash,
            role,
            name,
            phone_number,
            address,
            language_preference,
            first_login,
            created_at,
            updated_at
        ) VALUES (
            'user-admin-' || gen_random_uuid()::text,
            'admin@moh.lk',
            '000000000V',
            '$2a$10$Puf5Bp1O3xZ.41gmYdlGj.Qw5CDeU7fKIPZfX94WXSb3VTl69fbH.',
            'admin',
            'System Administrator',
            '+94711234567',
            'Colombo, Sri Lanka',
            'en',
            false,
            NOW(),
            NOW()
        );

        RAISE NOTICE 'Admin user created successfully';
    END IF;
END $$;

-- COMMIT TRANSACTION
COMMIT;

-- Step 3: Verify the admin user
RAISE NOTICE 'Verifying admin user...';
SELECT id, email, role, name, phone_number, created_at FROM users WHERE role = 'admin';

-- IMPORTANT NOTES:
-- ================================================================
-- 1. Change the default admin email if needed: 'admin@moh.lk'
-- 2. Change the default NIC if needed: '000000000V'
-- 3. Replace the password hash with your own secure password hash
--    To generate a bcrypt hash:
--    - Online: https://www.bcryptcalculator.com/
--    - Or use: SELECT crypt('YourPassword123', gen_salt('bf', 10));
--
-- 4. Security Best Practices:
--    - Use a STRONG password (min 12 characters, mixed case, numbers, symbols)
--    - Change password immediately after first login
--    - Do NOT commit the actual bcrypt hash to version control
--    - Use HTTPS in production
--
-- 5. Testing:
--    POST /api/v1/auth/login
--    Body: {
--      "usernameOrEmail": "admin@moh.lk",
--      "password": "YourAdminPassword123"
--    }
-- ================================================================

