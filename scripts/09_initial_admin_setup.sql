-- Initial Admin User Setup Guide
--
-- Since admin accounts cannot be created via the public API,
-- you must create the initial admin user directly in the database.
--
-- IMPORTANT: This should only be done during initial system setup!

-- Step 1: Generate a bcrypt hash of your desired admin password
-- You can use an online bcrypt generator or the following command:
-- echo -n "YourAdminPassword123" | htpasswd -bnBC 10 "" | tr -d ':\n' | sed 's/\$2y/\$2a/'
--
-- Or use any bcrypt tool. The hash should start with $2a$ or $2b$
-- Example hash (password: "admin123"): $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcg7b3XeKeUxWdeS86E36DRcx76

-- Step 2: Execute this query with your bcrypt hash
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
    'user-admin-' || gen_random_uuid()::text,  -- Generate unique ID
    'admin@moh.lk',                              -- Change this to your admin email
    '000000000V',                                -- Change this to actual NIC if needed
    '$2a$10$Puf5Bp1O3xZ.41gmYdlGj.Qw5CDeU7fKIPZfX94WXSb3VTl69fbH.',             -- Replace with bcrypt hash of your password
    'admin',                                     -- Role is admin
    'System Administrator',                      -- Admin name
    '+94711234567',                              -- Admin phone
    'Colombo, Sri Lanka',                        -- Admin address
    'en',                                        -- Language preference
    false,                                       -- Not first login (admin doesn't need to change password)
    NOW(),
    NOW()
);

-- Step 3: Verify the admin user was created
SELECT id, email, role, name, created_at FROM users WHERE role = 'admin';

-- Step 4: Test the admin can log in
-- Use: POST /api/v1/auth/login
-- Body: {
--   "usernameOrEmail": "admin@moh.lk",
--   "password": "YourAdminPassword123"
-- }

-------------------------------------------------------------------
-- How to Generate Bcrypt Hash (Multiple Options)
-------------------------------------------------------------------

-- Option 1: Linux/Mac command line
-- echo -n "YourPassword123" | htpasswd -bnBC 10 "" | tr -d ':\n' | sed 's/\$2y/\$2a/'

-- Option 2: Online bcrypt generator
-- Visit: https://www.bcryptcalculator.com/
-- Input your password, copy the hash starting with $2a$ or $2b$

-- Option 3: Go bcrypt (in your terminal)
-- go install github.com/xlab/bcrypt@latest
-- bcrypt "YourPassword123"

-- Option 4: Using PostgreSQL pgcrypto extension
-- SELECT crypt('YourPassword123', gen_salt('bf', 10));
-- (Returns something like: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcg7b3XeKeUxWdeS86E36DRcx76)

-------------------------------------------------------------------
-- Important Security Notes
-------------------------------------------------------------------
-- 1. NEVER use this in production without changing the default values
-- 2. Use a STRONG password (min 12 characters, mixed case, numbers, symbols)
-- 3. Do NOT commit the actual bcrypt hash to version control
-- 4. Change the default email and NIC to actual values
-- 5. Only one admin user should exist in the system
-- 6. The database trigger enforce_single_admin will prevent creating multiple admins
-- 7. After creation, log in and change password immediately
-- 8. Use HTTPS in production to protect credentials in transit

-------------------------------------------------------------------
-- Alternative: Bulk Create Initial Data
-------------------------------------------------------------------
-- If you need to recreate or have multiple test admins during development:

-- DELETE FROM users WHERE role = 'admin'; -- Only if needed to reset

-- Then create:
INSERT INTO users (id, email, nic, password_hash, role, name, phone_number, address, language_preference, first_login, created_at, updated_at)
VALUES
    ('user-admin-001', 'admin@moh.lk', '000000000V', '$2a$10$Puf5Bp1O3xZ.41gmYdlGj.Qw5CDeU7fKIPZfX94WXSb3VTl69fbH.', 'admin', 'Admin User', '+94711234567', 'Colombo', 'en', false, NOW(), NOW());

-------------------------------------------------------------------
-- Verify Setup
-------------------------------------------------------------------
-- After creating admin, verify the system is ready:

-- 1. Check admin user exists
SELECT id, email, role FROM users WHERE role = 'admin';

-- 2. Check moh_account_otps table exists
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'public' AND table_name = 'moh_account_otps';

-- 3. Verify the trigger exists
SELECT trigger_name FROM information_schema.triggers
WHERE trigger_schema = 'public' AND trigger_name = 'enforce_single_admin';

-- 4. Test creating another admin (should FAIL - expected behavior!)
-- This demonstrates the single admin constraint:
-- BEGIN;
-- INSERT INTO users (id, email, nic, password_hash, role, name, phone_number, address, language_preference, first_login)
-- VALUES ('user-admin-test', 'test-admin@moh.lk', '111111111V', 'hash', 'admin', 'Test', '+94712345678', 'Test', 'en', false);
-- -- You should get error: "Only one admin user is allowed in the system"
-- ROLLBACK;

-------------------------------------------------------------------
-- First Login for Admin (Optional if first_login = true)
-------------------------------------------------------------------
-- If you set first_login = true, admin must change password:
-- POST /api/v1/auth/change-password
-- Body: {
--   "oldPassword": "initial-password",
--   "newPassword": "new-secure-password",
--   "confirmPassword": "new-secure-password"
-- }

-- For new admin setup, recommended: set first_login = false (as in main example above)
-- This allows admin to start immediately without forced password change.

