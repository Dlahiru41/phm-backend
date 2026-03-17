-- Migration: Add PHM Account Authorization fields
-- Description: Adds support for PHM account creation with first-login tracking
-- Created: 2026-03-17
-- Status: Pending

-- Add new columns to users table for PHM support
ALTER TABLE users ADD COLUMN IF NOT EXISTS employee_id TEXT UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS assigned_area TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS first_login BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_by_moh TEXT REFERENCES users(id) ON DELETE SET NULL;

-- Create index for employee_id lookup (unique constraint with NULL handling)
-- PostgreSQL allows multiple NULLs in unique indexes by default
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_employee_id ON users(employee_id)
  WHERE employee_id IS NOT NULL;

-- Create index for created_by_moh foreign key
CREATE INDEX IF NOT EXISTS idx_users_created_by_moh ON users(created_by_moh);

-- Create index for first_login to optimize queries filtering first-login users
CREATE INDEX IF NOT EXISTS idx_users_first_login ON users(first_login)
  WHERE first_login = true;

-- Rollback script (if needed):
-- ALTER TABLE users DROP COLUMN IF EXISTS created_by_moh;
-- ALTER TABLE users DROP COLUMN IF EXISTS first_login;
-- ALTER TABLE users DROP COLUMN IF EXISTS assigned_area;
-- ALTER TABLE users DROP COLUMN IF EXISTS employee_id;
-- DROP INDEX IF EXISTS idx_users_employee_id;
-- DROP INDEX IF EXISTS idx_users_created_by_moh;
-- DROP INDEX IF EXISTS idx_users_first_login;

