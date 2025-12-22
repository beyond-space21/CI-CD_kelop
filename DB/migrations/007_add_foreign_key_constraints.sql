-- Migration: Add Foreign Key Constraints for Strong Schema Relationships
-- This migration adds referential integrity constraints to all tables
-- Uses DO blocks to check if constraints exist before adding them (idempotent)

-- ============================================================================
-- VIDEOS TABLES
-- ============================================================================

-- Add foreign key constraint for videos.user_uid
-- ON DELETE CASCADE: When a user is deleted, their videos are also deleted
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_videos_user_uid'
    ) THEN
        ALTER TABLE videos
        ADD CONSTRAINT fk_videos_user_uid
        FOREIGN KEY (user_uid) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- Add foreign key constraint for video_on_upload.user_uid
-- ON DELETE CASCADE: When a user is deleted, their incomplete uploads are deleted
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_video_on_upload_user_uid'
    ) THEN
        ALTER TABLE video_on_upload
        ADD CONSTRAINT fk_video_on_upload_user_uid
        FOREIGN KEY (user_uid) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- ============================================================================
-- SOCIAL TABLES
-- ============================================================================

-- Add foreign key constraints for followers table
-- ON DELETE CASCADE: When a user is deleted, all their follow relationships are deleted
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_followers_followed_by'
    ) THEN
        ALTER TABLE followers
        ADD CONSTRAINT fk_followers_followed_by
        FOREIGN KEY (followed_by) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_followers_followed_to'
    ) THEN
        ALTER TABLE followers
        ADD CONSTRAINT fk_followers_followed_to
        FOREIGN KEY (followed_to) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- Add foreign key constraints for blocklists table
-- ON DELETE CASCADE: When a user is deleted, all their block relationships are deleted
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_blocklists_blocked_by'
    ) THEN
        ALTER TABLE blocklists
        ADD CONSTRAINT fk_blocklists_blocked_by
        FOREIGN KEY (blocked_by) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_blocklists_blocked_to'
    ) THEN
        ALTER TABLE blocklists
        ADD CONSTRAINT fk_blocklists_blocked_to
        FOREIGN KEY (blocked_to) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- ============================================================================
-- ENGAGEMENT TABLES
-- ============================================================================

-- Add foreign key constraints for upvotes table
-- ON DELETE CASCADE: When a user is deleted, their upvotes are deleted
-- ON DELETE CASCADE: When a video is deleted, its upvotes are deleted
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_upvotes_upvoted_by'
    ) THEN
        ALTER TABLE upvotes
        ADD CONSTRAINT fk_upvotes_upvoted_by
        FOREIGN KEY (upvoted_by) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_upvotes_upvoted_to'
    ) THEN
        ALTER TABLE upvotes
        ADD CONSTRAINT fk_upvotes_upvoted_to
        FOREIGN KEY (upvoted_to) REFERENCES videos(video_id)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- Add foreign key constraints for downvotes table
-- ON DELETE CASCADE: When a user is deleted, their downvotes are deleted
-- ON DELETE CASCADE: When a video is deleted, its downvotes are deleted
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_downvotes_downvoted_by'
    ) THEN
        ALTER TABLE downvotes
        ADD CONSTRAINT fk_downvotes_downvoted_by
        FOREIGN KEY (downvoted_by) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_downvotes_downvoted_to'
    ) THEN
        ALTER TABLE downvotes
        ADD CONSTRAINT fk_downvotes_downvoted_to
        FOREIGN KEY (downvoted_to) REFERENCES videos(video_id)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- Add foreign key constraints for comments table
-- ON DELETE CASCADE: When a user is deleted, their comments are deleted
-- ON DELETE CASCADE: When a video is deleted, its comments are deleted
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_comments_commented_by'
    ) THEN
        ALTER TABLE comments
        ADD CONSTRAINT fk_comments_commented_by
        FOREIGN KEY (commented_by) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_comments_commented_to'
    ) THEN
        ALTER TABLE comments
        ADD CONSTRAINT fk_comments_commented_to
        FOREIGN KEY (commented_to) REFERENCES videos(video_id)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- Add foreign key constraints for replies table
-- ON DELETE CASCADE: When a user is deleted, their replies are deleted
-- ON DELETE CASCADE: When a comment is deleted, its replies are deleted
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_replies_replied_by'
    ) THEN
        ALTER TABLE replies
        ADD CONSTRAINT fk_replies_replied_by
        FOREIGN KEY (replied_by) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_replies_replied_to'
    ) THEN
        ALTER TABLE replies
        ADD CONSTRAINT fk_replies_replied_to
        FOREIGN KEY (replied_to) REFERENCES comments(comment_id)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- Add foreign key constraints for views table
-- ON DELETE CASCADE: When a user is deleted, their view records are deleted
-- ON DELETE CASCADE: When a video is deleted, its view records are deleted
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_views_viewed_by'
    ) THEN
        ALTER TABLE views
        ADD CONSTRAINT fk_views_viewed_by
        FOREIGN KEY (viewed_by) REFERENCES users(uid)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_views_viewed_to'
    ) THEN
        ALTER TABLE views
        ADD CONSTRAINT fk_views_viewed_to
        FOREIGN KEY (viewed_to) REFERENCES videos(video_id)
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- ============================================================================
-- NOTES
-- ============================================================================
-- All foreign keys use ON DELETE CASCADE to ensure data consistency:
-- - When a user is deleted, all their related data is automatically deleted
-- - When a video is deleted, all its engagement data is automatically deleted
-- - When a comment is deleted, all its replies are automatically deleted
--
-- ON UPDATE CASCADE ensures that if a user's UID changes (unlikely but possible),
-- all references are automatically updated.
--
-- This provides strong referential integrity at the database level.

