-- Add password column to users table for custom JWT authentication
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;

-- Add index for faster lookups (if needed for email-based login in future)
-- For now, we'll use username for login

