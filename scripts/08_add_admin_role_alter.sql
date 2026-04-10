-- ALTER script to add 'admin' role to existing users table
-- Run this if the users table is already created without the 'admin' role

-- Step 1: Drop the existing constraint
ALTER TABLE users DROP CONSTRAINT users_role_check;

-- Step 2: Add the new constraint with 'admin' role included
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('parent', 'phm', 'moh', 'admin'));

-- Verification: Check the constraint was added
SELECT constraint_name, constraint_type
FROM information_schema.table_constraints
WHERE table_name = 'users' AND constraint_name LIKE '%role%';

