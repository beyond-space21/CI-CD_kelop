-- Migration: Create System Counters Table
-- This migration creates a dedicated counter table for efficient admin statistics
-- Counters are automatically maintained by database triggers

-- ============================================================================
-- COUNTERS TABLE
-- ============================================================================

-- Create counters table with a single row (singleton pattern)
CREATE TABLE IF NOT EXISTS system_counters (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1), -- Ensures only one row
    users_count INTEGER DEFAULT 0 NOT NULL,
    videos_count INTEGER DEFAULT 0 NOT NULL,
    comments_count INTEGER DEFAULT 0 NOT NULL,
    replies_count INTEGER DEFAULT 0 NOT NULL,
    upvotes_count INTEGER DEFAULT 0 NOT NULL,
    downvotes_count INTEGER DEFAULT 0 NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Initialize with zero counts if table is empty
INSERT INTO system_counters (id, users_count, videos_count, comments_count, replies_count, upvotes_count, downvotes_count)
VALUES (1, 0, 0, 0, 0, 0, 0)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- TRIGGER FUNCTIONS
-- ============================================================================

-- Function to increment/decrement users counter
CREATE OR REPLACE FUNCTION update_users_counter()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE system_counters SET users_count = users_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE system_counters SET users_count = users_count - 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to increment/decrement videos counter
CREATE OR REPLACE FUNCTION update_videos_counter()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE system_counters SET videos_count = videos_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE system_counters SET videos_count = videos_count - 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to increment/decrement comments counter
CREATE OR REPLACE FUNCTION update_comments_counter()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE system_counters SET comments_count = comments_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE system_counters SET comments_count = comments_count - 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to increment/decrement replies counter
CREATE OR REPLACE FUNCTION update_replies_counter()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE system_counters SET replies_count = replies_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE system_counters SET replies_count = replies_count - 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to increment/decrement upvotes counter
CREATE OR REPLACE FUNCTION update_upvotes_counter()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE system_counters SET upvotes_count = upvotes_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE system_counters SET upvotes_count = upvotes_count - 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to increment/decrement downvotes counter
CREATE OR REPLACE FUNCTION update_downvotes_counter()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE system_counters SET downvotes_count = downvotes_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE system_counters SET downvotes_count = downvotes_count - 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Drop existing triggers if they exist (for idempotency)
DROP TRIGGER IF EXISTS trigger_users_counter ON users;
DROP TRIGGER IF EXISTS trigger_videos_counter ON videos;
DROP TRIGGER IF EXISTS trigger_comments_counter ON comments;
DROP TRIGGER IF EXISTS trigger_replies_counter ON replies;
DROP TRIGGER IF EXISTS trigger_upvotes_counter ON upvotes;
DROP TRIGGER IF EXISTS trigger_downvotes_counter ON downvotes;

-- Create triggers for users table
CREATE TRIGGER trigger_users_counter
    AFTER INSERT OR DELETE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_users_counter();

-- Create triggers for videos table
CREATE TRIGGER trigger_videos_counter
    AFTER INSERT OR DELETE ON videos
    FOR EACH ROW
    EXECUTE FUNCTION update_videos_counter();

-- Create triggers for comments table
CREATE TRIGGER trigger_comments_counter
    AFTER INSERT OR DELETE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_comments_counter();

-- Create triggers for replies table
CREATE TRIGGER trigger_replies_counter
    AFTER INSERT OR DELETE ON replies
    FOR EACH ROW
    EXECUTE FUNCTION update_replies_counter();

-- Create triggers for upvotes table
CREATE TRIGGER trigger_upvotes_counter
    AFTER INSERT OR DELETE ON upvotes
    FOR EACH ROW
    EXECUTE FUNCTION update_upvotes_counter();

-- Create triggers for downvotes table
CREATE TRIGGER trigger_downvotes_counter
    AFTER INSERT OR DELETE ON downvotes
    FOR EACH ROW
    EXECUTE FUNCTION update_downvotes_counter();

-- ============================================================================
-- INITIAL SYNC
-- ============================================================================

-- Sync counters with actual table counts (for existing data)
-- This ensures counters are accurate even if triggers weren't active during initial data load
-- First ensure the row exists (INSERT already did this, but this is a safety check)
INSERT INTO system_counters (id, users_count, videos_count, comments_count, replies_count, upvotes_count, downvotes_count)
VALUES (1, 0, 0, 0, 0, 0, 0)
ON CONFLICT (id) DO NOTHING;

-- Now sync with actual counts
UPDATE system_counters SET
    users_count = (SELECT COUNT(*) FROM users),
    videos_count = (SELECT COUNT(*) FROM videos),
    comments_count = (SELECT COUNT(*) FROM comments),
    replies_count = (SELECT COUNT(*) FROM replies),
    upvotes_count = (SELECT COUNT(*) FROM upvotes),
    downvotes_count = (SELECT COUNT(*) FROM downvotes),
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- ============================================================================
-- NOTES
-- ============================================================================
-- The system_counters table uses a singleton pattern (only one row with id=1)
-- All counters are automatically maintained by database triggers
-- Triggers fire AFTER INSERT/DELETE to ensure data consistency
-- Initial sync ensures counters match existing data
-- Counters are updated atomically within the same transaction as the data change
-- This provides 100% accurate, instant counter reads

