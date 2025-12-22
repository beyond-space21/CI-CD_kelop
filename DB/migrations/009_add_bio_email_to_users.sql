-- Migration: Add bio and email fields to users table
-- This migration adds bio (TEXT) and email (VARCHAR) columns to the users table

-- Add bio column (optional, can be NULL)
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS bio TEXT;

-- Add email column (optional, can be NULL, with unique constraint for email uniqueness)
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS email VARCHAR(255);

-- Create index on email for faster lookups (if email is used for authentication/search)
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;

-- Add unique constraint on email (only for non-null emails)
-- Note: PostgreSQL allows multiple NULL values in a UNIQUE column
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'users_email_unique'
    ) THEN
        ALTER TABLE users
        ADD CONSTRAINT users_email_unique UNIQUE (email);
    END IF;
END $$;

-- Also add bio and email to deleted_users table for archival purposes
ALTER TABLE deleted_users 
ADD COLUMN IF NOT EXISTS bio TEXT;

ALTER TABLE deleted_users 
ADD COLUMN IF NOT EXISTS email VARCHAR(255);

-- ============================================================================
-- NOTES
-- ============================================================================
-- bio: Optional text field for user biography/description
-- email: Optional email address with unique constraint (allows NULL values)
-- Both fields are nullable to support existing users
-- Email uniqueness is enforced at the database level
-- Index on email improves query performance for email-based lookups

