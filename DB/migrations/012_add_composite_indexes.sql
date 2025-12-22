-- Migration: Add Composite Indexes for Query Optimization
-- This migration adds composite indexes to optimize common query patterns
-- Note: Indexes on (col1, col2) are NOT added if UNIQUE(col1, col2) already exists
-- because UNIQUE constraints automatically create indexes

-- ============================================================================
-- FOLLOWERS TABLE
-- ============================================================================

-- Index for ordering followers by timestamp (used in ListFollowers)
-- UNIQUE(followed_by, followed_to) already creates index on (followed_by, followed_to)
-- So we only add the ordering index
CREATE INDEX IF NOT EXISTS idx_followers_followed_to_at 
    ON followers(followed_to, followed_at DESC);

-- Index for ordering following list by timestamp (used in ListFollowing)
CREATE INDEX IF NOT EXISTS idx_followers_followed_by_at 
    ON followers(followed_by, followed_at DESC);

-- ============================================================================
-- COMMENTS TABLE
-- ============================================================================

-- Composite index for listing comments on a video, ordered by timestamp
CREATE INDEX IF NOT EXISTS idx_comments_commented_to_at 
    ON comments(commented_to, commented_at DESC);

-- Index for general ordering by timestamp
CREATE INDEX IF NOT EXISTS idx_comments_commented_at 
    ON comments(commented_at DESC);

-- ============================================================================
-- REPLIES TABLE
-- ============================================================================

-- Composite index for listing replies to a comment, ordered by timestamp
CREATE INDEX IF NOT EXISTS idx_replies_replied_to_at 
    ON replies(replied_to, replied_at DESC);

-- Index for general ordering by timestamp
CREATE INDEX IF NOT EXISTS idx_replies_replied_at 
    ON replies(replied_at DESC);

-- ============================================================================
-- NOTES
-- ============================================================================
-- These indexes optimize:
-- 1. ORDER BY timestamp queries (newest first)
-- 2. Filtering by foreign key + ordering by timestamp
-- 
-- Redundant indexes NOT created (already exist via UNIQUE constraints):
-- - followers(followed_by, followed_to) - covered by UNIQUE constraint
-- - upvotes(upvoted_by, upvoted_to) - covered by UNIQUE constraint
-- - downvotes(downvoted_by, downvoted_to) - covered by UNIQUE constraint
-- - views(viewed_by, viewed_to) - covered by UNIQUE constraint
--
-- All indexes use IF NOT EXISTS for idempotency
-- Indexes are safe to add - they don't affect data integrity or constraints

