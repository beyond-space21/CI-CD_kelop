-- Migration: Add index on videos.created_at for efficient ordering
-- This migration adds an index to optimize queries that order videos by creation date
-- Particularly useful for the /videos/list/following endpoint

-- Add index on created_at for efficient ORDER BY created_at DESC queries
CREATE INDEX IF NOT EXISTS idx_videos_created_at ON videos(created_at DESC);

-- ============================================================================
-- NOTES
-- ============================================================================
-- This index optimizes queries that order videos by creation timestamp
-- The DESC order in the index matches the common query pattern (newest first)
-- Improves performance for:
--   - /videos/list/following (videos from followed users, newest first)
--   - /videos/list/{username} (user's videos, newest first)
--   - Any other queries ordering by created_at DESC

